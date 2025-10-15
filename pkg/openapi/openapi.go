package openapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"odin/pkg/service"
)

// Spec represents an OpenAPI 3.0 specification
type Spec struct {
	OpenAPI    string              `json:"openapi" yaml:"openapi"`
	Info       Info                `json:"info" yaml:"info"`
	Servers    []Server            `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths      map[string]PathItem `json:"paths" yaml:"paths"`
	Components Components          `json:"components,omitempty" yaml:"components,omitempty"`
}

// Info represents API information
type Info struct {
	Title       string   `json:"title" yaml:"title"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string   `json:"version" yaml:"version"`
	Contact     *Contact `json:"contact,omitempty" yaml:"contact,omitempty"`
}

// Contact represents contact information
type Contact struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
	URL   string `json:"url,omitempty" yaml:"url,omitempty"`
}

// Server represents an API server
type Server struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// PathItem represents operations available on a single path
type PathItem struct {
	Get     *Operation `json:"get,omitempty" yaml:"get,omitempty"`
	Post    *Operation `json:"post,omitempty" yaml:"post,omitempty"`
	Put     *Operation `json:"put,omitempty" yaml:"put,omitempty"`
	Delete  *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
	Patch   *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
	Options *Operation `json:"options,omitempty" yaml:"options,omitempty"`
}

// Operation represents an API operation
type Operation struct {
	Summary     string                `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Tags        []string              `json:"tags,omitempty" yaml:"tags,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses" yaml:"responses"`
	Security    []map[string][]string `json:"security,omitempty" yaml:"security,omitempty"`
}

// Parameter represents an operation parameter
type Parameter struct {
	Name        string  `json:"name" yaml:"name"`
	In          string  `json:"in" yaml:"in"` // query, header, path, cookie
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool    `json:"required,omitempty" yaml:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// RequestBody represents a request body
type RequestBody struct {
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                 `json:"required,omitempty" yaml:"required,omitempty"`
	Content     map[string]MediaType `json:"content" yaml:"content"`
}

// Response represents an API response
type Response struct {
	Description string               `json:"description" yaml:"description"`
	Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

// MediaType represents a media type
type MediaType struct {
	Schema *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// Schema represents a data schema
type Schema struct {
	Type       string             `json:"type,omitempty" yaml:"type,omitempty"`
	Properties map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items      *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	Required   []string           `json:"required,omitempty" yaml:"required,omitempty"`
	Example    interface{}        `json:"example,omitempty" yaml:"example,omitempty"`
}

// Components holds reusable objects
type Components struct {
	Schemas         map[string]*Schema        `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
}

// SecurityScheme defines a security scheme
type SecurityScheme struct {
	Type         string `json:"type" yaml:"type"` // apiKey, http, oauth2, openIdConnect
	Scheme       string `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"`
	In           string `json:"in,omitempty" yaml:"in,omitempty"`
	Name         string `json:"name,omitempty" yaml:"name,omitempty"`
}

// Generator generates OpenAPI documentation from service configurations
type Generator struct {
	spec *Spec
}

// NewGenerator creates a new OpenAPI generator
func NewGenerator(title, version, description string) *Generator {
	return &Generator{
		spec: &Spec{
			OpenAPI: "3.0.0",
			Info: Info{
				Title:       title,
				Version:     version,
				Description: description,
			},
			Paths: make(map[string]PathItem),
			Components: Components{
				Schemas:         make(map[string]*Schema),
				SecuritySchemes: make(map[string]SecurityScheme),
			},
		},
	}
}

// AddServer adds a server to the specification
func (g *Generator) AddServer(url, description string) {
	g.spec.Servers = append(g.spec.Servers, Server{
		URL:         url,
		Description: description,
	})
}

// AddSecurityScheme adds a security scheme
func (g *Generator) AddSecurityScheme(name string, scheme SecurityScheme) {
	g.spec.Components.SecuritySchemes[name] = scheme
}

// GenerateFromServices generates OpenAPI spec from service configurations
func (g *Generator) GenerateFromServices(services []*service.Config) error {
	for _, svc := range services {
		if err := g.addService(svc); err != nil {
			return fmt.Errorf("failed to add service %s: %w", svc.Name, err)
		}
	}
	return nil
}

// addService adds a service to the OpenAPI specification
func (g *Generator) addService(svc *service.Config) error {
	// Extract base path
	basePath := svc.BasePath
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	// Generate common operations for the service
	tag := svc.Name

	// List operation
	listPath := basePath
	g.addPath(listPath, "get", &Operation{
		Summary:     fmt.Sprintf("List %s", tag),
		Description: fmt.Sprintf("Retrieve a list of %s", tag),
		OperationID: fmt.Sprintf("list%s", toCamelCase(tag)),
		Tags:        []string{tag},
		Parameters: []Parameter{
			{
				Name:        "limit",
				In:          "query",
				Description: "Maximum number of items to return",
				Schema:      &Schema{Type: "integer"},
			},
			{
				Name:        "offset",
				In:          "query",
				Description: "Number of items to skip",
				Schema:      &Schema{Type: "integer"},
			},
		},
		Responses: map[string]Response{
			"200": {
				Description: "Successful response",
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "array",
							Items: &Schema{
								Type: "object",
							},
						},
					},
				},
			},
		},
	})

	// Create operation
	g.addPath(listPath, "post", &Operation{
		Summary:     fmt.Sprintf("Create %s", tag),
		Description: fmt.Sprintf("Create a new %s", tag),
		OperationID: fmt.Sprintf("create%s", toCamelCase(tag)),
		Tags:        []string{tag},
		RequestBody: &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/json": {
					Schema: &Schema{Type: "object"},
				},
			},
		},
		Responses: map[string]Response{
			"201": {
				Description: "Created successfully",
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Type: "object"},
					},
				},
			},
		},
	})

	// Get by ID operation
	idPath := basePath + "/{id}"
	g.addPath(idPath, "get", &Operation{
		Summary:     fmt.Sprintf("Get %s by ID", tag),
		Description: fmt.Sprintf("Retrieve a specific %s by ID", tag),
		OperationID: fmt.Sprintf("get%sById", toCamelCase(tag)),
		Tags:        []string{tag},
		Parameters: []Parameter{
			{
				Name:        "id",
				In:          "path",
				Description: fmt.Sprintf("%s ID", tag),
				Required:    true,
				Schema:      &Schema{Type: "string"},
			},
		},
		Responses: map[string]Response{
			"200": {
				Description: "Successful response",
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Type: "object"},
					},
				},
			},
			"404": {
				Description: "Not found",
			},
		},
	})

	// Update operation
	g.addPath(idPath, "put", &Operation{
		Summary:     fmt.Sprintf("Update %s", tag),
		Description: fmt.Sprintf("Update an existing %s", tag),
		OperationID: fmt.Sprintf("update%s", toCamelCase(tag)),
		Tags:        []string{tag},
		Parameters: []Parameter{
			{
				Name:        "id",
				In:          "path",
				Description: fmt.Sprintf("%s ID", tag),
				Required:    true,
				Schema:      &Schema{Type: "string"},
			},
		},
		RequestBody: &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/json": {
					Schema: &Schema{Type: "object"},
				},
			},
		},
		Responses: map[string]Response{
			"200": {
				Description: "Updated successfully",
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Type: "object"},
					},
				},
			},
		},
	})

	// Delete operation
	g.addPath(idPath, "delete", &Operation{
		Summary:     fmt.Sprintf("Delete %s", tag),
		Description: fmt.Sprintf("Delete a %s", tag),
		OperationID: fmt.Sprintf("delete%s", toCamelCase(tag)),
		Tags:        []string{tag},
		Parameters: []Parameter{
			{
				Name:        "id",
				In:          "path",
				Description: fmt.Sprintf("%s ID", tag),
				Required:    true,
				Schema:      &Schema{Type: "string"},
			},
		},
		Responses: map[string]Response{
			"204": {
				Description: "Deleted successfully",
			},
		},
	})

	// Add authentication if required
	if svc.Authentication {
		for path := range g.spec.Paths {
			if strings.HasPrefix(path, basePath) {
				pathItem := g.spec.Paths[path]
				operations := []*Operation{
					pathItem.Get, pathItem.Post, pathItem.Put,
					pathItem.Delete, pathItem.Patch, pathItem.Options,
				}
				for _, op := range operations {
					if op != nil {
						op.Security = []map[string][]string{
							{"bearerAuth": {}},
						}
					}
				}
				g.spec.Paths[path] = pathItem
			}
		}
	}

	return nil
}

// addPath adds or updates a path in the specification
func (g *Generator) addPath(path, method string, operation *Operation) {
	pathItem, exists := g.spec.Paths[path]
	if !exists {
		pathItem = PathItem{}
	}

	switch strings.ToLower(method) {
	case "get":
		pathItem.Get = operation
	case "post":
		pathItem.Post = operation
	case "put":
		pathItem.Put = operation
	case "delete":
		pathItem.Delete = operation
	case "patch":
		pathItem.Patch = operation
	case "options":
		pathItem.Options = operation
	}

	g.spec.Paths[path] = pathItem
}

// ToJSON converts the specification to JSON
func (g *Generator) ToJSON() ([]byte, error) {
	return json.MarshalIndent(g.spec, "", "  ")
}

// ToYAML converts the specification to YAML (returns JSON for now)
func (g *Generator) ToYAML() ([]byte, error) {
	// For simplicity, return JSON. In production, use gopkg.in/yaml.v3
	return g.ToJSON()
}

// GetSpec returns the OpenAPI specification
func (g *Generator) GetSpec() *Spec {
	return g.spec
}

// Helper functions

func toCamelCase(s string) string {
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	words := strings.Fields(s)
	for i := range words {
		words[i] = strings.Title(strings.ToLower(words[i]))
	}
	return strings.Join(words, "")
}

// Importer imports OpenAPI specifications and converts them to service configurations
type Importer struct{}

// NewImporter creates a new OpenAPI importer
func NewImporter() *Importer {
	return &Importer{}
}

// Import imports an OpenAPI specification and converts it to service configurations
func (i *Importer) Import(spec *Spec) ([]*service.Config, error) {
	var services []*service.Config
	serviceMap := make(map[string]*service.Config)

	// Group paths by tag (service)
	for path, pathItem := range spec.Paths {
		operations := map[string]*Operation{
			"GET":     pathItem.Get,
			"POST":    pathItem.Post,
			"PUT":     pathItem.Put,
			"DELETE":  pathItem.Delete,
			"PATCH":   pathItem.Patch,
			"OPTIONS": pathItem.Options,
		}

		for _, op := range operations {
			if op == nil {
				continue
			}

			// Use first tag as service name
			serviceName := "default-service"
			if len(op.Tags) > 0 {
				serviceName = op.Tags[0]
			}

			// Get or create service config
			svc, exists := serviceMap[serviceName]
			if !exists {
				basePath := extractBasePath(path)
				svc = &service.Config{
					Name:           serviceName,
					BasePath:       basePath,
					Targets:        []string{},
					Timeout:        30 * time.Second,
					Authentication: len(op.Security) > 0,
				}
				serviceMap[serviceName] = svc
			}
		}
	}

	// Convert map to slice
	for _, svc := range serviceMap {
		// Extract base URL from servers if available
		if len(spec.Servers) > 0 {
			svc.Targets = []string{spec.Servers[0].URL}
		}
		services = append(services, svc)
	}

	return services, nil
}

// ImportFromJSON imports an OpenAPI specification from JSON
func (i *Importer) ImportFromJSON(data []byte) ([]*service.Config, error) {
	var spec Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OpenAPI spec: %w", err)
	}

	return i.Import(&spec)
}

// extractBasePath extracts the base path from a full path
func extractBasePath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 0 {
		return "/" + parts[0]
	}
	return "/"
}
