import pandas as pd
import psycopg2
import deequ
import json

# Connect to the PostgreSQL database
conn = psycopg2.connect("host=<host> dbname=<dbname> user=<user> password=<password>")

# Query the list of tables in the database
query = "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'"
tables = pd.read_sql_query(query, conn)

# Loop over the list of tables and evaluate each one
for i, row in tables.iterrows():
    table_name = row["table_name"]
    query = f"SELECT * FROM {table_name}"
    data = pd.read_sql_query(query, conn)

    # Define the constraints for the data
    constraints = []
    for column_name in data.columns:
        constraints.append(deequ.Constraint(deequ.Check.HasCompleteness(column_name, completeness=1.0)))

    # Run the quality checks on the data
    result = deequ.verify(data, constraints)

    # Store the results in the `quality_check_results` table
    query = f"INSERT INTO quality_check_results (table_name, result) VALUES ({table_name}, '{json.dumps(result.to_json())}')"
    conn.execute(query)

# Commit the results to the database
conn.commit()

# Close the database connection
conn.close()
