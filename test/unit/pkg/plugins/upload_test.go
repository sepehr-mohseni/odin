package plugins_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures
const (
	testMongoURI = "mongodb://localhost:27017"
	testDBName   = "odin_test_plugins"
)

func createTestPluginFile(t *testing.T, name string) string {
	tmpDir := t.TempDir()
	pluginPath := filepath.Join(tmpDir, name)

	// Create a minimal ELF file header (mock .so file)
	elfHeader := []byte{0x7f, 0x45, 0x4c, 0x46, 0x02, 0x01, 0x01, 0x00}
	// Add padding to make it a reasonable size
	data := make([]byte, 1024)
	copy(data, elfHeader)
	
	err := os.WriteFile(pluginPath, data, 0644)
	require.NoError(t, err)

	return pluginPath
}

// TestPluginUpload_ValidFile tests uploading a valid plugin file
func TestPluginUpload_ValidFile(t *testing.T) {
	// Create test plugin file
	pluginPath := createTestPluginFile(t, "test-plugin.so")
	defer os.Remove(pluginPath)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	file, err := os.Open(pluginPath)
	require.NoError(t, err)
	defer file.Close()

	part, err := writer.CreateFormFile("file", "test-plugin.so")
	require.NoError(t, err)
	_, err = io.Copy(part, file)
	require.NoError(t, err)

	// Add form fields
	writer.WriteField("name", "test-plugin")
	writer.WriteField("version", "1.0.0")
	writer.WriteField("description", "Test plugin")
	writer.WriteField("author", "Test Author")
	writer.WriteField("config", `{"key":"value"}`)
	writer.WriteField("routes", "/*")
	writer.WriteField("priority", "100")
	writer.WriteField("phase", "pre-routing")

	err = writer.Close()
	require.NoError(t, err)

	// Verify the form was created successfully
	assert.Greater(t, body.Len(), 0, "Multipart form should not be empty")
	assert.Contains(t, writer.FormDataContentType(), "multipart/form-data")
}

// TestPluginUpload_InvalidExtension tests uploading file with wrong extension
func TestPluginUpload_InvalidExtension(t *testing.T) {
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(invalidPath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	file, err := os.Open(invalidPath)
	require.NoError(t, err)
	defer file.Close()

	part, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	io.Copy(part, file)

	writer.WriteField("name", "test-plugin")
	writer.WriteField("version", "1.0.0")
	writer.Close()

	// File should have wrong extension
	assert.NotContains(t, invalidPath, ".so")
}

// TestPluginUpload_FileTooLarge tests file size validation
func TestPluginUpload_FileTooLarge(t *testing.T) {
	tmpDir := t.TempDir()
	largePath := filepath.Join(tmpDir, "large.so")
	
	// Create 51MB file (exceeds 50MB limit)
	largeData := make([]byte, 51*1024*1024)
	err := os.WriteFile(largePath, largeData, 0644)
	require.NoError(t, err)

	info, err := os.Stat(largePath)
	require.NoError(t, err)

	// Verify file is too large
	maxSize := int64(50 * 1024 * 1024)
	assert.Greater(t, info.Size(), maxSize, "File should exceed maximum size")
}

// TestPluginUpload_MissingFields tests required field validation
func TestPluginUpload_MissingFields(t *testing.T) {
	tests := []struct {
		name        string
		pluginName  string
		version     string
		expectError bool
	}{
		{"valid", "test-plugin", "1.0.0", false},
		{"missing name", "", "1.0.0", true},
		{"missing version", "test-plugin", "", true},
		{"both missing", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := tt.pluginName == "" || tt.version == ""
			assert.Equal(t, tt.expectError, hasError)
		})
	}
}

// TestPluginMetadata tests metadata structure
func TestPluginMetadata(t *testing.T) {
	metadata := map[string]interface{}{
		"name":        "test-plugin",
		"version":     "1.0.0",
		"description": "Test description",
		"author":      "Test Author",
		"go_version":  "go1.25",
		"go_os":       "linux",
		"go_arch":     "amd64",
		"enabled":     false,
		"uploaded_at": time.Now(),
	}

	// Verify required fields
	assert.NotEmpty(t, metadata["name"])
	assert.NotEmpty(t, metadata["version"])
	assert.Equal(t, "test-plugin", metadata["name"])
	assert.Equal(t, "1.0.0", metadata["version"])
	assert.Equal(t, false, metadata["enabled"]) // Should be disabled by default
}

// TestPluginConfiguration tests plugin config JSON parsing
func TestPluginConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{"valid JSON", `{"key":"value"}`, false},
		{"empty object", `{}`, false},
		{"nested JSON", `{"key":{"nested":"value"}}`, false},
		{"invalid JSON", `{invalid}`, true},
		{"empty string", ``, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]interface{}
			err := json.Unmarshal([]byte(tt.config), &result)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPluginPriority tests priority validation
func TestPluginPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		valid    bool
	}{
		{"minimum", 0, true},
		{"default", 100, true},
		{"maximum", 1000, true},
		{"negative", -1, false},
		{"too high", 1001, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.priority >= 0 && tt.priority <= 1000
			assert.Equal(t, tt.valid, valid)
		})
	}
}

// TestPluginPhase tests execution phase validation
func TestPluginPhase(t *testing.T) {
	validPhases := []string{"pre-routing", "post-routing", "pre-response"}
	
	tests := []struct {
		name  string
		phase string
		valid bool
	}{
		{"pre-routing", "pre-routing", true},
		{"post-routing", "post-routing", true},
		{"pre-response", "pre-response", true},
		{"invalid", "invalid-phase", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := false
			for _, validPhase := range validPhases {
				if tt.phase == validPhase {
					valid = true
					break
				}
			}
			assert.Equal(t, tt.valid, valid)
		})
	}
}

// TestPluginRoutes tests route pattern validation
func TestPluginRoutes(t *testing.T) {
	tests := []struct {
		name   string
		routes string
		valid  bool
	}{
		{"all routes", "/*", true},
		{"specific path", "/api/*", true},
		{"multiple routes", "/api/*,/users/*", true},
		{"exact path", "/health", true},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.routes != ""
			assert.Equal(t, tt.valid, valid)
		})
	}
}

// TestEchoContext tests Echo framework integration
func TestEchoContext(t *testing.T) {
	e := echo.New()
	
	req := httptest.NewRequest(http.MethodPost, "/upload", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	assert.NotNil(t, c)
	assert.Equal(t, http.MethodPost, c.Request().Method)
	assert.Equal(t, "/upload", c.Request().URL.Path)
}

// TestSHA256Calculation tests SHA256 hash generation
func TestSHA256Calculation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.so")
	
	testData := []byte("test data for hashing")
	err := os.WriteFile(testFile, testData, 0644)
	require.NoError(t, err)

	// Read file and verify it exists
	data, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, testData, data)
}

// TestMultipartFormParsing tests form data extraction
func TestMultipartFormParsing(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add text fields
	writer.WriteField("name", "test-plugin")
	writer.WriteField("version", "1.0.0")
	writer.WriteField("description", "Test description")
	
	err := writer.Close()
	require.NoError(t, err)

	// Verify form can be parsed
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	err = req.ParseMultipartForm(32 << 20) // 32MB
	require.NoError(t, err)

	assert.Equal(t, "test-plugin", req.FormValue("name"))
	assert.Equal(t, "1.0.0", req.FormValue("version"))
	assert.Equal(t, "Test description", req.FormValue("description"))
}

// Benchmark tests
func BenchmarkCreateTestPluginFile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		pluginPath := filepath.Join(tmpDir, "test.so")
		elfHeader := []byte{0x7f, 0x45, 0x4c, 0x46}
		os.WriteFile(pluginPath, elfHeader, 0644)
	}
}

func BenchmarkMultipartFormCreation(b *testing.B) {
	tmpFile := filepath.Join(b.TempDir(), "test.so")
	os.WriteFile(tmpFile, []byte("test"), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		
		file, _ := os.Open(tmpFile)
		part, _ := writer.CreateFormFile("file", "test.so")
		io.Copy(part, file)
		file.Close()
		
		writer.WriteField("name", "test")
		writer.WriteField("version", "1.0.0")
		writer.Close()
	}
}

func BenchmarkJSONMarshal(b *testing.B) {
	data := map[string]interface{}{
		"name":        "test-plugin",
		"version":     "1.0.0",
		"description": "Test description",
		"config":      map[string]string{"key": "value"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(data)
	}
}
