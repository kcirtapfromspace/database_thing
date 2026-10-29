---
title: How the data gets here
---

1. **Storefront**: `shopgen` simulates shoppers. Orders, payments, shipments and refunds are written
   to **Postgres**; clickstream events are produced straight to **Redpanda** as Avro.
2. **Change data capture**: **Debezium** streams every insert, update and delete from Postgres into
   `cdc.shop.*` topics.
3. **Streaming SQL**: **RisingWave** consumes both streams and keeps live materialized views
   (funnel, revenue per minute, payment health). Sliding-window anomaly detection publishes
   incident signals back to Redpanda.
4. **Lakehouse**: the Kafka Connect S3 sink lands every topic in **MinIO** as Parquet (`raw/`).
5. **Event-driven orchestration**: **Argo Events** (its event bus also runs on Redpanda) reacts to
   new files in MinIO by submitting a **dbt + DuckDB** workflow in **Argo Workflows**, which rebuilds
   the marts in `marts/`; anomaly signals open incident-response workflows.
6. **Analytics**: this site queries the marts directly from MinIO.

```sql freshness
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select strftime(refreshed_at, '%Y-%m-%d %H:%M:%S') as refreshed_at_utc, triggered_by,
       strftime(latest_event_at, '%Y-%m-%d %H:%M:%S') as latest_event_utc,
       clickstream_events as clickstream_events_num0, orders as orders_num0, customers as customers_num0
from read_parquet('s3://lakehouse/marts/pipeline_freshness.parquet')
```


### Last refresh

<DataTable data={freshness}/>

The `triggered_by` value is the S3 key of the Parquet file whose arrival kicked off the dbt run.
