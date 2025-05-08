package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPClient is an interface for HTTP client implementations
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// DefaultHTTPClient returns an http client with sensible defaults
func DefaultHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

// ReadJSON reads JSON from a request body
func ReadJSON(body io.ReadCloser, target interface{}) error {
	defer body.Close()
	return json.NewDecoder(body).Decode(target)
}

// WriteJSON writes JSON to response
func WriteJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// IsJSON checks if a byte slice is valid JSON
func IsJSON(data []byte) bool {
	return json.Valid(data)
}

// BuildURL builds a URL from a base URL and path segments
func BuildURL(baseURL string, pathSegments ...string) string {
	base := strings.TrimRight(baseURL, "/")

	if len(pathSegments) == 0 {
		return base
	}

	var path string
	for _, segment := range pathSegments {
		segment = strings.Trim(segment, "/")
		if segment != "" {
			path += "/" + segment
		}
	}

	return base + path
}

// BuildQueryString builds a URL query string from a map
func BuildQueryString(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}

	query := "?"
	first := true
	for key, value := range params {
		if !first {
			query += "&"
		}
		query += fmt.Sprintf("%s=%s", key, value)
		first = false
	}

	return query
}
