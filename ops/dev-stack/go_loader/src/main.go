package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
    // Create a new Gin router
    router := gin.Default()

    // Create a route for the file upload form
    router.LoadHTMLGlob("templates/*") // load the HTML templates
	router.Static("/style", "./style/")

    router.GET("/", func(c *gin.Context) {
        c.HTML(http.StatusOK, "index.html", nil)
    })

    // Create a route to handle the file upload
    router.POST("/upload", handleUpload)

    // Start the Gin server
    router.Run(":8000")
}
