package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch <name>[:<semver>] | <tmid>",
	Short: "Fetch a TM by name or id",
	Long: `Fetch a TM by name, optionally accepting a semantic version, or id.
The semantic version can be full or partial, e.g. v1.2.3, v1.2, v1. The 'v' at the beginning of a version is optional.
When fetching by id, the returned TM's id might have a different timestamp than that in the requested id, because the timestamp
is considered irrelevant in this case. The TM name, semantic version, and content digest will match exactly, of course.`,
	Args:              cobra.ExactArgs(1),
	Run:               executeFetch,
	ValidArgsFunction: completion.CompleteFetchNames,
}

func init() {
	RootCmd.AddCommand(fetchCmd)
	AddRepoConstraintFlags(fetchCmd)
	fetchCmd.Flags().StringP("output", "o", "", "Write the fetched TM to output folder instead of stdout")
	_ = fetchCmd.MarkFlagDirname("output")
	fetchCmd.Flags().BoolP("restore-id", "R", false, "Restore the TM's original external id, if it had one")
}

func executeFetch(cmd *cobra.Command, args []string) {
	outputPath := cmd.Flag("output").Value.String()
	restoreId, _ := cmd.Flags().GetBool("restore-id")

	spec := RepoSpecFromFlags(cmd)

	err := cli.Fetch(context.Background(), spec, args[0], outputPath, restoreId)
	if err != nil {
		cli.Stderrf("fetch failed")
		os.Exit(1)
	}
}
