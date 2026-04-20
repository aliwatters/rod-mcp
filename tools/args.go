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

// getBoolArg extracts a required bool argument from the MCP request.
func getBoolArg(args map[string]interface{}, key string) (bool, error) {
	v, ok := args[key]
	if !ok {
		return false, fmt.Errorf("missing required argument: %s", key)
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("argument %s must be a boolean, got %T", key, v)
	}
	return b, nil
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

// getOptionalBoolPtr extracts an optional bool argument as a pointer, returning nil if not present.
// Use this when you need to distinguish "not provided" from false (e.g. for Reconfigure).
func getOptionalBoolPtr(args map[string]interface{}, key string) *bool {
	if v, ok := args[key].(bool); ok {
		return &v
	}
	return nil
}

// getOptionalStringPtr extracts an optional string argument as a pointer, returning nil if not present.
// Use this when you need to distinguish "not provided" from "" (e.g. for Reconfigure).
func getOptionalStringPtr(args map[string]interface{}, key string) *string {
	if v, ok := args[key].(string); ok {
		return &v
	}
	return nil
}
