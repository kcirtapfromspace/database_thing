"""Write paths: Postgres (the OLTP database Debezium captures) and Redpanda (clickstream)."""

import json
import logging
import time

import psycopg
from confluent_kafka import KafkaException, Producer
from confluent_kafka.admin import AdminClient, NewTopic
from confluent_kafka.schema_registry import SchemaRegistryClient
from confluent_kafka.schema_registry.avro import AvroSerializer
from confluent_kafka.serialization import MessageField, SerializationContext

log = logging.getLogger("shopgen")

DDL = """
CREATE SCHEMA IF NOT EXISTS shop;
CREATE SCHEMA IF NOT EXISTS shopgen;

CREATE TABLE IF NOT EXISTS shop.customers (
    customer_id         uuid PRIMARY KEY,
    email               text NOT NULL,
    full_name           text NOT NULL,
    country             text NOT NULL,
    city                text NOT NULL,
    segment             text NOT NULL,
    acquisition_channel text NOT NULL,
    marketing_opt_in    boolean NOT NULL,
    signup_at           timestamp NOT NULL,
    updated_at          timestamp NOT NULL
);
CREATE TABLE IF NOT EXISTS shop.products (
    product_id  integer PRIMARY KEY,
    sku         text NOT NULL,
    name        text NOT NULL,
    category    text NOT NULL,
    brand       text NOT NULL,
    price       numeric(10, 2) NOT NULL,
    unit_cost   numeric(10, 2) NOT NULL,
    is_active   boolean NOT NULL,
    updated_at  timestamp NOT NULL
);
CREATE TABLE IF NOT EXISTS shop.orders (
    order_id     uuid PRIMARY KEY,
    customer_id  uuid NOT NULL,
    session_id   text NOT NULL,
    status       text NOT NULL,
    currency     text NOT NULL,
    channel      text NOT NULL,
    device       text NOT NULL,
    ship_country text NOT NULL,
    item_count   integer NOT NULL,
    subtotal     numeric(12, 2) NOT NULL,
    discount     numeric(12, 2) NOT NULL,
    shipping_fee numeric(12, 2) NOT NULL,
    tax          numeric(12, 2) NOT NULL,
    total        numeric(12, 2) NOT NULL,
    promo_code   text,
    created_at   timestamp NOT NULL,
    updated_at   timestamp NOT NULL
);
CREATE TABLE IF NOT EXISTS shop.order_items (
    order_item_id bigserial PRIMARY KEY,
    order_id      uuid NOT NULL,
    product_id    integer NOT NULL,
    quantity      integer NOT NULL,
    unit_price    numeric(10, 2) NOT NULL,
    created_at    timestamp NOT NULL
);
CREATE TABLE IF NOT EXISTS shop.payments (
    payment_id     uuid PRIMARY KEY,
    order_id       uuid NOT NULL,
    customer_id    uuid NOT NULL,
    method         text NOT NULL,
    provider       text NOT NULL,
    status         text NOT NULL,
    decline_reason text,
    amount         numeric(12, 2) NOT NULL,
    currency       text NOT NULL,
    card_country   text NOT NULL,
    risk_score     double precision NOT NULL,
    attempted_at   timestamp NOT NULL,
    updated_at     timestamp NOT NULL
);
CREATE TABLE IF NOT EXISTS shop.shipments (
    shipment_id   uuid PRIMARY KEY,
    order_id      uuid NOT NULL,
    carrier       text NOT NULL,
    service_level text NOT NULL,
    status        text NOT NULL,
    shipped_at    timestamp,
    delivered_at  timestamp,
    created_at    timestamp NOT NULL,
    updated_at    timestamp NOT NULL
);
CREATE TABLE IF NOT EXISTS shop.refunds (
    refund_id  uuid PRIMARY KEY,
    order_id   uuid NOT NULL,
    payment_id uuid NOT NULL,
    amount     numeric(12, 2) NOT NULL,
    reason     text NOT NULL,
    created_at timestamp NOT NULL
);
-- full row images so CDC updates carry the previous state (status transitions, price changes)
ALTER TABLE shop.customers REPLICA IDENTITY FULL;
ALTER TABLE shop.products  REPLICA IDENTITY FULL;
ALTER TABLE shop.orders    REPLICA IDENTITY FULL;
ALTER TABLE shop.payments  REPLICA IDENTITY FULL;
ALTER TABLE shop.shipments REPLICA IDENTITY FULL;

CREATE TABLE IF NOT EXISTS shopgen.state (key text PRIMARY KEY, value text NOT NULL);
"""

CLICKSTREAM_SCHEMA = json.dumps({
    "type": "record", "name": "ClickstreamEvent", "namespace": "shop.events",
    "fields": [
        {"name": "event_id", "type": "string"},
        {"name": "event_type", "type": "string"},
        {"name": "event_time", "type": {"type": "long", "logicalType": "timestamp-millis"}},
        {"name": "session_id", "type": "string"},
        {"name": "customer_id", "type": ["null", "string"], "default": None},
        {"name": "anonymous_id", "type": "string"},
        {"name": "page", "type": "string"},
        {"name": "product_id", "type": ["null", "int"], "default": None},
        {"name": "category", "type": ["null", "string"], "default": None},
        {"name": "search_query", "type": ["null", "string"], "default": None},
        {"name": "order_id", "type": ["null", "string"], "default": None},
        {"name": "value", "type": ["null", "double"], "default": None},
        {"name": "device", "type": "string"},
        {"name": "country", "type": "string"},
        {"name": "utm_source", "type": "string"},
        {"name": "is_backfill", "type": "boolean", "default": False},
    ],
})


def retry(what, fn, attempts=60, delay=5):
    for i in range(attempts):
        try:
            return fn()
        except Exception as e:  # dependencies start in any order under Tilt
            if i == attempts - 1:
                raise
            log.info("waiting for %s (%s)", what, str(e).splitlines()[0][:120])
            time.sleep(delay)


class Postgres:
    """Buffers statements in order and flushes runs of identical SQL with executemany."""

    def __init__(self, dsn):
        self.conn = retry("postgres", lambda: psycopg.connect(dsn))
        self.buf = []
        self.ops = 0

    def migrate(self):
        with self.conn.cursor() as cur:
            cur.execute(DDL)
        self.conn.commit()

    def add(self, sql, params):
        self.buf.append((sql, params))

    def flush(self):
        if not self.buf:
            return
        with self.conn.cursor() as cur:
            run_sql, run = self.buf[0][0], []
            for sql, params in self.buf:
                if sql != run_sql:
                    cur.executemany(run_sql, run)
                    run_sql, run = sql, []
                run.append(params)
            cur.executemany(run_sql, run)
        self.conn.commit()
        self.ops += len(self.buf)
        self.buf.clear()

    def query(self, sql, params=()):
        with self.conn.cursor() as cur:
            cur.execute(sql, params)
            rows = cur.fetchall()
        self.conn.commit()
        return rows

    def get_state(self, key):
        rows = self.query("SELECT value FROM shopgen.state WHERE key = %s", (key,))
        return rows[0][0] if rows else None

    def set_state(self, key, value):
        with self.conn.cursor() as cur:
            cur.execute("INSERT INTO shopgen.state VALUES (%s, %s) "
                        "ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value", (key, value))
        self.conn.commit()


class Clickstream:
    def __init__(self, bootstrap, registry_url, topic, extra_topics, metrics):
        self.topic, self.m = topic, metrics
        admin = AdminClient({"bootstrap.servers": bootstrap})
        topics = [NewTopic(topic, num_partitions=3, replication_factor=1)]
        topics += [NewTopic(t, num_partitions=1, replication_factor=1) for t in extra_topics]
        retry("redpanda", lambda: admin.list_topics(timeout=10))
        for name, fut in admin.create_topics(topics).items():
            try:
                fut.result()
                log.info("created topic %s", name)
            except KafkaException as e:
                if "TOPIC_ALREADY_EXISTS" not in str(e):
                    raise
        registry = SchemaRegistryClient({"url": registry_url})
        self.serialize = AvroSerializer(registry, CLICKSTREAM_SCHEMA)
        self.ctx = SerializationContext(topic, MessageField.VALUE)
        self.producer = Producer({
            "bootstrap.servers": bootstrap, "linger.ms": 50, "compression.type": "lz4",
            "enable.idempotence": True, "queue.buffering.max.messages": 500000,
        })

    def _delivered(self, err, _msg):
        if err is not None:
            self.m.produce_errors.inc()

    def send(self, event):
        value = self.serialize(event, self.ctx)
        ts = int(event["event_time"].timestamp() * 1000)
        while True:
            try:
                self.producer.produce(self.topic, key=event["session_id"].encode(), value=value,
                                      timestamp=ts, on_delivery=self._delivered)
                break
            except BufferError:
                self.producer.poll(0.5)

    def poll(self):
        self.producer.poll(0)

    def flush(self):
        self.producer.flush(60)
