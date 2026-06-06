package types

import (
	"context"
	"strings"
	"testing"
)

func TestConfigureLauncherHeadlessFlag(t *testing.T) {
	l, err := configureLauncher(context.Background(), Config{
		Headless:       true,
		BrowserBinPath: "/bin/echo",
	}, t.TempDir())
	if err != nil {
		t.Fatalf("configureLauncher: %v", err)
	}
	if !launcherArgsContain(l.FormatArgs(), "--headless") {
		t.Fatalf("headless launcher args = %v, want --headless", l.FormatArgs())
	}
}

func TestConfigureLauncherWindowedOmitsHeadlessFlag(t *testing.T) {
	l, err := configureLauncher(context.Background(), Config{
		Headless:       false,
		BrowserBinPath: "/bin/echo",
	}, t.TempDir())
	if err != nil {
		t.Fatalf("configureLauncher: %v", err)
	}
	if launcherArgsContain(l.FormatArgs(), "--headless") {
		t.Fatalf("windowed launcher args = %v, want no --headless", l.FormatArgs())
	}
}

func launcherArgsContain(args []string, want string) bool {
	for _, arg := range args {
		if arg == want || strings.HasPrefix(arg, want+"=") {
			return true
		}
	}
	return false
}
