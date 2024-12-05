package repo

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

// repoConfigDescriptionCmd represents the 'repo con' command
var repoConfigDescriptionCmd = &cobra.Command{
	Use:     "description <repo-name> <description>",
	Short:   "Set description of a repository",
	Long:    `Set description of a repository`,
	Example: "tmc repo config description myrepo \"my default TM repository\"",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		repoName := args[0]
		description := args[1]
		err := cli.RepoSetDescription(context.Background(), repoName, description)
		if err != nil {
			_ = cmd.Usage()
			os.Exit(1)
		}
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		switch len(args) {
		case 0:
			return completion.CompleteRepoNames(cmd, args, toComplete)
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	},
}

func init() {
	repoConfigCmd.AddCommand(repoConfigDescriptionCmd)
}
