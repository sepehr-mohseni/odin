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
	err := errors.NewHTTPError(http.StatusBadRequest, "test error", "validation failed")

	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "test error", err.Message)
	assert.Equal(t, "validation failed", err.Details)
}

func TestHTTPErrorError(t *testing.T) {
	err := errors.NewHTTPError(http.StatusBadRequest, "test error", "validation failed")
	assert.Equal(t, "test error: validation failed", err.Error())

	errNoDetails := errors.NewHTTPError(http.StatusBadRequest, "test error", "")
	assert.Equal(t, "test error", errNoDetails.Error())
}

func TestErrorHandler(t *testing.T) {
	logger := logrus.New()
	handler := errors.ErrorHandler(logger)

	e := echo.New()
	e.HTTPErrorHandler = handler

	// Test with custom HTTPError
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	customErr := errors.NewHTTPError(http.StatusBadRequest, "custom error", "details")
	handler(customErr, c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "custom error")
	assert.Contains(t, rec.Body.String(), "details")
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
	assert.Contains(t, rec.Body.String(), "not found")
}

func TestErrorHandlerWithGenericError(t *testing.T) {
	logger := logrus.New()
	handler := errors.ErrorHandler(logger)

	e := echo.New()
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := assert.AnError
	handler(err, c)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Internal server error")
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
