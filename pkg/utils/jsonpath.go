package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

// JSONPath extracts a value from JSON data using a simple JSONPath-like syntax
// Example: $.users[0].name
func JSONPath(data []byte, path string) (interface{}, error) {
	if !strings.HasPrefix(path, "$.") {
		return nil, fmt.Errorf("path must start with $.")
	}

	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, err
	}

	// Remove $. prefix
	path = strings.TrimPrefix(path, "$.")
	if path == "" {
		return jsonData, nil
	}

	// Split path components
	components := strings.Split(path, ".")
	result := jsonData

	for _, component := range components {
		// Handle array indexing
		if strings.Contains(component, "[") && strings.Contains(component, "]") {
			parts := strings.SplitN(component, "[", 2)
			fieldName := parts[0]
			indexStr := strings.TrimRight(parts[1], "]")

			// Get the map at field name
			mapObj, ok := result.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("expected object at path '%s', got %T", fieldName, result)
			}

			result, ok = mapObj[fieldName]
			if !ok {
				return nil, fmt.Errorf("field '%s' not found", fieldName)
			}

			// Parse array index
			var index int
			if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
				return nil, fmt.Errorf("invalid array index '%s'", indexStr)
			}

			// Access array element
			array, ok := result.([]interface{})
			if !ok {
				return nil, fmt.Errorf("expected array at path '%s[%d]', got %T", fieldName, index, result)
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
func SetJSONPath(data []byte, path string, value interface{}) ([]byte, error) {
	if !strings.HasPrefix(path, "$.") {
		return nil, fmt.Errorf("path must start with $.")
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, err
	}

	// Remove $. prefix
	path = strings.TrimPrefix(path, "$.")
	if path == "" {
		// Setting the root object - just marshal the value
		return json.Marshal(value)
	}

	// Split path components
	components := strings.Split(path, ".")

	// Recursive function to set the value
	var setNestedValue func(obj map[string]interface{}, pathComponents []string, val interface{}) error
	setNestedValue = func(obj map[string]interface{}, pathComponents []string, val interface{}) error {
		if len(pathComponents) == 1 {
			// We're at the last component, set the value
			obj[pathComponents[0]] = val
			return nil
		}

		// We need to navigate deeper
		component := pathComponents[0]
		remaining := pathComponents[1:]

		// Check if this key exists
		nextObj, exists := obj[component]
		if !exists {
			// Create a new map for this path
			newMap := make(map[string]interface{})
			obj[component] = newMap
			return setNestedValue(newMap, remaining, val)
		}

		// Key exists, check if it's a map
		nextMap, ok := nextObj.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cannot set value: path component '%s' is not an object", component)
		}

		return setNestedValue(nextMap, remaining, val)
	}

	if err := setNestedValue(jsonData, components, value); err != nil {
		return nil, err
	}

	return json.Marshal(jsonData)
}
