import os
import random

from faker import Faker
import psycopg2
import uuid

from time import sleep

## may need to verify this code actually works
def main():
    if os.getenv("RUNTIME_ENVIRONMENT") == "DOCKER":
        postgres_host = "postgres-db"
    else:
        postgres_host = "localhost"

    # Use a context manager for the database connection
    with psycopg2.connect(host=postgres_host, database="postgres", user="postgres", password="postgres") as conn:
        with conn.cursor() as cur:
            fake = Faker()

            num_iterations = 0  # Initialize a counter for the loop iterations

            while num_iterations < 10000:  # Set a maximum number of iterations
                customer_id = str(uuid.uuid4())

                # Use parameterized queries and error handling
                try:
                    cur.execute(
                        "INSERT INTO public.customers (id, first_name, last_name) VALUES (%s, %s, %s);",
                        (customer_id, fake.first_name(), fake.last_name())
                    )

                    order_id = str(uuid.uuid4())
                    order_date = fake.date_this_decade()
                    status = random.choice(["pending", "shipped", "delivered"])
                    cur.execute(
                        "INSERT INTO public.orders (id, user_id, order_date, status) VALUES (%s, %s, %s, %s);",
                        (order_id, customer_id, order_date, status)
                    )

                    payments = []
                    for i in range(random.randint(5, 15)):
                        payment_id = str(uuid.uuid4())
                        payment_method = random.randint(1, 3)  # Assuming there are three payment methods
                        amount = random.randint(10, 1000)
                        payments.append((payment_id, order_id, payment_method, amount))

                    cur.executemany(
                        "INSERT INTO public.payments (id, order_id, payment_method, amount) VALUES (%s, %s, %s, %s);",
                        payments
                    )

                    conn.commit()

                    # Add a log statement to indicate that the insert was successful
                    print(f"Inserted customer {customer_id} with order {order_id} and {len(payments)} payments.")

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
