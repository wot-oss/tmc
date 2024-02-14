package remote

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/cmd/completion"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

// remoteShowCmd represents the 'remote show' command
var remoteShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Shows settings for the remote <name>",
	Long:  `Shows settings for the remote <name>`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := cli.RemoteShow(args[0])
		if err != nil {
			os.Exit(1)
		}
	},
	ValidArgsFunction: completion.CompleteRemoteNames,
}

func init() {
	remoteCmd.AddCommand(remoteShowCmd)
}
