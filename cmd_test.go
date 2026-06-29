package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/aliwatters/rod-mcp/types"
)

type registryEntry struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

func TestApplyOverridesCanForceHeadful(t *testing.T) {
	cfg := types.DefaultConfig
	applyOverrides(&cfg, &SubCfg{
		Headless:    false,
		HeadlessSet: true,
	})

	if cfg.Headless {
		t.Fatal("Headless = true, want false")
	}
}

func TestParseCommandArgsGUIForcesHeadful(t *testing.T) {
	subCfg, err := parseCommandArgs([]string{"rod-mcp", "--gui", "--compact-snapshot"})
	if err != nil {
		t.Fatalf("parseCommandArgs: %v", err)
	}
	if !subCfg.HeadlessSet {
		t.Fatal("HeadlessSet = false, want true")
	}
	if subCfg.Headless {
		t.Fatal("Headless = true, want false")
	}

	cfg := types.DefaultConfig
	applyOverrides(&cfg, subCfg)
	if cfg.Headless {
		t.Fatal("registry-style --gui launch stayed headless")
	}
}

func TestParseCommandArgsRejectsConflictingHeadlessAndGUI(t *testing.T) {
	if _, err := parseCommandArgs([]string{"rod-mcp", "--headless", "--gui"}); err == nil {
		t.Fatal("parseCommandArgs accepted conflicting --headless and --gui")
	}
}

func TestGUIServerRegistryLaunchesHeadful(t *testing.T) {
	data, err := os.ReadFile("mcp-registry.json")
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	var registry map[string]registryEntry
	if err := json.Unmarshal(data, &registry); err != nil {
		t.Fatalf("parse registry: %v", err)
	}
	entry, ok := registry["rod-mcp-gui"]
	if !ok {
		t.Fatal("registry missing rod-mcp-gui entry")
	}

	if !containsArg(entry.Args, "--gui") {
		t.Fatalf("rod-mcp-gui args = %v, want --gui", entry.Args)
	}
	if containsArg(entry.Args, "--headless") {
		t.Fatalf("rod-mcp-gui args = %v, must not include --headless", entry.Args)
	}

	subCfg, err := parseCommandArgs(append([]string{"rod-mcp"}, entry.Args...))
	if err != nil {
		t.Fatalf("parse registry args: %v", err)
	}
	cfg := types.DefaultConfig
	applyOverrides(&cfg, subCfg)
	if cfg.Headless {
		t.Fatalf("rod-mcp-gui registry args yielded Headless=true: %v", entry.Args)
	}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
