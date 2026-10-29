"""Event-driven shop simulation.

Sessions and order lifecycles are generator coroutines that yield waits; a heap
scheduler resumes them on a virtual clock. The same code backfills history (the
clock jumps from event to event) and then runs live (the clock is wall time,
with in-session waits and back-office waits compressed so a demo shows activity).
"""

import heapq
import itertools
import logging
import math
import random
import time
import uuid
from collections import deque
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone

import catalog
from catalog import weighted

log = logging.getLogger("shopgen")

HOUR, DAY = 3600, 86400
INCIDENTS = ("payment_outage", "fraud_burst", "traffic_spike")

CHANNEL_CONV = {c[0]: c[2] for c in catalog.CHANNELS}
DEVICE_CONV = {d[0]: d[2] for d in catalog.DEVICES}
CITIES = {c[0]: c[3] for c in catalog.COUNTRIES}
SEARCH_TERMS = {c[0]: [n.lower() for n in c[5]] for c in catalog.CATEGORIES}


def funnel(seconds):
    """In-session wait between clicks."""
    return ("funnel", seconds)


def ops(seconds):
    """Back-office wait: picking, shipping, refunds."""
    return ("ops", seconds)


def utcnow():
    return datetime.now(timezone.utc)


@dataclass
class Customer:
    customer_id: str
    email: str
    full_name: str
    country: str
    city: str
    segment: str
    opt_in: bool
    orders: int = 0
    spend: float = 0.0


@dataclass
class Session:
    session_id: str
    anonymous_id: str
    customer: Customer | None
    channel: str
    device: str
    country: str
    promo_code: str | None = None
    fraud: bool = False


INS_CUSTOMER = ("INSERT INTO shop.customers (customer_id, email, full_name, country, city, segment, "
                "acquisition_channel, marketing_opt_in, signup_at, updated_at) "
                "VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)")
INS_PRODUCT = ("INSERT INTO shop.products (product_id, sku, name, category, brand, price, unit_cost, "
               "is_active, updated_at) VALUES (%s, %s, %s, %s, %s, %s, %s, true, %s) "
               "ON CONFLICT (product_id) DO NOTHING")
INS_ORDER = ("INSERT INTO shop.orders (order_id, customer_id, session_id, status, currency, channel, device, "
             "ship_country, item_count, subtotal, discount, shipping_fee, tax, total, promo_code, "
             "created_at, updated_at) VALUES (%s, %s, %s, 'placed', 'USD', %s, %s, %s, %s, %s, %s, %s, %s, "
             "%s, %s, %s, %s)")
INS_ITEM = ("INSERT INTO shop.order_items (order_id, product_id, quantity, unit_price, created_at) "
            "VALUES (%s, %s, %s, %s, %s)")
INS_PAYMENT = ("INSERT INTO shop.payments (payment_id, order_id, customer_id, method, provider, status, "
               "decline_reason, amount, currency, card_country, risk_score, attempted_at, updated_at) "
               "VALUES (%s, %s, %s, %s, %s, %s, %s, %s, 'USD', %s, %s, %s, %s)")
INS_SHIPMENT = ("INSERT INTO shop.shipments (shipment_id, order_id, carrier, service_level, status, "
                "created_at, updated_at) VALUES (%s, %s, %s, %s, 'label_created', %s, %s)")
INS_REFUND = ("INSERT INTO shop.refunds (refund_id, order_id, payment_id, amount, reason, created_at) "
              "VALUES (%s, %s, %s, %s, %s, %s)")
UPD_ORDER = "UPDATE shop.orders SET status = %s, updated_at = %s WHERE order_id = %s"
UPD_PAYMENT = "UPDATE shop.payments SET status = %s, updated_at = %s WHERE payment_id = %s"
UPD_SHIP_TRANSIT = ("UPDATE shop.shipments SET status = 'in_transit', shipped_at = %s, updated_at = %s "
                    "WHERE shipment_id = %s")
UPD_SHIP_DELIVERED = ("UPDATE shop.shipments SET status = 'delivered', delivered_at = %s, updated_at = %s "
                      "WHERE shipment_id = %s")
UPD_SEGMENT = "UPDATE shop.customers SET segment = %s, updated_at = %s WHERE customer_id = %s"
UPD_PROFILE = ("UPDATE shop.customers SET city = %s, marketing_opt_in = %s, updated_at = %s "
               "WHERE customer_id = %s")
UPD_PRICE = "UPDATE shop.products SET price = %s, updated_at = %s WHERE product_id = %s"
DEL_CUSTOMER = "DELETE FROM shop.customers WHERE customer_id = %s"


class Shop:
    def __init__(self, db, stream, metrics, cfg):
        self.db, self.stream, self.m, self.cfg = db, stream, metrics, cfg
        self.rng = random.Random()
        self.products = catalog.build_products(random.Random(cfg.seed), cfg.products)
        self.weights = [p["popularity"] for p in self.products]
        self.by_category = {}
        for p in self.products:
            self.by_category.setdefault(p["category"], []).append(p)
        self.customers = {}
        self.customer_list = []
        self.recent_buyers = deque(maxlen=400)
        self.heap, self.seq = [], itertools.count()
        self.now = utcnow()
        self.live = False
        self.incidents = {}  # name -> expiry (wall clock)
        self.pending_fraud = 0
        self.backfill_progress = 0.0

    # ---- scheduler -------------------------------------------------------
    def at(self, when, coro):
        heapq.heappush(self.heap, (when, next(self.seq), coro))

    def spawn(self, coro):
        self.at(self.now, coro)

    def _delay(self, kind, seconds):
        if not self.live:
            return seconds
        if kind == "funnel":
            return seconds / self.cfg.time_scale
        return seconds * self.cfg.live_seconds_per_day / DAY

    def step(self, until, on_batch=None):
        n = 0
        while self.heap and self.heap[0][0] <= until:
            when, _, coro = heapq.heappop(self.heap)
            self.now = when
            try:
                kind, seconds = next(coro)
            except StopIteration:
                pass
            else:
                self.at(self.now + timedelta(seconds=self._delay(kind, seconds)), coro)
            n += 1
            if on_batch and n % 2000 == 0:
                on_batch()

    def ts(self):
        return self.now.replace(tzinfo=None)

    def incident(self, name):
        return self.live and self.incidents.get(name, utcnow()) > utcnow()

    # ---- reference data ----------------------------------------------------
    def load(self):
        for p in self.products:
            self.db.add(INS_PRODUCT, (p["product_id"], p["sku"], p["name"], p["category"], p["brand"],
                                      p["price"], p["unit_cost"], self.ts()))
        self.db.flush()
        prices = dict(self.db.query("SELECT product_id, price::float FROM shop.products"))
        for p in self.products:
            p["price"] = prices.get(p["product_id"], p["price"])
        rows = self.db.query(
            "SELECT c.customer_id::text, c.email, c.full_name, c.country, c.city, c.segment, "
            "c.marketing_opt_in, count(o.order_id), coalesce(sum(o.total), 0)::float "
            "FROM shop.customers c LEFT JOIN shop.orders o ON o.customer_id = c.customer_id "
            "AND o.status IN ('paid', 'shipped', 'delivered') GROUP BY 1, 2, 3, 4, 5, 6, 7")
        for r in rows:
            self._remember(Customer(*r))
        log.info("loaded %d products, %d customers", len(self.products), len(self.customers))

    def _remember(self, c):
        self.customers[c.customer_id] = c
        self.customer_list.append(c)

    # ---- helpers -------------------------------------------------------------
    def emit(self, s, event_type, product=None, page=None, value=None, order_id=None, search_query=None):
        self.stream.send({
            "event_id": str(uuid.uuid4()), "event_type": event_type, "event_time": self.now,
            "session_id": s.session_id, "customer_id": s.customer.customer_id if s.customer else None,
            "anonymous_id": s.anonymous_id,
            "page": page or (f"/p/{product['product_id']}" if product else "/"),
            "product_id": product["product_id"] if product else None,
            "category": product["category"] if product else None,
            "search_query": search_query, "order_id": order_id,
            "value": round(value, 2) if value is not None else None,
            "device": s.device, "country": s.country, "utm_source": s.channel,
            "is_backfill": not self.live,
        })
        self.m.events.labels(event_type).inc()

    def pick_product(self, category=None):
        if category and self.rng.random() < 0.7:
            pool = self.by_category[category]
            return self.rng.choices(pool, weights=[p["popularity"] for p in pool], k=1)[0]
        return self.rng.choices(self.products, weights=self.weights, k=1)[0]

    def pick_returning(self):
        rng = self.rng
        for _ in range(5):
            if self.recent_buyers and rng.random() < 0.55:
                c = rng.choice(self.recent_buyers)
            else:
                c = rng.choice(self.customer_list)
            if c.customer_id in self.customers:
                return c
        return None

    def new_session(self, promo=None, customer=None, fraud=False):
        rng = self.rng
        if customer is None and self.customer_list and rng.random() < 0.42:
            customer = self.pick_returning()
        if customer and rng.random() < 0.5:
            channel = "email" if customer.opt_in and rng.random() < 0.5 else "organic"
        else:
            channel = weighted(rng, catalog.CHANNELS)[0]
        country = customer.country if customer else weighted(rng, catalog.COUNTRIES)[0]
        return Session(uuid.uuid4().hex, uuid.uuid4().hex, customer, channel,
                       weighted(rng, catalog.DEVICES)[0], country, promo, fraud)

    def signup(self, s, fraud=False):
        rng = self.rng
        first, last = rng.choice(catalog.FIRST_NAMES), rng.choice(catalog.LAST_NAMES)
        domain = rng.choice(["mailbox.test", "tempmail.test"]) if fraud else "example.com"
        segment = "business" if rng.random() < 0.1 else "consumer"
        c = Customer(str(uuid.uuid4()), f"{first}.{last}.{rng.randint(1, 99999)}@{domain}".lower(),
                     f"{first} {last}", s.country, rng.choice(CITIES[s.country]), segment,
                     rng.random() < 0.6)
        self._remember(c)
        self.db.add(INS_CUSTOMER, (c.customer_id, c.email, c.full_name, c.country, c.city, c.segment,
                                   s.channel, c.opt_in, self.ts(), self.ts()))
        s.customer = c
        self.emit(s, "sign_up", page="/account/new")
        return c

    def set_status(self, order, status):
        order["status"] = status
        self.db.add(UPD_ORDER, (status, self.ts(), order["order_id"]))
        self.m.orders.labels(status).inc()

    # ---- shopper journey -------------------------------------------------------
    def session(self, promo=None):
        rng = self.rng
        s = self.new_session(promo)
        self.m.sessions.inc()
        self.emit(s, "session_start", page=rng.choice(["/", "/", "/deals", "/new-arrivals"]))
        yield funnel(rng.uniform(2, 15))
        if rng.random() < 0.12:
            return  # bounce
        if s.customer is None and rng.random() < 0.03:
            self.signup(s)  # newsletter / account signup without buying
        affinity = rng.choice(list(self.by_category)) if rng.random() < 0.5 else None
        viewed = []
        for _ in range(1 + min(int(rng.expovariate(0.45)), 9)):
            if rng.random() < 0.25:
                cat = affinity or rng.choice(list(self.by_category))
                self.emit(s, "search", page="/search", search_query=rng.choice(SEARCH_TERMS[cat]))
                yield funnel(rng.uniform(4, 20))
            p = self.pick_product(affinity)
            viewed.append(p)
            self.emit(s, "product_view", product=p)
            yield funnel(rng.uniform(10, 80))

        conv = CHANNEL_CONV[s.channel] * DEVICE_CONV[s.device]
        conv *= 1.25 if s.customer and s.customer.orders else 1.0
        conv *= 1.4 if s.promo_code else 1.0
        if rng.random() > 0.08 * conv:
            return
        cart = []
        for p in rng.sample(viewed, k=min(len(viewed), rng.choice([1, 1, 1, 2, 2, 3]))):
            qty = 1 if rng.random() < 0.85 else rng.randint(2, 3)
            cart.append((p, qty))
            self.emit(s, "add_to_cart", product=p, value=p["price"] * qty, page="/cart")
            yield funnel(rng.uniform(5, 30))
        if len(cart) > 1 and rng.random() < 0.1:
            p, _ = cart.pop()
            self.emit(s, "remove_from_cart", product=p, page="/cart")
            yield funnel(rng.uniform(3, 15))
        if rng.random() > min(0.9, 0.47 * conv ** 0.5):
            return  # cart abandonment
        self.emit(s, "checkout_start", page="/checkout", value=sum(p["price"] * q for p, q in cart))
        yield funnel(rng.uniform(25, 120))
        yield from self.checkout(s, cart)

    def checkout(self, s, cart):
        rng = self.rng
        if s.customer is None:
            self.signup(s, fraud=s.fraud)
        order = self.place_order(s, cart)
        paid = False
        for attempt in range(2):
            if self.attempt_payment(order, s):
                paid = True
                break
            if attempt == 0 and rng.random() < 0.45 and not s.fraud:
                self.emit(s, "payment_retry", page="/checkout", order_id=order["order_id"])
                yield funnel(rng.uniform(15, 60))
            else:
                break
        if not paid:
            self.set_status(order, "payment_failed")
            self.emit(s, "checkout_error", page="/checkout", order_id=order["order_id"])
            return
        self.set_status(order, "paid")
        self.emit(s, "purchase", page="/checkout/complete", order_id=order["order_id"], value=order["total"])
        s.customer.orders += 1
        self.recent_buyers.append(s.customer)
        self.spawn(self.fulfillment(order, s.customer))

    def place_order(self, s, cart):
        subtotal = round(sum(p["price"] * q for p, q in cart), 2)
        rate = {"FLASH25": 0.25}.get(s.promo_code, 0.2) if s.promo_code else 0.0
        discount = round(subtotal * rate, 2)
        shipping = 0.0 if subtotal - discount >= 50 else 5.99
        tax = round((subtotal - discount) * catalog.TAX_RATE[s.country], 2)
        order = {
            "order_id": str(uuid.uuid4()), "customer_id": s.customer.customer_id, "status": "placed",
            "total": round(subtotal - discount + shipping + tax, 2), "country": s.country,
            "category": max(cart, key=lambda c: c[0]["price"] * c[1])[0]["category"], "fraud": s.fraud,
        }
        self.db.add(INS_ORDER, (order["order_id"], s.customer.customer_id, s.session_id, s.channel, s.device,
                                s.country, sum(q for _, q in cart), subtotal, discount, shipping, tax,
                                order["total"], s.promo_code, self.ts(), self.ts()))
        for p, q in cart:
            self.db.add(INS_ITEM, (order["order_id"], p["product_id"], q, p["price"], self.ts()))
        self.m.orders.labels("placed").inc()
        return order

    def attempt_payment(self, order, s):
        rng = self.rng
        method, _, providers = weighted(rng, catalog.PAYMENT_METHODS)
        provider = weighted(rng, providers)[0]
        if s.fraud:
            method, provider = "card", rng.choice(["stripe", "adyen"])
            card_country = rng.choice([c[0] for c in catalog.COUNTRIES if c[0] != s.country])
            risk = round(rng.uniform(0.75, 0.99), 3)
        else:
            card_country = s.country if rng.random() < 0.97 else weighted(rng, catalog.COUNTRIES)[0]
            risk = round(min(0.99, rng.betavariate(1.2, 12) + (0.1 if card_country != s.country else 0)), 3)
        status, reason = "authorized", None
        if self.incident("payment_outage") and provider == "adyen" and rng.random() < 0.8:
            status, reason = "declined", "processor_unavailable"
        elif s.fraud and rng.random() < 0.65:
            status, reason = "declined", "fraud_suspected"
        elif not s.fraud and rng.random() < 0.04 + risk * 0.3:
            status, reason = "declined", weighted(rng, catalog.DECLINE_REASONS)[0]
        order["payment_id"] = str(uuid.uuid4())
        self.db.add(INS_PAYMENT, (order["payment_id"], order["order_id"], order["customer_id"], method, provider,
                                  status, reason, order["total"], card_country, risk, self.ts(), self.ts()))
        self.m.payments.labels(provider, status).inc()
        return status == "authorized"

    def fulfillment(self, order, customer):
        rng = self.rng
        yield ops(rng.uniform(2, 26) * HOUR)
        if rng.random() < 0.03:
            self.set_status(order, "cancelled")
            self.db.add(UPD_PAYMENT, ("voided", self.ts(), order["payment_id"]))
            return
        options = [c for c in catalog.CARRIERS if c[3] is None or order["country"] in c[3]]
        if order["country"] != "US":
            options = [c for c in options if c[0] in ("DHL", "UPS", "FedEx")]
        carrier, level, (lo, hi), _ = rng.choice(options)
        shipment_id = str(uuid.uuid4())
        self.db.add(INS_SHIPMENT, (shipment_id, order["order_id"], carrier, level, self.ts(), self.ts()))
        self.db.add(UPD_PAYMENT, ("captured", self.ts(), order["payment_id"]))
        yield ops(rng.uniform(3, 18) * HOUR)
        self.db.add(UPD_SHIP_TRANSIT, (self.ts(), self.ts(), shipment_id))
        self.set_status(order, "shipped")
        yield ops(rng.uniform(lo, hi) * DAY)
        self.db.add(UPD_SHIP_DELIVERED, (self.ts(), self.ts(), shipment_id))
        self.set_status(order, "delivered")
        customer.spend += order["total"]
        if customer.segment != "vip" and customer.spend > 600 and customer.customer_id in self.customers:
            customer.segment = "vip"
            self.db.add(UPD_SEGMENT, ("vip", self.ts(), customer.customer_id))
        if order["fraud"]:
            yield ops(rng.uniform(3, 20) * DAY)
            if rng.random() < 0.6:
                self.db.add(UPD_PAYMENT, ("chargeback", self.ts(), order["payment_id"]))
                self.m.payments.labels("chargeback", "chargeback").inc()
            return
        if rng.random() < catalog.REFUND_RATE[order["category"]]:
            yield ops(rng.uniform(1, 12) * DAY)
            self.db.add(INS_REFUND, (str(uuid.uuid4()), order["order_id"], order["payment_id"], order["total"],
                                     weighted(rng, catalog.REFUND_REASONS)[0], self.ts()))
            self.db.add(UPD_PAYMENT, ("refunded", self.ts(), order["payment_id"]))
            self.set_status(order, "refunded")

    def fraud_actor(self, until):
        """A card-testing ring: throwaway accounts hammering checkout with mismatched cards."""
        rng = self.rng
        s = self.new_session(fraud=True)
        s.customer, s.channel = None, "affiliate"
        while utcnow() < until:
            p = rng.choice(self.by_category["Electronics"])
            self.emit(s, "product_view", product=p)
            self.emit(s, "add_to_cart", product=p, value=p["price"], page="/cart")
            self.emit(s, "checkout_start", page="/checkout", value=p["price"])
            yield from self.checkout(s, [(p, rng.randint(1, 3))])
            yield funnel(rng.uniform(15, 45))

    # ---- background changes (CDC updates and deletes) ----------------------------
    def change_price(self):
        p = self.rng.choice(self.products)
        p["price"] = max(1.99, round(p["price"] * self.rng.uniform(0.85, 1.12), 2))
        self.db.add(UPD_PRICE, (p["price"], self.ts(), p["product_id"]))

    def edit_profile(self):
        c = self.pick_returning() if self.customer_list else None
        if c:
            c.city, c.opt_in = self.rng.choice(CITIES[c.country]), not c.opt_in
            self.db.add(UPD_PROFILE, (c.city, c.opt_in, self.ts(), c.customer_id))

    def delete_account(self):
        c = self.pick_returning() if self.customer_list else None
        if c and c.orders == 0:
            del self.customers[c.customer_id]
            self.db.add(DEL_CUSTOMER, (c.customer_id,))

    def background(self, until):
        rng = self.rng
        while self.now < until:
            yield ops(rng.expovariate(1 / (3 * HOUR)))
            roll = rng.random()
            if roll < 0.45:
                self.change_price()
            elif roll < 0.9:
                self.edit_profile()
            else:
                self.delete_account()

    # ---- drivers -------------------------------------------------------------------
    def traffic(self, start, days, end):
        rng = self.rng
        promo_days = set(rng.sample(range(3, days - 1), k=2)) if days > 6 else set()
        for d in range(days):
            day = start + timedelta(days=d)
            mult = catalog.WEEKDAY[day.weekday()] * (1 + 0.012 * d) * rng.lognormvariate(0, 0.08)
            promo = None
            if d in promo_days:
                mult, promo = mult * 2.2, rng.choice(["SPRING20", "WELCOME20"])
            for hour in rng.choices(range(24), weights=catalog.HOURLY, k=int(self.cfg.sessions_per_day * mult)):
                t = day + timedelta(hours=hour, seconds=rng.uniform(0, HOUR))
                if t < end:
                    self.at(t, self.session(promo if promo and rng.random() < 0.6 else None))
            yield ops(DAY)

    def backfill(self, days, end, on_batch):
        start = (end - timedelta(days=days)).replace(minute=0, second=0, microsecond=0)
        self.now = start
        log.info("backfilling %d days of history from %s", days, start.isoformat())
        self.spawn(self.traffic(start, days, end))
        self.spawn(self.background(end))
        span = (end - start).total_seconds()

        def batch():
            self.backfill_progress = (self.now - start).total_seconds() / span
            self.m.backfill_progress.set(self.backfill_progress)
            on_batch()

        self.step(end, on_batch=batch)
        self.backfill_progress = 1.0
        self.m.backfill_progress.set(1.0)

    def run_live(self, on_tick):
        """Wall-clock mode. Lifecycles still pending from the backfill resume within minutes."""
        rng = self.rng
        self.live = True
        self.m.mode.set(1)
        now = utcnow()
        pending, self.heap = self.heap, []
        for when, _, coro in pending:
            self.at(now + timedelta(seconds=rng.uniform(1, 240)), coro)
        log.info("live: resumed %d in-flight lifecycles", len(pending))
        last = time.monotonic()
        while True:
            time.sleep(0.2)
            tick = time.monotonic()
            dt, last = tick - last, tick
            self.now = utcnow()
            spike = self.incident("traffic_spike")
            rate = self.cfg.live_sessions_per_min / 60 * (4 if spike else 1)
            for _ in range(_poisson(rng, rate * dt)):
                promo = "FLASH25" if spike and rng.random() < 0.7 else None
                self.spawn(self.session(promo))
            while self.pending_fraud:
                self.pending_fraud -= 1
                self.spawn(self.fraud_actor(self.incidents["fraud_burst"]))
            if rng.random() < dt / 45:
                self.change_price()
            if rng.random() < dt / 20:
                self.edit_profile()
            if rng.random() < dt / 300:
                self.delete_account()
            self.step(self.now)
            on_tick()


def _poisson(rng, lam):
    k, p, limit = 0, 1.0, math.exp(-lam)
    while True:
        p *= rng.random()
        if p <= limit:
            return k
        k += 1
