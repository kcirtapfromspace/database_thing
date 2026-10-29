-- Streaming layer: RisingWave consumes the Redpanda topics directly and keeps these
-- materialized views incrementally up to date. Grafana queries them over the Postgres protocol.
-- Idempotent: applied (and re-applied) by the risingwave-ddl job.

------------------------------------------------------------------------------------------
-- Inputs
------------------------------------------------------------------------------------------
-- Clickstream events produced straight to Redpanda by shopgen (Avro, schema registry)
CREATE SOURCE IF NOT EXISTS clickstream
WITH (
    connector = 'kafka',
    topic = 'shop.clickstream',
    properties.bootstrap.server = 'redpanda-0.redpanda.kafka.svc.cluster.local:9093',
    scan.startup.mode = 'earliest'
) FORMAT PLAIN ENCODE AVRO (schema.registry = 'http://redpanda-0.redpanda.kafka.svc.cluster.local:8081');

-- OLTP tables mirrored from Postgres via Debezium change events (inserts, updates, deletes)
CREATE TABLE IF NOT EXISTS orders (PRIMARY KEY (order_id))
WITH (
    connector = 'kafka',
    topic = 'cdc.shop.orders',
    properties.bootstrap.server = 'redpanda-0.redpanda.kafka.svc.cluster.local:9093',
    scan.startup.mode = 'earliest'
) FORMAT DEBEZIUM ENCODE AVRO (schema.registry = 'http://redpanda-0.redpanda.kafka.svc.cluster.local:8081');

CREATE TABLE IF NOT EXISTS payments (PRIMARY KEY (payment_id))
WITH (
    connector = 'kafka',
    topic = 'cdc.shop.payments',
    properties.bootstrap.server = 'redpanda-0.redpanda.kafka.svc.cluster.local:9093',
    scan.startup.mode = 'earliest'
) FORMAT DEBEZIUM ENCODE AVRO (schema.registry = 'http://redpanda-0.redpanda.kafka.svc.cluster.local:8081');

CREATE TABLE IF NOT EXISTS products (PRIMARY KEY (product_id))
WITH (
    connector = 'kafka',
    topic = 'cdc.shop.products',
    properties.bootstrap.server = 'redpanda-0.redpanda.kafka.svc.cluster.local:9093',
    scan.startup.mode = 'earliest'
) FORMAT DEBEZIUM ENCODE AVRO (schema.registry = 'http://redpanda-0.redpanda.kafka.svc.cluster.local:8081');

------------------------------------------------------------------------------------------
-- Live metrics
------------------------------------------------------------------------------------------
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_traffic_per_minute AS
SELECT window_start, event_type, device,
       count(*) AS events,
       count(DISTINCT session_id) AS sessions
FROM TUMBLE(clickstream, event_time, INTERVAL '1 minute')
GROUP BY window_start, event_type, device;

-- one row per session, updated as the shopper moves through the funnel
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_session_funnel AS
SELECT session_id,
       min(event_time) AS started_at,
       max(event_time) AS last_seen,
       max(device) AS device,
       max(utm_source) AS channel,
       bool_or(event_type = 'product_view') AS viewed_product,
       bool_or(event_type = 'add_to_cart') AS added_to_cart,
       bool_or(event_type = 'checkout_start') AS started_checkout,
       bool_or(event_type = 'purchase') AS purchased,
       coalesce(sum(value) FILTER (WHERE event_type = 'purchase'), 0) AS revenue
FROM clickstream
GROUP BY session_id;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_product_views_per_minute AS
SELECT window_start, product_id, category, count(*) AS views
FROM TUMBLE(clickstream, event_time, INTERVAL '1 minute')
WHERE event_type = 'product_view'
GROUP BY window_start, product_id, category;

-- orders change status over time (paid -> shipped -> refunded); the view follows the CDC updates
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_sales_per_minute AS
SELECT window_start,
       count(*) AS orders,
       sum(total) AS gmv,
       sum(item_count) AS units,
       count(*) FILTER (WHERE promo_code IS NOT NULL) AS promo_orders
FROM TUMBLE(orders, created_at, INTERVAL '1 minute')
WHERE status IN ('paid', 'shipped', 'delivered', 'refunded')
GROUP BY window_start;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_payments_per_minute AS
SELECT window_start, provider, method,
       count(*) AS attempts,
       count(*) FILTER (WHERE status = 'declined') AS declines,
       count(*) FILTER (WHERE decline_reason = 'processor_unavailable') AS processor_errors,
       count(*) FILTER (WHERE decline_reason = 'fraud_suspected') AS fraud_declines
FROM TUMBLE(payments, attempted_at, INTERVAL '1 minute')
GROUP BY window_start, provider, method;

------------------------------------------------------------------------------------------
-- Anomaly detection (5-minute sliding windows, advancing every minute)
------------------------------------------------------------------------------------------
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_provider_health_5m AS
SELECT window_start, window_end, provider,
       count(*) AS attempts,
       count(*) FILTER (WHERE status = 'declined') AS declines
FROM HOP(payments, attempted_at, INTERVAL '1 minute', INTERVAL '5 minutes')
GROUP BY window_start, window_end, provider;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_customer_velocity_5m AS
SELECT window_start, window_end, customer_id,
       count(*) AS attempts,
       count(*) FILTER (WHERE status = 'declined') AS declines,
       count(DISTINCT card_country) AS card_countries,
       max(risk_score) AS max_risk,
       sum(amount) AS amount
FROM HOP(payments, attempted_at, INTERVAL '1 minute', INTERVAL '5 minutes')
GROUP BY window_start, window_end, customer_id;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_alerts AS
SELECT 'payment_decline_spike' AS alert_type,
       provider AS subject,
       window_end AS observed_at,
       attempts,
       declines,
       round(declines::numeric / attempts, 3) AS decline_rate,
       provider || ' declined ' || declines::varchar || ' of ' || attempts::varchar
           || ' payments in a 5 minute window' AS summary
FROM mv_provider_health_5m
WHERE attempts >= 5 AND declines::numeric / attempts > 0.4
UNION ALL
SELECT 'fraud_velocity' AS alert_type,
       customer_id AS subject,
       window_end AS observed_at,
       attempts,
       declines,
       round(declines::numeric / attempts, 3) AS decline_rate,
       'customer made ' || attempts::varchar || ' payment attempts with cards from '
           || card_countries::varchar || ' countries in 5 minutes (max risk '
           || round(max_risk::numeric, 2)::varchar || ')' AS summary
FROM mv_customer_velocity_5m
WHERE attempts >= 4 AND (card_countries >= 2 OR declines >= 3);

-- One signal per alert type, subject and 10-minute window. These rows never change once
-- created, so each becomes exactly one event (alert rows keep updating as windows fill).
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_incident_signals AS
SELECT alert_type, subject, window_start AS incident_window
FROM TUMBLE(mv_alerts, observed_at, INTERVAL '10 minutes')
GROUP BY alert_type, subject, window_start;

-- the first version of this file streamed every alert update
DROP SINK IF EXISTS alerts_to_kafka;

-- each signal becomes an event on Redpanda; Argo Events opens an incident for it.
-- snapshot = false: re-applying this file never replays past signals.
CREATE SINK IF NOT EXISTS incident_signals_to_kafka FROM mv_incident_signals
WITH (
    connector = 'kafka',
    properties.bootstrap.server = 'redpanda-0.redpanda.kafka.svc.cluster.local:9093',
    topic = 'alerts.anomalies',
    snapshot = 'false'
) FORMAT PLAIN ENCODE JSON (force_append_only = 'true');

-- incidents opened by the Argo incident-response workflow, written back for the dashboards
CREATE TABLE IF NOT EXISTS incidents (
    workflow varchar PRIMARY KEY,
    alert_type varchar,
    subject varchar,
    incident_window timestamptz,
    summary varchar,
    opened_at timestamptz
);
