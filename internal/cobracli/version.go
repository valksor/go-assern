package cobracli

import (
	"github.com/spf13/cobra"

	"github.com/valksor/go-assern/internal/version"
)

// NewVersionCommand creates a version command for the given app name.
func NewVersionCommand(appName string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(version.Info(appName))
		},
	}
}
