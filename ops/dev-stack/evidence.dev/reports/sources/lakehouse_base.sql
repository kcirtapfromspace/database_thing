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