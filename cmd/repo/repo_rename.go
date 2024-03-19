package repo

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

// repoRenameCmd represents the 'repo show' command
var repoRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Renames repository <old-name> to <new-name>",
	Long:  `Renames repository <old-name> to <new-name>`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		err := cli.RepoRename(args[0], args[1])
		if err != nil {
			os.Exit(1)
		}
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completion.CompleteRepoNames(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	repoCmd.AddCommand(repoRenameCmd)
}
