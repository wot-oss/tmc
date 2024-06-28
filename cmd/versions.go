package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var versionsCmd = &cobra.Command{
	Use:               "versions <name>",
	Short:             "List available versions of the TM with given name",
	Long:              `List available versions of the TM with given name`,
	Args:              cobra.ExactArgs(1),
	Run:               listVersions,
	ValidArgsFunction: completion.CompleteTMNames,
}

func init() {
	RootCmd.AddCommand(versionsCmd)
	versionsCmd.Flags().StringP("repo", "r", "", "Name of the repository to search for versions. Searches all if omitted")
	_ = versionsCmd.RegisterFlagCompletionFunc("repo", completion.CompleteRepoNames)
	versionsCmd.Flags().StringP("directory", "d", "", "Use the specified directory as repository. This option allows directly using a directory as a local TM repository, forgoing creating a named repository.")
	_ = versionsCmd.MarkFlagDirname("directory")
}

func listVersions(cmd *cobra.Command, args []string) {
	spec := RepoSpec(cmd)

	name := args[0]
	err := cli.ListVersions(context.Background(), spec, name)
	if err != nil {
		cli.Stderrf("versions failed")
		os.Exit(1)
	}
}
