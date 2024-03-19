package repo

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

// repoRemoveCmd represents the 'repo remove' command
var repoRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove the named repository from config",
	Long:  `Remove the named repository from config`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := cli.RepoRemove(args[0])
		if err != nil {
			os.Exit(1)
		}
	},
	ValidArgsFunction: completion.CompleteRepoNames,
}

func init() {
	repoCmd.AddCommand(repoRemoveCmd)
}
