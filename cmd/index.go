package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/model"
)

var indexCmd = &cobra.Command{
	Use:   "index [<id1> <id2> ...]",
	Short: "Update the repository's index file",
	Long: `Update the repository's index file listing all paths to Thing Model files. Used for simple search functionality.
Optionally, TM id arguments can be provided to only update the table of contents with the given ids. Ids referring to non-existing files are removed from index.
Specifying the catalog with --directory or --repo is optional if there's exactly one catalog configured`,
	Run: executeCreateIndex,
}

func init() {
	RootCmd.AddCommand(indexCmd)
	indexCmd.Flags().StringP("repo", "r", "", "Name of the repository to update")
	indexCmd.Flags().StringP("directory", "d", "", "Use the specified directory as repository. This option allows directly using a directory as a local TM repository, forgoing creating a named repository.")

}

func executeCreateIndex(cmd *cobra.Command, args []string) {
	var log = slog.Default()

	repoName := cmd.Flag("repo").Value.String()
	dir := cmd.Flag("directory").Value.String()
	spec, err := model.NewSpec(repoName, dir)
	if errors.Is(err, model.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --repo and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}
	log.Debug(fmt.Sprintf("creating table of contents for repository %s", spec))

	err = cli.Index(context.Background(), spec, args)
	if err != nil {
		os.Exit(1)
	}
}
