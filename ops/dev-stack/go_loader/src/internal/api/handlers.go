package handlers

import (
	"app/validation"
	"encoding/csv"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/internal/config"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/pkg/errors"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/pkg/logger"
)

// Upload handles the file upload and data insertion
func Upload(c *gin.Context) {
    // Get the configuration
    cfg := config.Get()

    // Initialize the logger
    logger := logger.New(cfg.LogLevel)

    // Initialize the error handler
    errorHandler := errors.New(logger)

    // Initialize Multer with progress bar
    uploader, err := multer.New(multer.Options{
        MaxSize:       cfg.MaxFileSize,
        AllowedFormats: []string{"csv"},
        OnProgress: func(info multer.FileInfo, progress int64) {
            c.JSON(http.StatusOK, gin.H{"message": "Upload in progress", "progress": progress})
        },
    })
    if err != nil {
        errorHandler.Handle(c, http.StatusInternalServerError, "Failed to initialize Multer: "+err.Error())
        return
    }

    // Handle the uploaded file
    file, err := uploader.Handle(c)
    if err != nil {
        errorHandler.Handle(c, http.StatusBadRequest, "Failed to handle uploaded file: "+err.Error())
        return
    }

    // Validate the file
    if err := validation.ValidateFile(file); err != nil {
        errorHandler.Handle(c, http.StatusBadRequest, err.Error())
        return
    }

    // Write the file to the volume
    if err := volume.WriteToVolume(file); err != nil {
        errorHandler.Handle(c, http.StatusInternalServerError, "Failed to write file to volume: "+err.Error())
        return
    }

    // Connect to the Postgres database
    db, err := db.Open(cfg.DatabaseURL)
    if err != nil {
        errorHandler.Handle(c, http.StatusInternalServerError, "Failed to connect to database: "+err.Error())
        return
    }
    defer db.Close()

    // Create a new file
    out, err := file.Create(cfg.FilePath + file.Filename)
    if err != nil {
        errorHandler.Handle(c, http.StatusInternalServerError, "Failed to create new file: "+err.Error())
		return
		}
		defer out.Close()
			
	// Copy the uploaded file to the new file
	if _, err := io.Copy(out, file); err != nil {
		errorHandler.Handle(c, http.StatusInternalServerError, "Failed to copy uploaded file: "+err.Error())
		return
	}

	// Open the CSV file
	csvfile, err := os.Open(cfg.FilePath + file.Filename)
	if err != nil {
		errorHandler.Handle(c, http.StatusInternalServerError, "Failed to open CSV file: "+err.Error())
		return
	}
	defer csvfile.Close()

	// Parse the CSV file
	reader := csv.NewReader(csvfile)
	records, err := reader.ReadAll()
	if err != nil {
		errorHandler.Handle(c, http.StatusInternalServerError, "Failed to parse CSV file: "+err.Error())
		return
	}

	// Create the table
	if err := CreateTable(db, file.Filename, records); err != nil {
		errorHandler.Handle(c, http.StatusInternalServerError, "Failed to create table: "+err.Error())
		return
	}

	// Remove the temporary file from the volume
	if err := os.Remove(cfg.FilePath + file.Filename); err != nil {
		errorHandler.Handle(c, http.StatusInternalServerError, "Failed to remove temporary file: "+err.Error())
		return
	}

	// Return success message
	c.JSON(http.StatusOK, gin.H{"message": "File uploaded and data inserted successfully"})
}