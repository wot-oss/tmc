package repo

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

// repoConfigHeadersCmd represents the 'repo config auth' command
var repoConfigHeadersCmd = &cobra.Command{
	Use:   "headers <repo-name> (<header>=<value> ...)",
	Short: "Set fixed HTTP headers for a repository",
	Long: `Set fixed HTTP headers for a repository.
Headers are set as a list of key-value pairs. Header keys can be repeated any number of times.`,
	Example: "tmc repo config headers http-repo Authorization=\"Bearer fd98fdsnmr4iudsn\"",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoName := args[0]
		var headers []string
		if len(args) > 1 {
			headers = args[1:]
		}
		err := cli.RepoSetHeaders(repoName, headers)
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
	repoConfigCmd.AddCommand(repoConfigHeadersCmd)
}
