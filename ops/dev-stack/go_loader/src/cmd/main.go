package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/internal/api/handlers"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/internal/api/middleware"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/internal/api/routes"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/internal/config"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/pkg/errors"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/pkg/logger"
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
    logger.InitializeLogger()

    // Initialize the error handler
    errorHandler := errors.ErrorHandler()

    // Register the middleware
    r.Use(middleware.LoggingMiddleware(logger.Logger))
    r.Use(middleware.Recovery(errorHandler))

    // Register the routes
    routes.Register(r, handlers.Upload)

    // Start the server
    r.Run(fmt.Sprintf(":%d", cfg.Port))
}

