import deequ
import sqlalchemy as sa
import json

def connect_to_database(user, password, host, port, database):
    return sa.create_engine("postgresql://{}:{}@{}:{}/{}".format(user, password, host, port, database))

def get_tables(engine):
    metadata = sa.MetaData()
    tables = sa.Table(metadata, autoload=True, autoload_with=engine)
    return [table.name for table in tables]

def define_completeness_check(columns):
    return deequ.Check(
        check_name="Completeness check",
        condition=deequ.check_functions.completeness(columns),
    )

def define_approximate_uniqueness_check(columns, rate=0.1):
    return deequ.Check(
        check_name="Approximate uniqueness check",
        condition=deequ.check_functions.approximate_unique(columns, rate),
    )

def define_value_distribution_check(columns, min_value, max_value):
    return deequ.Check(
        check_name="Value distribution check",
        condition=deequ.check_functions.has_value_distribution(columns, 
        distribution=deequ.ValueDistribution(min=min_value, max=max_value)),
    )

def define_entropy_check(columns, entropy):
    return deequ.Check(
        check_name="Entropy check",
        condition=deequ.check_functions.has_entropy(columns, entropy),
    )

def define_mutual_information_check(table):
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


def define_checks(table):
    all_columns = deequ.check_functions.all_columns_of_dataframe(table)
    checks = [
        define_completeness_check(all_columns),
        define_approximate_uniqueness_check(all_columns),
        define_value_distribution_check(all_columns, 0, 100),
        define_entropy_check(all_columns, 1.0),
    ]
    checks += define_mutual_information_check(table)
    return checks


def evaluate_table(engine, table_name, checks):
    data = deequ.DataFrame.from_postgres(table_name, engine)
    analysis = deequ.Analysis().add_checks(checks).run(data)
    return analysis.check_results

def store_results(engine, results):
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

def run_checks_on_database(user, password, host, port, database):
    engine = connect_to_database(user, password, host, port, database)
    tables = get_tables(engine)
    checks = define_checks()
    for table in tables:
        results = evaluate_table(engine, table, checks)
        store_results(engine, results)

