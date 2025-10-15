package transform

import (
"bytes"
"encoding/json"
"fmt"
"io"
"net/http"
"net/url"
"strings"
"text/template"

"github.com/sirupsen/logrus"
)

// Engine handles request and response transformations using Go templates
type Engine struct {
logger *logrus.Logger
}

// NewEngine creates a new transformation engine
func NewEngine(logger *logrus.Logger) *Engine {
return &Engine{
logger: logger,
}
}

// TransformRequest applies transformations to the request
func (e *Engine) TransformRequest(req *http.Request, config *RequestTransform) error {
if config == nil {
return nil
}

// Transform headers
if err := e.transformHeaders(req.Header, config.Headers); err != nil {
return fmt.Errorf("failed to transform headers: %w", err)
}

// Transform query parameters
if err := e.transformQueryParams(req.URL, config.QueryParams); err != nil {
return fmt.Errorf("failed to transform query params: %w", err)
}

// Transform body if present
if config.Body != "" && req.Body != nil {
if err := e.transformBody(req, config.Body); err != nil {
return fmt.Errorf("failed to transform body: %w", err)
}
}

return nil
}

// TransformResponse applies transformations to the response
func (e *Engine) TransformResponse(body []byte, statusCode int, headers http.Header, config *ResponseTransform) ([]byte, http.Header, error) {
if config == nil {
return body, headers, nil
}

// Transform headers
transformedHeaders := headers.Clone()
if err := e.transformHeaders(transformedHeaders, config.Headers); err != nil {
return body, headers, fmt.Errorf("failed to transform headers: %w", err)
}

// Transform body if template provided
if config.Body != "" {
transformedBody, err := e.applyBodyTemplate(body, statusCode, config.Body)
if err != nil {
return body, headers, fmt.Errorf("failed to transform body: %w", err)
}
return transformedBody, transformedHeaders, nil
}

return body, transformedHeaders, nil
}

// transformHeaders applies template transformations to headers
func (e *Engine) transformHeaders(headers http.Header, transforms map[string]string) error {
if len(transforms) == 0 {
return nil
}

// Create template data from existing headers
data := make(map[string]interface{})
for key, values := range headers {
if len(values) > 0 {
data[key] = values[0]
}
}

for key, tmplStr := range transforms {
value, err := e.executeTemplate(tmplStr, data)
if err != nil {
return fmt.Errorf("failed to transform header %s: %w", key, err)
}
headers.Set(key, value)
}

return nil
}

// transformQueryParams applies template transformations to query parameters
func (e *Engine) transformQueryParams(u *url.URL, transforms map[string]string) error {
if len(transforms) == 0 {
return nil
}

query := u.Query()

// Create template data from existing query params
data := make(map[string]interface{})
for key, values := range query {
if len(values) > 0 {
data[key] = values[0]
}
}

for key, tmplStr := range transforms {
value, err := e.executeTemplate(tmplStr, data)
if err != nil {
return fmt.Errorf("failed to transform query param %s: %w", key, err)
}
query.Set(key, value)
}

u.RawQuery = query.Encode()
return nil
}

// transformBody applies template transformation to request body
func (e *Engine) transformBody(req *http.Request, tmplStr string) error {
// Read body
bodyBytes, err := io.ReadAll(req.Body)
if err != nil {
return fmt.Errorf("failed to read body: %w", err)
}

// Parse as JSON for template data
var data map[string]interface{}
if len(bodyBytes) > 0 {
if err := json.Unmarshal(bodyBytes, &data); err != nil {
// If not JSON, use raw body as string
data = map[string]interface{}{"body": string(bodyBytes)}
}
}

// Apply template
result, err := e.executeTemplate(tmplStr, data)
if err != nil {
return fmt.Errorf("failed to execute template: %w", err)
}

// Set new body
req.Body = io.NopCloser(strings.NewReader(result))
req.ContentLength = int64(len(result))

return nil
}

// applyBodyTemplate applies template to response body
func (e *Engine) applyBodyTemplate(body []byte, statusCode int, tmplStr string) ([]byte, error) {
// Parse body as JSON for template data
var data map[string]interface{}
if len(body) > 0 {
if err := json.Unmarshal(body, &data); err != nil {
// If not JSON, use raw body
data = map[string]interface{}{
"body":       string(body),
"statusCode": statusCode,
}
} else {
data["statusCode"] = statusCode
}
} else {
data = map[string]interface{}{"statusCode": statusCode}
}

result, err := e.executeTemplate(tmplStr, data)
if err != nil {
return nil, err
}

return []byte(result), nil
}

// executeTemplate executes a Go template with the given data
func (e *Engine) executeTemplate(tmplStr string, data interface{}) (string, error) {
// Create template with helper functions
tmpl, err := template.New("transform").Funcs(templateFuncs()).Parse(tmplStr)
if err != nil {
return "", fmt.Errorf("failed to parse template: %w", err)
}

var buf bytes.Buffer
if err := tmpl.Execute(&buf, data); err != nil {
return "", fmt.Errorf("failed to execute template: %w", err)
}

return buf.String(), nil
}

// templateFuncs returns helper functions for templates
func templateFuncs() template.FuncMap {
return template.FuncMap{
"upper":    strings.ToUpper,
"lower":    strings.ToLower,
"title":    strings.Title,
"trim":     strings.TrimSpace,
"replace":  strings.ReplaceAll,
"contains": strings.Contains,
"split":    strings.Split,
"join":     strings.Join,
"default": func(defaultVal, val interface{}) interface{} {
if val == nil || val == "" {
return defaultVal
}
return val
},
"toJson": func(v interface{}) string {
b, _ := json.Marshal(v)
return string(b)
},
"fromJson": func(s string) map[string]interface{} {
var m map[string]interface{}
json.Unmarshal([]byte(s), &m)
return m
},
}
}
