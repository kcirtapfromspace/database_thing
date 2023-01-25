package error

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Error represents an error that can be returned in the API response
type Error struct {
    Message string `json:"message"`
    Code    int    `json:"code"`
}

func (e *Error) Error() string {
    return e.Message
}

// NewError creates a new error with the given message and code
func NewError(message string, code int) *Error {
    return &Error{
        Message: message,
        Code:    code,
    }
}

// ErrorHandler is a middleware that handles errors and returns them in the API response
func ErrorHandler(c *gin.Context, err error) {
    var apiError *Error
    if ok := errors.As(err, &apiError); ok {
        c.JSON(apiError.Code, gin.H{"error": apiError.Message})
    } else {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
        fmt.Printf("Unexpected error: %v\n", err)
    }
}

func Recovery() gin.HandlerFunc {
    return func(c *gin.Context) {
        // catch any panics and handle the error
        defer func() {
            if r := recover(); r != nil {
                var err error
                switch x := r.(type) {
                case string:
                    err = errors.New(x)
                case error:
                    err = x
                default:
                    err = errors.New("Unknown error")
                }
                // handle the error
                utils.ErrorHandler(c, err)
            }
        }()
        // continue with the request handlers
        c.Next()
    }
}

