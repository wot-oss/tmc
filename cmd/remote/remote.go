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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// remoteCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// remoteCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
