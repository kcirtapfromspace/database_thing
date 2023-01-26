package validation

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// maxFileSize is the maximum file size allowed for upload
const maxFileSize int64 = 1024 * 1024 * 5 // 5MB

// validateRequest validates the request data
func ValidateRequest(c *gin.Context) (string, string, error) {
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
func ValidateFile(c *gin.Context, file *multer.File) error {
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
