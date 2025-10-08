package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ProxyConfig holds gRPC proxy configuration
type ProxyConfig struct {
	Target           string        `yaml:"target"`
	MaxMessageSize   int           `yaml:"maxMessageSize"`
	Timeout          time.Duration `yaml:"timeout"`
	EnableTLS        bool          `yaml:"enableTLS"`
	TLSCertFile      string        `yaml:"tlsCertFile"`
	TLSKeyFile       string        `yaml:"tlsKeyFile"`
	EnableReflection bool          `yaml:"enableReflection"`
}

// Proxy handles gRPC requests and HTTP-gRPC transcoding
type Proxy struct {
	config *ProxyConfig
	logger *logrus.Logger
	conn   *grpc.ClientConn
}

// NewProxy creates a new gRPC proxy
func NewProxy(config *ProxyConfig, logger *logrus.Logger) (*Proxy, error) {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxMessageSize == 0 {
		config.MaxMessageSize = 4 * 1024 * 1024 // 4MB default
	}

	// Set up gRPC dial options
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(config.MaxMessageSize)),
	}

	if !config.EnableTLS {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Establish connection to gRPC service
	conn, err := grpc.Dial(config.Target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC service: %w", err)
	}

	return &Proxy{
		config: config,
		logger: logger,
		conn:   conn,
	}, nil
}

// Close closes the gRPC connection
func (p *Proxy) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

// HTTPRequest represents an HTTP request for gRPC transcoding
type HTTPRequest struct {
	Method  string                 `json:"method"`
	Service string                 `json:"service"`
	Message map[string]interface{} `json:"message"`
}

// Handle processes HTTP to gRPC transcoding requests
func (p *Proxy) Handle(c echo.Context) error {
	// Extract service and method from path
	// Expected format: /grpc/{service}/{method}
	path := c.Request().URL.Path
	pathParts := strings.Split(strings.TrimPrefix(path, "/"), "/")

	if len(pathParts) < 3 || pathParts[0] != "grpc" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid gRPC path format. Expected: /grpc/{service}/{method}",
		})
	}

	serviceName := pathParts[1]
	methodName := pathParts[2]
	fullMethod := fmt.Sprintf("/%s/%s", serviceName, methodName)

	// Parse request body
	var reqBody map[string]interface{}
	if c.Request().Method == http.MethodPost {
		if err := c.Bind(&reqBody); err != nil {
			p.logger.WithError(err).Error("Failed to parse request body")
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid JSON request body",
			})
		}
	}

	// Create gRPC context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), p.config.Timeout)
	defer cancel()

	// Convert HTTP headers to gRPC metadata
	md := p.httpHeadersToMetadata(c.Request().Header)
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Perform gRPC call using dynamic invocation
	resp, err := p.invokeGRPCMethod(ctx, fullMethod, reqBody)
	if err != nil {
		return p.handleGRPCError(c, err)
	}

	// Return JSON response
	return c.JSON(http.StatusOK, resp)
}

// invokeGRPCMethod performs dynamic gRPC method invocation
func (p *Proxy) invokeGRPCMethod(ctx context.Context, method string, req map[string]interface{}) (interface{}, error) {
	// Convert request to JSON bytes for generic handling
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create a generic message holder
	var resp json.RawMessage

	// Invoke the method using grpc.ClientConn.Invoke
	err = p.conn.Invoke(ctx, method, reqBytes, &resp)
	if err != nil {
		return nil, err
	}

	// Parse response
	var result interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// httpHeadersToMetadata converts HTTP headers to gRPC metadata
func (p *Proxy) httpHeadersToMetadata(headers http.Header) metadata.MD {
	md := metadata.New(nil)

	for key, values := range headers {
		// Skip certain HTTP-specific headers
		if p.shouldSkipHeader(key) {
			continue
		}

		// Convert to lowercase (gRPC metadata keys are case-insensitive)
		key = strings.ToLower(key)

		for _, value := range values {
			md.Append(key, value)
		}
	}

	return md
}

// shouldSkipHeader determines if an HTTP header should be skipped
func (p *Proxy) shouldSkipHeader(header string) bool {
	skipHeaders := []string{
		"Content-Length",
		"Content-Type",
		"Host",
		"User-Agent",
		"Accept-Encoding",
		"Connection",
	}

	headerLower := strings.ToLower(header)
	for _, skip := range skipHeaders {
		if strings.ToLower(skip) == headerLower {
			return true
		}
	}

	return false
}

// handleGRPCError converts gRPC errors to appropriate HTTP responses
func (p *Proxy) handleGRPCError(c echo.Context, err error) error {
	st, ok := status.FromError(err)
	if !ok {
		p.logger.WithError(err).Error("Non-gRPC error occurred")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	// Map gRPC status codes to HTTP status codes
	httpStatus := p.grpcCodeToHTTPStatus(st.Code())

	response := map[string]interface{}{
		"error": st.Message(),
		"code":  st.Code().String(),
	}

	// Include details if available
	if len(st.Details()) > 0 {
		response["details"] = st.Details()
	}

	return c.JSON(httpStatus, response)
}

// grpcCodeToHTTPStatus maps gRPC status codes to HTTP status codes
func (p *Proxy) grpcCodeToHTTPStatus(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return 499 // Client Closed Request
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusBadRequest
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// RegisterRoutes registers gRPC proxy routes
func (p *Proxy) RegisterRoutes(e *echo.Echo, basePath string) {
	// Handle both GET and POST for different gRPC methods
	e.POST(basePath+"/*", p.Handle)
	e.GET(basePath+"/*", p.Handle)

	// Health check for gRPC service
	e.GET(basePath+"/health", func(c echo.Context) error {
		// Simple connection state check
		state := p.conn.GetState()
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":                "UP",
			"grpc_connection_state": state.String(),
		})
	})
}

// StreamHandler handles gRPC streaming (placeholder for future implementation)
func (p *Proxy) StreamHandler(c echo.Context) error {
	// This would handle gRPC streaming over WebSockets or Server-Sent Events
	// For production use, consider implementing proper streaming support
	return c.JSON(http.StatusNotImplemented, map[string]string{
		"error": "gRPC streaming not yet implemented",
	})
}
