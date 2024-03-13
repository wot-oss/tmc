package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

var updateTocCmd = &cobra.Command{
	Use:   "update-toc [<id1> <id2> ...]",
	Short: "Update the Table of Contents",
	Long: `Update the Table of Contents file listing all paths to Thing Model files. Used for simple search functionality.
Optionally, TM id arguments can be provided to only update the table of contents with the given ids. Invalid/non-existing ids are ignored.
Specifying the repository with --directory or --remote is optional if there's exactly one remote configured`,
	Run: executeCreateTOC,
}

func init() {
	RootCmd.AddCommand(updateTocCmd)
	updateTocCmd.Flags().StringP("remote", "r", "", "name of the remote")
	updateTocCmd.Flags().StringP("directory", "d", "", "TM repository directory")

}

func executeCreateTOC(cmd *cobra.Command, args []string) {
	var log = slog.Default()

	remoteName := cmd.Flag("remote").Value.String()
	dir := cmd.Flag("directory").Value.String()
	spec, err := model.NewSpec(remoteName, dir)
	if errors.Is(err, model.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --remote and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}
	log.Debug(fmt.Sprintf("creating table of contents for remote %s", spec))

	err = cli.UpdateToc(spec, args)
	if err != nil {
		os.Exit(1)
	}
}
