```sql customers
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

SELECT 
name as customer_name,

FROM read_parquet('s3://lakehouse/export/user_latest_payments.parquet')
group by 1
```

{#each customers as customer}

- [{customer.customer_name}](./{customer.customer_name})

{/each}