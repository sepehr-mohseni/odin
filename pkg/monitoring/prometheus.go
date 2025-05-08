package monitoring

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_gateway_requests_total",
			Help: "Total number of requests processed by the API gateway",
		},
		[]string{"service", "method", "status"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_gateway_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method"},
	)

	responseSizes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_gateway_response_size_bytes",
			Help:    "Response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"service"},
	)

	activeRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_gateway_active_requests",
			Help: "Currently active requests",
		},
		[]string{"service"},
	)

	cacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_gateway_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"service"},
	)

	cacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_gateway_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"service"},
	)

	rateLimited = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_gateway_rate_limited_total",
			Help: "Total number of rate limited requests",
		},
		[]string{"service"},
	)
)

func Register(e *echo.Echo, path string) {
	e.GET(path, echo.WrapHandler(promhttp.Handler()))

	e.Use(MetricsMiddleware)
}

func MetricsMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		path := c.Request().URL.Path
		service := "unknown"

		parts := splitPath(path)
		if len(parts) > 1 {
			service = parts[1]
		}

		activeRequests.WithLabelValues(service).Inc()
		defer activeRequests.WithLabelValues(service).Dec()

		start := time.Now()

		err := next(c)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Response().Status)

		requestCounter.WithLabelValues(service, c.Request().Method, status).Inc()
		requestDuration.WithLabelValues(service, c.Request().Method).Observe(duration)

		if c.Response().Size > 0 {
			responseSizes.WithLabelValues(service).Observe(float64(c.Response().Size))
		}

		if c.Response().Status == http.StatusTooManyRequests {
			rateLimited.WithLabelValues(service).Inc()
		}

		if c.Response().Header().Get("X-Cache") == "HIT" {
			cacheHits.WithLabelValues(service).Inc()
		} else if c.Response().Header().Get("X-Cache") == "MISS" {
			cacheMisses.WithLabelValues(service).Inc()
		}

		return err
	}
}

func splitPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if start < i {
				parts = append(parts, path[start:i])
			}
			parts = append(parts, "/")
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}
