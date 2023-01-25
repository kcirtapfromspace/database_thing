package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func LoggingMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        end := time.Now()
        Logger.Info("request details",
            zap.String("Method", c.Request.Method),
            zap.String("URL", c.Request.URL.String()),
            zap.Int("Status", c.Writer.Status()),
            zap.Duration("elapsed time", end.Sub(start)),
        )
    }
}


// AuthenticationMiddleware checks if the user is authenticated
func AuthenticationMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // check if the user is authenticated
        if !isAuthenticated(c) {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            c.Abort()
            return
        }
        c.Next()
    }
}

func isAuthenticated(c *gin.Context) bool {
    // check if the user is authenticated here
    // ...
    return true
}
