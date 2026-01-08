import os
import pandas as pd
from sqlalchemy import create_engine, MetaData
from tabulate import tabulate

pd.options.display.max_rows = None
pd.options.display.max_columns = None

# Load database configuration from environment variables
postgres_user = os.getenv("POSTGRES_USER", "postgres")
postgres_password = os.getenv("POSTGRES_PASSWORD")
postgres_host = os.getenv("POSTGRES_HOST", "localhost")
postgres_port = os.getenv("POSTGRES_PORT", "5432")

if not postgres_password:
    raise ValueError("POSTGRES_PASSWORD environment variable must be set")

# Build connection string from environment variables
connection_string = f"postgresql://{postgres_user}:{postgres_password}@{postgres_host}:{postgres_port}"
engine = create_engine(connection_string + '/postgres')
metadata = MetaData()
metadata.reflect(bind=engine)

# list the databases
df = pd.read_sql_query("SELECT datname FROM pg_database WHERE datistemplate = false", engine)
print(df)

# select the database you want to inspect
database_name = input("Enter the name of the database you want to inspect:")
engine = create_engine(connection_string + f'/{database_name}')

# list the tables in the selected database
df = pd.read_sql_query("SELECT tablename FROM pg_tables where schemaname = 'public'", engine)
tables = df['tablename'].tolist()

# iterate over all the tables in the database
for table_name in tables:
    # get column names of the selected table
    columns = ', '.join([col.name for col in metadata.tables[table_name].columns])

    # create a report of column names
    with open(f'table_columns_{database_name}.txt', 'a') as f:
        f.write(f'\n\n{table_name} table columns:\n')
        f.write(columns)

    missing_data = df.isnull().sum()

    # check for duplicate data
    duplicate_data = df.duplicated().sum()

    # Add missing data & duplicate data to report
    # with open(f'table_columns_{database_name}.txt', 'a') as f:
    #     f.write('\n\nMissing Data:\n')
    #     missing_data = missing_data.to_dict()
    #     missing_data = [[col, val] for col, val in missing_data.items()]
    #     f.write(tabulate(missing_data, headers=["", "Missing Values"], tablefmt="pipe"))
    #     f.write('\n\nDuplicate Data:\n')
    #     f.write(tabulate([["Duplicate Rows", duplicate_data]], headers=["", "Count"], tablefmt="pipe"))