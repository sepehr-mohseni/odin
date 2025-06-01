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
	assert.Equal(t, logrus.InfoLevel, logger.Level)
}

func TestConfigureLogger(t *testing.T) {
	logger := logrus.New()

	config := logging.Config{
		Level:  "debug",
		Format: "text",
	}

	logging.ConfigureLogger(logger, config)

	assert.Equal(t, logrus.DebugLevel, logger.Level)
}

func TestConfigureLoggerWithJSON(t *testing.T) {
	logger := logrus.New()

	config := logging.Config{
		Level: "warn",
		JSON:  true,
	}

	logging.ConfigureLogger(logger, config)

	assert.Equal(t, logrus.WarnLevel, logger.Level)
	assert.IsType(t, &logrus.JSONFormatter{}, logger.Formatter)
}

func TestConfigureLoggerWithJSONFormat(t *testing.T) {
	logger := logrus.New()

	config := logging.Config{
		Level:  "error",
		Format: "json",
	}

	logging.ConfigureLogger(logger, config)

	assert.Equal(t, logrus.ErrorLevel, logger.Level)
	assert.IsType(t, &logrus.JSONFormatter{}, logger.Formatter)
}

func TestConfigureLoggerWithText(t *testing.T) {
	logger := logrus.New()

	config := logging.Config{
		Level:  "error",
		Format: "text",
	}

	logging.ConfigureLogger(logger, config)

	assert.Equal(t, logrus.ErrorLevel, logger.Level)
	assert.IsType(t, &logrus.TextFormatter{}, logger.Formatter)
}

func TestConfigureLoggerLegacy(t *testing.T) {
	logger := logrus.New()

	logging.ConfigureLoggerLegacy(logger, "debug", true)

	assert.Equal(t, logrus.DebugLevel, logger.Level)
	assert.IsType(t, &logrus.JSONFormatter{}, logger.Formatter)
}

func TestConfigureLoggerInvalidLevel(t *testing.T) {
	logger := logrus.New()

	config := logging.Config{
		Level: "invalid-level",
		JSON:  false,
	}

	logging.ConfigureLogger(logger, config)

	// Should default to info level
	assert.Equal(t, logrus.InfoLevel, logger.Level)
}
