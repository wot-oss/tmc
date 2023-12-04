package remote

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

// remoteToggleEnabledCmd represents the 'remote toggle-enabled' command
var remoteToggleEnabledCmd = &cobra.Command{
	Use:   "toggle-enabled <name>",
	Short: "Toggle enabled status of the named remote repository",
	Long:  `Toggle enabled status of the named remote repository`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := cli.RemoteToggleEnabled(args[0])
		if err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	remoteCmd.AddCommand(remoteToggleEnabledCmd)
}
