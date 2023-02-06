import pydeequ
import sqlalchemy as sa
import json

def connect_to_database(user, password, host, port, database):
    try:
        return sa.create_engine("postgresql://{}:{}@{}:{}/{}".format(user, password, host, port, database))
    except Exception as e:
        print(f"Failed to connect to the database: {e}")
        return None

def get_tables(engine):
    try:
        metadata = sa.MetaData()
        tables = sa.Table(metadata, autoload=True, autoload_with=engine)
        return [table.name for table in tables]
    except Exception as e:
        print(f"Failed to get tables from the database: {e}")
        return []

def define_completeness_check(columns):
    try:
        return deequ.Check(
            check_name="Completeness check",
            condition=deequ.check_functions.completeness(columns),
        )
    except Exception as e:
        print(f"Failed to define completeness check: {e}")
        return None

def define_approximate_uniqueness_check(columns, rate=0.1):
    try:
        return deequ.Check(
            check_name="Approximate uniqueness check",
            condition=deequ.check_functions.approximate_unique(columns, rate),
        )
    except Exception as e:
        print(f"Failed to define approximate uniqueness check: {e}")
        return None

def define_value_distribution_check(columns, min_value, max_value):
    try:
        return deequ.Check(
            check_name="Value distribution check",
            condition=deequ.check_functions.has_value_distribution(columns, 
            distribution=deequ.ValueDistribution(min=min_value, max=max_value)),
        )
    except Exception as e:
        print(f"Failed to define value distribution check: {e}")
        return None

def define_entropy_check(columns, entropy):
    try:
        return deequ.Check(
            check_name="Entropy check",
            condition=deequ.check_functions.has_entropy(columns, entropy),
        )
    except Exception as e:
        print(f"Failed to define entropy check: {e}")
        return None

def define_mutual_information_check(table):
    try:
        all_columns = deequ.check_functions.all_columns_of_dataframe(table)
        checks = []
        for i, column1 in enumerate(all_columns):
            for column2 in all_columns[i+1:]:
                checks.append(deequ.Check(
                    check_name=f"Mutual information check between {column1} and {column2}",
                    condition=deequ.check_functions.has_mutual_information(
                        deequ.check_functions.column(column1), 
                        deequ.check_functions.column(column2), 
                        min_mutual_information=0.1
                    )
                ))
        return checks
    except Exception as e:
        print("Error in define_mutual_information_check:", str(e))
        return []

def define_checks(table):
    try:
        all_columns = deequ.check_functions.all_columns_of_dataframe(table)
        checks = [
            define_completeness_check(all_columns),
            define_approximate_uniqueness_check(all_columns),
            define_value_distribution_check(all_columns, 0, 100),
            define_entropy_check(all_columns, 1.0),
        ]
        checks += define_mutual_information_check(table)
        return checks
    except Exception as e:
        print("Error in define_checks:", str(e))
        return []

def evaluate_table(engine, table_name, checks):
    try:
        data = deequ.DataFrame.from_postgres(table_name, engine)
        analysis = deequ.Analysis().add_checks(checks).run(data)
        return analysis.check_results
    except Exception as e:
        print("Error in evaluate_table:", str(e))
        return []

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
            conn.execute(results_table.insert().values(
                check_name=result.check_name,
                status=result.status,
                message=result.message,
                attributes=json.dumps(result.attributes)))
        conn.close()
    except Exception as e:
        print("Error in store_results:", str(e))

def run_checks_on_database(user, password, host, port, database):
    try:
        engine = connect_to_database(user, password, host, port, database)
    except Exception as e:
        print(f"Error connecting to the database: {e}")
        return

    try:
        tables = get_tables(engine)
    except Exception as e:
        print(f"Error getting tables: {e}")
        return

    try:
        checks = define_checks()
    except Exception as e:
        print(f"Error defining checks: {e}")
        return

    for table in tables:
        try:
            results = evaluate_table(engine, table, checks)
        except Exception as e:
            print(f"Error evaluating table '{table}': {e}")
            continue

        try:
            store_results(engine, results)
        except Exception as e:
            print(f"Error storing results for table '{table}': {e}")


