package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

var createTOCCmd = &cobra.Command{
	Use:   "create-toc",
	Short: "Creates a Table of Contents",
	Long: `Creates a Table of Contents file listing all paths to Thing Model files. Used for simple search functionality.
Specifying the repository with --directory or --remote is optional if there's exactly one remote configured`,
	Run: executeCreateTOC,
}

func init() {
	RootCmd.AddCommand(createTOCCmd)
	createTOCCmd.Flags().StringP("remote", "r", "", "name of the remote")
	createTOCCmd.Flags().StringP("directory", "d", "", "TM repository directory")

}

func executeCreateTOC(cmd *cobra.Command, _ []string) {
	var log = slog.Default()

	remoteName := cmd.Flag("remote").Value.String()
	dir := cmd.Flag("directory").Value.String()
	spec, err := remotes.NewSpec(remoteName, dir)
	if errors.Is(err, remotes.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --remote and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}
	log.Debug(fmt.Sprintf("creating table of contents for remote %s", spec))

	err = cli.UpdateToc(remotes.DefaultManager(), spec)
	if err != nil {
		os.Exit(1)
	}
}
