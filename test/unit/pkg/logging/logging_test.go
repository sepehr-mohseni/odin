package logging

import (
	"testing"

	"odin/pkg/logging"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	logger := logging.NewLogger()
	assert.NotNil(t, logger)
	assert.IsType(t, &logrus.Logger{}, logger)
}

func TestConfigureLogger(t *testing.T) {
	logger := logrus.New()

	// Test debug level
	logging.ConfigureLogger(logger, "debug", false)
	assert.Equal(t, logrus.DebugLevel, logger.Level)

	// Test info level
	logging.ConfigureLogger(logger, "info", false)
	assert.Equal(t, logrus.InfoLevel, logger.Level)

	// Test warn level
	logging.ConfigureLogger(logger, "warn", false)
	assert.Equal(t, logrus.WarnLevel, logger.Level)

	// Test error level
	logging.ConfigureLogger(logger, "error", false)
	assert.Equal(t, logrus.ErrorLevel, logger.Level)

	// Test invalid level (should default to info)
	logging.ConfigureLogger(logger, "invalid", false)
	assert.Equal(t, logrus.InfoLevel, logger.Level)
}

func TestConfigureLoggerWithJSON(t *testing.T) {
	logger := logrus.New()

	logging.ConfigureLogger(logger, "info", true)
	assert.IsType(t, &logrus.JSONFormatter{}, logger.Formatter)
}

func TestConfigureLoggerWithText(t *testing.T) {
	logger := logrus.New()

	logging.ConfigureLogger(logger, "info", false)
	assert.IsType(t, &logrus.TextFormatter{}, logger.Formatter)
}
