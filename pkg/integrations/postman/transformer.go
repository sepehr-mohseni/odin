package postman

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"odin/pkg/config"
	"odin/pkg/service"
)

// Transformer handles conversion between Postman and Odin formats
type Transformer struct{}

// NewTransformer creates a new transformer
func NewTransformer() *Transformer {
	return &Transformer{}
}

// PostmanToOdinService converts a Postman collection to an Odin service configuration
func (t *Transformer) PostmanToOdinService(collection *PostmanCollection) (*config.ServiceConfig, error) {
	if collection == nil {
		return nil, fmt.Errorf("collection is nil")
	}

	// Extract base URL from collection if available
	baseURL, err := t.extractBaseURL(collection)
	if err != nil {
		return nil, fmt.Errorf("failed to extract base URL: %w", err)
	}

	// Create service config
	svc := &config.ServiceConfig{
		Name:           t.sanitizeName(collection.Info.Name),
		BasePath:       t.extractBasePath(collection),
		Targets:        []string{baseURL},
		Protocol:       "http",
		Timeout:        30000, // 30 seconds default (milliseconds, will be converted)
		RetryCount:     3,
		RetryDelay:     1000,
		LoadBalancing:  "round-robin",
		Authentication: false, // Will be set to true if auth is configured
	}

	// Extract authentication if available
	if collection.Auth != nil {
		svc.Authentication = true
	} // Extract headers from collection level
	svc.Headers = t.extractHeaders(collection)

	return svc, nil
}

// OdinServiceToPostman converts an Odin service to a Postman collection
func (t *Transformer) OdinServiceToPostman(svc *config.ServiceConfig) (*PostmanCollection, error) {
	if svc == nil {
		return nil, fmt.Errorf("service is nil")
	}

	collection := &PostmanCollection{
		Info: CollectionInfo{
			Name:        svc.Name,
			Description: fmt.Sprintf("Generated from Odin service: %s", svc.Name),
			Schema:      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Item: []CollectionItem{},
	}

	// Create example requests for the service
	baseURL := ""
	if len(svc.Targets) > 0 {
		baseURL = svc.Targets[0]
	}

	// Add basic GET request
	collection.Item = append(collection.Item, t.createExampleRequest(
		"GET "+svc.BasePath,
		"GET",
		baseURL,
		svc.BasePath,
		svc.Headers,
	))

	// Add basic POST request
	collection.Item = append(collection.Item, t.createExampleRequest(
		"POST "+svc.BasePath,
		"POST",
		baseURL,
		svc.BasePath,
		svc.Headers,
	))

	return collection, nil
}

// extractBaseURL extracts the base URL from collection items
func (t *Transformer) extractBaseURL(collection *PostmanCollection) (string, error) {
	// Try to find a base URL from the first request
	for _, item := range collection.Item {
		if item.Request != nil && item.Request.URL != nil {
			if item.Request.URL.Raw != "" {
				u, err := url.Parse(item.Request.URL.Raw)
				if err == nil && u.Host != "" {
					return fmt.Sprintf("%s://%s", u.Scheme, u.Host), nil
				}
			}

			// Build from parts
			if len(item.Request.URL.Host) > 0 {
				protocol := item.Request.URL.Protocol
				if protocol == "" {
					protocol = "http"
				}
				host := strings.Join(item.Request.URL.Host, ".")
				return fmt.Sprintf("%s://%s", protocol, host), nil
			}
		}

		// Recursively check sub-items (folders)
		if len(item.Item) > 0 {
			subCollection := &PostmanCollection{Item: item.Item}
			if baseURL, err := t.extractBaseURL(subCollection); err == nil {
				return baseURL, nil
			}
		}
	}

	// Default fallback
	return "http://localhost:8080", nil
}

// extractBasePath extracts the common base path from collection requests
func (t *Transformer) extractBasePath(collection *PostmanCollection) string {
	var paths []string

	// Collect all paths
	for _, item := range collection.Item {
		if item.Request != nil && item.Request.URL != nil {
			if len(item.Request.URL.Path) > 0 {
				paths = append(paths, "/"+strings.Join(item.Request.URL.Path, "/"))
			} else if item.Request.URL.Raw != "" {
				u, err := url.Parse(item.Request.URL.Raw)
				if err == nil {
					paths = append(paths, u.Path)
				}
			}
		}
	}

	if len(paths) == 0 {
		return "/"
	}

	// Find common prefix
	commonPath := t.findCommonPrefix(paths)
	if commonPath == "" {
		return "/"
	}

	return commonPath
}

// findCommonPrefix finds the common path prefix among all paths
func (t *Transformer) findCommonPrefix(paths []string) string {
	if len(paths) == 0 {
		return ""
	}

	if len(paths) == 1 {
		// Return directory path
		return path.Dir(paths[0])
	}

	// Split paths into segments
	segments := make([][]string, len(paths))
	minLen := len(strings.Split(paths[0], "/"))

	for i, p := range paths {
		segments[i] = strings.Split(strings.Trim(p, "/"), "/")
		if len(segments[i]) < minLen {
			minLen = len(segments[i])
		}
	}

	// Find common segments
	var common []string
	for i := 0; i < minLen; i++ {
		seg := segments[0][i]
		allMatch := true
		for j := 1; j < len(segments); j++ {
			if segments[j][i] != seg {
				allMatch = false
				break
			}
		}
		if allMatch {
			common = append(common, seg)
		} else {
			break
		}
	}

	if len(common) == 0 {
		return "/"
	}

	return "/" + strings.Join(common, "/")
}

// extractHeaders extracts common headers from collection
func (t *Transformer) extractHeaders(collection *PostmanCollection) map[string]string {
	headers := make(map[string]string)

	// Check collection-level headers
	for _, item := range collection.Item {
		if item.Request != nil {
			for _, h := range item.Request.Header {
				if !h.Disabled {
					// Only add common headers (not auth-related)
					key := strings.ToLower(h.Key)
					if key != "authorization" && key != "x-api-key" {
						headers[h.Key] = h.Value
					}
				}
			}
		}
	}

	return headers
}

// createExampleRequest creates an example Postman request
func (t *Transformer) createExampleRequest(name, method, baseURL, basePath string, headers map[string]string) CollectionItem {
	item := CollectionItem{
		Name: name,
		Request: &Request{
			Method: method,
			URL: &URL{
				Raw:      baseURL + basePath,
				Protocol: "http",
				Host:     strings.Split(strings.TrimPrefix(baseURL, "http://"), "."),
				Path:     strings.Split(strings.Trim(basePath, "/"), "/"),
			},
		},
	}

	// Add headers
	for k, v := range headers {
		item.Request.Header = append(item.Request.Header, Header{
			Key:   k,
			Value: v,
		})
	}

	// Add body for POST/PUT requests
	if method == "POST" || method == "PUT" || method == "PATCH" {
		item.Request.Body = &RequestBody{
			Mode: "raw",
			Raw:  "{\n  \"example\": \"data\"\n}",
			Options: map[string]interface{}{
				"raw": map[string]interface{}{
					"language": "json",
				},
			},
		}
		item.Request.Header = append(item.Request.Header, Header{
			Key:   "Content-Type",
			Value: "application/json",
		})
	}

	return item
}

// sanitizeName sanitizes a name to be valid for Odin
func (t *Transformer) sanitizeName(name string) string {
	// Replace spaces and special chars with hyphens
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")

	// Remove invalid characters
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// EnhancedTransform provides more detailed transformation with routes
func (t *Transformer) EnhancedTransform(collection *PostmanCollection, serviceName string) (*service.Config, []RouteMapping, error) {
	routes := []RouteMapping{}

	// Extract all requests from collection
	t.extractRoutes(collection.Item, &routes, "")

	// Create base service config
	baseURL, _ := t.extractBaseURL(collection)

	svc := &service.Config{
		Name:       serviceName,
		BasePath:   t.extractBasePath(collection),
		Targets:    []string{baseURL},
		Timeout:    30000,
		RetryCount: 3,
		Protocol:   "http",
	}

	return svc, routes, nil
}

// RouteMapping represents a mapping between Postman request and Odin route
type RouteMapping struct {
	Name        string
	Method      string
	Path        string
	Description string
	Headers     []Header
	Auth        *Auth
	Body        *RequestBody
	Tests       []string // Test scripts
}

// extractRoutes recursively extracts routes from collection items
func (t *Transformer) extractRoutes(items []CollectionItem, routes *[]RouteMapping, prefix string) {
	for _, item := range items {
		if item.Request != nil {
			// This is a request
			mapping := RouteMapping{
				Name:        item.Name,
				Description: item.Description,
				Method:      item.Request.Method,
				Headers:     item.Request.Header,
				Auth:        item.Request.Auth,
				Body:        item.Request.Body,
			}

			// Extract path
			if item.Request.URL != nil {
				if len(item.Request.URL.Path) > 0 {
					mapping.Path = "/" + strings.Join(item.Request.URL.Path, "/")
				} else if item.Request.URL.Raw != "" {
					u, err := url.Parse(item.Request.URL.Raw)
					if err == nil {
						mapping.Path = u.Path
					}
				}
			}

			// Extract test scripts
			for _, event := range item.Event {
				if event.Listen == "test" {
					mapping.Tests = event.Script.Exec
				}
			}

			*routes = append(*routes, mapping)
		} else if len(item.Item) > 0 {
			// This is a folder, recurse
			folderPrefix := prefix
			if prefix != "" {
				folderPrefix = prefix + " / " + item.Name
			} else {
				folderPrefix = item.Name
			}
			t.extractRoutes(item.Item, routes, folderPrefix)
		}
	}
}

// TransformEnvironment converts Postman environment to Odin config
func (t *Transformer) TransformEnvironment(env *Environment) (map[string]string, error) {
	if env == nil {
		return nil, fmt.Errorf("environment is nil")
	}

	config := make(map[string]string)
	for _, val := range env.Values {
		if val.Enabled {
			config[val.Key] = val.Value
		}
	}

	return config, nil
}

// OdinConfigToEnvironment converts Odin config to Postman environment
func (t *Transformer) OdinConfigToEnvironment(name string, config map[string]string) *Environment {
	env := &Environment{
		Name:   name,
		Values: []EnvValue{},
	}

	for k, v := range config {
		env.Values = append(env.Values, EnvValue{
			Key:     k,
			Value:   v,
			Type:    "default",
			Enabled: true,
		})
	}

	return env
}
