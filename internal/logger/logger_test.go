package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"log/slog"
)

func newTestLog(buf *bytes.Buffer) Log {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logger := slog.New(slog.NewJSONHandler(buf, opts))
	return Log{original: logger}
}

func TestLog_Error(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLog(&buf)

	log.Error("error message", "key1", "value1", "key2", "value2")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)

	assert.Equal(t, "error message", logEntry["msg"])
	assert.Equal(t, "value1", logEntry["key1"])
	assert.Equal(t, "value2", logEntry["key2"])
	assert.Equal(t, "ERROR", logEntry["level"])
}

func TestLog_WithError(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLog(&buf)
	err := assert.AnError

	log.WithError(err, "error message", "key1", "value1", "key2", "value2")

	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)

	assert.Equal(t, "error message", logEntry["msg"])
	assert.Equal(t, "assert.AnError general error for testing", logEntry["error"])
	assert.Equal(t, "value1", logEntry["key1"])
	assert.Equal(t, "value2", logEntry["key2"])
	assert.Equal(t, "ERROR", logEntry["level"])
}

func TestLog_Info(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLog(&buf)

	log.Info("info message", "key1", "value1", "key2", "value2")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)

	assert.Equal(t, "info message", logEntry["msg"])
	assert.Equal(t, "value1", logEntry["key1"])
	assert.Equal(t, "value2", logEntry["key2"])
	assert.Equal(t, "INFO", logEntry["level"])
}

func TestLog_Debug(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLog(&buf)

	log.Debug("debug message", "key1", "value1", "key2", "value2")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)

	assert.Equal(t, "debug message", logEntry["msg"])
	assert.Equal(t, "value1", logEntry["key1"])
	assert.Equal(t, "value2", logEntry["key2"])
	assert.Equal(t, "DEBUG", logEntry["level"])
}

func TestLog_DebugContext(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLog(&buf)
	ctx := context.Background()

	log.DebugContext(ctx, "debug context message", "key1", "value1", "key2", "value2")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)

	assert.Equal(t, "debug context message", logEntry["msg"])
	assert.Equal(t, "value1", logEntry["key1"])
	assert.Equal(t, "value2", logEntry["key2"])
	assert.Equal(t, "DEBUG", logEntry["level"])
}
