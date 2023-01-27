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
	"go.uber.org/fx"
)

var Module = fx.Module("db", fx.Provide(ConnectDB))

const (
	host     = "postgres-db"
	port     = 5432
	user     = "go_loader"
	dbname   = "postgres"
	password = "go_loader"
)

// Connect to the PostgreSQL database
func ConnectDB() (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
		os.Exit(1)
	}
	return db, nil
}

func SanitizeName(columnName string) string {
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

func InsertRecord(db *sql.DB, tableName string, data [][]string) error {
	// Prepare insert query
	query := "INSERT INTO " + tableName + "("
	for i, col := range data[0] {
		query += SanitizeName(col)
		if i < len(data[0])-1 {
			query += ","
		}
	}
	query += ") VALUES("
	for i := 0; i < len(data[0]); i++ {
		query += "$" + strconv.Itoa(i+1)
		if i < len(data[0])-1 {
			query += ","
		}
	}
	query += ")"
	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("error preparing insert query: %v", err)
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
			log.Printf("Error: %v", err)
		}
	}
	return nil
}

func CreateTable(db *sql.DB, tableName string, columns []string) error {
	// Create table query
	query := "CREATE TABLE IF NOT EXISTS " + tableName + "("
	query += "id SERIAL PRIMARY KEY,"
	for i, col := range columns {
		columns[i] = SanitizeName(col)
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
	return err
}

func GetAllData(db *sql.DB, tableName string) ([]map[string]interface{}, error) {
	rows, err := db.Query("SELECT * FROM " + tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []map[string]interface{}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			row[col] = val
		}
		data = append(data, row)
	}
	return data, nil
}
