package remote

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/cmd/completion"
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
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completion.CompleteRemoteNames(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	remoteCmd.AddCommand(remoteRenameCmd)
}
