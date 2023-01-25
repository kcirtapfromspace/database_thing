package logger

import (
	"log"
	"time"

	"go.uber.org/zap"
)
var Logger *zap.Logger

func InitializeLogger() {
	var err error
	// config := zap.NewProductionEncoderConfig()
	Logger, _ = zap.NewProduction() // NewProduction, or NewDevelopment
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer Logger.Sync()
	sugar := Logger.Sugar()
	sugar.Infow("Failed to fetch URL.",
		// Structured context as loosely typed key-value pairs.
		"url", "https://example.com",
		"attempt", 3,
		"backoff", time.Second,
	)
	sugar.Infof("Failed to fetch URL: %s", "https://example.com")
}