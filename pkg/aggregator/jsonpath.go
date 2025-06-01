package aggregator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

type JSONPathExpressionData struct {
	arrayName      string
	expressionType string
	index          int
	filterField    string
	resultField    string
}

func (a *Aggregator) extractValues(data map[string]interface{}, path string) []string {
	path = strings.TrimPrefix(path, "$.")

	arrayWildcardPattern := regexp.MustCompile(`(.+)\[(\*|\d+)\]\.(.+)`)
	if match := arrayWildcardPattern.FindStringSubmatch(path); match != nil {
		arrayName := match[1]
		propertyName := match[3]

		array, ok := data[arrayName].([]interface{})
		if !ok {
			return nil
		}

		values := make([]string, 0, len(array))
		for _, item := range array {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if val, exists := itemMap[propertyName]; exists {
					switch v := val.(type) {
					case string:
						values = append(values, v)
					case float64:
						values = append(values, fmt.Sprintf("%v", v))
					case int:
						values = append(values, fmt.Sprintf("%d", v))
					}
				}
			}
		}
		return values
	}

	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if i == len(parts)-1 {
			if val, exists := current[part]; exists {
				switch v := val.(type) {
				case string:
					return []string{v}
				case float64:
					return []string{fmt.Sprintf("%v", v)}
				case int:
					return []string{fmt.Sprintf("%d", v)}
				}
			}
		} else {
			next, ok := current[part].(map[string]interface{})
			if !ok {
				return nil
			}
			current = next
		}
	}

	return nil
}

func (a *Aggregator) mapData(target map[string]interface{}, source map[string]interface{}, fromPath, toPath, paramValue, paramName string) {
	if fromPath == "$" {
		toPath = strings.TrimPrefix(toPath, "$.")

		if a.isJSONPathExpression(toPath) {
			a.processJSONPathMapping(target, source, toPath, paramValue, paramName)
			return
		}

		parts := strings.Split(toPath, ".")
		current := target

		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = source
			} else {
				next, exists := current[part].(map[string]interface{})
				if !exists {
					next = make(map[string]interface{})
					current[part] = next
				}
				current = next
			}
		}
	}
}

func (a *Aggregator) isJSONPathExpression(path string) bool {
	return strings.Contains(path, "[") && strings.Contains(path, "]")
}

func (a *Aggregator) parseJSONPathExpression(path string) *JSONPathExpressionData {
	wildcardPattern := regexp.MustCompile(`^(.+)\[(\*)\]\.(.+)$`)
	if matches := wildcardPattern.FindStringSubmatch(path); matches != nil {
		return &JSONPathExpressionData{
			arrayName:      matches[1],
			expressionType: "wildcard",
			resultField:    matches[3],
		}
	}

	filterPattern := regexp.MustCompile(`^(.+)\[\?\(@\.([^=]+)=(?:\{([^}]+)\}|([^}]+))\)\]\.(.+)$`)
	if matches := filterPattern.FindStringSubmatch(path); matches != nil {
		return &JSONPathExpressionData{
			arrayName:      matches[1],
			expressionType: "filter",
			filterField:    matches[2],
			resultField:    matches[5],
		}
	}

	indexPattern := regexp.MustCompile(`^(.+)\[(\d+)\]\.(.+)$`)
	if matches := indexPattern.FindStringSubmatch(path); matches != nil {
		index, _ := strconv.Atoi(matches[2])
		return &JSONPathExpressionData{
			arrayName:      matches[1],
			expressionType: "index",
			index:          index,
			resultField:    matches[3],
		}
	}

	return nil
}

func (a *Aggregator) processJSONPathMapping(target map[string]interface{}, source map[string]interface{}, path string, paramValue, paramName string) {
	expressionData := a.parseJSONPathExpression(path)
	if expressionData == nil {
		a.logger.WithField("path", path).Warn("Failed to parse JSONPath expression")
		return
	}

	a.logger.WithFields(logrus.Fields{
		"expression": path,
		"parsed":     expressionData,
	}).Debug("Parsed JSONPath expression")

	arrayPath := expressionData.arrayName
	array, ok := getNestedValue(target, strings.Split(arrayPath, "."))
	if !ok {
		a.logger.WithField("arrayPath", arrayPath).Warn("Array not found for JSONPath expression")
		return
	}

	arrayItems, ok := array.([]interface{})
	if !ok {
		a.logger.WithField("arrayPath", arrayPath).Warn("Value is not an array")
		return
	}

	switch expressionData.expressionType {
	case "wildcard":
		a.processWildcardExpression(target, arrayPath, arrayItems, expressionData, source, paramValue)
	case "filter":
		a.processFilterExpression(target, arrayPath, arrayItems, expressionData, source, paramValue, paramName)
	case "index":
		a.processIndexExpression(target, arrayPath, arrayItems, expressionData, source)
	}
}

func (a *Aggregator) processWildcardExpression(target map[string]interface{}, arrayPath string,
	arrayItems []interface{}, expressionData *JSONPathExpressionData, source map[string]interface{}, paramValue string) {

	matchingIndices := []int{}
	for i, item := range arrayItems {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		for key, val := range itemMap {
			if strings.HasSuffix(strings.ToLower(key), "id") {
				if fmt.Sprintf("%v", val) == paramValue {
					matchingIndices = append(matchingIndices, i)
					break
				}
			}
		}
	}

	if len(matchingIndices) > 0 {
		for _, idx := range matchingIndices {
			a.setResultOnArrayItem(arrayItems, idx, expressionData.resultField, source)
		}
		setNestedValue(target, strings.Split(arrayPath, "."), arrayItems)
	} else {
		a.logger.WithFields(logrus.Fields{
			"paramValue": paramValue,
			"arrayPath":  arrayPath,
		}).Debug("No array items matched the parameter value in wildcard expression")
	}
}

func (a *Aggregator) processFilterExpression(target map[string]interface{}, arrayPath string,
	arrayItems []interface{}, expressionData *JSONPathExpressionData, source map[string]interface{}, paramValue, paramName string) {

	itemUpdated := false

	for i, item := range arrayItems {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		fieldValue, exists := itemMap[expressionData.filterField]
		if !exists {
			continue
		}

		fieldValueStr := fmt.Sprintf("%v", fieldValue)
		if fieldValueStr == paramValue {
			a.logger.WithFields(logrus.Fields{
				"index":       i,
				"field":       expressionData.filterField,
				"value":       fieldValueStr,
				"resultField": expressionData.resultField,
			}).Debug("Found matching item for filter expression")

			a.setResultOnArrayItem(arrayItems, i, expressionData.resultField, source)
			itemUpdated = true
		}
	}

	if itemUpdated {
		setNestedValue(target, strings.Split(arrayPath, "."), arrayItems)
	}

	keysToDelete := make([]string, 0)
	for k := range target {
		if strings.Contains(k, "?(@") || strings.HasPrefix(k, "[") {
			keysToDelete = append(keysToDelete, k)
		}
	}

	for _, key := range keysToDelete {
		delete(target, key)
	}
}

func (a *Aggregator) processIndexExpression(target map[string]interface{}, arrayPath string,
	arrayItems []interface{}, expressionData *JSONPathExpressionData, source map[string]interface{}) {

	if expressionData.index < 0 || expressionData.index >= len(arrayItems) {
		a.logger.WithFields(logrus.Fields{
			"index":       expressionData.index,
			"arrayLength": len(arrayItems),
		}).Warn("Index out of bounds in JSONPath expression")
		return
	}

	a.setResultOnArrayItem(arrayItems, expressionData.index, expressionData.resultField, source)
	setNestedValue(target, strings.Split(arrayPath, "."), arrayItems)
}

func (a *Aggregator) setResultOnArrayItem(arrayItems []interface{}, index int, resultField string, resultValue map[string]interface{}) {
	if index < 0 || index >= len(arrayItems) {
		return
	}

	item, ok := arrayItems[index].(map[string]interface{})
	if !ok {
		a.logger.WithField("index", index).Warn("Array item is not an object")
		return
	}

	resultParts := strings.Split(resultField, ".")
	currentMap := item

	for i, part := range resultParts {
		if i == len(resultParts)-1 {
			currentMap[part] = resultValue
		} else {
			next, exists := currentMap[part].(map[string]interface{})
			if !exists {
				next = make(map[string]interface{})
				currentMap[part] = next
			}
			currentMap = next
		}
	}

	arrayItems[index] = item
}

func setNestedValue(data map[string]interface{}, path []string, value interface{}) {
	if len(path) == 0 {
		return
	}

	current := data
	for i, part := range path {
		if i == len(path)-1 {
			current[part] = value
			return
		}

		nextMap, ok := current[part].(map[string]interface{})
		if !ok {
			nextMap = make(map[string]interface{})
			current[part] = nextMap
		}
		current = nextMap
	}
}

func getNestedValue(data map[string]interface{}, path []string) (interface{}, bool) {
	if len(path) == 0 {
		return nil, false
	}

	current := data
	for i, part := range path {
		if i == len(path)-1 {
			val, exists := current[part]
			return val, exists
		}

		next, ok := current[part].(map[string]interface{})
		if !ok {
			nextAsInterface, exists := current[part]
			if !exists {
				return nil, false
			}
			next, ok = nextAsInterface.(map[string]interface{})
			if !ok {
				return nil, false
			}
		}
		current = next
	}

	return nil, false
}

func parseArrayIndex(segment string) (int, bool) {
	if !strings.HasPrefix(segment, "[") || !strings.HasSuffix(segment, "]") {
		return 0, false
	}

	indexStr := segment[1 : len(segment)-1]
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return 0, false
	}

	return index, true
}
