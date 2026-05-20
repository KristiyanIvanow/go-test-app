package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Level is the SDK logger level abstraction used by examples and constructors.
type Level int

const (
	Debug Level = iota
	Information
	Warning
	Error
)

// Info is kept as an alias for callers that use the shorter name.
const Info = Information

// ILogger is the SDK logging interface used throughout the Go SDK.
type ILogger interface {
	Information(message string)
	Warning(message string)
	Error(message string)
	Debug(message string)
}

// Default is the package-level logger used when no custom logger is supplied.
var Default ILogger = NewConsoleLogger()

type zeroLogger struct {
	logger zerolog.Logger
}

// New creates a zerolog.Logger configured from environment variables.
//
// LOG_LEVEL  — one of trace, debug, info, warn, error, fatal, panic (default: info)
// LOG_FORMAT — set to "json" for structured JSON output (default: colored console)
func New() zerolog.Logger {
	level := parseLevel()
	zerolog.SetGlobalLevel(level)

	writer := buildWriter()
	return zerolog.New(writer).With().Timestamp().Logger()
}

// NewConsoleLogger returns an ILogger backed by zerolog.
//
// When a level is provided, it overrides LOG_LEVEL for this logger instance.
func NewConsoleLogger(levels ...Level) ILogger {
	base := New()
	if len(levels) > 0 {
		base = base.Level(toZeroLevel(levels[0]))
	}
	return &zeroLogger{logger: base}
}

func (z *zeroLogger) Information(message string) {
	z.logger.Info().Msg(message)
}

func (z *zeroLogger) Warning(message string) {
	z.logger.Warn().Msg(message)
}

func (z *zeroLogger) Error(message string) {
	z.logger.Error().Msg(message)
}

func (z *zeroLogger) Debug(message string) {
	z.logger.Debug().Msg(message)
}

func parseLevel() zerolog.Level {
	raw := os.Getenv("LOG_LEVEL")
	if raw == "" {
		return zerolog.InfoLevel
	}

	level, err := zerolog.ParseLevel(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: unknown LOG_LEVEL %q, falling back to info\n", raw)
		return zerolog.InfoLevel
	}

	return level
}

func buildWriter() zerolog.LevelWriter {
	if os.Getenv("LOG_FORMAT") == "json" {
		return zerolog.MultiLevelWriter(os.Stdout)
	}

	return zerolog.MultiLevelWriter(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.Stamp,
	})
}

func toZeroLevel(level Level) zerolog.Level {
	switch level {
	case Debug:
		return zerolog.DebugLevel
	case Warning:
		return zerolog.WarnLevel
	case Error:
		return zerolog.ErrorLevel
	case Information:
		fallthrough
	default:
		return zerolog.InfoLevel
	}
}
