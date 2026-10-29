"""shopgen: synthetic e-commerce traffic for the tributary stack.

Writes orders, payments, shipments and refunds to Postgres (captured by Debezium)
and streams clickstream events straight to Redpanda as Avro. Backfills history
once, then runs live. Incidents can be injected over HTTP:

    curl -X POST 'localhost:8000/incident/payment_outage?minutes=5'
"""

import json
import logging
import os
import random
import threading
from dataclasses import dataclass
from datetime import timedelta
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse

from prometheus_client import CONTENT_TYPE_LATEST, Counter, Gauge, generate_latest

import sim
from sinks import Clickstream, Postgres

log = logging.getLogger("shopgen")


@dataclass
class Config:
    seed: int = int(os.getenv("SEED", "42"))
    products: int = int(os.getenv("PRODUCTS", "240"))
    backfill_days: int = int(os.getenv("BACKFILL_DAYS", "30"))
    sessions_per_day: int = int(os.getenv("SESSIONS_PER_DAY", "2500"))
    live_sessions_per_min: float = float(os.getenv("LIVE_SESSIONS_PER_MIN", "150"))
    # live mode compresses time: clicks happen TIME_SCALE x faster, and a day of
    # back-office work (picking, shipping, refunds) takes LIVE_SECONDS_PER_DAY seconds
    time_scale: float = float(os.getenv("TIME_SCALE", "3"))
    live_seconds_per_day: float = float(os.getenv("LIVE_SECONDS_PER_DAY", "60"))
    auto_incidents: bool = os.getenv("AUTO_INCIDENTS", "false") == "true"
    kafka: str = os.getenv("KAFKA_BOOTSTRAP", "redpanda-0.redpanda.kafka.svc.cluster.local:9093")
    registry: str = os.getenv("SCHEMA_REGISTRY_URL", "http://redpanda-0.redpanda.kafka.svc.cluster.local:8081")
    clickstream_topic: str = os.getenv("CLICKSTREAM_TOPIC", "shop.clickstream")
    alerts_topic: str = os.getenv("ALERTS_TOPIC", "alerts.anomalies")
    port: int = int(os.getenv("PORT", "8000"))


class Metrics:
    events = Counter("shopgen_events_total", "Clickstream events produced", ["event_type"])
    sessions = Counter("shopgen_sessions_total", "Shopper sessions started")
    orders = Counter("shopgen_order_transitions_total", "Order status transitions", ["status"])
    payments = Counter("shopgen_payments_total", "Payment attempts", ["provider", "status"])
    produce_errors = Counter("shopgen_produce_errors_total", "Failed Kafka deliveries")
    db_ops = Gauge("shopgen_db_statements", "Statements written to Postgres")
    backfill_progress = Gauge("shopgen_backfill_progress", "Backfill progress (0-1)")
    mode = Gauge("shopgen_live", "1 when running live, 0 while backfilling")
    incident = Gauge("shopgen_incident_active", "Injected incident in progress", ["incident"])


def control_server(shop, port):
    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *_):
            pass

        def _send(self, code, body, ctype="application/json"):
            data = body if isinstance(body, bytes) else json.dumps(body, indent=2).encode()
            self.send_response(code)
            self.send_header("Content-Type", ctype)
            self.end_headers()
            self.wfile.write(data)

        def do_GET(self):
            path = urlparse(self.path).path
            if path == "/metrics":
                self._send(200, generate_latest(), CONTENT_TYPE_LATEST)
            elif path == "/healthz":
                self._send(200, {"ok": True})
            elif path in ("/", "/status"):
                now = sim.utcnow()
                self._send(200, {
                    "mode": "live" if shop.live else "backfill",
                    "backfill_progress": round(shop.backfill_progress, 3),
                    "customers": len(shop.customers), "in_flight": len(shop.heap),
                    "postgres_statements": shop.db.ops,
                    "incidents": {k: round((v - now).total_seconds()) for k, v in shop.incidents.items() if v > now},
                    "inject": [f"POST /incident/{n}?minutes=5" for n in sim.INCIDENTS] + ["POST /incident/clear"],
                })
            else:
                self._send(404, {"error": "not found"})

        def do_POST(self):
            url = urlparse(self.path)
            name = url.path.rsplit("/", 1)[-1]
            if url.path == "/incident/clear":
                shop.incidents.clear()
                return self._send(200, {"cleared": True})
            if not url.path.startswith("/incident/") or name not in sim.INCIDENTS:
                return self._send(404, {"error": f"unknown incident; choose from {list(sim.INCIDENTS)}"})
            minutes = float(parse_qs(url.query).get("minutes", ["5"])[0])
            start_incident(shop, name, minutes)
            self._send(200, {"incident": name, "minutes": minutes})

    server = ThreadingHTTPServer(("0.0.0.0", port), Handler)
    threading.Thread(target=server.serve_forever, daemon=True).start()


def start_incident(shop, name, minutes):
    shop.incidents[name] = sim.utcnow() + timedelta(minutes=minutes)
    if name == "fraud_burst":
        shop.pending_fraud += 3
    log.warning("incident started: %s for %.1f minutes", name, minutes)


def main():
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    cfg = Config()
    m = Metrics()
    dsn = (f"host={os.getenv('POSTGRES_HOST', 'postgres-db')} port={os.getenv('POSTGRES_PORT', '5432')} "
           f"dbname={os.getenv('POSTGRES_DBNAME', 'postgres')} user={os.getenv('POSTGRES_USER', 'postgres')} "
           f"password={os.environ['POSTGRES_PASSWORD']}")
    db = Postgres(dsn)
    db.migrate()
    stream = Clickstream(cfg.kafka, cfg.registry, cfg.clickstream_topic, [cfg.alerts_topic], m)
    shop = sim.Shop(db, stream, m, cfg)
    control_server(shop, cfg.port)
    shop.load()

    def flush():
        db.flush()
        stream.poll()
        m.db_ops.set(db.ops)

    if db.get_state("backfill_done") is None:
        shop.backfill(cfg.backfill_days, sim.utcnow(), on_batch=flush)
        flush()
        stream.flush()
        db.set_state("backfill_done", sim.utcnow().isoformat())
        log.info("backfill complete: %d customers, %d statements", len(shop.customers), db.ops)
    else:
        log.info("history already backfilled; going straight to live")

    rng = random.Random()
    next_auto = [sim.utcnow() + timedelta(minutes=rng.uniform(10, 20))]

    def tick():
        flush()
        now = sim.utcnow()
        for name in sim.INCIDENTS:
            m.incident.labels(name).set(1 if shop.incident(name) else 0)
        if cfg.auto_incidents and now > next_auto[0]:
            start_incident(shop, rng.choice(sim.INCIDENTS), 4)
            next_auto[0] = now + timedelta(minutes=rng.uniform(15, 30))

    shop.run_live(tick)


if __name__ == "__main__":
    main()
