import os
import random

from faker import Faker
import psycopg2
import uuid

from time import sleep


def main():
    # Load database configuration from environment variables
    postgres_host = os.getenv("POSTGRES_HOST", "postgres-db" if os.getenv("RUNTIME_ENVIRONMENT") == "DOCKER" else "localhost")
    postgres_db = os.getenv("POSTGRES_DBNAME", "postgres")
    postgres_user = os.getenv("POSTGRES_USER", "postgres")
    postgres_password = os.getenv("POSTGRES_PASSWORD")

    if not postgres_password:
        raise ValueError("POSTGRES_PASSWORD environment variable must be set")

    # Use a context manager for the database connection
    with psycopg2.connect(host=postgres_host, database=postgres_db, user=postgres_user, password=postgres_password) as conn:
        with conn.cursor() as cur:
            fake = Faker()

            num_iterations = 0  # Initialize a counter for the loop iterations

            while num_iterations < 10000:  # Set a maximum number of iterations
                user_id = str(uuid.uuid4())

                # Use parameterized queries and error handling
                try:
                    cur.execute(
                        "INSERT INTO public.user (id, name, email, address) VALUES (%s, %s, %s, %s);",
                        (user_id, fake.name(), fake.email(), fake.address())
                    )

                    payments = []
                    for i in range(random.randint(5, 15)):
                        payments.append((uuid.uuid4().hex, user_id, random.randint(1, 100)))

                    cur.executemany(
                        "INSERT INTO public.payment (id, user_id, amount) VALUES (%s, %s, %s);",
                        payments
                    )

                    conn.commit()

                    # Add a log statement to indicate that the insert was successful
                    print(f"Inserted user {user_id} with {len(payments)} payments.")

                except Exception as e:
                    # Add a log statement to indicate that an error occurred
                    print(f"Error: {e}")
                    conn.rollback()

                # Add a sleep timer
                sleep(1)
                # Increment the counter
                num_iterations += 1
                # Add a log statement to indicate that the loop is still running
            print("finished generating data!")


if __name__ == "__main__":
    main()
