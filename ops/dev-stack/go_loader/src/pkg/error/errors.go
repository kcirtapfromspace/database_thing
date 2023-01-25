package error

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/pkg/logger"
	"go.uber.org/zap"
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
        logger.Logger.Error("Unexpected error: %v\n", zap.Error(err))
    }
}

