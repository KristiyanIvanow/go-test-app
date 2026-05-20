package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  zerolog.Level
	}{
		{"default when unset", "", zerolog.InfoLevel},
		{"trace", "trace", zerolog.TraceLevel},
		{"debug", "debug", zerolog.DebugLevel},
		{"info", "info", zerolog.InfoLevel},
		{"warn", "warn", zerolog.WarnLevel},
		{"error", "error", zerolog.ErrorLevel},
		{"fatal", "fatal", zerolog.FatalLevel},
		{"panic", "panic", zerolog.PanicLevel},
		{"invalid falls back to info", "notvalid", zerolog.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", tt.level)

			got := parseLevel()

			if got != tt.want {
				t.Errorf("parseLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLevel_InvalidWritesWarningToStderr(t *testing.T) {
	t.Setenv("LOG_LEVEL", "notvalid")

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStderr := os.Stderr
	os.Stderr = w

	parseLevel()

	w.Close()
	os.Stderr = origStderr

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "notvalid") {
		t.Errorf("expected stderr to contain the invalid level name, got: %q", buf.String())
	}
}

func TestNew_SetsGlobalLevel(t *testing.T) {
	tests := []struct {
		level string
		want  zerolog.Level
	}{
		{"trace", zerolog.TraceLevel},
		{"debug", zerolog.DebugLevel},
		{"info", zerolog.InfoLevel},
		{"warn", zerolog.WarnLevel},
		{"error", zerolog.ErrorLevel},
		{"", zerolog.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", tt.level)
			// Redirect stdout so New() doesn't pollute test output.
			t.Setenv("LOG_FORMAT", "json")

			New()

			if got := zerolog.GlobalLevel(); got != tt.want {
				t.Errorf("zerolog.GlobalLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew_ReturnsLoggerWithTimestamp(t *testing.T) {
	t.Setenv("LOG_FORMAT", "json")

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStdout := os.Stdout
	os.Stdout = w

	l := New()
	l.Info().Msg("ping")

	w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), `"time"`) {
		t.Errorf("expected timestamp field in log output, got: %q", buf.String())
	}
}

func TestBuildWriter_JSONFormat(t *testing.T) {
	t.Setenv("LOG_FORMAT", "json")

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStdout := os.Stdout
	os.Stdout = w

	writer := buildWriter()
	l := zerolog.New(writer)
	l.Info().Msg("test")

	w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Errorf("expected JSON output (starting with '{'), got: %q", output)
	}
}

func TestBuildWriter_ConsoleFormat(t *testing.T) {
	t.Setenv("LOG_FORMAT", "")

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStdout := os.Stdout
	os.Stdout = w

	writer := buildWriter()
	l := zerolog.New(writer)
	l.Info().Msg("test")

	w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}

	output := strings.TrimSpace(buf.String())
	if strings.HasPrefix(output, "{") {
		t.Errorf("expected console (non-JSON) output, got: %q", output)
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name        string
		setLevel    string
		logFunc     func(*zerolog.Logger)
		shouldLog   bool
		description string
	}{
		{
			name:        "error level filters info logs",
			setLevel:    "error",
			logFunc:     func(l *zerolog.Logger) { l.Info().Msg("info message") },
			shouldLog:   false,
			description: "info message",
		},
		{
			name:        "error level filters warn logs",
			setLevel:    "error",
			logFunc:     func(l *zerolog.Logger) { l.Warn().Msg("warn message") },
			shouldLog:   false,
			description: "warn message",
		},
		{
			name:        "error level allows error logs",
			setLevel:    "error",
			logFunc:     func(l *zerolog.Logger) { l.Error().Msg("error message") },
			shouldLog:   true,
			description: "error message",
		},
		{
			name:        "debug level allows info logs",
			setLevel:    "debug",
			logFunc:     func(l *zerolog.Logger) { l.Info().Msg("info message") },
			shouldLog:   true,
			description: "info message",
		},
		{
			name:        "warn level filters debug logs",
			setLevel:    "warn",
			logFunc:     func(l *zerolog.Logger) { l.Debug().Msg("debug message") },
			shouldLog:   false,
			description: "debug message",
		},
		{
			name:        "warn level allows warn logs",
			setLevel:    "warn",
			logFunc:     func(l *zerolog.Logger) { l.Warn().Msg("warn message") },
			shouldLog:   true,
			description: "warn message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", tt.setLevel)
			t.Setenv("LOG_FORMAT", "json")

			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			origStdout := os.Stdout
			os.Stdout = w

			logger := New()
			tt.logFunc(&logger)

			w.Close()
			os.Stdout = origStdout

			var buf bytes.Buffer
			if _, err := buf.ReadFrom(r); err != nil {
				t.Fatal(err)
			}

			output := buf.String()
			logPresent := strings.Contains(output, tt.description)

			if tt.shouldLog && !logPresent {
				t.Errorf("expected %q to be logged at level %s, but it wasn't", tt.description, tt.setLevel)
			}
			if !tt.shouldLog && logPresent {
				t.Errorf("expected %q not to be logged at level %s, but it was", tt.description, tt.setLevel)
			}
		})
	}
}
