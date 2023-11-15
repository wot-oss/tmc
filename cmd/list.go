package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

var listCmd = &cobra.Command{
	Use:   "list [PATTERN]",
	Short: "List TMs in catalog",
	Long:  `List TMs and filter for PATTERN in all mandatory fields`,
	Args:  cobra.MaximumNArgs(1),
	Run:   listRemote,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("remote", "r", "", "use named remote instead of default")
}

func listRemote(cmd *cobra.Command, args []string) {
	log := slog.Default()

	// TODO: if not specified returns "remote"?
	remoteName := cmd.Flag("remote").Value.String()
	remote, err := remotes.Get(remoteName)
	if err != nil {
		// TODO: error seems specific to remotes.Get()
		log.Error(fmt.Sprintf("could not initialize a remote instance for %s. check config", remoteName), "error", err)
		os.Exit(1)
	}

	filter := ""
	if len(args) > 0 {
		filter = args[0]
	}

	toc, err := remote.List(filter)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	commands.PrintToC(toc, filter)
}
