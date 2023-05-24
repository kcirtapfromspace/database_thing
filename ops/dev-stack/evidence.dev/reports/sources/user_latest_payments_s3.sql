
INSTALL 'httpfs';
INSTALL 'parquet';
LOAD 'httpfs';
LOAD 'parquet';

-- Set up MinIO connection parameters
SET s3_endpoint=' minio.minio.svc.cluster.local:9000';
SET s3_url_style='path';
SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa';
SET s3_secret_access_key='minio123';

-- Replace 'my-bucket' with the name of your MinIO bucket and
-- 'path/to/your/parquet_file.parquet' with the path to your Parquet file within the bucket
SELECT *
FROM read_parquet('s3://lakehouse/export/latest_payments_per_user.parquet');
