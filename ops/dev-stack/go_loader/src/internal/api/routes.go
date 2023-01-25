package routes

import (
	"app/file"
	"app/upload"
	"app/validation"

	"github.com/gin-gonic/gin"
)

// SetupRoutes sets up the application's routes
func SetupRoutes(router *gin.Engine) {
    // Serve the index page
    router.LoadHTMLGlob("templates/*")
    router.GET("/", func(c *gin.Context) {
        c.HTML(200, "index.html", nil)
    })

    // Handle file uploads
    router.POST("/upload", validation.ValidateFile, upload.HandleUpload,file.RemoveFile)
}
