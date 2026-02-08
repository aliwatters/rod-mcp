package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveOutputDir_Default(t *testing.T) {
	cfg := Config{}
	dir, err := ResolveOutputDir(cfg)
	if err != nil {
		t.Fatalf("ResolveOutputDir() error = %v", err)
	}
	expected := filepath.Join(os.TempDir(), defaultOutputSubdir)
	if dir != expected {
		t.Errorf("ResolveOutputDir() = %q, want %q", dir, expected)
	}
	// Verify directory was created
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("output dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("output dir is not a directory")
	}
}

func TestResolveOutputDir_Custom(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "my-output")
	cfg := Config{OutputDir: custom}

	got, err := ResolveOutputDir(cfg)
	if err != nil {
		t.Fatalf("ResolveOutputDir() error = %v", err)
	}
	if got != custom {
		t.Errorf("ResolveOutputDir() = %q, want %q", got, custom)
	}
	info, err := os.Stat(custom)
	if err != nil {
		t.Fatalf("custom output dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("custom output dir is not a directory")
	}
}

func TestSaveOutput(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{OutputDir: dir}
	data := []byte("test content")

	path, err := SaveOutput(cfg, data, "screenshot", "png")
	if err != nil {
		t.Fatalf("SaveOutput() error = %v", err)
	}

	// Verify file path structure
	if !strings.HasPrefix(path, dir) {
		t.Errorf("path %q does not start with %q", path, dir)
	}
	if !strings.HasPrefix(filepath.Base(path), "screenshot-") {
		t.Errorf("filename %q does not start with 'screenshot-'", filepath.Base(path))
	}
	if !strings.HasSuffix(path, ".png") {
		t.Errorf("path %q does not end with '.png'", path)
	}

	// Verify file contents
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("file content = %q, want %q", got, data)
	}
}

func TestSaveOutput_PDF(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{OutputDir: dir}
	data := []byte("%PDF-1.4 fake pdf content")

	path, err := SaveOutput(cfg, data, "page", "pdf")
	if err != nil {
		t.Fatalf("SaveOutput() error = %v", err)
	}

	if !strings.HasPrefix(filepath.Base(path), "page-") {
		t.Errorf("filename %q does not start with 'page-'", filepath.Base(path))
	}
	if !strings.HasSuffix(path, ".pdf") {
		t.Errorf("path %q does not end with '.pdf'", path)
	}
}

func TestImageResponsesMode_Defaults(t *testing.T) {
	// Default config should have ImageResponses = "allow"
	if DefaultConfig.ImageResponses != ImageResponsesAllow {
		t.Errorf("DefaultConfig.ImageResponses = %q, want %q", DefaultConfig.ImageResponses, ImageResponsesAllow)
	}

	// Zero-value Config should have empty ImageResponses (treated as allow by handlers)
	cfg := Config{}
	if cfg.ImageResponses == ImageResponsesOmit {
		t.Error("zero-value Config.ImageResponses should not be 'omit'")
	}
}
