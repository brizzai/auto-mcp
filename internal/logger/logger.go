package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/brizzai/auto-mcp/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var globalLogger = zap.NewNop()

// getConsoleEncoder returns a console encoder with optional color support
func getConsoleEncoder(cfg *config.LoggingConfig) zapcore.EncoderConfig {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")

	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	encoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	return encoderConfig
}

// getJSONEncoder returns a JSON encoder
func getJSONEncoder() zapcore.EncoderConfig {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return encoderConfig
}

// InitLogger initializes the global logger with the given configuration
func InitLogger(cfg *config.LoggingConfig) error {
	logger, err := NewLogger(cfg)
	if err != nil {
		return err
	}

	globalLogger = logger
	return nil
}

// NewLogger creates a new zap logger with the given configuration
func NewLogger(cfg *config.LoggingConfig) (*zap.Logger, error) {
	// Set log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %v", err)
	}

	// Configure encoder based on format
	var encoding string
	var encoderConfig zapcore.EncoderConfig
	switch cfg.Format {
	case "json":
		encoding = "json"
		encoderConfig = getJSONEncoder()
	case "console", "":
		encoding = "console"
		encoderConfig = getConsoleEncoder(cfg)
	}

	// Configure output paths
	var outputPaths []string
	var errorOutputPaths []string

	// Add console output if not disabled
	if !cfg.DisableConsole {
		outputPaths = append(outputPaths, "stdout")
		errorOutputPaths = append(errorOutputPaths, "stderr")
	}

	// Handle file output if path is specified
	if cfg.OutputPath != "" {
		// Ensure the directory exists
		dir := filepath.Dir(cfg.OutputPath)
		if dir != "." && dir != "" {
			if err = os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("failed to create log directory %s: %v", dir, err)
			}
		}
		// If append is disabled and file exists, remove it first
		if !cfg.AppendToFile {
			_ = os.Remove(cfg.OutputPath)
		}
		outputPaths = append(outputPaths, cfg.OutputPath)
		errorOutputPaths = append(errorOutputPaths, cfg.OutputPath)
	}

	// Ensure we have at least one output path
	if len(outputPaths) == 0 {
		outputPaths = append(outputPaths, "stdout")
	}
	if len(errorOutputPaths) == 0 {
		errorOutputPaths = append(errorOutputPaths, "stderr")
	}

	zapConfig := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      encoding == "console",
		Encoding:         encoding,
		OutputPaths:      outputPaths,
		ErrorOutputPaths: errorOutputPaths,
		EncoderConfig:    encoderConfig,
	}

	// Build with or without stacktrace based on configuration
	var logger *zap.Logger
	if cfg.DisableStacktrace {
		logger, err = zapConfig.Build(zap.AddCallerSkip(1))
	} else {
		logger, err = zapConfig.Build(
			zap.AddCallerSkip(1),
			zap.AddStacktrace(zapcore.ErrorLevel),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %v", err)
	}

	return logger, nil
}

// GetLogger returns the global logger instance
func GetLogger() *zap.Logger {
	return globalLogger
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	globalLogger.Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	globalLogger.Info(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	globalLogger.Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	globalLogger.Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	globalLogger.Fatal(msg, fields...)
}

// With creates a child logger with additional fields
func With(fields ...zap.Field) *zap.Logger {
	return globalLogger.With(fields...)
}

// Sync flushes any buffered log entries
func Sync() error {
	return globalLogger.Sync()
}
