package filepkg

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"go.uber.org/fx"

	"database_thing/pkg/db"

	"github.com/gin-gonic/gin"
)

var Module = fx.Module("file", fx.Provide(FileUploadHandler))

func FileUploadHandler(c *gin.Context, file *multipart.FileHeader, dst string, fileName string) bool {
	// Upload file
	c.SaveUploadedFile(file, dst)

	// Process CSV file
	if err := ProcessCSVFile(dst, fileName); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Error processing CSV file: %s", err))
		return false
	}

	// Respond with success message
	c.String(http.StatusOK, "File uploaded and processed successfully")
	return true
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

func ProcessCSVFile(filePath string, fileName string) error {
	tableName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	cleanTableName := SanitizeName(tableName)
	if tableName == "" {
		return fmt.Errorf("Error: empty file name")
	}
	dbConnect, err := db.ConnectDB()
	if err != nil {
		return fmt.Errorf("Error connecting to the database: %s", err)
	}
	defer dbConnect.Close()

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

	// Use functions from db package to insert records
	records, err := ReadCSV(data)
	if err != nil {
		return fmt.Errorf("Error reading CSV file: %s", err)
	}

	// Use functions from db package to create table
	if err := db.CreateTable(dbConnect, cleanTableName, records[0]); err != nil {
		return fmt.Errorf("Error creating table: %s", err)
	}

	// Use functions from db package to insert records
	if err := db.InsertRecord(dbConnect, cleanTableName, records[1:]); err != nil {
		return fmt.Errorf("Error inserting records: %s", err)
	}

	// Use functions from file package to remove temp file
	if err := Remove(filePath); err != nil {
		return fmt.Errorf("Error removing temp file: %s", err)
	}

	return nil
}

func SanitizeName(dirtyName string) string {
	dirtyName = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return -1
	}, dirtyName)
	dirtyName = strings.ToLower(dirtyName)
	dirtyName = strings.Replace(dirtyName, ".", "_", -1)
	dirtyName = strings.Replace(dirtyName, " ", "_", -1)
	cleanName := dirtyName
	return cleanName
}
