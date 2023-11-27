package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

// remoteSetDefaultCmd represents the 'remote set-default' command
var remoteSetDefaultCmd = &cobra.Command{
	Use:   "set-default <name>",
	Short: "Set named remote repository as default",
	Long:  `Set named remote repository as default`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := cli.RemoteSetDefault(args[0])
		if err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	remoteCmd.AddCommand(remoteSetDefaultCmd)
}
