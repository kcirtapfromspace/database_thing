package config

import (
	"fmt"
	"os"

	"go.uber.org/fx"
)

type ConfigData struct {
	Config *Config
	Err    error
}
type Config struct {
	// DatabaseURL is the URL to connect to the database
	DatabaseURL string
	// DatabaseURL is the URL to connect to the database
	DatabasePassword string

	// MaxFileSize is the maximum allowed file size for uploads
	MaxFileSize int64

	// VolumePath is the path to the volume for storing files
	VolumePath string

	// Debug if the application is running in debug mode
	Debug bool

	// LogLevel is the level of logging for the application
	LogLevel string

	// Port is the port the application is running on
	Port int64

	// Port is the port the web server is listening on
	ListenAddr string
}

var Module = fx.Module("config", fx.Provide(Load))

func Load() (ConfigData, error) {
	// Load the environment variables
	// maxFileSize, _ := strconv.ParseInt(os.Getenv("MAX_FILE_SIZE"), 10, 64)
	// isDebug, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	config := Config{
		// DatabaseURL:      os.Getenv("DATABASE_URL"),
		VolumePath: os.Getenv("VOLUME_MOUNT_PATH"),
		// MaxFileSize:      maxFileSize,
		// Debug:            isDebug,
		// LogLevel:         os.Getenv("LOG_LEVEL"),
		Port:             8000,
		ListenAddr:       ":8000",
		DatabasePassword: os.Getenv("DATABASE_PASSWORD"),
	}
	// Check if all the required environment variables are set
	if config.VolumePath == "" {
		return ConfigData{}, fmt.Errorf("one or more required environment variables are not set")
	}
	return ConfigData{Config: &config, Err: nil}, nil
}
