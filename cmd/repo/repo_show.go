package repo

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

// repoShowCmd represents the 'repo show' command
var repoShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Shows settings for the named repository",
	Long:  `Shows settings for the named repository`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := cli.RepoShow(args[0])
		if err != nil {
			os.Exit(1)
		}
	},
	ValidArgsFunction: completion.CompleteRepoNames,
}

func init() {
	repoCmd.AddCommand(repoShowCmd)
}
