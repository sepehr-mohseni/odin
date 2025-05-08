package logging

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func NewLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)
	return logger
}

func ConfigureLogger(logger *logrus.Logger, level string, jsonFormat bool) {
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logger.Warnf("Invalid log level '%s', defaulting to 'info'", level)
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	if jsonFormat {
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
