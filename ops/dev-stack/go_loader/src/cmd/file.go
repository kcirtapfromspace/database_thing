package file

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type MyStruct struct {
	// struct fields
}

// FileUploadHandler handles file uploads
func FileUploadHandler(c *gin.Context, db *sqlx.DB) {
	// Get file from form
	file, _ := c.FormFile("file")
	// Create destination file
	dst := "static/files/" + file.Filename
	// Upload file
	c.SaveUploadedFile(file, dst)
	// Process CSV file
	if err := ProcessCSVFile(db, dst); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Error processing CSV file: %s", err))
		return
	}
	// Respond with success message
	c.String(http.StatusOK, "File uploaded and processed successfully")
}

// ReadCSV reads a CSV file from a byte slice and returns a slice of records
func ReadCSV(data []byte) ([][]string, error) {
	// Create a new CSV reader
	reader := csv.NewReader(strings.NewReader(string(data)))
	// Read all records from the CSV file
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	return records, nil
}

func Remove(filePath string) error {
	return os.Remove(filePath)
}

func ProcessCSVFile(db *sqlx.DB, filePath string) error {
	// Open CSV file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Error opening CSV file: %s", err)
	}
	defer file.Close()

	// Read CSV file into a byte slice
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("Error reading CSV file: %s", err)
	}

	// Use functions from file package to process the CSV file
	records, err := ReadCSV(data)
	if err != nil {
		return fmt.Errorf("Error processing CSV file: %s", err)
	}

	// Use functions from db package to create table and insert data
	if err := db.CreateTable(file, records[0]); err != nil {
		return fmt.Errorf("Error creating table: %s", err)
	}
	if err := db.InsertData(file, records[1:]); err != nil {
		return fmt.Errorf("Error inserting data: %s", err)
	}

	// Use functions from file package to remove temp file
	if err := Remove(filePath); err != nil {
		return fmt.Errorf("Error removing temp file: %s", err)
	}

	return nil
}
