package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

// Init initializes the application logger
func Init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Print directly to console for Docker
	config.OutputPaths = []string{"stdout"}

	// Pretty format the level (INFO, WARN etc)
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// Allow overriding log level via env
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		switch level {
		case "debug", "DEBUG":
			config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		case "info", "INFO":
			config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		case "warn", "WARN":
			config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
		case "error", "ERROR":
			config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
		}
	}

	logger, err := config.Build()
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	Log = logger
}
