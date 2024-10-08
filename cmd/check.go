package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var checkCmd = &cobra.Command{
	Use:   "check [<resource-name>...]",
	Short: "Check the integrity of all or only named resources in repository's internal storage",
	Long: `The check command verifies a repository for internal consistency and integrity of the storage.
If arguments are given, only the named resources are checked. If a named resource does not exist, no error is returned.
Resource names correspond to the relative paths of TMs and attachments as they are stored in a file repository. I.e. for TMs the
resource names are same as their ids and for attachments they have the format: <tm-name> "/.attachments/" [<tm-version-string> "/"] <attachment-name>.

The check command with arguments is meant for use in CI pipeline scripts. They could get a list of modified files from git and
pass it to the check command and thus verify the integrity of only the modified parts of a repository.`,
	Args: cobra.ArbitraryArgs,
	Run:  checkIntegrity,
}

func init() {
	RootCmd.AddCommand(checkCmd)
	AddRepoDisambiguatorFlags(checkCmd)
}

func checkIntegrity(cmd *cobra.Command, args []string) {
	spec := RepoSpecFromFlags(cmd)
	err := cli.CheckIntegrity(context.Background(), spec, args)

	if err != nil {
		cli.Stderrf("integrity check failed")
		cli.Stderrf("Hint: make sure you did not change any files directly, bypassing TMC CLI. If you did, consider reverting the files and/or running `tmc index` on the repository")
		os.Exit(1)
	}
}
