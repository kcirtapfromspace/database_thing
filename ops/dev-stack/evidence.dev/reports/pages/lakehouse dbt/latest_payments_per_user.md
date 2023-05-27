
# latest_payments_per_user

```latest_payments_per_user_via_local
-- Install and load necessary extensions
INSTALL 'httpfs';
INSTALL 'parquet';
LOAD 'httpfs';
LOAD 'parquet';

-- Replace 'my-bucket' with the name of your MinIO bucket and
-- 'path/to/your/parquet_file.parquet' with the path to your Parquet file within the bucket
WITH data AS (
  SELECT *
FROM read_parquet('../../latest_payments_per_user.parquet')
)
SELECT 
    value."before"."id" AS before_id,
    value."before"."user_id" AS before_user_id,
    value."before"."amount" AS before_amount,
    value."after"."id" AS after_id,
    value."after"."user_id" AS after_user_id,
    value."after"."amount" AS after_amount,
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
    value.op,
    value.ts_ms,
    value."transaction"."id" AS transaction_id,
    value."transaction".total_order AS transaction_total_order,
    value."transaction".data_collection_order AS transaction_data_collection_order
FROM data;
```

<BarChart 
    data={latest_payments_per_user_via_local} 
    y=after_amount 
    x=after_id
    series=after_user_id
/>

<!-- <DataTable
  data={latest_payments_per_user_via_local}
  columns={[
    { accessor: 'id', Header: 'ID' },
    { accessor: 'name', Header: 'Name' },
    { accessor: 'email', Header: 'Email' },
    { accessor: 'address', Header: 'Address' },
    { accessor: 'before_id', Header: 'Before ID' },
    { accessor: 'before_name', Header: 'Before Name' },
    { accessor: 'before_email', Header: 'Before Email' },
    { accessor: 'before_address', Header: 'Before Address' },
    { accessor: 'op', Header: 'Operation' },
    { accessor: 'ts_ms', Header: 'Timestamp' },
    { accessor: 'transaction_id', Header: 'Transaction ID' }
  ]}
/> -->



latest_payments_per_user parquet in lakehouse
```latest_payments_per_user_via_lakehouse
-- Install and load necessary extensions
INSTALL 'httpfs';
INSTALL 'parquet';
LOAD 'httpfs';
LOAD 'parquet';


-- Set up MinIO connection parameters
SET s3_endpoint='minio.minio.svc.cluster.local:9000';
SET s3_url_style='path';
SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa';
SET s3_secret_access_key='minio123';

-- Replace 'my-bucket' with the name of your MinIO bucket and
-- 'path/to/your/parquet_file.parquet' with the path to your Parquet file within the bucket
WITH data AS (
  SELECT *
FROM read_parquet('s3://lakehouse/export/latest_payments_per_user.parquet')
)
SELECT 
    value."after"."id" AS after_id,
    value."after"."amount" AS after_amount,
    value."after"."user_id" AS after_user_id
FROM data;
```

<!-- <BarChart 
    data={latest_payments_per_user_via_lakehouse} 
    y=after_id
    x=after_amount
    series=after_user_id
/>  -->


<!-- <DataTable
  data={latest_payments_per_user_via_lakehouse}
  columns={[
    { accessor: 'id', Header: 'ID' },
    { accessor: 'name', Header: 'Name' },
    { accessor: 'email', Header: 'Email' },
    { accessor: 'address', Header: 'Address' },
    { accessor: 'before_id', Header: 'Before ID' },
    { accessor: 'before_name', Header: 'Before Name' },
    { accessor: 'before_email', Header: 'Before Email' },
    { accessor: 'before_address', Header: 'Before Address' },
    { accessor: 'op', Header: 'Operation' },
    { accessor: 'ts_ms', Header: 'Timestamp' },
    { accessor: 'transaction_id', Header: 'Transaction ID' },
  ]}
/> -->

