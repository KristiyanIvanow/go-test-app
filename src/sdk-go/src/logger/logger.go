package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type Logger = zerolog.Logger

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
