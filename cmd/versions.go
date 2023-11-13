package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

var versionsCmd = &cobra.Command{
	Use:   "versions [name]",
	Short: "List available versions of the TM",
	Long:  `List available versions of the TM`,
	Args:  cobra.ExactArgs(1),
	Run:   Versions,
}

func init() {
	rootCmd.AddCommand(versionsCmd)
	versionsCmd.Flags().StringP("remote", "r", "", "use named remote instead of default")
}

func Versions(cmd *cobra.Command, args []string) {
	log := slog.Default()

	// TODO: if not specified returns "remote"?
	remoteName := cmd.Flag("remote").Value.String()
	remote, err := remotes.Get(remoteName)
	if err != nil {
		// TODO: error seems specific to remotes.Get()
		log.Error(fmt.Sprintf("could not initialize a remote instance for %s. check config", remoteName), "error", err)
		os.Exit(1)
	}

	name := args[0]
	tocThing, err := remote.Versions(name)
	if err != nil {
		fmt.Errorf(err.Error())
		os.Exit(1)
	}
	commands.PrintToCThing(name, tocThing)
}
