package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

var versionsCmd = &cobra.Command{
	Use:   "versions <name> [--remote <remoteName>]",
	Short: "List available versions of the TM with given name",
	Long:  `List available versions of the TM with given name. --remote is optional if there's only one remote configured'`,
	Args:  cobra.ExactArgs(1),
	Run:   listVersions,
}

func init() {
	RootCmd.AddCommand(versionsCmd)
	versionsCmd.Flags().StringP("remote", "r", "", "name of the remote to search for versions")
}

func listVersions(cmd *cobra.Command, args []string) {
	remoteName := cmd.Flag("remote").Value.String()
	name := args[0]
	err := cli.ListVersions(remoteName, name)
	if err != nil {
		os.Exit(1)
	}
}
