package remote

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/cmd/completion"
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
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		switch len(args) {
		case 0:
			return completion.CompleteRemoteNames(cmd, args, toComplete)
		case 1:
			return []string{"bearer"}, cobra.ShellCompDirectiveNoFileComp
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	},
}

func init() {
	remoteCmd.AddCommand(remoteSetAuthCmd)
}
