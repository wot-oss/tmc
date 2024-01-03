package remote

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

// remoteSetConfigCmd represents the 'remote add' command
var remoteSetAuthCmd = &cobra.Command{
	Use:     "set-auth <remote-name> <auth-type> <auth-data>",
	Short:   "Set authentication config for a remote repository",
	Long:    `Overwrite auth config of a remote repository. <auth-type> must be one of: bearer`,
	Example: "set-auth http-remote bearer qfdhjf83cblkju",
	Args:    cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		err := cli.RemoteSetAuth(args[0], args[1], args[2])
		if err != nil {
			_ = cmd.Usage()
			os.Exit(1)
		}
	},
}

func init() {
	remoteCmd.AddCommand(remoteSetAuthCmd)
}
