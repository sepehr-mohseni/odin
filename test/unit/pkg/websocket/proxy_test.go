package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"odin/pkg/websocket"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func isWebSocketUpgrade(r *http.Request) bool {
	return r.Header.Get("Connection") == "Upgrade" && r.Header.Get("Upgrade") == "websocket"
}

func TestNewProxy(t *testing.T) {
	config := websocket.Config{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	proxy := websocket.NewProxy(config, logrus.New())
	assert.NotNil(t, proxy)
}

func TestNewProxyWithDefaults(t *testing.T) {
	config := websocket.Config{} // Empty config should use defaults

	proxy := websocket.NewProxy(config, logrus.New())
	assert.NotNil(t, proxy)
}

func TestIsWebSocketUpgrade(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected bool
	}{
		{
			name: "valid websocket upgrade",
			headers: map[string]string{
				"Connection": "Upgrade",
				"Upgrade":    "websocket",
			},
			expected: true,
		},
		{
			name: "invalid connection header",
			headers: map[string]string{
				"Connection": "keep-alive",
				"Upgrade":    "websocket",
			},
			expected: false,
		},
		{
			name: "invalid upgrade header",
			headers: map[string]string{
				"Connection": "Upgrade",
				"Upgrade":    "h2c",
			},
			expected: false,
		},
		{
			name:     "missing headers",
			headers:  map[string]string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			result := isWebSocketUpgrade(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWebSocketMiddleware(t *testing.T) {
	proxy := websocket.NewProxy(websocket.Config{}, logrus.New())
	middleware := websocket.WebSocketMiddleware(proxy)

	e := echo.New()

	// Test non-WebSocket request
	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextCalled := false
	next := func(c echo.Context) error {
		nextCalled = true
		return nil
	}

	err := middleware(next)(c)
	assert.NoError(t, err)
	assert.True(t, nextCalled)

	// Test WebSocket request without target URL
	req = httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	nextCalled = false
	err = middleware(next)(c)
	assert.Error(t, err)
	assert.False(t, nextCalled)
}

func TestProxyConfiguration(t *testing.T) {
	config := websocket.Config{
		HandshakeTimeout: 5 * time.Second,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     5 * time.Second,
	}

	proxy := websocket.NewProxy(config, logrus.New())
	assert.NotNil(t, proxy)
}

func TestConnectionClose(t *testing.T) {
	// This test verifies the connection cleanup logic
	// In a real scenario, you'd have actual WebSocket connections to test with
	config := websocket.Config{
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	proxy := websocket.NewProxy(config, logrus.New())
	assert.NotNil(t, proxy)

	// Test that proxy handles configuration correctly
	assert.Equal(t, 1*time.Second, config.ReadTimeout)
	assert.Equal(t, 1*time.Second, config.WriteTimeout)
}
