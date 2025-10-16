package ai

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// TrafficMiddleware collects traffic data for AI analysis
type TrafficMiddleware struct {
	collector *TrafficCollector
	logger    *logrus.Logger
}

// NewTrafficMiddleware creates a new traffic collection middleware
func NewTrafficMiddleware(collector *TrafficCollector, logger *logrus.Logger) *TrafficMiddleware {
	return &TrafficMiddleware{
		collector: collector,
		logger:    logger,
	}
}

// Middleware returns the HTTP middleware function
func (tm *TrafficMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Create response recorder to capture status code and size
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call next handler
		next.ServeHTTP(recorder, r)

		// Calculate latency
		latency := time.Since(startTime).Seconds() * 1000 // Convert to milliseconds

		// Extract service name from context or path
		serviceName := tm.extractServiceName(r)
		endpoint := r.URL.Path
		method := r.Method

		// Collect traffic data
		data := &TrafficData{
			Timestamp:    time.Now(),
			ServiceName:  serviceName,
			Endpoint:     endpoint,
			Method:       method,
			StatusCode:   recorder.statusCode,
			Latency:      latency,
			RequestSize:  r.ContentLength,
			ResponseSize: recorder.size,
			SourceIP:     tm.extractIP(r),
			UserAgent:    r.UserAgent(),
			UserID:       tm.extractUserID(r),
		}

		// Handle errors
		if recorder.statusCode >= 400 {
			data.Error = http.StatusText(recorder.statusCode)
		}

		// Send to collector
		tm.collector.Collect(data)
	})
}

// extractServiceName extracts service name from request
func (tm *TrafficMiddleware) extractServiceName(r *http.Request) string {
	// Try to get from context first
	if svc := r.Context().Value("service_name"); svc != nil {
		if serviceName, ok := svc.(string); ok {
			return serviceName
		}
	}

	// Try to extract from path (e.g., /api/users -> users)
	// This is a simplified version - adapt based on your routing
	return "unknown"
}

// extractIP extracts client IP from request
func (tm *TrafficMiddleware) extractIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Use RemoteAddr
	return r.RemoteAddr
}

// extractUserID extracts user ID from request (if authenticated)
func (tm *TrafficMiddleware) extractUserID(r *http.Request) string {
	// Try to get from context
	if uid := r.Context().Value("user_id"); uid != nil {
		if userID, ok := uid.(string); ok {
			return userID
		}
	}

	return ""
}

// responseRecorder wraps http.ResponseWriter to capture status code and response size
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	size       int64
}

// WriteHeader captures the status code
func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

// Write captures the response size
func (rr *responseRecorder) Write(b []byte) (int, error) {
	n, err := rr.ResponseWriter.Write(b)
	rr.size += int64(n)
	return n, err
}
