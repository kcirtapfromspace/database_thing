package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	// DatabaseURL is the URL to connect to the database
	DatabaseURL string

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
}

func Load() (config Config, err error) {
	// Load the environment variables
	maxFileSize, _ := strconv.ParseInt(os.Getenv("MAX_FILE_SIZE"), 10, 64)
	isDebug, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	config.DatabaseURL = os.Getenv("DATABASE_URL")
	config.VolumePath = os.Getenv("VOLUME_MOUNT_PATH")
	config.MaxFileSize = maxFileSize
	config.Debug = isDebug
	config.Port = 8000
	// Check if all the required environment variables are set
	if config.DatabaseURL == "" || config.MaxFileSize == 0 || config.VolumePath == "" || config.LogLevel == "" || config.Port == 0 || config.Debug == false {
		return config, fmt.Errorf("one or more required environment variables are not set")
	}
	return config, nil
}
