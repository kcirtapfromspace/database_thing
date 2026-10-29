-- The shop schema (customers, orders, payments, ...) is created and owned by shopgen.
-- Debezium captures it via logical replication (wal_level=logical, set in db_deployment.yaml).
CREATE SCHEMA IF NOT EXISTS shop;
