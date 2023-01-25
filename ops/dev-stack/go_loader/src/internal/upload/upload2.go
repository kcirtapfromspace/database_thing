package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/stdlib"
)

const maxFileSize = 5 * 1024 * 1024 // 5MB

// handleFileUpload handles the file upload process and writes the file to the volume
func handleFileUpload(c *gin.Context) (*multer.File, error) {
    // Initialize Multer with progress bar
    uploader, err := multer.New(multer.Options{
        MaxSize:       maxFileSize,
        AllowedFormats: []string{"csv"},
        OnProgress: func(info multer.FileInfo, progress int64) {
            c.JSON(http.StatusOK, gin.H{"message": "Upload in progress", "progress": progress})
        },
    })
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to initialize Multer: "+err.Error())
        return nil, err
    }

    // Handle the uploaded file
    file, err := uploader.Handle(c)
    if err != nil {
        c.String(http.StatusBadRequest, "Failed to handle uploaded file: "+err.Error())
        return nil, err
    }

    // Write the file to the volume
    if err := WriteToVolume(file); err != nil {
        c.String(http.StatusInternalServerError, "Failed to write file to volume: "+err.Error())
        return nil, err
    }
    return file, nil
}

// openDb connects to the database
func openDb() (*sqlx.DB, error) {
    db, err := sqlx.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        return nil, err
    }
    return db, nil
}

// validateRequest validates the request data
func validateRequest(c *gin.Context) (string, string, error) {
    storageMethod := c.PostForm("storage_method")
    customName := c.PostForm("custom_name")
    if storageMethod != "database" {
        return "", "", fmt.Errorf("Invalid storage method")
    }
    if customName == "" {
        return storageMethod, customName, nil
    }
    return storageMethod, customName, nil
}

// validateFile validates the file
func validateFile(c *gin.Context, file *multer.File) error {
    if !strings.HasSuffix(file.Filename, ".csv") {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file format"})
        return fmt.Errorf("Invalid file format")
    }
    if file.Size > maxFileSize {
        c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds the limit"})
        return fmt.Errorf("File size exceeds the limit")
    }
    return nil
}

// parseCSVFile parses the csv file
func parseCSVFile(filepath string) ([][]string, error)
    // Open the CSV file
    csvfile, err := os.Open(filepath)
    if err != nil {
        return nil, err
    }
    defer csvfile.Close()

    // Parse the CSV file
    reader := csv.NewReader(csvfile)
    records, err := reader.ReadAll()
    if err != nil {
        return nil, err
    }

    return records, nil
}

// createTable creates the table and inserts the data into the table
func createTable(db *sqlx.DB, tableName string, records [][]string) error {
    func CreateTable(c *gin.Context, db *sqlx.DB, tableName string, filepath string) error {
        f, err := os.Open(filepath)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file: "+err.Error()})
            return err
        }
        defer f.Close()
    
        // check file format
        if !strings.HasSuffix(filepath, ".csv") {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid file format, only csv is supported"})
            return errors.New("Invalid file format")
        }
        reader := csv.NewReader(f)
    
        // Get the header row
        header, err := reader.Read()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read header: "+err.Error()})
            return err
        }
    
        // Create the table
        tx, err := db.Begin()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction: "+err.Error()})
            return err
        }
    
        _, err = tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to drop table: "+err.Error()})
            return err
        }
    
        query := fmt.Sprintf("CREATE TABLE %s (", tableName)
        for _, column := range header {
            query += fmt.Sprintf("%s TEXT,", column)
        }
        query = strings.TrimSuffix(query, ",") + ");"
    
        _, err = tx.Exec(query)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create table: "+err.Error()})
            return err
        }
    
        // Insert the data into the table
        stmt, err := tx.Prepare(fmt.Sprintf("INSERT INTO %s VALUES (", tableName))
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare statement: "+err.Error()})
            return err
        }
        defer stmt.Close()
    
        for {
            record, err := reader.Read()
            if err == io.EOF {
                break
            } else if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read CSV file: "+err.Error()})
                return err
            }
            var query string
            for _, value := range record {
                query += fmt.Sprintf("?,")
            }
            query = strings.TrimSuffix(query, ",") + ")"
            _, err = stmt.Exec(record...)
            if err != nil {c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert data into table: "+err.Error()})
            return err
        }
    }

    if err := tx.Commit(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: "+err.Error()})
        return err
    }

    // Remove the temporary file from the volume
    if err := os.Remove(filepath); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove temporary file: "+err.Error()})
        return err
    }

    // Return success message
    c.JSON(http.StatusOK, gin.H{"message": "File uploaded and data inserted successfully"})
    return nil
}