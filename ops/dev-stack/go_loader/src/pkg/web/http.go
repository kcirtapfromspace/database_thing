package web

import (
	"expvar"
	"fmt"
	"log"
	"net/http"

	"go.uber.org/fx"

	"database_thing/pkg/filepkg"

	"github.com/gin-gonic/gin"

	_ "github.com/lib/pq"
)

func expvarHandler(c *gin.Context) {
	w := c.Writer
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{\n")
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}

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
	r := gin.New()
	// r.Use(gin.Recovery())
	// if gin.IsDebugging() {
	// 	r.HTMLRender = r.NewDebug("resources")
	// } else {
	// 	r.HTMLRender = r.NewProduction("resources")
	// }
	r.Static("/static", "/static")
	r.LoadHTMLGlob("./static/templates/*")
	r.GET("/debug/vars", expvarHandler)
	// Serve file upload form
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", nil)
	})

	// Handle file upload
	r.POST("/upload", func(c *gin.Context) {
		file, _ := c.FormFile("file")
		// Check file type
		allowedTypes := map[string]bool{
			"text/csv": true,
			// "text/css":        true,
			// "text/javascript": true,
		}
		if !allowedTypes[file.Header.Get("Content-Type")] {
			c.JSON(http.StatusBadRequest, gin.H{"message": "File type not allowed"})
			return
		}
		dst := "./data/"
		// dst := "/tmp/" //+ file.Filename
		fileName := file.Filename
		if filepkg.FileUploadHandler(c, file, dst, fileName) {
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
