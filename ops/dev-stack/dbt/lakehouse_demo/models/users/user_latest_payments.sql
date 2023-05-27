WITH latest_payments AS (
    SELECT *
    FROM {{ ref('latest_payments_per_user') }}
)

SELECT
    u.value.after.name,
    p.value.after.amount
FROM read_parquet('s3://lakehouse/user-payments/debezium.public.user-*') AS u
JOIN latest_payments AS p ON u.value.after.id = p.value.after.user_id

