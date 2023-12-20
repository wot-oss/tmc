package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

var createTOCCmd = &cobra.Command{
	Use:   "create-toc DIRECTORY [--remote <remoteName>]",
	Short: "Creates a Table of Contents",
	Long: `Creates a Table of Contents listing all paths to Thing Model files. Used for simple search functionality.
--remote is optional if there's only one remote configured`,
	Run: executeCreateTOC,
}

func init() {
	RootCmd.AddCommand(createTOCCmd)
	createTOCCmd.Flags().StringP("remote", "r", "", "name of the remote")
}

func executeCreateTOC(cmd *cobra.Command, args []string) {
	var log = slog.Default()

	remoteName := cmd.Flag("remote").Value.String()
	log.Debug(fmt.Sprintf("creating table of contents for remote %s", remoteName))

	remote, err := remotes.DefaultManager().Get(remoteName)
	if err != nil {
		//TODO: log to stderr or logger ?
		cli.Stderrf("could not initialize a remote instance for %s: %v. check config", remoteName, err)
		os.Exit(1)
	}

	err = remote.CreateToC()

	if err != nil {
		//TODO: log to stderr or logger ?
		cli.Stderrf("could not create TOC: %v", err)
		os.Exit(1)
	}
}
