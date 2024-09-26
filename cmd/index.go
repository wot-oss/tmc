package cmd

import (
	"context"
	"fmt"
	"log/slog"
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

func executeRefreshIndex(cmd *cobra.Command, args []string) {
	var log = slog.Default()

	spec := RepoSpecFromFlags(cmd)
	log.Debug(fmt.Sprintf("Refreshing index for repository %s", spec))

	err := cli.Index(context.Background(), spec)
	if err != nil {
		os.Exit(1)
	}
}
