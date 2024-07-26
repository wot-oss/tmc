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
	Short: "Update the repository's index file",
	Long: `Update the repository's index file listing all paths to Thing Model files. Used for simple search functionality.
Specifying the catalog with --directory or --repo is optional if there's exactly one catalog configured`,
	Run:  executeCreateIndex,
	Args: cobra.NoArgs,
}

func init() {
	RootCmd.AddCommand(indexCmd)
	indexCmd.Flags().StringP("repo", "r", "", "Name of the repository to update")
	indexCmd.Flags().StringP("directory", "d", "", "Use the specified directory as repository. This option allows directly using a directory as a local TM repository, forgoing creating a named repository.")

}

func executeCreateIndex(cmd *cobra.Command, args []string) {
	var log = slog.Default()

	spec := RepoSpec(cmd)
	log.Debug(fmt.Sprintf("creating table of contents for repository %s", spec))

	err := cli.Index(context.Background(), spec)
	if err != nil {
		os.Exit(1)
	}
}
