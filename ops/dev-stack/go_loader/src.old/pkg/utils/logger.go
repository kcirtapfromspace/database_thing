package logger

import (
	"log"

	"go.uber.org/zap"
)
var Logger *zap.Logger

func InitializeLogger() *zap.Logger {
    var err error
    Logger, err = zap.NewProduction() // NewProduction, or NewDevelopment
    if err != nil {
        log.Fatalf("can't initialize zap logger: %v", err)
    }
    defer Logger.Sync()
    return Logger
}
