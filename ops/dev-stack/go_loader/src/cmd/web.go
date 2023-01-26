package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/internal/db"
	_ "github.com/lib/pq"
)

const (
	host   = "postgres-db"
	port   = 5432
	user   = "postgres"
	dbname = "postgres"
)

func main() {

	// Initialize Gin web server
	r := gin.Default()
	r.LoadHTMLGlob("static/templates/*")
	r.Static("/static", "./static")
	// Serve file upload form
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "upload.tmpl", nil)
	})
	// Check connection status
	r.GET("/status", func(c *gin.Context) {
		if err := db.Ping(); err != nil {
			log.Fatalf("Error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"status": "disconnected"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "connected"})
		if err := db.Ping(); err != nil {
			fmt.Println("Error:", err)
			return
		}
	})

	// Handle file upload
	r.POST("/upload", func(c *gin.Context) {
		// Get file from form
		file, _ := c.FormFile("file")
		// Create destination file
		dst := "./data/files/" + file.Filename
		// Upload file
		c.SaveUploadedFile(file, dst)
		// Read file contents
		data, err := ioutil.ReadFile(dst)
		if err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("Error reading file: %s", err))
			return
		}

		// Use functions from file package to process the CSV file
		records, err := file.ReadCSV(data)
		if err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("Error processing CSV file: %s", err))
			return
		}

		// Use functions from db package to create table and insert data
		if err := db.CreateTable(file.Filename, records[0]); err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("Error creating table: %s", err))
			return
		}
		if err := db.InsertData(file.Filename, records[1:]); err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("Error inserting data: %s", err))
			return
		}
		// Use functions from file package to remove temp file
		if err := file.Remove("./data/" + file.Filename); err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("Error removing temp file: %s", err))
			return
		}
		// Send success message to user
		c.JSON(http.StatusOK, gin.H{"message": "File uploaded and processed successfully"})
	})
	// Start server
	r.Run(":8000")
}
