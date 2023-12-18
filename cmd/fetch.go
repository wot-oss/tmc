package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch NAME[:SEMVER|DIGEST]",
	Short: "Fetches the TM by name",
	Long:  "Fetches TM by name, optionally accepting semantic version or digest",
	Args:  cobra.ExactArgs(1),
	Run:   executeFetch,
}

func init() {
	RootCmd.AddCommand(fetchCmd)
	fetchCmd.Flags().StringP("remote", "r", "", "use named remote instead of default")
}

func executeFetch(cmd *cobra.Command, args []string) {
	remoteName := cmd.Flag("remote").Value.String()
	err := cli.Fetch(remoteName, args[0])
	if err != nil {
		cli.Stderrf("fetch failed")
		os.Exit(1)
	}
}
