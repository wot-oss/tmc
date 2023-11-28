package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

var versionsCmd = &cobra.Command{
	Use:   "versions <name>",
	Short: "List available versions of the TM with given name",
	Long:  `List available versions of the TM with given name`,
	Args:  cobra.ExactArgs(1),
	Run:   listVersions,
}

func init() {
	RootCmd.AddCommand(versionsCmd)
	versionsCmd.Flags().StringP("remote", "r", "", "use named remote instead of default")
}

func listVersions(cmd *cobra.Command, args []string) {
	remoteName := cmd.Flag("remote").Value.String()
	name := args[0]
	err := cli.ListVersions(remoteName, name)
	if err != nil {
		os.Exit(1)
	}
}
