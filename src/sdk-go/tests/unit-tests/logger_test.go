package unit_tests_test

import (
	"testing"

	"github.com/KristiyanIvanow/go-test-app/src/logger"
)

func TestDefaultLoggerImplementsInterface(t *testing.T) {
	var _ logger.ILogger = logger.Default
}

func TestNewConsoleLoggerImplementsInterface(t *testing.T) {
	var _ logger.ILogger = logger.NewConsoleLogger()
}

// captureLogger records all messages it receives so we can verify
// callers route the right messages to the right level.
type captureLogger struct {
	info, warn, err, debug []string
}

func (c *captureLogger) Information(m string) { c.info = append(c.info, m) }
func (c *captureLogger) Warning(m string)     { c.warn = append(c.warn, m) }
func (c *captureLogger) Error(m string)       { c.err = append(c.err, m) }
func (c *captureLogger) Debug(m string)       { c.debug = append(c.debug, m) }

func TestCustomLoggerSatisfiesInterface(t *testing.T) {
	var _ logger.ILogger = (*captureLogger)(nil)

	l := &captureLogger{}
	l.Information("a")
	l.Warning("b")
	l.Error("c")
	l.Debug("d")
	if len(l.info)+len(l.warn)+len(l.err)+len(l.debug) != 4 {
		t.Errorf("expected exactly one message per level")
	}
}
