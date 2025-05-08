package middleware

import (
	"crypto/sha256"
	"encoding/hex" // Added missing import for fmt
	"io"
	"net/http"
	"odin/pkg/cache"
	"odin/pkg/ratelimit"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func RateLimiterMiddleware(limiter ratelimit.RateLimiter, logger *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := c.RealIP()

			if user, ok := c.Get("user").(map[string]interface{}); ok {
				if userID, exists := user["user_id"].(string); exists && userID != "" {
					key = userID
				}
			}

			if !limiter.Allow(key) {
				logger.WithFields(logrus.Fields{
					"ip":  c.RealIP(),
					"uri": c.Request().RequestURI,
				}).Warn("Rate limit exceeded")
				return echo.NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded")
			}

			return next(c)
		}
	}
}

func CacheMiddleware(store cache.Store, logger *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()

			if req.Method != http.MethodGet {
				return next(c)
			}

			key := generateCacheKey(c)

			if cachedResp, found := store.Get(key); found {
				logger.WithFields(logrus.Fields{
					"uri": req.RequestURI,
					"key": key,
				}).Debug("Cache hit")

				for k, v := range cachedResp.Headers {
					c.Response().Header().Set(k, v[0])
				}
				c.Response().WriteHeader(cachedResp.StatusCode)
				_, err := c.Response().Write(cachedResp.Body)
				return err
			}

			resWriter := &responseWriterWrapper{
				ResponseWriter: c.Response().Writer,
				statusCode:     http.StatusOK,
				body:           strings.Builder{},
				headers:        make(http.Header),
			}
			c.Response().Writer = resWriter

			err := next(c)

			if err == nil {
				cachedResp := &cache.CachedResponse{
					Headers:    resWriter.headers,
					StatusCode: resWriter.statusCode,
					Body:       []byte(resWriter.body.String()),
				}
				store.Set(key, cachedResp)

				logger.WithFields(logrus.Fields{
					"uri": req.RequestURI,
					"key": key,
				}).Debug("Cached response")
			}

			return err
		}
	}
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	body       strings.Builder
	headers    http.Header
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriterWrapper) Header() http.Header {
	return w.ResponseWriter.Header()
}

func generateCacheKey(c echo.Context) string {
	req := c.Request()

	keyParts := []string{req.Method, req.URL.Path, req.URL.RawQuery}

	if req.Body != nil && (req.Method == http.MethodPost || req.Method == http.MethodPut) {
		if req.ContentLength > 0 && req.ContentLength < 1024*10 {
			bodyBytes, _ := io.ReadAll(req.Body)
			req.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
			keyParts = append(keyParts, string(bodyBytes))
		}
	}

	hasher := sha256.New()
	hasher.Write([]byte(strings.Join(keyParts, "|")))
	return hex.EncodeToString(hasher.Sum(nil))
}
