package remote

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/cmd"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

// remoteCmd represents the remote command
var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage remote repositories",
	Long: `The command remote and its subcommands allow to manage the list of remote repositories and their settings.
When no subcommand is given, defaults to list.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := cli.RemoteList()
		if err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	cmd.RootCmd.AddCommand(remoteCmd)
}
