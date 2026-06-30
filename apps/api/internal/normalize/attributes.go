package normalize

import (
	"fmt"
	"strconv"
)

func getStringValue(attrs map[string]any, key string) string {
	if attrs == nil {
		return ""
	}
	v, ok := attrs[key]
	if !ok {
		return ""
	}
	return extractStringValue(v)
}

func extractStringValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case map[string]any:
		if sv, ok := val["stringValue"]; ok {
			return extractStringValue(sv)
		}
		if iv, ok := val["intValue"]; ok {
			return strconv.FormatInt(extractIntValue(iv), 10)
		}
		if bv, ok := val["boolValue"]; ok {
			if b, ok := bv.(bool); ok {
				return strconv.FormatBool(b)
			}
		}
		if dv, ok := val["doubleValue"]; ok {
			if d, ok := dv.(float64); ok {
				return strconv.FormatFloat(d, 'f', -1, 64)
			}
		}
		return ""
	default:
		return ""
	}
}

func extractIntValue(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		var n int64
		fmt.Sscanf(val, "%d", &n)
		return n
	case map[string]any:
		if iv, ok := val["intValue"]; ok {
			return extractIntValue(iv)
		}
		if sv, ok := val["stringValue"]; ok {
			var n int64
			fmt.Sscanf(extractStringValue(sv), "%d", &n)
			return n
		}
		return 0
	default:
		return 0
	}
}

func extractBoolValue(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case map[string]any:
		if bv, ok := val["boolValue"]; ok {
			if b, ok := bv.(bool); ok {
				return b
			}
		}
		return false
	default:
		return false
	}
}

func getAllStringAttributes(attrs map[string]any) map[string]string {
	result := make(map[string]string)
	for k, v := range attrs {
		result[k] = extractStringValue(v)
	}
	return result
}

func getIntAttribute(attrs map[string]any, key string) int64 {
	if attrs == nil {
		return 0
	}
	v, ok := attrs[key]
	if !ok {
		return 0
	}
	return extractIntValue(v)
}