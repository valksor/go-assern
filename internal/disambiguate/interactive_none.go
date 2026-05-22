//go:build none

package disambiguate

import (
	"fmt"
)

// IsInteractive returns false when interactive support is disabled.
func IsInteractive() bool {
	return false
}

// SelectCommand returns an error when interactive support is disabled.
func SelectCommand(matches []CommandMatch, prefix string) (*CommandMatch, error) {
	return nil, fmt.Errorf("ambiguous command %q matches: %s (interactive support disabled)",
		prefix, FormatMatchNames(matches))
}
