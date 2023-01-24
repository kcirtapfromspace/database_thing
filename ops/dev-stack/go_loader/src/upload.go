package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/stdlib"
)

func handleUpload(c *gin.Context) {
    // Connect to the Postgres database
    db, err :=openDB()
    if err != nil {
        log.Println(err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to connect to database"})
        return
    }
    defer db.Close()

    // Get the uploaded file
    file, err := c.FormFile("file")
    if err != nil {
        log.Println(err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get uploaded file"})
        return
    }
    storageMethod := c.PostForm("storage_method")
    customName := c.PostForm("custom_name")
    var tableName string
    if customName == "" {
    tableName = file.Filename
    }else{
    tableName = customName
    }

    if storageMethod == "table" {
        // Create a new table with the tableName for the CSV file
        createTableQuery := fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY, name VARCHAR(255), email VARCHAR(255))", tableName)
        _, err := db.Exec(createTableQuery, tableName)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
    } else if storageMethod == "database" {
        // Create a new database with the tableName for the CSV file
        createDbQuery := fmt.Sprint("CREATE DATABASE ", tableName)
        _, err := db.Exec(createDbQuery, tableName)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
    } else if storageMethod == "update" {
        // Update existing table or database
        updateTableQuery := fmt.Sprintf("UPDATE %s SET column1 = value1 WHERE condition", tableName)
        _, err := db.Exec(updateTableQuery)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
    }else {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid storage method"})
        return
    }

    filename := file.Filename
    if filename == "" {
        log.Println("filename is not defined or initialized")
        c.JSON(http.StatusBadRequest, gin.H{"error": "filename is not defined or initialized"})
        return
    }

    // Create a new file
    out, err := os.Create(filename)
    if err != nil {
        log.Println(err)
    }
    defer out.Close()

    // Copy the uploaded file to the new file
    err = c.SaveUploadedFile(file, filename)
    if err != nil {
        log.Println(err)
    }
    c.JSON(http.StatusOK, gin.H{"message": "File uploaded successfully"})
    // Open the CSV file
    csvfile, err := os.Open(filename)
    if err != nil {
        log.Println(err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to open CSV file"})
    }
    defer csvfile.Close()

    // Parse the CSV file
    reader := csv.NewReader(csvfile)
    records, err := reader.ReadAll()
    if err != nil {
        log.Println(err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse CSV file"})
        return
    }


    // Get the column names from the first row of the CSV
    var columns []string
    if len(records) >= 1 {
        columns = records[0]
    } else {
        // If no rows in the file, use numerical column names
        columns = make([]string, len(records[0]))
        for i := 0; i < len(records[0]); i++ {
            columns[i] = fmt.Sprintf("column%d", i+1)
        }
    }

    // Insert the data into the table/database
    if storageMethod == "table" {
        for i, record := range records {
            if i == 0 {
                continue
            }
            // Convert record from []string to []interface{}
            interfaceRecord := make([]interface{}, len(record))
            for i, v := range record {
                interfaceRecord[i] = v
            }
            insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, strings.Join(columns, ","), placeholders(len(columns)))
            _, err := db.Exec(insertQuery, interfaceRecord...)
            if err != nil {
                log.Println(err)
                c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to insert data into table"})
                return
            }
        }
    } else if storageMethod == "database" {
        // Insert the data into the newly created database
        for i, record := range records {
            if i == 0 {
                continue
            }
            // Convert record from []string to []interface{}
            interfaceRecord := make([]interface{}, len(record))
            for i, v := range record {
                interfaceRecord[i] = v
            }
            insertQuery := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s)", tableName, tableName, strings.Join(columns, ","), placeholders(len(columns)))
            _, err := db.Exec(insertQuery, interfaceRecord...)
            if err != nil {
                log.Println(err)
                c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to insert data into database"})
                return
            }
        }
    } else if storageMethod == "update" {
        // update data in the existing table or database
        for i, record := range records {
            if i == 0 {
                continue
            }
              // Convert record from []string to []interface{}
              interfaceRecord := make([]interface{}, len(record))
              for i, v := range record {
                  interfaceRecord[i] = v
              }
            updateQuery := fmt.Sprintf("UPDATE %s SET column1 = value1 WHERE condition", tableName)
            _, err := db.Exec(updateQuery, interfaceRecord...)
            if err != nil {
                log.Println(err)
                c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to update data in table/database"})
                return
            }
        }
    }

    // Iterate through the remaining rows of the CSV
    colVals := make([]interface{}, len(columns))
    args := make([]interface{}, len(colVals))
    for i := range columns {
        args[i] = &colVals[i]
    }
    insertDataQuery := "INSERT INTO " + tableName + " (" + strings.Join(columns, ",") + ") VALUES (" + strings.TrimRight(strings.Repeat("$", len(columns)), ",") + ")"
    stmt, err := db.Prepare(insertDataQuery)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    defer stmt.Close()

    for _, record := range records[1:] {
        for i, val := range record {
            colVals[i] = val
        }
        _, err := stmt.Exec(args...)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
    }
}