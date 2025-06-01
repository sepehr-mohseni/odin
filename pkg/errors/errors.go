package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// Standard error types
var (
	ErrNotFound           = errors.New("resource not found")
	ErrUnauthorized       = errors.New("unauthorized access")
	ErrForbidden          = errors.New("forbidden access")
	ErrBadRequest         = errors.New("bad request")
	ErrInternalServer     = errors.New("internal server error")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrTimeout            = errors.New("request timed out")
)

// HTTPError represents an error with HTTP status code and details
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error returns the error message
func (e *HTTPError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}
	return e.Message
}

// NewHTTPError creates a new HTTPError
func NewHTTPError(code int, message string, details string) *HTTPError {
	return &HTTPError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// StatusCodeFromError returns an appropriate HTTP status code for an error
func StatusCodeFromError(err error) int {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Code
	}

	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, ErrTimeout), errors.Is(err, ErrServiceUnavailable):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// ErrorHandler creates a custom error handler for Echo
func ErrorHandler(logger *logrus.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		var he *HTTPError
		var ee *echo.HTTPError

		switch {
		case errors.As(err, &he):
			logger.WithFields(logrus.Fields{
				"method":  c.Request().Method,
				"path":    c.Request().URL.Path,
				"code":    he.Code,
				"message": he.Message,
				"details": he.Details,
			}).Error("HTTP error occurred")

			if err := c.JSON(he.Code, map[string]interface{}{
				"error":   he.Message,
				"details": he.Details,
			}); err != nil {
				logger.WithError(err).Error("Failed to write error response")
			}

		case errors.As(err, &ee):
			logger.WithFields(logrus.Fields{
				"method":  c.Request().Method,
				"path":    c.Request().URL.Path,
				"code":    ee.Code,
				"message": ee.Message,
			}).Error("Echo HTTP error occurred")

			if err := c.JSON(ee.Code, map[string]interface{}{
				"error": ee.Message,
			}); err != nil {
				logger.WithError(err).Error("Failed to write error response")
			}

		default:
			logger.WithFields(logrus.Fields{
				"method": c.Request().Method,
				"path":   c.Request().URL.Path,
				"error":  err.Error(),
			}).Error("Unhandled error occurred")

			if err := c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Internal server error",
			}); err != nil {
				logger.WithError(err).Error("Failed to write error response")
			}
		}
	}
}
