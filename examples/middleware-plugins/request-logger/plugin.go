package main

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// RequestLogger logs all incoming requests
type RequestLogger struct {
	logger         *logrus.Logger
	prefix         string
	includeHeaders bool
	includeBody    bool
}

// Export the middleware instance (required for Go plugins)
var Middleware RequestLogger

// Name returns the middleware name
func (m *RequestLogger) Name() string {
	return "request-logger"
}

// Version returns the middleware version
func (m *RequestLogger) Version() string {
	return "1.0.0"
}

// Initialize sets up the middleware with configuration
func (m *RequestLogger) Initialize(config map[string]interface{}) error {
	m.logger = logrus.New()
	m.logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})

	// Parse configuration
	if prefix, ok := config["prefix"].(string); ok {
		m.prefix = prefix
	} else {
		m.prefix = "[REQUEST]"
	}

	if logLevel, ok := config["logLevel"].(string); ok {
		level, err := logrus.ParseLevel(logLevel)
		if err != nil {
			return fmt.Errorf("invalid log level: %w", err)
		}
		m.logger.SetLevel(level)
	} else {
		m.logger.SetLevel(logrus.InfoLevel)
	}

	if includeHeaders, ok := config["includeHeaders"].(bool); ok {
		m.includeHeaders = includeHeaders
	}

	if includeBody, ok := config["includeBody"].(bool); ok {
		m.includeBody = includeBody
	}

	m.logger.WithFields(logrus.Fields{
		"prefix":         m.prefix,
		"includeHeaders": m.includeHeaders,
		"includeBody":    m.includeBody,
	}).Info("Request logger middleware initialized")

	return nil
}

// Handle processes each request
func (m *RequestLogger) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		req := c.Request()

		// Prepare log fields
		fields := logrus.Fields{
			"method": req.Method,
			"path":   req.URL.Path,
			"ip":     c.RealIP(),
			"host":   req.Host,
		}

		// Add headers if configured
		if m.includeHeaders {
			headers := make(map[string]string)
			for key, values := range req.Header {
				if len(values) > 0 {
					headers[key] = values[0]
				}
			}
			fields["headers"] = headers
		}

		// Log incoming request
		m.logger.WithFields(fields).Info(fmt.Sprintf("%s Incoming request", m.prefix))

		// Process request
		err := next(c)

		// Log response
		duration := time.Since(start)
		status := c.Response().Status

		responseFields := logrus.Fields{
			"method":   req.Method,
			"path":     req.URL.Path,
			"status":   status,
			"duration": duration.String(),
			"size":     c.Response().Size,
		}

		logEntry := m.logger.WithFields(responseFields)

		// Log based on status code
		if status >= 500 {
			logEntry.Error(fmt.Sprintf("%s Request failed (5xx)", m.prefix))
		} else if status >= 400 {
			logEntry.Warn(fmt.Sprintf("%s Client error (4xx)", m.prefix))
		} else if status >= 300 {
			logEntry.Info(fmt.Sprintf("%s Redirect (3xx)", m.prefix))
		} else {
			logEntry.Info(fmt.Sprintf("%s Request completed successfully", m.prefix))
		}

		return err
	}
}

// Cleanup releases resources
func (m *RequestLogger) Cleanup() error {
	m.logger.Info("Request logger middleware cleaned up")
	return nil
}

// Required for Go plugins
func main() {}
