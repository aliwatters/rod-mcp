package utils

import "strings"

func IsHttp(raw string) bool {
	return strings.HasPrefix(strings.TrimSpace(raw), "http://") || strings.HasPrefix(strings.TrimSpace(raw), "https://")
}

// HeaderMapToSlice converts a map of HTTP headers to a flat key-value slice
// suitable for Rod's SetExtraHeaders.
func HeaderMapToSlice(m map[string]string) []string {
	headers := make([]string, 0, len(m)*2)
	for k, v := range m {
		headers = append(headers, k, v)
	}
	return headers
}
