package upload

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
)

const maxFileSize = 5 * 1024 * 1024 // 5MB

func handleUpload(c *gin.Context) {
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
        return
    }

    // Handle the uploaded file
    file, err := uploader.Handle(c)
    if err != nil {
        c.String(http.StatusBadRequest, "Failed to handle uploaded file: "+err.Error())
        return
    }

    // Write the file to the volume
    if err := WriteToVolume(file); err != nil {
        c.String(http.StatusInternalServerError, "Failed to write file to volume: "+err.Error())
        return
    }

    // Connect to the Postgres database
    db, err :=openDB( os.Getenv("POSTGRES_DBNAME"))
    if err != nil {
        log.Println(err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to connect to database"})
        return;
    }
    defer db.Close()

    // Check if the file format is csv
    if !strings.HasSuffix(file.Filename, ".csv") {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file format"})
        return;
    }

    // Get the uploaded file
    // file, err := c.FormFile("file")
    if err != nil {
        log.Println(err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get uploaded file"})
        return;
    }
    // Check file size
    if file.Size > maxFileSize {
        c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds the limit"})
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

    if storageMethod != "database" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid storage method"})
    } 

    // Create a new file
    out, err := os.Create("/mnt/data/" + file.Filename)
    if err != nil {
        log.Println(err)
    }
    defer out.Close()

    // Copy the uploaded file to the new file
    if _, err := io.Copy(out, file); err != nil {
        log.Println(err)
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Open the CSV file
    csvfile, err := os.Open("/mnt/data/" + file.Filename)
    if err != nil {
        log.Println(err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to open CSV file"})
        return
    }
    defer csvfile.Close()


    // Parse the CSV file
    reader := csv.NewReader(csvfile)
    records, err := reader.ReadAll()
    if err != nil {
        log.Println(err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse CSV file"})
        return;
    }
    // Initialize the CSV reader
    reader := csv.NewReader(csvfile)

    if err := CreateTable(db, tableName, reader); err != nil {
        log.Println(err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create table in database"})
        return
        }


    // Get the column names from the first row of the CSV
    // var columns []string
    // if len(records) < 1 {
    //     c.JSON(http.StatusBadRequest, gin.H{"error": "The CSV file is empty"})
    //     return;
    // } else if len(records) >= 1 {
    //     columns = records[0]
    // } else {
    //     // If no rows in the file, use numerical column names
    //     columns = make([]string, len(records[0]))
    //     for i := 0; i < len(records[0]); i++ {
    //         columns[i] = fmt.Sprintf("column%d", i+1)
    //     }
    // }

    // Insert the data into the table
    if err := InsertData(db, tableName, reader); err != nil {
        log.Println(err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to insert data into table"})
        return
        }
    // Show upload progress using gin's context
    c.JSON(http.StatusOK, gin.H{"message": "File uploaded and data inserted successfully"})

    //Remove temp file from volume
    if err := RemoveFileFromVolume("/mnt/data/" + filename); err != nil {
        log.Println(err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to remove temp file from volume"})
        return
        }
    }

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
