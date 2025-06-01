package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// JSONPath extracts a value from JSON data using a simple JSONPath-like syntax
// Example: $.users[0].name
func JSONPath(data []byte, path string) (interface{}, error) {
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Remove $. prefix
	if strings.HasPrefix(path, "$.") {
		path = path[2:]
	}

	// Split path components
	components := strings.Split(path, ".")

	for _, component := range components {
		// Handle array indexing
		if strings.Contains(component, "[") && strings.Contains(component, "]") {
			// Extract field name and index
			parts := strings.Split(component, "[")
			fieldName := parts[0]
			indexPart := strings.TrimSuffix(parts[1], "]")

			// Get the map at field name
			mapObj, ok := result.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("expected object for field access '%s', got %T", fieldName, result)
			}

			arrayValue, exists := mapObj[fieldName]
			if !exists {
				return nil, fmt.Errorf("field '%s' not found", fieldName)
			}

			// Parse array index
			index, err := strconv.Atoi(indexPart)
			if err != nil {
				return nil, fmt.Errorf("invalid array index '%s': %w", indexPart, err)
			}

			// Access array element
			array, ok := arrayValue.([]interface{})
			if !ok {
				return nil, fmt.Errorf("expected array for field '%s', got %T", fieldName, arrayValue)
			}

			if index < 0 || index >= len(array) {
				return nil, fmt.Errorf("array index out of bounds: %d (array length: %d)", index, len(array))
			}

			result = array[index]
		} else {
			// Regular object property access
			mapObj, ok := result.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("expected object at path '%s', got %T", component, result)
			}

			result, ok = mapObj[component]
			if !ok {
				return nil, fmt.Errorf("field '%s' not found", component)
			}
		}
	}

	return result, nil
}

// SetJSONPath sets a value at a specific path in a JSON object
func SetJSONPath(data map[string]interface{}, path string, value interface{}) error {
	if strings.HasPrefix(path, "$.") {
		path = path[2:]
	}

	components := strings.Split(path, ".")
	current := data

	for i, component := range components {
		if i == len(components)-1 {
			// Last component, set the value
			current[component] = value
			return nil
		}

		// Navigate or create intermediate objects
		if next, exists := current[component]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return fmt.Errorf("cannot set path: intermediate component '%s' is not an object", component)
			}
		} else {
			// Create new object
			newObj := make(map[string]interface{})
			current[component] = newObj
			current = newObj
		}
	}

	return nil
}
