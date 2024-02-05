package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch <NAME>[:<SEMVER>] | <TMID>)",
	Short: "Fetches a TM by name or id",
	Long: `Fetches a TM by name, optionally accepting a semantic version, or id.
The semantic version can be full or partial, e.g. v1.2.3, v1.2, v1. The 'v' at the beginning of a version is optional.`,
	Args: cobra.ExactArgs(1),
	Run:  executeFetch,
}

func init() {
	RootCmd.AddCommand(fetchCmd)
	fetchCmd.Flags().StringP("remote", "r", "", "name of the remote to fetch from")
	fetchCmd.Flags().StringP("directory", "d", "", "TM repository directory")
	fetchCmd.Flags().StringP("output", "o", "", "write the fetched TM to output folder instead of stdout")
}

func executeFetch(cmd *cobra.Command, args []string) {
	remoteName := cmd.Flag("remote").Value.String()
	dirName := cmd.Flag("directory").Value.String()
	outputPath := cmd.Flag("output").Value.String()

	spec, err := remotes.NewSpec(remoteName, dirName)
	if errors.Is(err, remotes.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --remote and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}

	err = cli.NewFetchExecutor(remotes.DefaultManager()).Fetch(spec, args[0], outputPath)
	if err != nil {
		cli.Stderrf("fetch failed")
		os.Exit(1)
	}
}
