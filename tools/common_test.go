package tools

import (
	"testing"

	"github.com/go-rod/rod/lib/input"
)

func TestParseKey(t *testing.T) {
	tests := []struct {
		name    string
		keyStr  string
		want    input.Key
		wantErr bool
	}{
		// Named keys
		{"Tab", "Tab", input.Tab, false},
		{"Enter", "Enter", input.Enter, false},
		{"Return alias", "Return", input.Enter, false},
		{"Escape", "Escape", input.Escape, false},
		{"Backspace", "Backspace", input.Backspace, false},
		{"Delete", "Delete", input.Delete, false},
		{"Space", "Space", input.Space, false},

		// Arrow keys
		{"ArrowUp", "ArrowUp", input.ArrowUp, false},
		{"ArrowDown", "ArrowDown", input.ArrowDown, false},
		{"ArrowLeft", "ArrowLeft", input.ArrowLeft, false},
		{"ArrowRight", "ArrowRight", input.ArrowRight, false},

		// Function keys
		{"F1", "F1", input.F1, false},
		{"F12", "F12", input.F12, false},

		// Modifiers
		{"Shift", "Shift", input.ShiftLeft, false},
		{"Control", "Control", input.ControlLeft, false},
		{"Alt", "Alt", input.AltLeft, false},
		{"Meta", "Meta", input.MetaLeft, false},

		// Single characters
		{"lowercase a", "a", input.Key('a'), false},
		{"uppercase A", "A", input.Key('A'), false},
		{"digit 1", "1", input.Key('1'), false},
		{"special char @", "@", input.Key('@'), false},

		// Errors
		{"unknown key", "UnknownKey", 0, true},
		{"multi-char non-key", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseKey(tt.keyStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseKey(%q) error = %v, wantErr %v", tt.keyStr, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseKey(%q) = %v, want %v", tt.keyStr, got, tt.want)
			}
		})
	}
}
