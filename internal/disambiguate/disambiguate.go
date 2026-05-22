// Package disambiguate provides Symfony-style command prefix matching
// and colon notation for Cobra CLI applications.
//
// This package enables shorthand command syntax like "c:v" → "config validate",
// with case-insensitive prefix matching and optional interactive disambiguation.
//
// Basic usage:
//
//	rootCmd := &cobra.Command{Use: "mytool"}
//	// ... add commands ...
//
//	// Pre-process args for colon notation
//	args := os.Args[1:]
//	if len(args) > 0 && strings.Contains(args[0], ":") {
//		resolved, matches, err := disambiguate.ResolveColonPath(rootCmd, args[0])
//		if err == nil {
//			rootCmd.SetArgs(append(resolved, args[1:]...))
//		}
//	}
package disambiguate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// CommandMatch represents a command that matches a prefix.
type CommandMatch struct {
	Command     *cobra.Command // The matched Cobra command
	MatchedName string         // The name that matched (command name)
	Path        []string       // Full path for subcommands (e.g., ["config", "validate"])
}

// FindPrefixMatches finds all available commands matching the given prefix.
// Matching is case-insensitive.
//
// An empty prefix matches all available commands.
func FindPrefixMatches(parent *cobra.Command, prefix string) []CommandMatch {
	var matches []CommandMatch
	prefix = strings.ToLower(prefix)

	for _, cmd := range parent.Commands() {
		if !cmd.IsAvailableCommand() || cmd.IsAdditionalHelpTopicCommand() {
			continue
		}

		name := strings.ToLower(cmd.Name())
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, CommandMatch{
				Command:     cmd,
				MatchedName: cmd.Name(),
				Path:        []string{cmd.Name()},
			})
		}
	}

	return matches
}

// ResolveColonPath resolves a colon-separated command path like "c:v" to ["config", "validate"].
//
// Returns:
//   - resolved: the resolved path segments
//   - matches: ambiguous matches at the last level (empty if unambiguous)
//   - error: if resolution fails
//
// If the path is unambiguous, the matches slice will be empty.
// If ambiguous, the caller can use SelectCommand for interactive selection.
func ResolveColonPath(root *cobra.Command, path string) ([]string, []CommandMatch, error) {
	if !strings.Contains(path, ":") {
		return nil, nil, errors.New("not a colon path")
	}

	segments := strings.Split(path, ":")
	resolved := make([]string, 0, len(segments))
	current := root

	for i, segment := range segments {
		if segment == "" {
			// Trailing colon - list subcommands of current
			if i == len(segments)-1 {
				matches := FindPrefixMatches(current, "")

				return resolved, matches, nil
			}

			continue
		}

		matches := FindPrefixMatches(current, segment)

		switch len(matches) {
		case 0:
			return resolved, nil, fmt.Errorf("no command matching %q in %s", segment, commandPath(resolved))
		case 1:
			resolved = append(resolved, matches[0].Command.Name())
			current = matches[0].Command
		default:
			// Ambiguous - return matches for user selection
			// Update paths to include full resolution so far
			for j := range matches {
				matches[j].Path = append(append([]string{}, resolved...), matches[j].Command.Name())
			}

			return resolved, matches, nil
		}
	}

	return resolved, nil, nil
}

// FormatAmbiguousError returns a formatted error message for ambiguous commands.
func FormatAmbiguousError(prefix string, matches []CommandMatch) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Command %q is ambiguous. Did you mean one of these?\n", prefix)
	for _, m := range matches {
		fmt.Fprintf(&sb, "  %s - %s\n", m.Command.Name(), m.Command.Short)
	}

	return sb.String()
}

// FindPrefixMatchesInPath searches for commands matching the prefix at any level
// of the command tree. It returns all unique command paths that match.
func FindPrefixMatchesInPath(root *cobra.Command, prefix string) []CommandMatch {
	return findPrefixMatchesRecursive(root, prefix, []string{})
}

func findPrefixMatchesRecursive(cmd *cobra.Command, prefix string, path []string) []CommandMatch {
	var matches []CommandMatch

	// Check current command's children
	for _, child := range cmd.Commands() {
		if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
			continue
		}

		childPath := append(path, child.Name())
		name := strings.ToLower(child.Name())

		if strings.HasPrefix(name, prefix) {
			matches = append(matches, CommandMatch{
				Command:     child,
				MatchedName: child.Name(),
				Path:        append([]string{}, childPath...),
			})
		}

		// Recursively search subcommands
		matches = append(matches, findPrefixMatchesRecursive(child, prefix, childPath)...)
	}

	return matches
}

func commandPath(segments []string) string {
	if len(segments) == 0 {
		return "root"
	}

	return strings.Join(segments, " ")
}

// FormatMatchNames returns a comma-separated list of command names.
func FormatMatchNames(matches []CommandMatch) string {
	names := make([]string, len(matches))
	for i, m := range matches {
		names[i] = m.Command.Name()
	}

	return strings.Join(names, ", ")
}
