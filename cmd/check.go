package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var checkCmd = &cobra.Command{
	Use:   "check [<name1> <name2> <name3> ...]",
	Short: "Check the integrity of all or only named resources in internal repository storage.",
	Long: `The check command verifies a repository for internal consistency and integrity of the storage.
If arguments are given, only the named resources are checked. If a named resource does not exist, no error is returned.
The check command with arguments is meant for use in CI pipeline scripts. They could get a list of modified files from git and
pass it to the check command and thus verify the integrity of only modified parts of a repository.`,
	Args: cobra.ArbitraryArgs,
	Run:  checkIntegrity,
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
	err := cli.CheckIntegrity(context.Background(), spec, args)

	if err != nil {
		cli.Stderrf("integrity check failed")
		os.Exit(1)
	}
}
