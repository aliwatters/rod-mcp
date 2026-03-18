package types

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultOutputSubdir   = "rod-mcp"
	outputTimestampFormat = "20060102-150405"
)

// ResolveOutputDir returns the output directory, creating it if necessary.
// If cfg.OutputDir is set, uses that. Otherwise uses os.TempDir()/rod-mcp.
func ResolveOutputDir(cfg Config) (string, error) {
	dir := cfg.OutputDir
	if dir == "" {
		dir = filepath.Join(os.TempDir(), defaultOutputSubdir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create output directory %s: %w", dir, err)
	}
	return dir, nil
}

// SaveOutput writes data to a timestamped file in the output directory.
// Returns the absolute path to the saved file.
func SaveOutput(cfg Config, data []byte, prefix, ext string) (string, error) {
	dir, err := ResolveOutputDir(cfg)
	if err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s-%s.%s", prefix, time.Now().Format(outputTimestampFormat), ext)
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", path, err)
	}
	return path, nil
}
