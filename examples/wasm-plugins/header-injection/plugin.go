//go:build tinygo.wasm
// +build tinygo.wasm

// Example WASM Plugin: Header Injection
// This plugin adds custom headers to requests
// Compile with: tinygo build -o plugin.wasm -target=wasi plugin.go

package main

import (
	"encoding/json"
	"unsafe"
)

// Input represents the input data from the gateway
type Input struct {
	Context struct {
		RequestID string                 `json:"requestId"`
		ServiceID string                 `json:"serviceId"`
		RoutePath string                 `json:"routePath"`
		Config    map[string]interface{} `json:"config"`
		Metadata  map[string]string      `json:"metadata"`
	} `json:"context"`
	Input struct {
		Method  string              `json:"method"`
		URL     string              `json:"url"`
		Headers map[string][]string `json:"headers"`
		Body    []byte              `json:"body"`
		Query   map[string][]string `json:"query"`
	} `json:"input"`
	Config map[string]interface{} `json:"config"`
}

// Result represents the output data to the gateway
type Result struct {
	Modified bool              `json:"modified"`
	Continue bool              `json:"continue"`
	Response *Response         `json:"response,omitempty"`
	Error    string            `json:"error,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Response represents an HTTP response
type Response struct {
	StatusCode int                 `json:"statusCode"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body"`
}

// Host functions (imported from gateway)
//
//go:wasm-module env
//export log
func log(ptr uint32, size uint32)

//go:wasm-module env
//export get_time
func get_time() int64

// Plugin functions (exported to gateway)

//export init
func init() {
	logMessage("Plugin initialized")
}

//export execute
func execute(inputPtr uint32, inputSize uint32) uint64 {
	// Read input from memory
	inputData := readMemory(inputPtr, inputSize)

	var input Input
	if err := json.Unmarshal(inputData, &input); err != nil {
		return returnError("Failed to parse input: " + err.Error())
	}

	// Get custom headers from config
	addHeaders := make(map[string]string)
	if headers, ok := input.Config["addHeaders"].(map[string]interface{}); ok {
		for k, v := range headers {
			if str, ok := v.(string); ok {
				addHeaders[k] = str
			}
		}
	}

	// Add configured headers
	if input.Input.Headers == nil {
		input.Input.Headers = make(map[string][]string)
	}

	for k, v := range addHeaders {
		input.Input.Headers[k] = []string{v}
	}

	// Add request ID header
	if input.Context.RequestID != "" {
		input.Input.Headers["X-Request-ID"] = []string{input.Context.RequestID}
	}

	// Add timestamp header
	timestamp := get_time()
	input.Input.Headers["X-Gateway-Timestamp"] = []string{string(timestamp)}

	logMessage("Headers injected successfully")

	// Return modified request
	result := Result{
		Modified: true,
		Continue: true,
		Response: &Response{
			StatusCode: 200,
			Headers:    input.Input.Headers,
			Body:       input.Input.Body,
		},
		Metadata: map[string]string{
			"plugin": "header-injection",
		},
	}

	return returnResult(&result)
}

//export cleanup
func cleanup() {
	logMessage("Plugin cleanup")
}

// Helper functions

func logMessage(msg string) {
	ptr, size := writeMemory([]byte(msg))
	log(ptr, size)
}

func readMemory(ptr uint32, size uint32) []byte {
	data := make([]byte, size)
	copy(data, (*[1 << 30]byte)(unsafe.Pointer(uintptr(ptr)))[:size:size])
	return data
}

func writeMemory(data []byte) (uint32, uint32) {
	size := uint32(len(data))
	ptr := uint32(uintptr(unsafe.Pointer(&data[0])))
	return ptr, size
}

func returnResult(result *Result) uint64 {
	data, _ := json.Marshal(result)
	ptr, size := writeMemory(data)
	return uint64(ptr) | (uint64(size) << 32)
}

func returnError(errMsg string) uint64 {
	result := Result{
		Modified: false,
		Continue: false,
		Error:    errMsg,
	}
	return returnResult(&result)
}

// Required for WASI
func main() {}
