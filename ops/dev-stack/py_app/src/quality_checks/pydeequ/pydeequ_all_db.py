import deequ
import sqlalchemy as sa
import json

def connect_to_database(user, password, host, port, database):
    return sa.create_engine("postgresql://{}:{}@{}:{}/{}".format(user, password, host, port, database))

def get_tables(engine):
    metadata = sa.MetaData()
    tables = sa.Table(metadata, autoload=True, autoload_with=engine)
    return [table.name for table in tables]

def define_checks():
    return [
        deequ.Check(
            check_name="Completeness check",
            condition=deequ.check_functions.completeness(deequ.check_functions.all_columns),
        ),
        deequ.Check(
            check_name="Approximate uniqueness check",
            condition=deequ.check_functions.approximate_unique(deequ.check_functions.all_columns, rate=0.1),
        ),
        deequ.Check(
            check_name="Value distribution check",
            condition=deequ.check_functions.has_value_distribution(deequ.check_functions.all_columns, 
            distribution=deequ.ValueDistribution(min=0, max=100)),
        ),
        deequ.Check(
            check_name="Entropy check",
            condition=deequ.check_functions.has_entropy(deequ.check_functions.all_columns, entropy=1.0),
        ),
        deequ.Check(
            check_name="Mutual information check",
            condition=deequ.check_functions.has_mutual_information(deequ.check_functions.column("column_name_1"), 
                deequ.check_functions.column("column_name_2"), 
                min_mutual_information=0.1),
        ),
    ]

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

