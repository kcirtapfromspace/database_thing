# Inspiration:
https://redpanda.com/blog/kafka-streaming-data-pipeline-from-postgres-to-duckdb

```sql
SELECT count(value.after.id) as user_count FROM read_parquet('s3://lakehouse/user-payments/debezium.public.user-*');
```
