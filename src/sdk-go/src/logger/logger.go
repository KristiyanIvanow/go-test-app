// Package logger provides the logging interface used by the SDK
// and a default stdout-based implementation.
package logger

import (
	"log"
	"os"
)

// ILogger is the logging interface used across the SDK.
type ILogger interface {
	Information(message string)
	Warning(message string)
	Error(message string)
	Debug(message string)
}

// ConsoleLogger is a default ILogger implementation that writes to stdout.
type ConsoleLogger struct {
	logger *log.Logger
}

// NewConsoleLogger creates a new ConsoleLogger.
func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (l *ConsoleLogger) Information(message string) { l.logger.Println("[INFO] " + message) }
func (l *ConsoleLogger) Warning(message string)     { l.logger.Println("[WARN] " + message) }
func (l *ConsoleLogger) Error(message string)       { l.logger.Println("[ERROR] " + message) }
func (l *ConsoleLogger) Debug(message string)       { l.logger.Println("[DEBUG] " + message) }

// Default is the default logger instance used by the SDK when no
// custom logger is provided.
var Default ILogger = NewConsoleLogger()
