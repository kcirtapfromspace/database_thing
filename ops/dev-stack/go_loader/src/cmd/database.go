package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"

	_ "github.com/lib/pq"
)

const (
	host   = "postgres-db"
	port   = 5432
	user   = "postgres"
	dbname = "postgres"
)

// Connect to the PostgreSQL database
func Connect() (*sql.DB, error) {
	password := os.Getenv("POSTGRES_PASSWORD")
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
		os.Exit(1)
	}
	return db, nil
}

func sanitizeColumnName(columnName string) string {
	columnName = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return -1
	}, columnName)
	columnName = strings.ToLower(columnName)
	columnName = strings.Replace(columnName, ".", "_", -1)
	columnName = strings.Replace(columnName, " ", "_", -1)
	return columnName
}

func insertData(db *sql.DB, tableName string, data [][]string) {
	// Prepare insert query
	query := "INSERT INTO " + tableName + " VALUES("
	for i := 0; i < len(data[0]); i++ {
		query += "$" + strconv.Itoa(i+1)
		if i < len(data[0])-1 {
			query += ","
		}
	}
	query += ")"
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Fatalf("Error: %v", err)
		return
	}
	defer stmt.Close()

	// Insert data
	for _, record := range data {
		interfaces := make([]interface{}, len(record))
		for i, v := range record {
			interfaces[i] = interface{}(v)
		}
		_, err = stmt.Exec(interfaces...)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
	}
}

func createTable(db *sql.DB, tableName string, columns []string) {
	if _, err := db.Exec("SELECT 1 FROM " + tableName + " LIMIT 1"); err == sql.ErrNoRows {
		log.Fatalf("Error: %v", err)
		// table does not exist, create it
	}

	// Create table query
	if _, err := db.Exec("SELECT 1 FROM " + tableName + " LIMIT 1"); err == sql.ErrNoRows {
		// table does not exist, create it
		query := "CREATE TABLE IF NOT EXISTS " + tableName + "("
		query += "id SERIAL PRIMARY KEY,"
		for i, col := range columns {
			columns[i] = sanitizeColumnName(col)
			query += col + " VARCHAR(255)"
			if i < len(columns)-1 {
				query += ","
			}
		}
		query += ")"

		// Execute query
		_, err := db.Exec(query)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
	}
}
