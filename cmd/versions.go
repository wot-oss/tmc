package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var versionsCmd = &cobra.Command{
	Use:               "versions <tm-name>",
	Short:             "List available versions of the TM with given name",
	Long:              `List available versions of the TM with given name`,
	Args:              cobra.ExactArgs(1),
	Run:               listVersions,
	ValidArgsFunction: completion.CompleteTMNames,
}

func init() {
	RootCmd.AddCommand(versionsCmd)
	AddRepoConstraintFlags(versionsCmd)
	AddOutputFormatFlag(versionsCmd)
}

func listVersions(cmd *cobra.Command, args []string) {
	spec := RepoSpecFromFlags(cmd)
	format := cmd.Flag("format").Value.String()

	name := args[0]
	err := cli.ListVersions(context.Background(), spec, name, format)
	if err != nil {
		cli.Stderrf("versions failed")
		os.Exit(1)
	}
}
