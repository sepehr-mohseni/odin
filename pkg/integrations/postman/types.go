package postman

import "time"

// PostmanCollection represents a Postman collection structure
type PostmanCollection struct {
	Info  CollectionInfo   `json:"info"`
	Item  []CollectionItem `json:"item"`
	Auth  *Auth            `json:"auth,omitempty"`
	Event []Event          `json:"event,omitempty"`
}

// CollectionInfo contains collection metadata
type CollectionInfo struct {
	ID          string `json:"_postman_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema"`
	Version     string `json:"version,omitempty"`
}

// CollectionItem represents a request or folder in the collection
type CollectionItem struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Request     *Request         `json:"request,omitempty"`
	Item        []CollectionItem `json:"item,omitempty"` // For folders
	Event       []Event          `json:"event,omitempty"`
}

// Request represents a Postman request
type Request struct {
	Method      string       `json:"method"`
	Header      []Header     `json:"header,omitempty"`
	Body        *RequestBody `json:"body,omitempty"`
	URL         *URL         `json:"url"`
	Auth        *Auth        `json:"auth,omitempty"`
	Description string       `json:"description,omitempty"`
}

// Header represents an HTTP header
type Header struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Type        string `json:"type,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
	Description string `json:"description,omitempty"`
}

// RequestBody represents request body
type RequestBody struct {
	Mode       string                 `json:"mode"` // raw, formdata, urlencoded, file, graphql
	Raw        string                 `json:"raw,omitempty"`
	URLEncoded []KeyValue             `json:"urlencoded,omitempty"`
	FormData   []FormDataItem         `json:"formdata,omitempty"`
	GraphQL    map[string]interface{} `json:"graphql,omitempty"`
	Options    map[string]interface{} `json:"options,omitempty"`
}

// KeyValue represents a key-value pair
type KeyValue struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Type        string `json:"type,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
	Description string `json:"description,omitempty"`
}

// FormDataItem represents form data item
type FormDataItem struct {
	Key         string `json:"key"`
	Value       string `json:"value,omitempty"`
	Src         string `json:"src,omitempty"` // For file uploads
	Type        string `json:"type"`          // text or file
	Disabled    bool   `json:"disabled,omitempty"`
	Description string `json:"description,omitempty"`
}

// URL represents a request URL
type URL struct {
	Raw      string     `json:"raw"`
	Protocol string     `json:"protocol,omitempty"`
	Host     []string   `json:"host,omitempty"`
	Path     []string   `json:"path,omitempty"`
	Port     string     `json:"port,omitempty"`
	Query    []KeyValue `json:"query,omitempty"`
	Variable []Variable `json:"variable,omitempty"`
}

// Variable represents a URL variable
type Variable struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

// Auth represents authentication configuration
type Auth struct {
	Type   string                 `json:"type"` // apikey, bearer, basic, oauth2, etc.
	APIKey []AuthAttribute        `json:"apikey,omitempty"`
	Bearer []AuthAttribute        `json:"bearer,omitempty"`
	Basic  []AuthAttribute        `json:"basic,omitempty"`
	OAuth2 []AuthAttribute        `json:"oauth2,omitempty"`
	Custom map[string]interface{} `json:"-"`
}

// AuthAttribute represents an authentication attribute
type AuthAttribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

// Event represents a pre-request or test script
type Event struct {
	Listen string `json:"listen"` // prerequest, test
	Script Script `json:"script"`
}

// Script represents JavaScript code
type Script struct {
	ID   string   `json:"id,omitempty"`
	Type string   `json:"type,omitempty"` // text/javascript
	Exec []string `json:"exec"`           // Lines of code
}

// Environment represents a Postman environment
type Environment struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Values    []EnvValue `json:"values"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// EnvValue represents an environment variable
type EnvValue struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Enabled bool   `json:"enabled"`
}

// Workspace represents a Postman workspace
type Workspace struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // personal, team, private, public
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Monitor represents a Postman monitor
type Monitor struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	CollectionID  string    `json:"collection"`
	EnvironmentID string    `json:"environment,omitempty"`
	Schedule      Schedule  `json:"schedule"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// Schedule represents a monitor schedule
type Schedule struct {
	Cron     string `json:"cron"`
	Timezone string `json:"timezone"`
}

// MockServer represents a Postman mock server
type MockServer struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	CollectionID  string    `json:"collection"`
	EnvironmentID string    `json:"environment,omitempty"`
	URL           string    `json:"mockUrl"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// API Response structures

// CollectionsResponse represents the response from /collections endpoint
type CollectionsResponse struct {
	Collections []CollectionSummary `json:"collections"`
}

// CollectionSummary is a summary of a collection
type CollectionSummary struct {
	ID        string `json:"id"`
	UID       string `json:"uid"`
	Name      string `json:"name"`
	Owner     string `json:"owner"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// CollectionResponse represents a single collection response
type CollectionResponse struct {
	Collection PostmanCollection `json:"collection"`
}

// EnvironmentsResponse represents the response from /environments endpoint
type EnvironmentsResponse struct {
	Environments []Environment `json:"environments"`
}

// EnvironmentResponse represents a single environment response
type EnvironmentResponse struct {
	Environment Environment `json:"environment"`
}

// WorkspacesResponse represents the response from /workspaces endpoint
type WorkspacesResponse struct {
	Workspaces []Workspace `json:"workspaces"`
}

// WorkspaceResponse represents a single workspace response
type WorkspaceResponse struct {
	Workspace Workspace `json:"workspace"`
}

// MonitorsResponse represents the response from /monitors endpoint
type MonitorsResponse struct {
	Monitors []Monitor `json:"monitors"`
}

// MockServersResponse represents the response from /mocks endpoint
type MockServersResponse struct {
	Mocks []MockServer `json:"mocks"`
}

// Error response
type ErrorResponse struct {
	Error struct {
		Name    string `json:"name"`
		Message string `json:"message"`
		Details string `json:"details,omitempty"`
	} `json:"error"`
}

// Integration configuration
type IntegrationConfig struct {
	Enabled      bool                `yaml:"enabled" json:"enabled" bson:"enabled"`
	APIKey       string              `yaml:"apiKey" json:"apiKey" bson:"apiKey"`
	WorkspaceID  string              `yaml:"workspaceId" json:"workspaceId" bson:"workspaceId"`
	AutoSync     bool                `yaml:"autoSync" json:"autoSync" bson:"autoSync"`
	SyncInterval string              `yaml:"syncInterval" json:"syncInterval" bson:"syncInterval"`
	Sync         SyncConfig          `yaml:"sync" json:"sync" bson:"sync"`
	Newman       NewmanConfig        `yaml:"newman" json:"newman" bson:"newman"`
	Mappings     []CollectionMapping `yaml:"mappings" json:"mappings" bson:"mappings"`
	Provider     string              `yaml:"provider" json:"provider" bson:"provider"` // Always "postman"
	CreatedAt    time.Time           `yaml:"createdAt" json:"createdAt" bson:"createdAt"`
	UpdatedAt    time.Time           `yaml:"updatedAt" json:"updatedAt" bson:"updatedAt"`
}

// SyncConfig defines what to sync
type SyncConfig struct {
	Collections  bool `yaml:"collections" json:"collections"`
	Environments bool `yaml:"environments" json:"environments"`
	Monitors     bool `yaml:"monitors" json:"monitors"`
	MockServers  bool `yaml:"mockServers" json:"mockServers"`
}

// NewmanConfig defines Newman test runner settings
type NewmanConfig struct {
	Enabled     bool     `yaml:"enabled" json:"enabled"`
	RunOnDeploy bool     `yaml:"runOnDeploy" json:"runOnDeploy"`
	Reporters   []string `yaml:"reporters" json:"reporters"`
	Timeout     int      `yaml:"timeout" json:"timeout"`
	Iterations  int      `yaml:"iterations" json:"iterations"`
}

// CollectionMapping maps Postman collections to Odin services
type CollectionMapping struct {
	PostmanCollectionID string `yaml:"postmanCollection" json:"postmanCollection"`
	OdinServiceName     string `yaml:"odinService" json:"odinService"`
	SyncDirection       string `yaml:"syncDirection" json:"syncDirection"` // postman_to_odin, odin_to_postman, bidirectional
	AutoTest            bool   `yaml:"autoTest" json:"autoTest"`
	AutoSync            bool   `yaml:"autoSync" json:"autoSync"`
}

// Newman test result
type NewmanResult struct {
	ID           string             `json:"id" bson:"_id,omitempty"`
	CollectionID string             `json:"collectionId" bson:"collectionId"`
	ServiceName  string             `json:"serviceName" bson:"serviceName"`
	Status       string             `json:"status" bson:"status"` // passed, failed, running
	Summary      NewmanSummary      `json:"summary" bson:"summary"`
	Results      []NewmanTestResult `json:"results" bson:"results"`
	Duration     int64              `json:"duration" bson:"duration"` // milliseconds
	RunAt        time.Time          `json:"runAt" bson:"runAt"`
	Error        string             `json:"error,omitempty" bson:"error,omitempty"`
	Output       string             `json:"output,omitempty" bson:"output,omitempty"` // Raw Newman output
}

// NewmanSummary contains test execution summary
type NewmanSummary struct {
	Total      int `json:"total" bson:"total"`
	Passed     int `json:"passed" bson:"passed"`
	Failed     int `json:"failed" bson:"failed"`
	Skipped    int `json:"skipped" bson:"skipped"`
	Requests   int `json:"requests" bson:"requests"`
	Iterations int `json:"iterations" bson:"iterations"`
}

// NewmanTestResult contains individual test result
type NewmanTestResult struct {
	Name           string       `json:"name" bson:"name"`
	Status         string       `json:"status" bson:"status"`
	Method         string       `json:"method,omitempty" bson:"method,omitempty"`
	URL            string       `json:"url,omitempty" bson:"url,omitempty"`
	ResponseCode   int          `json:"responseCode,omitempty" bson:"responseCode,omitempty"`
	ResponseStatus string       `json:"responseStatus,omitempty" bson:"responseStatus,omitempty"`
	Response       TestResponse `json:"response" bson:"response"`
	Assertions     []Assertion  `json:"assertions" bson:"assertions"`
	Duration       int64        `json:"duration" bson:"duration"`
	Error          string       `json:"error,omitempty" bson:"error,omitempty"`
}

// NewmanStats represents Newman execution statistics
type NewmanStats struct {
	Total   int `json:"total"`
	Failed  int `json:"failed"`
	Pending int `json:"pending"`
}

// TestResponse contains response details
type TestResponse struct {
	Code    int               `json:"code" bson:"code"`
	Status  string            `json:"status" bson:"status"`
	Headers map[string]string `json:"headers" bson:"headers"`
	Body    string            `json:"body" bson:"body"`
	Time    int64             `json:"time" bson:"time"`
}

// Assertion represents a test assertion
type Assertion struct {
	Name   string `json:"name" bson:"name"`
	Passed bool   `json:"passed" bson:"passed"`
	Error  string `json:"error,omitempty" bson:"error,omitempty"`
}
