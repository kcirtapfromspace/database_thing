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
	if _, err := os.Stat("/data"); os.IsNotExist(err) {
		os.Mkdir(dst, os.ModePerm)
	}
	dst = dst + fileName
	// Save file to disk
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
		return false
	}
	// Process CSV file
	if err := ProcessCSVFile(dst, fileName); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("error processing CSV file: %s", err))
		return false
	}

	// Respond with success message
	c.String(http.StatusOK, fmt.Sprintf("file %s uploaded successfully with fields .", file.Filename))
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

func ProcessCSVFile(dst string, fileName string) error {
	columnTypes := make(map[string]string)
	tableName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	cleanTableName := SanitizeName(tableName)
	if tableName == "" {
		return fmt.Errorf("error: empty file name")
	}
	dbConnect, err := db.ConnectDB()
	if err != nil {
		return fmt.Errorf("error connecting to the database: %s", err)
	}
	defer dbConnect.Close()

	// Open CSV fil
	file, err := os.Open(dst)
	if err != nil {
		return fmt.Errorf("error opening CSV file: %s", err)
	}
	defer file.Close()

	// Read CSV file into a byte slice
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("error reading CSV file: %s", err)
	}

	// Use functions from db package to insert records
	records, err := ReadCSV(data)
	if err != nil {
		return fmt.Errorf("error reading CSV file: %s", err)
	}
	records[0] = db.SanitizeHeaders(records[0])

	// Use functions from db package determine column name for map columnTypes
	columnNames := db.DetermineColumnNames(records)
	if err != nil {
		return fmt.Errorf("error determining the column names and types of csv file: %s", err)
	}
	columnNamesSlice := make([]string, 0, len(columnNames))
	for key := range columnNames {
		columnNamesSlice = append(columnNamesSlice, key)
	}
	columnTypes = db.InferColumnTypes(records, columnNamesSlice)

	// Use functions from db package determine update map columnTypes with appropriate column types
	if err != nil {
		return fmt.Errorf("error determining the column types of csv file: %s", err)
	}

	// Use functions from db package to create table
	if err := db.CreateTable(dbConnect, cleanTableName, records[0], columnTypes); err != nil {
		return fmt.Errorf("error creating table: %s", err)
	}

	interfaceRecords := make([][]interface{}, len(records[1:]))
	for i, record := range records[1:] {
		interfaceRecord := make([]interface{}, len(record))
		for j, item := range record {
			interfaceRecord[j] = item
		}
		interfaceRecords[i] = interfaceRecord
	}

	columnNamesSlice = make([]string, 0, len(columnNames))
	for columnName := range columnNames {
		columnNamesSlice = append(columnNamesSlice, columnName)
	}

	// Use functions from db package to insert records
	if err := db.PopulateTable(dbConnect, cleanTableName, interfaceRecords, columnNamesSlice, 1000); err != nil {
		return fmt.Errorf("error inserting records: %s", err)
	}

	// Use functions from file package to remove temp file
	if err := Remove(dst); err != nil {
		return fmt.Errorf("error removing temp file: %s", err)
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
