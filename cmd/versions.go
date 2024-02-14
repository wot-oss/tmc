package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/cmd/completion"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
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
	versionsCmd.Flags().StringP("remote", "r", "", "name of the remote to search for versions")
	_ = versionsCmd.RegisterFlagCompletionFunc("remote", completion.CompleteRemoteNames)
	versionsCmd.Flags().StringP("directory", "d", "", "TM repository directory")
	_ = versionsCmd.MarkFlagDirname("directory")
}

func listVersions(cmd *cobra.Command, args []string) {
	remoteName := cmd.Flag("remote").Value.String()
	dirName := cmd.Flag("directory").Value.String()
	spec, err := remotes.NewSpec(remoteName, dirName)
	if errors.Is(err, remotes.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --remote and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}

	name := args[0]
	err = cli.ListVersions(spec, name)
	if err != nil {
		os.Exit(1)
	}
}
