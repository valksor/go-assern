// Package cli provides interactive CLI components for assern.
package cli

import (
	"testing"
)

func TestValidateServerName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid name with letters",
			input:   "myserver",
			wantErr: false,
		},
		{
			name:    "valid name with hyphens",
			input:   "my-server",
			wantErr: false,
		},
		{
			name:    "valid name with underscores",
			input:   "my_server",
			wantErr: false,
		},
		{
			name:    "valid name with numbers",
			input:   "server123",
			wantErr: false,
		},
		{
			name:        "empty name",
			input:       "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "name too long",
			input:       string(make([]byte, 65)), // 65 characters
			wantErr:     true,
			errContains: "64 characters or less",
		},
		{
			name:        "reserved name: all",
			input:       "all",
			wantErr:     true,
			errContains: "reserved",
		},
		{
			name:        "reserved name: global",
			input:       "global",
			wantErr:     true,
			errContains: "reserved",
		},
		{
			name:        "reserved name: project",
			input:       "project",
			wantErr:     true,
			errContains: "reserved",
		},
		{
			name:        "starts with number",
			input:       "123server",
			wantErr:     true,
			errContains: "start with a letter",
		},
		{
			name:        "contains invalid characters",
			input:       "my.server",
			wantErr:     true,
			errContains: "letters, numbers, hyphens, and underscores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure long name is properly initialized
			if tt.name == "name too long" {
				tt.input = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 65 'a' chars
			}

			err := ValidateServerName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateServerName() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil {
					t.Errorf("ValidateServerName() expected error containing %q, got nil", tt.errContains)

					return
				}
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("ValidateServerName() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid HTTP URL",
			input:   "http://example.com/mcp",
			wantErr: false,
		},
		{
			name:    "valid HTTPS URL",
			input:   "https://example.com/mcp",
			wantErr: false,
		},
		{
			name:    "valid HTTPS URL with port",
			input:   "https://example.com:8080/mcp",
			wantErr: false,
		},
		{
			name:        "empty URL",
			input:       "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "no scheme",
			input:       "example.com/mcp",
			wantErr:     true,
			errContains: "include a scheme",
		},
		{
			name:        "invalid scheme",
			input:       "ftp://example.com/mcp",
			wantErr:     true,
			errContains: "http or https",
		},
		{
			name:        "no host",
			input:       "https://",
			wantErr:     true,
			errContains: "include a host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil {
					t.Errorf("ValidateURL() expected error containing %q, got nil", tt.errContains)

					return
				}
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("ValidateURL() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestValidateHTTPSURL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid HTTPS URL",
			input:   "https://example.com/mcp",
			wantErr: false,
		},
		{
			name:        "HTTP URL not allowed",
			input:       "http://example.com/mcp",
			wantErr:     true,
			errContains: "OAuth requires HTTPS",
		},
		{
			name:        "empty URL",
			input:       "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHTTPSURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHTTPSURL() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil {
					t.Errorf("ValidateHTTPSURL() expected error containing %q, got nil", tt.errContains)

					return
				}
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("ValidateHTTPSURL() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestValidateEnvVarKey(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid uppercase key",
			input:   "API_KEY",
			wantErr: false,
		},
		{
			name:    "valid mixed case key",
			input:   "ApiKey",
			wantErr: false,
		},
		{
			name:    "valid key with underscores",
			input:   "MY_API_KEY",
			wantErr: false,
		},
		{
			name:        "empty key",
			input:       "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "key with hyphens",
			input:       "MY-API-KEY",
			wantErr:     true,
			errContains: "letters, numbers, and underscores",
		},
		{
			name:        "key with dots",
			input:       "MY.API.KEY",
			wantErr:     true,
			errContains: "letters, numbers, and underscores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnvVarKey(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEnvVarKey() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil {
					t.Errorf("ValidateEnvVarKey() expected error containing %q, got nil", tt.errContains)

					return
				}
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("ValidateEnvVarKey() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestValidateTransport(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
	}{
		{
			name:    "stdio",
			input:   "stdio",
			wantErr: false,
		},
		{
			name:    "http",
			input:   "http",
			wantErr: false,
		},
		{
			name:    "sse",
			input:   "sse",
			wantErr: false,
		},
		{
			name:    "oauth-http",
			input:   "oauth-http",
			wantErr: false,
		},
		{
			name:    "oauth-sse",
			input:   "oauth-sse",
			wantErr: false,
		},
		{
			name:    "empty (auto-detect)",
			input:   "",
			wantErr: false,
		},
		{
			name:        "invalid transport",
			input:       "websocket",
			wantErr:     true,
			errContains: "invalid transport type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransport(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransport() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil {
					t.Errorf("ValidateTransport() expected error containing %q, got nil", tt.errContains)

					return
				}
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("ValidateTransport() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestIsReservedName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "all is reserved",
			input: "all",
			want:  true,
		},
		{
			name:  "global is reserved",
			input: "global",
			want:  true,
		},
		{
			name:  "project is reserved",
			input: "project",
			want:  true,
		},
		{
			name:  "ALL is reserved (case insensitive)",
			input: "ALL",
			want:  true,
		},
		{
			name:  "regular name not reserved",
			input: "myserver",
			want:  false,
		},
		{
			name:  "empty string not reserved",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsReservedName(tt.input); got != tt.want {
				t.Errorf("IsReservedName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		fieldName string
		wantErr   bool
	}{
		{
			name:      "non-empty value",
			value:     "test",
			fieldName: "Test Field",
			wantErr:   false,
		},
		{
			name:      "empty value",
			value:     "",
			fieldName: "Test Field",
			wantErr:   true,
		},
		{
			name:      "whitespace only value",
			value:     "   ",
			fieldName: "Test Field",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.value, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequired() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to check if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
