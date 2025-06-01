package logging

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	JSON   bool   `yaml:"json"`
}

func NewLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	return logger
}

func ConfigureLogger(logger *logrus.Logger, config Config) {
	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set formatter
	if config.JSON || config.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}
}

func LoggerMiddleware(logger *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()

			err := next(c)

			res := c.Response()
			latency := time.Since(start)

			fields := logrus.Fields{
				"method":     req.Method,
				"uri":        req.RequestURI,
				"status":     res.Status,
				"latency_ms": latency.Milliseconds(),
				"user_agent": req.UserAgent(),
				"ip":         c.RealIP(),
			}

			if err != nil {
				fields["error"] = err.Error()
				logger.WithFields(fields).Error("Request error")
			} else {
				logger.WithFields(fields).Info("Request processed")
			}

			return err
		}
	}
}

// Legacy function for backward compatibility
func ConfigureLoggerLegacy(logger *logrus.Logger, level string, json bool) {
	config := Config{
		Level: level,
		JSON:  json,
	}
	ConfigureLogger(logger, config)
}
