package cmd

import (
	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"log/slog"
	"os"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push [file.tm.json]",
	Short: "Push TM to remote",
	Long:  `Push TM to remote`,
	Args:  cobra.ExactArgs(1),
	Run:   executePush,
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().StringP("remote", "r", "", "use named remote instead of default")
}

func executePush(cmd *cobra.Command, args []string) {
	var log = slog.Default()

	log.Debug("executing push", "args", args)
	remoteName := cmd.Flag("remote").Value.String()

	err := commands.PushToRemote(remoteName, args[0])
	if err != nil {
		log.Error("push failed", "error", err)
		os.Exit(1)
	}
}
