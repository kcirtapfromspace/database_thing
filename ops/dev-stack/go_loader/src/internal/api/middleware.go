package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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

func Recovery(errorHandler func(*gin.Context, error)) gin.HandlerFunc {
    return func(c *gin.Context) {
        // catch any panics and handle the error
        defer func() {
            if r := recover(); r != nil {
                var err error
                switch x := r.(type) {
                case string:
                    err = errors.NewError(x, http.StatusInternalServerError)
                case error:
                    err = x
                default:
                    err = errors.NewError("Unknown error", http.StatusInternalServerError)
                }
                // handle the error
                errorHandler(c, err)
            }
        }()
        // continue with the request handlers
        c.Next()
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
