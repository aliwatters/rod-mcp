package tools

import "fmt"

// getStringArg extracts a required string argument from the MCP request.
func getStringArg(args map[string]interface{}, key string) (string, error) {
	v, ok := args[key]
	if !ok {
		return "", fmt.Errorf("missing required argument: %s", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("argument %s must be a string, got %T", key, v)
	}
	return s, nil
}

// getFloatArg extracts a required float64 argument from the MCP request.
func getFloatArg(args map[string]interface{}, key string) (float64, error) {
	v, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("missing required argument: %s", key)
	}
	f, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("argument %s must be a number, got %T", key, v)
	}
	return f, nil
}

// getOptionalStringArg extracts an optional string argument, returning "" if not present.
func getOptionalStringArg(args map[string]interface{}, key string) string {
	v, _ := args[key].(string)
	return v
}

// getOptionalFloatArg extracts an optional float64 argument, returning the fallback if not present.
func getOptionalFloatArg(args map[string]interface{}, key string, fallback float64) float64 {
	if v, ok := args[key].(float64); ok {
		return v
	}
	return fallback
}

// getOptionalBoolArg extracts an optional bool argument, returning the fallback if not present.
func getOptionalBoolArg(args map[string]interface{}, key string, fallback bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return fallback
}
