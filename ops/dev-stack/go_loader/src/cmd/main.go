package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/internal/config"
)

func main() {
    // Initialize Gin
    r := gin.New()

    // Load the configuration
    cfg, err := config.Load()   
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Initialize the logger
    log := logger.InitializeLogger()

    // Initialize the error handler
    errorHandler := errors.New(log)

    // Register the middleware
    r.Use(middleware.LoggingMiddleware(log))
    r.Use(middleware.Recovery(errorHandler))

    // Register the routes
    routes.Register(r, handlers.Upload)

    // Start the server
    r.Run(fmt.Sprintf(":%d", cfg.Port))
}

