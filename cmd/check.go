package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var checkCmd = &cobra.Command{
	Use:               "check",
	Short:             "Check the integrity of internal repository storage",
	Long:              `The check command verifies a repository for internal consistency and integrity of the storage.`,
	Args:              cobra.NoArgs,
	Run:               checkIntegrity,
	ValidArgsFunction: completion.CompleteTMNames,
}

func init() {
	RootCmd.AddCommand(checkCmd)

	checkCmd.Flags().StringP("repo", "r", "", "Name of the repository to check")
	checkCmd.Flags().StringP("directory", "d", "", "Use the specified directory as repository. This option allows directly using a directory as a local TM repository, forgoing creating a named repository.")
	_ = checkCmd.RegisterFlagCompletionFunc("repo", completion.CompleteRepoNames)
	_ = checkCmd.MarkFlagDirname("directory")
}

func checkIntegrity(cmd *cobra.Command, args []string) {
	spec := RepoSpec(cmd)
	err := cli.CheckIntegrity(context.Background(), spec)

	if err != nil {
		cli.Stderrf("integrity check failed")
		os.Exit(1)
	}
}
