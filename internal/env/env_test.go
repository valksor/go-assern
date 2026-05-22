package env

import (
	"os"
	"testing"
)

func TestExpandEnv(t *testing.T) {
	// Set test environment variables
	t.Setenv("TEST_VAR", "test_value")
	t.Setenv("NESTED_VAR", "nested_value")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "braced syntax",
			input:    "${TEST_VAR}",
			expected: "test_value",
		},
		{
			name:     "unbraced syntax",
			input:    "$TEST_VAR",
			expected: "test_value",
		},
		{
			name:     "partial expansion",
			input:    "prefix-${TEST_VAR}-suffix",
			expected: "prefix-test_value-suffix",
		},
		{
			name:     "nested variables",
			input:    "${TEST_VAR}/${NESTED_VAR}",
			expected: "test_value/nested_value",
		},
		{
			name:     "unset variable",
			input:    "${UNSET_VAR}",
			expected: "",
		},
		{
			name:     "no variables",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "mixed syntax",
			input:    "$TEST_VAR/${NESTED_VAR}",
			expected: "test_value/nested_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnv(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandEnv(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandEnvInMap(t *testing.T) {
	t.Setenv("VAR1", "value1")
	t.Setenv("VAR2", "value2")

	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name:     "nil map",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "simple expansion",
			input: map[string]string{
				"path": "${VAR1}/docs",
			},
			expected: map[string]string{
				"path": "value1/docs",
			},
		},
		{
			name: "multiple keys",
			input: map[string]string{
				"path1": "$VAR1",
				"path2": "${VAR2}",
				"path3": "no vars",
			},
			expected: map[string]string{
				"path1": "value1",
				"path2": "value2",
				"path3": "no vars",
			},
		},
		{
			name: "nested values",
			input: map[string]string{
				"home": "${VAR1}",
				"docs": "${VAR1}/${VAR2}",
			},
			expected: map[string]string{
				"home": "value1",
				"docs": "value1/value2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnvInMap(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("ExpandEnvInMap() = %v, want nil", result)
				}

				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("ExpandEnvInMap() length = %d, want %d", len(result), len(tt.expected))
			}

			for k, expectedVal := range tt.expected {
				if result[k] != expectedVal {
					t.Errorf("ExpandEnvInMap()[%q] = %q, want %q", k, result[k], expectedVal)
				}
			}
		})
	}
}

func TestExpandEnvInMapDoesNotModifyInput(t *testing.T) {
	t.Setenv("TEST_VAR", "expanded")

	input := map[string]string{
		"key": "${TEST_VAR}",
	}

	originalValue := input["key"]
	_ = ExpandEnvInMap(input)

	if input["key"] != originalValue {
		t.Error("ExpandEnvInMap() should not modify input map")
	}
}

func TestGetenv(t *testing.T) {
	// Set a test variable
	t.Setenv("GETENV_TEST", "test_value")

	tests := []struct {
		name         string
		key          string
		defaultValue string
		expected     string
	}{
		{
			name:         "existing variable",
			key:          "GETENV_TEST",
			defaultValue: "default",
			expected:     "test_value",
		},
		{
			name:         "non-existing variable",
			key:          "NONEXISTENT_VAR",
			defaultValue: "default_value",
			expected:     "default_value",
		},
		{
			name:         "existing variable with empty default",
			key:          "GETENV_TEST",
			defaultValue: "",
			expected:     "test_value",
		},
		{
			name:         "non-existing variable with empty default",
			key:          "NONEXISTENT_VAR",
			defaultValue: "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Getenv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Getenv(%q, %q) = %q, want %q", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestExpandEnvWithSpecialCharacters(t *testing.T) {
	t.Setenv("SPECIAL", "a:b/c-d")

	result := ExpandEnv("${SPECIAL}/path")
	if result != "a:b/c-d/path" {
		t.Errorf("ExpandEnv() with special chars = %q, want 'a:b/c-d/path'", result)
	}
}

func TestExpandEnvWithHome(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}

	result := ExpandEnv("${HOME}/.config")
	if result != home+"/.config" {
		t.Errorf("ExpandEnv($HOME) = %q, want %q", result, home+"/.config")
	}
}
