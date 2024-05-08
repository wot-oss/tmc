package cmd

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/model"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check catalog",
	Long:  `The command check and its subcommands allow to validate a catalog for consistency.`,
}

var checkResCmd = &cobra.Command{
	Use:               "resources [<name1> <name2> <name3> ...]",
	Short:             "Check resources in the catalog for consistency.",
	Long:              `Check all or multiple named resources in the catalog for consistency.`,
	Args:              cobra.MinimumNArgs(0),
	Run:               checkResources,
	ValidArgsFunction: completion.CompleteTMNames,
}

var checkIndexCmd = &cobra.Command{
	Use:   "index",
	Short: "Check the index of a catalog for consistency.",
	Long:  `Check the index of a catalog for consistency.`,
	Args:  cobra.MaximumNArgs(0),
	Run:   checkIndex,
}

func init() {
	RootCmd.AddCommand(checkCmd)

	checkCmd.AddCommand(checkResCmd)
	checkResCmd.Flags().StringP("repo", "r", "", "Name of the repository to check")
	checkResCmd.Flags().StringP("directory", "d", "", "Use the specified directory as repository. This option allows directly using a directory as a local TM repository, forgoing creating a named repository.")
	_ = checkResCmd.RegisterFlagCompletionFunc("repo", completion.CompleteRepoNames)
	_ = checkResCmd.MarkFlagDirname("directory")

	checkCmd.AddCommand(checkIndexCmd)
	checkIndexCmd.Flags().StringP("repo", "r", "", "Name of the repository to check")
	checkIndexCmd.Flags().StringP("directory", "d", "", "Use the specified directory as repository. This option allows directly using a directory as a local TM repository, forgoing creating a named repository.")
	_ = checkIndexCmd.RegisterFlagCompletionFunc("repo", completion.CompleteRepoNames)
	_ = checkIndexCmd.MarkFlagDirname("directory")
}

func checkResources(cmd *cobra.Command, args []string) {
	spec := repoSpec(cmd)
	err := cli.CheckResources(context.Background(), spec, args)

	if err != nil {
		cli.Stderrf("check resources failed")
		os.Exit(1)
	}
}

func checkIndex(cmd *cobra.Command, args []string) {
	spec := repoSpec(cmd)
	err := cli.CheckIndex(context.Background(), spec)

	if err != nil {
		cli.Stderrf("check index failed")
		os.Exit(1)
	}
}

func repoSpec(cmd *cobra.Command) model.RepoSpec {
	repoName := cmd.Flag("repo").Value.String()
	dir := cmd.Flag("directory").Value.String()
	spec, err := model.NewSpec(repoName, dir)
	if errors.Is(err, model.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --repo and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}
	return spec
}
