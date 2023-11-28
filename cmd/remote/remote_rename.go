package remote

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

// remoteRenameCmd represents the 'remote show' command
var remoteRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Renames remote <old-name> to <new-name>",
	Long:  `Renames remote <old-name> to <new-name>`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		err := cli.RemoteRename(args[0], args[1])
		if err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	remoteCmd.AddCommand(remoteRenameCmd)
}
