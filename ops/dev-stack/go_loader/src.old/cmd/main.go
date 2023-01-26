package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/internal/config"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/internal/middleware"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/internal/routes"
	"github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/pkg/logger"
	api_error "github.com/kcirtapfromspace/database_thing/ops/dev-stack/go_loader/src/pkg/utils"
)

func main() {
    // Initialize Gin
    r := gin.Default()

    // Initialize the logger
    logger.InitializeLogger()
    
    // Load the configuration
    cfg, err := config.Load()   
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Initialize the error handler
    errorHandler := api_error.ErrorHandler

    // Register the middleware
    r.Use(middleware.LoggingMiddleware(logger.Logger))
    r.Use(middleware.Recovery(errorHandler))

    // Register the routes
    routes.SetupRoutes(r)

    // Start the server
    r.Run(fmt.Sprintf(":%d", cfg.Port))
}

