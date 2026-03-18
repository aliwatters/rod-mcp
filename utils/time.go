package utils

import "time"

const (
	DefaultTimeFormat = "2006/01/02 15:04:05"
	DefaultDateFormat = "2006-01-02"
)

func FormatBuildTime(timeStr string) string {
	if timeStr == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return timeStr
	}
	return t.Local().Format("2006-01-02 15:04:05")
}
