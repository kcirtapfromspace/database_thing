package db

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func openDB(dbname string) (*sqlx.DB, error) {
    user := os.Getenv("POSTGRES_USER")
    password := os.Getenv("POSTGRES_PASSWORD")
    host := os.Getenv("POSTGRES_HOST")
    port := os.Getenv("POSTGRES_PORT")
    // check if the environment variables are set 
    if user == "" || password == "" || host == "" || port == "" || dbname == "" {
        log.Fatal("Missing environment variables")
    }
    connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
    // Connect to the Postgres database
    db, err := sqlx.Open("postgres", connStr)
    if err != nil {
        log.Fatal(err)
    }
    return db, err

}
