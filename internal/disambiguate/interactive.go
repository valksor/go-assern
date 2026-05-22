//go:build !none

package disambiguate

import (
	"errors"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"golang.org/x/term"
)

// IsInteractive returns true if stdin is a terminal (TTY).
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// SelectCommand prompts the user to select from matching commands.
// Returns an error if not in interactive mode or user cancels.
func SelectCommand(matches []CommandMatch, prefix string) (*CommandMatch, error) {
	if !IsInteractive() {
		return nil, fmt.Errorf("ambiguous command %q matches: %s (non-interactive mode)",
			prefix, FormatMatchNames(matches))
	}

	// Build options with Cancel at the end
	options := make([]string, len(matches)+1)
	for i, m := range matches {
		options[i] = fmt.Sprintf("%s - %s", m.Command.Name(), m.Command.Short)
	}
	options[len(matches)] = "[Cancel]"

	var selected int
	prompt := &survey.Select{
		Message: fmt.Sprintf("Command %q is ambiguous. Select one:", prefix),
		Options: options,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		return nil, errors.New("cancelled")
	}

	// Check if Cancel was selected
	if selected == len(matches) {
		return nil, errors.New("cancelled")
	}

	return &matches[selected], nil
}
