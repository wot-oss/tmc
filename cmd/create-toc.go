package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

var createTOCCmd = &cobra.Command{
	Use:   "create-toc DIRECTORY",
	Short: "Creates a Table of Contents",
	Long:  "Creates a Table of Contents listing all paths to Thing Model files. Used for simple search functionality.",
	Run:   executeCreateTOC,
}

func init() {
	rootCmd.AddCommand(createTOCCmd)
	createTOCCmd.Flags().StringP("remote", "r", "", "use named remote instead of default")
}

func executeCreateTOC(cmd *cobra.Command, args []string) {
	var log = slog.Default()

	remoteName := cmd.Flag("remote").Value.String()
	log.Debug(fmt.Sprintf("creating table of contents for remote %s", remoteName))

	remote, err := remotes.Get(remoteName)
	if err != nil {
		log.Error(fmt.Sprintf("could not ìnitialize a remote instance for %s. check config", remoteName), "error", err)
	}

	err = remote.CreateToC()
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
