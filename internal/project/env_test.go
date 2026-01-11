package project_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/valksor/go-assern/internal/project"
)

func TestNewEnvLoader(t *testing.T) {
	t.Parallel()

	loader := project.NewEnvLoader()

	if loader == nil {
		t.Fatal("NewEnvLoader returned nil")
	}

	// Should have base environment from os.Environ()
	combined := loader.GetCombinedEnv()
	if combined == nil {
		t.Error("GetCombinedEnv returned nil")
	}

	// Should have at least PATH in most environments
	if _, ok := combined["PATH"]; !ok {
		// This might not exist in all test environments, so we'll just check
		// that we got some environment variables
		if len(combined) == 0 {
			t.Error("Expected at least some environment variables")
		}
	}
}

func TestExpand_BasicVar(t *testing.T) {
	t.Parallel()

	loader := project.NewEnvLoader()
	loader.SetGlobalEnv("TEST_VAR", "test_value")

	tests := []struct {
		name  string
		input string
		want  string
		setup func(*project.EnvLoader)
	}{
		{
			name:  "simple $VAR expansion",
			input: "$TEST_VAR",
			want:  "test_value",
			setup: func(l *project.EnvLoader) {
				l.SetGlobalEnv("TEST_VAR", "test_value")
			},
		},
		{
			name:  "multiple $VAR in string",
			input: "prefix_$TEST_VAR suffix",
			want:  "prefix_test_value suffix",
			setup: func(l *project.EnvLoader) {
				l.SetGlobalEnv("TEST_VAR", "test_value")
			},
		},
		{
			name:  "adjacent $VAR",
			input: "$FOO$BAR",
			want:  "foobar",
			setup: func(l *project.EnvLoader) {
				l.SetGlobalEnv("FOO", "foo")
				l.SetGlobalEnv("BAR", "bar")
			},
		},
		{
			name:  "no variables",
			input: "plain string",
			want:  "plain string",
			setup: func(l *project.EnvLoader) {},
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
			setup: func(l *project.EnvLoader) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := project.NewEnvLoader()
			tt.setup(l)

			got := l.Expand(tt.input)
			if got != tt.want {
				t.Errorf("Expand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpand_BraceVar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
		setup func(*project.EnvLoader)
	}{
		{
			name:  "simple ${VAR} expansion",
			input: "${TEST_VAR}",
			want:  "test_value",
			setup: func(l *project.EnvLoader) {
				l.SetGlobalEnv("TEST_VAR", "test_value")
			},
		},
		{
			name:  "multiple ${VAR} in string",
			input: "prefix_${TEST_VAR}_suffix",
			want:  "prefix_test_value_suffix",
			setup: func(l *project.EnvLoader) {
				l.SetGlobalEnv("TEST_VAR", "test_value")
			},
		},
		{
			name:  "mixed ${VAR} and $VAR",
			input: "${FOO}_$BAR",
			want:  "foo_bar",
			setup: func(l *project.EnvLoader) {
				l.SetGlobalEnv("FOO", "foo")
				l.SetGlobalEnv("BAR", "bar")
			},
		},
		{
			name:  "adjacent ${VAR}",
			input: "${FOO}${BAR}",
			want:  "foobar",
			setup: func(l *project.EnvLoader) {
				l.SetGlobalEnv("FOO", "foo")
				l.SetGlobalEnv("BAR", "bar")
			},
		},
		{
			name:  "unclosed brace returns original",
			input: "${UNCLOSED",
			want:  "${UNCLOSED",
			setup: func(l *project.EnvLoader) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := project.NewEnvLoader()
			tt.setup(l)

			got := l.Expand(tt.input)
			if got != tt.want {
				t.Errorf("Expand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpand_UnknownVar(t *testing.T) {
	t.Parallel()

	l := project.NewEnvLoader()

	tests := []string{
		"$UNKNOWN_VAR",
		"${UNKNOWN_VAR}",
		"prefix_$UNKNOWN_VAR",
		"${UNKNOWN_VAR}_suffix",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			got := l.Expand(input)
			// Unknown variables should be preserved as-is
			if got != input {
				t.Errorf("Expand(%q) = %q, want %q (unknown vars preserved)", input, got, input)
			}
		})
	}
}

func TestExpand_ResolutionOrder(t *testing.T) {
	// Resolution order: project → global → base (system)
	t.Run("project overrides global", func(t *testing.T) {
		t.Parallel()

		l := project.NewEnvLoader()
		l.SetGlobalEnv("TEST_VAR", "global_value")
		l.SetProjectEnv("TEST_VAR", "project_value")

		got := l.Expand("$TEST_VAR")
		if got != "project_value" {
			t.Errorf("Expand($TEST_VAR) = %q, want project_value", got)
		}
	})

	t.Run("global overrides base", func(t *testing.T) {
		// Set a base environment variable
		t.Setenv("TEST_OVERRIDE_VAR", "base_value")

		l := project.NewEnvLoader()
		l.SetGlobalEnv("TEST_OVERRIDE_VAR", "global_value")

		got := l.Expand("$TEST_OVERRIDE_VAR")
		if got != "global_value" {
			t.Errorf("Expand($TEST_OVERRIDE_VAR) = %q, want global_value", got)
		}
	})

	t.Run("base used when no project or global", func(t *testing.T) {
		// Set a base environment variable
		t.Setenv("TEST_BASE_VAR", "base_value")

		l := project.NewEnvLoader()

		got := l.Expand("$TEST_BASE_VAR")
		if got != "base_value" {
			t.Errorf("Expand($TEST_BASE_VAR) = %q, want base_value", got)
		}
	})
}

func TestExpandMap(t *testing.T) {
	t.Parallel()

	t.Run("nil map returns nil", func(t *testing.T) {
		t.Parallel()

		l := project.NewEnvLoader()
		l.SetGlobalEnv("VAR", "value")

		got := l.ExpandMap(nil)
		if got != nil {
			t.Errorf("ExpandMap(nil) = %v, want nil", got)
		}
	})

	t.Run("expands all values", func(t *testing.T) {
		t.Parallel()

		l := project.NewEnvLoader()
		l.SetGlobalEnv("FOO", "foo_value")
		l.SetGlobalEnv("BAR", "bar_value")

		input := map[string]string{
			"key1": "$FOO",
			"key2": "prefix_$BAR",
			"key3": "static",
		}

		want := map[string]string{
			"key1": "foo_value",
			"key2": "prefix_bar_value",
			"key3": "static",
		}

		got := l.ExpandMap(input)
		if !maps.Equal(got, want) {
			t.Errorf("ExpandMap() = %v, want %v", got, want)
		}
	})

	t.Run("does not modify original map", func(t *testing.T) {
		t.Parallel()

		l := project.NewEnvLoader()
		l.SetGlobalEnv("VAR", "value")

		input := map[string]string{
			"key": "$VAR",
		}

		originalValue := input["key"]
		_ = l.ExpandMap(input)

		if input["key"] != originalValue {
			t.Error("ExpandMap modified the original map")
		}
	})
}

func TestBuildServerEnv(t *testing.T) {
	t.Parallel()

	t.Run("with server env and project name", func(t *testing.T) {
		t.Parallel()

		l := project.NewEnvLoader()
		l.SetGlobalEnv("API_KEY", "global_key")

		serverEnv := map[string]string{
			"API_KEY": "$API_KEY",
			"PORT":    "8080",
		}

		got := l.BuildServerEnv(serverEnv, "myproject")

		// Check that API_KEY was expanded
		hasAPIKey := false
		hasPort := false
		hasProject := false

		for _, env := range got {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				switch parts[0] {
				case "API_KEY":
					if parts[1] == "global_key" {
						hasAPIKey = true
					}
				case "PORT":
					if parts[1] == "8080" {
						hasPort = true
					}
				case "ASSERN_PROJECT":
					if parts[1] == "myproject" {
						hasProject = true
					}
				}
			}
		}

		if !hasAPIKey {
			t.Error("BuildServerEnv did not include expanded API_KEY")
		}
		if !hasPort {
			t.Error("BuildServerEnv did not include PORT")
		}
		if !hasProject {
			t.Error("BuildServerEnv did not include ASSERN_PROJECT")
		}
	})

	t.Run("with nil server env", func(t *testing.T) {
		t.Parallel()

		l := project.NewEnvLoader()

		got := l.BuildServerEnv(nil, "myproject")

		// Should still have ASSERN_PROJECT
		hasProject := false
		for _, env := range got {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 && parts[0] == "ASSERN_PROJECT" && parts[1] == "myproject" {
				hasProject = true
			}
		}

		if !hasProject {
			t.Error("BuildServerEnv with nil env did not include ASSERN_PROJECT")
		}
	})

	t.Run("with empty project name", func(t *testing.T) {
		t.Parallel()

		l := project.NewEnvLoader()

		serverEnv := map[string]string{
			"KEY": "value",
		}

		got := l.BuildServerEnv(serverEnv, "")

		// Should not have ASSERN_PROJECT
		for _, env := range got {
			if strings.HasPrefix(env, "ASSERN_PROJECT=") {
				t.Error("BuildServerEnv included ASSERN_PROJECT when project name is empty")
			}
		}
	})
}

func TestGetCombinedEnv(t *testing.T) {
	t.Run("all layers combined", func(t *testing.T) {
		// Set a base environment variable
		t.Setenv("TEST_COMBINED", "base_value")

		l := project.NewEnvLoader()
		l.SetGlobalEnv("GLOBAL_VAR", "global_value")
		l.SetProjectEnv("PROJ_VAR", "proj_value")

		got := l.GetCombinedEnv()

		if got["TEST_COMBINED"] != "base_value" {
			t.Error("GetCombinedEnv missing base layer")
		}
		if got["GLOBAL_VAR"] != "global_value" {
			t.Error("GetCombinedEnv missing global layer")
		}
		if got["PROJ_VAR"] != "proj_value" {
			t.Error("GetCombinedEnv missing project layer")
		}
	})

	t.Run("project overrides global and base", func(t *testing.T) {
		t.Setenv("OVERRIDE_VAR", "base_value")

		l := project.NewEnvLoader()
		l.SetGlobalEnv("OVERRIDE_VAR", "global_value")
		l.SetProjectEnv("OVERRIDE_VAR", "proj_value")

		got := l.GetCombinedEnv()

		if got["OVERRIDE_VAR"] != "proj_value" {
			t.Errorf("GetCombinedEnv() = %v, want proj_value to override", got)
		}
	})
}

func TestSetProjectEnv(t *testing.T) {
	t.Parallel()

	l := project.NewEnvLoader()
	l.SetProjectEnv("TEST_VAR", "test_value")

	combined := l.GetCombinedEnv()
	if combined["TEST_VAR"] != "test_value" {
		t.Error("SetProjectEnv did not set the value")
	}
}

func TestSetGlobalEnv(t *testing.T) {
	t.Parallel()

	l := project.NewEnvLoader()
	l.SetGlobalEnv("TEST_VAR", "test_value")

	combined := l.GetCombinedEnv()
	if combined["TEST_VAR"] != "test_value" {
		t.Error("SetGlobalEnv did not set the value")
	}
}

func TestMergeEnvSlices(t *testing.T) {
	t.Parallel()

	base := []string{
		"FOO=bar",
		"BAZ=qux",
	}

	override := []string{
		"FOO=overridden",
		"NEW=value",
	}

	got := project.MergeEnvSlices(base, override)

	// Check FOO was overridden
	hasFoo := false
	fooVal := ""
	for _, env := range got {
		parts := strings.SplitN(env, "=", 2)
		if parts[0] == "FOO" {
			hasFoo = true
			fooVal = parts[1]
		}
	}

	if !hasFoo {
		t.Error("MergeEnvSlices did not include FOO")
	}
	if fooVal != "overridden" {
		t.Errorf("FOO = %s, want overridden", fooVal)
	}

	// Check BAZ was preserved
	hasBaz := false
	for _, env := range got {
		parts := strings.SplitN(env, "=", 2)
		if parts[0] == "BAZ" && parts[1] == "qux" {
			hasBaz = true
		}
	}
	if !hasBaz {
		t.Error("MergeEnvSlices did not preserve BAZ")
	}

	// Check NEW was added
	hasNew := false
	for _, env := range got {
		parts := strings.SplitN(env, "=", 2)
		if parts[0] == "NEW" && parts[1] == "value" {
			hasNew = true
		}
	}
	if !hasNew {
		t.Error("MergeEnvSlices did not add NEW")
	}
}

func TestMergeEnvSlices_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty base", func(t *testing.T) {
		t.Parallel()

		got := project.MergeEnvSlices([]string{}, []string{"KEY=value"})
		if !slices.Contains(got, "KEY=value") {
			t.Error("MergeEnvSlices with empty base failed")
		}
	})

	t.Run("empty override", func(t *testing.T) {
		t.Parallel()

		base := []string{"KEY=value"}
		got := project.MergeEnvSlices(base, []string{})
		if !slices.Contains(got, "KEY=value") {
			t.Error("MergeEnvSlices with empty override failed")
		}
	})

	t.Run("both empty", func(t *testing.T) {
		t.Parallel()

		got := project.MergeEnvSlices([]string{}, []string{})
		if len(got) != 0 {
			t.Errorf("MergeEnvSlices with both empty = %v, want empty", got)
		}
	})

	t.Run("malformed env entry", func(t *testing.T) {
		t.Parallel()

		base := []string{"VALID=value"}
		override := []string{"INVALID"}

		got := project.MergeEnvSlices(base, override)
		// Malformed entry should be ignored
		if !slices.Contains(got, "VALID=value") {
			t.Error("MergeEnvSlices lost valid entry")
		}
	})
}
