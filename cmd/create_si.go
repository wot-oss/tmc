package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
)

// createSiCmd represents the create-si command
var createSiCmd = &cobra.Command{
	Use:   "create-si",
	Short: "Create or update search index",
	Long: `Create or update a bleve search index for deep searching TMs with the 'search' command. Usually needs to
be called only once per repository. Afterwards, updates are performed automatically when an outdated search index is detected.`,
	Run: executeCreateSearchIndex,
}

func init() {
	RootCmd.AddCommand(createSiCmd)
	AddRepoConstraintFlags(createSiCmd)
}

func executeCreateSearchIndex(cmd *cobra.Command, args []string) {
	spec := RepoSpecFromFlags(cmd)
	err := cli.CreateSearchIndex(context.Background(), spec)
	if err != nil {
		os.Exit(1)
	}
}
