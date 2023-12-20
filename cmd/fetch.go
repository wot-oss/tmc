package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch NAME[:SEMVER|DIGEST] [--remote <remoteName>]",
	Short: "Fetches the TM by name",
	Long:  "Fetches TM by name, optionally accepting semantic version or digest. --remote is optional if there's only one remote configured",
	Args:  cobra.ExactArgs(1),
	Run:   executeFetch,
}

func init() {
	RootCmd.AddCommand(fetchCmd)
	fetchCmd.Flags().StringP("remote", "r", "", "name of the remote to fetch from")
}

func executeFetch(cmd *cobra.Command, args []string) {
	remoteName := cmd.Flag("remote").Value.String()
	err := cli.Fetch(args[0], remoteName)
	if err != nil {
		cli.Stderrf("fetch failed")
		os.Exit(1)
	}
}
