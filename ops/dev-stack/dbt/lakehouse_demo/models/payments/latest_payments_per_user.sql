WITH payments AS (
    SELECT *,
           row_number() OVER (PARTITION BY value.after.user_id ORDER BY timestamp DESC) AS rn
    FROM read_parquet('s3://lakehouse/user-payments/debezium.public.payment-*')
)

SELECT *
FROM payments
WHERE rn <= 3
