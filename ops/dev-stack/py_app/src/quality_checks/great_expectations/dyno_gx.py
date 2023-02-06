import great_expectations as ge
import pandas as pd
import sqlalchemy as sa
import json
import logging
import configparser

def connect_to_database(user, password, host, port, database):
    try:
        engine = sa.create_engine("postgresql://{}:{}@{}:{}/{}".format(user, password, host, port, database))
        logging.info("Successfully connected to database")
        return engine
    except Exception as e:
        logging.error("Failed to connect to database: {}".format(e))
        raise

def get_tables(engine):
    try:
        metadata = sa.MetaData()
        tables = sa.Table(metadata, autoload=True, autoload_with=engine)
        logging.info("Successfully retrieved tables from database")
        return [table.name for table in tables]
    except Exception as e:
        logging.error("Failed to retrieve tables from database: {}".format(e))
        raise

def define_completeness_check(columns):
    return ge.expect_df_columns_to_not_be_null(columns)

def define_approximate_uniqueness_check(columns, rate=0.1):
    return ge.expect_column_unique_value_count_to_be_between(columns, 1, 1 / rate)

def define_value_distribution_check(columns, min_value, max_value):
    return ge.expect_column_values_to_be_between(columns, min_value=min_value, max_value=max_value)

def define_entropy_check(columns, entropy):
    return ge.expect_column_entropy_to_be_between(columns, min_entropy=entropy, max_entropy=entropy)

def define_mutual_information_check(table):
    all_columns = table.columns
    checks = []
    for i, column1 in enumerate(all_columns):
        for column2 in all_columns[i+1:]:
            checks.append(ge.expect_column_pair_wise_correlation_to_be_between(
                table,
                column1,
                column2,
                min_value=0.1
            ))
    return checks

def evaluate_table(engine, table, checks):
    try:
        data = pd.read_sql_table(table, engine)
    except Exception as e:
        print("Error reading data from table: ", e)
        return {}
    results = {}
    for check in checks:
        try:
            result = check(data)
            result_key = str(check).split("\n")[0]
            results[result_key] = result.results
        except Exception as e:
            print("Error evaluating check: ", e)
    return results


def store_results(engine, results):
    try:
        metadata = sa.MetaData()
        results_table = sa.Table("quality_check_results", metadata,
        sa.Column("check_name", sa.String),
        sa.Column("status", sa.String),
        sa.Column("message", sa.String),
        sa.Column("attributes", sa.String))
        conn = engine.connect()
        for result in results:
            try:
                conn.execute(results_table.insert().values(
                    check_name=result.check_name,
                    status=result.status,
                    message=result.message,
                    attributes=json.dumps(result.attributes)))
            except Exception as e:
                print("Error storing result: ", e)
        conn.close()
    except Exception as e:
        print("Error connecting to database: ", e)
    

def run_checks_on_database(user, password, host, port, database):
    try:
        engine = connect_to_database(user, password, host, port, database)
    except Exception as e:
        print("Error connecting to database: ", e)
        return
    try:
        tables = get_tables(engine)
    except Exception as e:
        print("Error getting tables: ", e)
        return
    try:
        checks = define_checks()
    except Exception as e:
        print("Error defining checks: ", e)
        return
    for table in tables:
        try:
            results = evaluate_table(engine, table, checks)
            store_results(engine, results)
        except Exception as e:
            print("Error running checks: ", e)
        return


