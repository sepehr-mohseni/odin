package errors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"odin/pkg/errors"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewHTTPError(t *testing.T) {
	err := errors.NewHTTPError(http.StatusBadRequest, "Bad request", "Invalid input")

	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "Bad request", err.Message)
	assert.Equal(t, "Invalid input", err.Details)
}

func TestHTTPErrorError(t *testing.T) {
	err := errors.NewHTTPError(http.StatusNotFound, "Not found", "Resource not found")

	assert.Equal(t, "Not found: Resource not found", err.Error())
}

func TestErrorHandler(t *testing.T) {
	logger := logrus.New()
	handler := errors.ErrorHandler(logger)

	e := echo.New()
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	customErr := errors.NewHTTPError(http.StatusBadRequest, "custom error", "details")

	handler(customErr, c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestErrorHandlerWithEchoHTTPError(t *testing.T) {
	logger := logrus.New()
	handler := errors.ErrorHandler(logger)

	e := echo.New()
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	echoErr := echo.NewHTTPError(http.StatusNotFound, "not found")

	handler(echoErr, c)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestErrorHandlerWithGenericError(t *testing.T) {
	logger := logrus.New()
	handler := errors.ErrorHandler(logger)

	e := echo.New()
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	genericErr := assert.AnError

	handler(genericErr, c)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestStatusCodeFromError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "HTTPError",
			err:      errors.NewHTTPError(http.StatusBadRequest, "bad request", ""),
			expected: http.StatusBadRequest,
		},
		{
			name:     "ErrNotFound",
			err:      errors.ErrNotFound,
			expected: http.StatusNotFound,
		},
		{
			name:     "ErrUnauthorized",
			err:      errors.ErrUnauthorized,
			expected: http.StatusUnauthorized,
		},
		{
			name:     "ErrForbidden",
			err:      errors.ErrForbidden,
			expected: http.StatusForbidden,
		},
		{
			name:     "ErrBadRequest",
			err:      errors.ErrBadRequest,
			expected: http.StatusBadRequest,
		},
		{
			name:     "ErrTimeout",
			err:      errors.ErrTimeout,
			expected: http.StatusServiceUnavailable,
		},
		{
			name:     "ErrServiceUnavailable",
			err:      errors.ErrServiceUnavailable,
			expected: http.StatusServiceUnavailable,
		},
		{
			name:     "Generic error",
			err:      assert.AnError,
			expected: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := errors.StatusCodeFromError(tt.err)
			assert.Equal(t, tt.expected, code)
		})
	}
}
