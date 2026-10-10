package sorx

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// ============================================================================
// Transform Actions
// ============================================================================

func transformMapFields(ctx context.Context, ac ActionContext) (interface{}, error) {
	mapping, _ := ac.Params["mapping"].(string)
	source, _ := ac.Params["source"].(string)

	slog.Info("transformMapFields", "mapping", mapping, "source", source)

	// Get data from previous step
	sourceData := ac.Execution.StepResults[source]
	if sourceData == nil {
		return map[string]interface{}{
			"mapping":     mapping,
			"transformed": []interface{}{},
			"error":       "source data not found",
		}, nil
	}

	// Parse mapping (format: "field1:targetField1,field2:targetField2")
	mappingPairs := strings.Split(mapping, ",")
	fieldMap := make(map[string]string)
	for _, pair := range mappingPairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) == 2 {
			fieldMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	// Transform data
	transformed := make([]interface{}, 0)
	if dataList, ok := sourceData.([]interface{}); ok {
		for _, item := range dataList {
			if itemMap, ok := item.(map[string]interface{}); ok {
				newItem := make(map[string]interface{})
				for srcField, targetField := range fieldMap {
					if val, exists := itemMap[srcField]; exists {
						newItem[targetField] = val
					}
				}
				transformed = append(transformed, newItem)
			}
		}
	}

	return map[string]interface{}{
		"mapping":     mapping,
		"transformed": transformed,
		"count":       len(transformed),
	}, nil
}

func transformFilter(ctx context.Context, ac ActionContext) (interface{}, error) {
	condition, _ := ac.Params["condition"].(string)
	source, _ := ac.Params["source"].(string)

	slog.Info("transformFilter", "condition", condition, "source", source)

	// Get data from previous step
	sourceData := ac.Execution.StepResults[source]
	if sourceData == nil {
		return map[string]interface{}{
			"condition": condition,
			"filtered":  []interface{}{},
			"error":     "source data not found",
		}, nil
	}

	// Parse condition (format: "field operator value", e.g., "status equals active")
	parts := strings.Fields(condition)
	if len(parts) < 3 {
		return map[string]interface{}{
			"condition": condition,
			"filtered":  []interface{}{},
			"error":     "invalid condition format",
		}, nil
	}

	field := parts[0]
	operator := parts[1]
	value := strings.Join(parts[2:], " ")

	// Filter data
	filtered := make([]interface{}, 0)
	if dataList, ok := sourceData.([]interface{}); ok {
		for _, item := range dataList {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if fieldVal, exists := itemMap[field]; exists {
					match := false
					switch operator {
					case "equals", "=", "==":
						match = fmt.Sprintf("%v", fieldVal) == value
					case "contains":
						match = strings.Contains(fmt.Sprintf("%v", fieldVal), value)
					case "starts_with":
						match = strings.HasPrefix(fmt.Sprintf("%v", fieldVal), value)
					}
					if match {
						filtered = append(filtered, item)
					}
				}
			}
		}
	}

	return map[string]interface{}{
		"condition": condition,
		"filtered":  filtered,
		"count":     len(filtered),
	}, nil
}
