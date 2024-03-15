package repo

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/cmd/completion"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

// repoToggleEnabledCmd represents the 'repo toggle-enabled' command
var repoToggleEnabledCmd = &cobra.Command{
	Use:   "toggle-enabled <name>",
	Short: "Toggle enabled status of the named repository",
	Long:  `Toggle enabled status of the named repository`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := cli.RepoToggleEnabled(args[0])
		if err != nil {
			os.Exit(1)
		}
	},
	ValidArgsFunction: completion.CompleteRepoNames,
}

func init() {
	repoCmd.AddCommand(repoToggleEnabledCmd)
}
