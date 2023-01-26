package handlers

import (
	"encoding/csv"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/kcirtapfromspace/database_thing/internal/config"
	"github.com/kcirtapfromspace/database_thing/internal/db"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/internal/validation"
	"github.com/kcirtapfromspace/database_thing/pkg/logger"
	"github.com/kcirtapfromspace/database_thing/pkg/validation"
	"github.com/kcirtapfromspace/database_thing/pkg/volume"
	"go.uber.org/zap"
)

func Upload(c *gin.Context) {
	// Get the configuration
	cfg := config.Load()

	// Initialize the logger
	log := logger.New(cfg.LogLevel)

	// Get the file from the form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		log.Error("Failed to get file from form", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"message": "Failed to get file from form: " + err.Error()})
		return
	}
	defer file.Close()

	// Validate the file
	if err := validation.ValidateFile(header); err != nil {
		log.Error("Failed to validate file", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	// Write the file to the volume
	if err := volume.WriteToVolume(header.Filename, file); err != nil {
		log.Error("Failed to write file to volume", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to write file to volume: " + err.Error()})
		return
	}

	// Connect to the Postgres database
	db, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Error("Failed to connect to database", zap.Error(err), zap.String("error",err.Error()))
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to connect to database: "+err.Error()})
        return
        }
    // Create a new file
    out, err := os.Create(cfg.FilePath + file.Filename)
    if err != nil {
        log.Error("Failed to create new file", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create new file: "+err.Error()})
        return
    }
    defer out.Close()

    // Copy the uploaded file to the new file
    if _, err := io.Copy(out, file); err != nil {
        log.Error("Failed to copy uploaded file", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to copy uploaded file: "+err.Error()})
        return
    }

    // Open the CSV file
    f, err := os.Open(cfg.FilePath + file.Filename)
    if err != nil {
        log.Error("Failed to open CSV file", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to open CSV file: "+err.Error()})
        return
    }
    defer f.Close()

    // Read the CSV data
    data, err := csv.NewReader(f).ReadAll()
    if err != nil {
        log.Error("Failed to read CSV data", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to read CSV data: "+err.Error()})
        return
    }

    // Create a new table in the database
    if err := db.CreateTable(file.Filename, data[0]); err != nil {
        log.Error("Failed to create new table", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create new table: "+err.Error()})
        return
    }
    // Open the CSV file
    f, err := os.Open(cfg.FilePath + file.Filename)
    if err != nil {
    log.Error("Failed to open CSV file", zap.Error(err))
    c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to open CSV file: "+err.Error()})
    return
    }
    defer f.Close()
    // Create a new table in the database
    if err := db.CreateTable(file.Filename); err != nil {
        log.Error("Failed to create table in database", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create table in database: "+err.Error()})
        return
    }

    // Parse the CSV file
    rows, err := csv.NewReader(f).ReadAll()
    if err != nil {
        log.Error("Failed to parse CSV file", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse CSV file: "+err.Error()})
        return
    }

    // Insert the data into the table
    if err := db.InsertData(file.Filename, rows); err != nil {
        log.Error("Failed to insert data into table", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to insert data into table: "+err.Error()})
        return
    }

    // Remove the file from the volume
    if err := volume.RemoveFromVolume(file); err != nil {
        log.Error("Failed to remove file from volume", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to remove file from volume: "+err.Error()})
        return
    }

    // Return success
    c.JSON(http.StatusOK, gin.H{"message": "Upload succeeded"})
}
