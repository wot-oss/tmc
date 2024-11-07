package repo

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

// repoSetAuthCmd represents the 'repo add' command
var repoSetAuthCmd = &cobra.Command{
	Use:     "set-auth <repo-name> <auth-type> <auth-data>",
	Short:   "Set authentication config for a repository",
	Long:    `Overwrite auth config of a repository. <auth-type> must be one of: bearer`,
	Example: "set-auth http-repo bearer qfdhjf83cblkju",
	Args:    cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		err := cli.RepoSetAuth(context.Background(), args[0], args[1], args[2])
		if err != nil {
			_ = cmd.Usage()
			os.Exit(1)
		}
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		switch len(args) {
		case 0:
			return completion.CompleteRepoNames(cmd, args, toComplete)
		case 1:
			return []string{"bearer"}, cobra.ShellCompDirectiveNoFileComp
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	},
}

func init() {
	repoCmd.AddCommand(repoSetAuthCmd)
}
