import great_expectations as gx
from sqlalchemy import create_engine
import pandas as pd
from great_expectations.core.expectation_configuration import ExpectationConfiguration
from great_expectations.data_context.types.resource_identifiers import (
    ExpectationSuiteIdentifier,
)
from great_expectations.exceptions import DataContextError
# add great_expectations/plugins to path
import sys, os
sys.path.append(os.path.join(os.path.dirname(os.path.realpath(__file__)), 'great_expectations'))

from plugins import column_custom_max_expectation

config = """
name: postgres-db
class_name: Datasource
execution_engine:
  class_name: SqlAlchemyExecutionEngine
  credentials:
    host: postgres
    port: '5432'
    username: postgres
    password: postgres
    database: postgres
    drivername: postgresql
data_connectors:
  default_runtime_data_connector_name:
    class_name: RuntimeDataConnector
    batch_identifiers:
      - default_identifier_name
  default_inferred_data_connector_name:
    class_name: InferredAssetSqlDataConnector
    include_schema_name: True
    introspection_directives:
      schema_name: public
  default_configured_data_connector_name:
    class_name: ConfiguredAssetSqlDataConnector
    assets:
      turbotax:
        class_name: Asset
        schema_name: public
"""
my_context.test_yaml_config(
    config=config
)

context = gx.get_context()
context.run_checkpoint(checkpoint_name="my_checkpoint_with_custom_expectation")
context.open_data_docs()


def create_data_asset(df, table_name):
    # Create a Great Expectations Data Asset for the DataFrame
    data_asset = gx.DataAsset.create(df, dataset_name=table_name)
    return data_asset

def validate_data_asset(data_asset):
    # Get the column names from the DataFrame
    column_names = data_asset.get_dataframe().columns

    # Loop over the column names
    expectations = []
    for column_name in column_names:
        # Define the Expectations for the column
        expectations += [
            gx.expect_column_to_exist(column_name),
            gx.expect_column_not_to_be_null(column_name),
            gx.expect_column_values_to_be_between(column_name, 0, 100),
            gx.expect_column_values_to_match_regex(column_name, r"^[a-z]+$"),
        ]

    # Validate the Data Asset
    validation_result = data_asset.validate(expectations)

    # Print the validation results
    print(f"Validation results for table '{data_asset.dataset_name}':")
    print(validation_result)

class PostgreSQLValidator:
    def __init__(self, database_url):
        self.engine = create_engine(database_url)
        
    def validate_all_tables(self):
        # Get the table names from the database
        table_names = self.engine.table_names()

        # Loop over the table names
        for table_name in table_names:
            # Load data from the database into a pandas DataFrame
            df = pd.read_sql_table(table_name, self.engine)

            # Create a Great Expectations Data Asset for the DataFrame
            data_asset = create_data_asset(df, table_name)

            # Validate the Data Asset
            validate_data_asset(data_asset)

# Connect to your PostgreSQL database
validator = PostgreSQLValidator('postgresql://user:password@host:port/database')

# Validate all tables in the database
validator.validate_all_tables()
