package repo

import (
	"github.com/spf13/cobra"
)

// repoConfigCmd represents the 'repo config' command
var repoConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Update config of a named repository",
	Long:  `Update config of a named repository`,
}

func init() {
	repoCmd.AddCommand(repoConfigCmd)
}
