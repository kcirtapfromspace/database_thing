package web

import (
	"log"
	"net/http"

	"go.uber.org/fx"

	"database_thing/pkg/filepkg"

	"github.com/gin-gonic/gin"

	_ "github.com/lib/pq"
)

var Module = fx.Module("web", fx.Provide(ListenAndServe))

type Server struct {
}

// func ListenAndServe(address string) {
func ListenAndServe(address string) *Server {
	// srv := &http.Server{Addr: address}
	// go func() {
	// 	if err := srv.ListenAndServe(); err != nil {
	// 		log.Println(err)
	// 	}
	// }()
	r := gin.Default()
	r.LoadHTMLGlob("static/templates/*")
	r.Static("/static", "./static")
	// Serve file upload form
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "upload.tmpl", nil)
	})
	// Handle file upload
	r.POST("/upload", func(c *gin.Context) {
		file, _ := c.FormFile("file")
		dst := "./data/files/" + file.Filename
		fileName := file.Filename
		if filepkg.FileUploadHandler(c, file, fileName, dst) {
			// Send success message to user
			c.JSON(http.StatusOK, gin.H{"message": "File uploaded and processed successfully"})
		} else {
			// Send error message to user
			c.JSON(http.StatusBadRequest, gin.H{"message": "Error processing file"})
		}
	})
	// start the server
	go func() {
		if err := r.Run(address); err != nil {
			log.Println(err)
		}
	}()

	return &Server{}
}

// Check connection status
// r.GET("/status", func(c *gin.Context) {
// 	if err := db.Ping(); err != nil {
// 		log.Fatalf("Error: %v", err)
// 		c.JSON(http.StatusInternalServerError, gin.H{"status": "disconnected"})
// 		return
// 	} else {
// 		c.JSON(http.StatusOK, gin.H{"status": "connected"})
// 		return
// 	}
// })
// Start server
