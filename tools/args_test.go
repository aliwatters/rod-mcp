package tools

import "testing"

func TestGetStringArg(t *testing.T) {
	args := map[string]interface{}{
		"name":  "hello",
		"count": 42.0,
	}

	tests := []struct {
		name    string
		key     string
		want    string
		wantErr bool
	}{
		{"present string", "name", "hello", false},
		{"missing key", "missing", "", true},
		{"wrong type", "count", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getStringArg(args, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("getStringArg(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getStringArg(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestGetFloatArg(t *testing.T) {
	args := map[string]interface{}{
		"x":    42.5,
		"name": "hello",
	}

	tests := []struct {
		name    string
		key     string
		want    float64
		wantErr bool
	}{
		{"present float", "x", 42.5, false},
		{"missing key", "missing", 0, true},
		{"wrong type", "name", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getFloatArg(args, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("getFloatArg(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getFloatArg(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestGetOptionalStringArg(t *testing.T) {
	args := map[string]interface{}{
		"name":  "hello",
		"count": 42.0,
	}

	if got := getOptionalStringArg(args, "name"); got != "hello" {
		t.Errorf("getOptionalStringArg(name) = %q, want %q", got, "hello")
	}
	if got := getOptionalStringArg(args, "missing"); got != "" {
		t.Errorf("getOptionalStringArg(missing) = %q, want empty", got)
	}
	if got := getOptionalStringArg(args, "count"); got != "" {
		t.Errorf("getOptionalStringArg(count) = %q, want empty (wrong type)", got)
	}
}

func TestGetOptionalFloatArg(t *testing.T) {
	args := map[string]interface{}{
		"x":    42.5,
		"name": "hello",
	}

	if got := getOptionalFloatArg(args, "x", 0); got != 42.5 {
		t.Errorf("getOptionalFloatArg(x) = %v, want 42.5", got)
	}
	if got := getOptionalFloatArg(args, "missing", 99.0); got != 99.0 {
		t.Errorf("getOptionalFloatArg(missing) = %v, want 99.0", got)
	}
	if got := getOptionalFloatArg(args, "name", 99.0); got != 99.0 {
		t.Errorf("getOptionalFloatArg(name) = %v, want 99.0 (wrong type)", got)
	}
}

func TestGetOptionalBoolArg(t *testing.T) {
	args := map[string]interface{}{
		"flag": true,
		"name": "hello",
	}

	if got := getOptionalBoolArg(args, "flag", false); got != true {
		t.Errorf("getOptionalBoolArg(flag) = %v, want true", got)
	}
	if got := getOptionalBoolArg(args, "missing", false); got != false {
		t.Errorf("getOptionalBoolArg(missing) = %v, want false", got)
	}
	if got := getOptionalBoolArg(args, "name", false); got != false {
		t.Errorf("getOptionalBoolArg(name) = %v, want false (wrong type)", got)
	}
}
