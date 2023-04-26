-- Install and load necessary extensions
INSTALL 'httpfs';
INSTALL 'parquet';
LOAD 'httpfs';
LOAD 'parquet';

-- Set up MinIO connection parameters
SET s3_endpoint='127.0.0.1:9000';
SET s3_url_style='path';
SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa';
SET s3_secret_access_key='minio123';

-- Replace 'my-bucket' with the name of your MinIO bucket and
-- 'path/to/your/parquet_file.parquet' with the path to your Parquet file within the bucket
WITH data AS (
  SELECT *
FROM read_parquet('s3://lakehouse/user-payments/debezium.public.user-*')
)

SELECT
  value."after".id AS id,
  value."after"."name" AS name,
  value."after".email AS email,
  value."after".address AS address,
  value.before.id AS before_id,
  value.before."name" AS before_name,
  value.before.email AS before_email,
  value.before.address AS before_address,
  value.source."version" AS source_version,
  value.source.connector AS source_connector,
  value.source."name" AS source_name,
  value.source.ts_ms AS source_ts_ms,
  value.source."snapshot" AS source_snapshot,
  value.source.db AS source_db,
  value.source."sequence" AS source_sequence,
  value.source."schema" AS source_schema,
  value.source."table" AS source_table,
  value.source."txId" AS source_txId,
  value.source.lsn AS source_lsn,
  value.source.xmin AS source_xmin,
  value.op AS op,
  value.ts_ms AS ts_ms,
  value."transaction".id AS transaction_id,
  value."transaction".total_order AS transaction_total_order,
  value."transaction".data_collection_order AS transaction_data_collection_order
FROM data;


WITH data AS (
  SELECT *
FROM read_parquet('s3://lakehouse/user-payments/debezium.public.payments-*')
)
SELECT
  value."after".id AS id,
  value."after"."name" AS name,
  value."after".email AS email,
  value."after".address AS address,
  value.before.id AS before_id,
  value.before."name" AS before_name,
  value.before.email AS before_email,
  value.before.address AS before_address,
  value.source."version" AS source_version,
  value.source.connector AS source_connector,
  value.source."name" AS source_name,
  value.source.ts_ms AS source_ts_ms,
  value.source."snapshot" AS source_snapshot,
  value.source.db AS source_db,
  value.source."sequence" AS source_sequence,
  value.source."schema" AS source_schema,
  value.source."table" AS source_table,
  value.source."txId" AS source_txId,
  value.source.lsn AS source_lsn,
  value.source.xmin AS source_xmin,
  value.op AS op,
  value.ts_ms AS ts_ms,
  value."transaction".id AS transaction_id,
  value."transaction".total_order AS transaction_total_order,
  value."transaction".data_collection_order AS transaction_data_collection_order
FROM data;

DESCRIBE SELECT *
FROM read_parquet('s3://lakehouse/user-payments/debezium.public.user-*');
DESCRIBE SELECT *
FROM read_parquet('s3://lakehouse/user-payments/debezium.public.payment-*');

DESCRIBE SELECT *
FROM read_parquet('s3://lakehouse/export/latest_payments_per_user.parquet');

-- Replace 'my-bucket' with the name of your MinIO bucket and
-- 'path/to/your/parquet_file.parquet' with the path to your Parquet file within the bucket
WITH data AS (
  SELECT *
FROM read_parquet('s3://lakehouse/export/latest_payments_per_user.parquet')
)

SELECT 
    "before"."id" AS before_id,
    "before"."user_id" AS before_user_id,
    "before"."amount" AS before_amount,
    "after"."id" AS after_id,
    "after"."user_id" AS after_user_id,
    "after"."amount" AS after_amount,
    source."version" AS source_version,
    source.connector AS source_connector,
    source."name" AS source_name,
    source.ts_ms AS source_ts_ms,
    source."snapshot" AS source_snapshot,
    source.db AS source_db,
    source."sequence" AS source_sequence,
    source."schema" AS source_schema,
    source."table" AS source_table,
    source."txId" AS source_txId,
    source.lsn AS source_lsn,
    source.xmin AS source_xmin,
    op,
    ts_ms,
    "transaction"."id" AS transaction_id,
    "transaction".total_order AS transaction_total_order,
    "transaction".data_collection_order AS transaction_data_collection_order
FROM data;