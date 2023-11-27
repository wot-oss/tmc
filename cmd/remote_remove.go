package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

// remoteRemoveCmd represents the 'remote remove' command
var remoteRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove the named remote repository from config",
	Long:  `Remove the named remote repository from config`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := cli.RemoteRemove(args[0])
		if err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	remoteCmd.AddCommand(remoteRemoveCmd)
}
