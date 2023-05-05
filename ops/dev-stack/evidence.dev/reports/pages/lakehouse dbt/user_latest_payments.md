# user_latest_payments

```user_latest_payments_via_local

WITH data AS (
  SELECT *
FROM read_parquet('../../user_latest_payments.parquet')
-- FROM read_parquet('s3://lakehouse/export/user_latest_payments.parquet')
)