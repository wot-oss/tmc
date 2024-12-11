package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Refresh the repository's internal index, if it has one",
	Long: `Refresh the repository's internal index, if it has one.
Specifying the repository with --directory or --repo is optional if there's exactly one repository configured.`,
	Run:  executeRefreshIndex,
	Args: cobra.NoArgs,
}

func init() {
	RootCmd.AddCommand(indexCmd)
	AddRepoDisambiguatorFlags(indexCmd)
}

func executeRefreshIndex(cmd *cobra.Command, _ []string) {
	spec := RepoSpecFromFlags(cmd)

	err := cli.Index(context.Background(), spec)
	if err != nil {
		cli.Stderrf("index failed")
		os.Exit(1)
	}
}
