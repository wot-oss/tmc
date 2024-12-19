package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
)

// createSiCmd represents the createSi command
var createSiCmd = &cobra.Command{
	Use:   "create-si",
	Short: "Create or update a bleve search index",
	Long: `Create or update a bleve search index for deep searching TMs with '--search <query> --deep'. Usually needs to
be called only once per repository. Afterwards, updates are performed automatically an outdated search index is detected.`,
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
