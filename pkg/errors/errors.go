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
		if c.Response().Committed {
			return
		}

		var httpErr *HTTPError
		var echoErr *echo.HTTPError

		switch {
		case errors.As(err, &httpErr):
			// Custom HTTPError
			logger.WithFields(logrus.Fields{
				"code":    httpErr.Code,
				"message": httpErr.Message,
				"details": httpErr.Details,
				"path":    c.Request().URL.Path,
				"method":  c.Request().Method,
			}).Error("HTTP error occurred")

			c.JSON(httpErr.Code, map[string]interface{}{
				"error":   httpErr.Message,
				"details": httpErr.Details,
				"code":    httpErr.Code,
			})

		case errors.As(err, &echoErr):
			// Echo HTTPError
			logger.WithFields(logrus.Fields{
				"code":    echoErr.Code,
				"message": echoErr.Message,
				"path":    c.Request().URL.Path,
				"method":  c.Request().Method,
			}).Error("Echo HTTP error occurred")

			message := echoErr.Message
			if message == nil {
				message = http.StatusText(echoErr.Code)
			}

			c.JSON(echoErr.Code, map[string]interface{}{
				"error": message,
				"code":  echoErr.Code,
			})

		default:
			// Generic error
			logger.WithFields(logrus.Fields{
				"error":  err.Error(),
				"path":   c.Request().URL.Path,
				"method": c.Request().Method,
			}).Error("Unhandled error occurred")

			c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Internal Server Error",
				"code":  http.StatusInternalServerError,
			})
		}
	}
}
