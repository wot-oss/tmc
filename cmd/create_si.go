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
	Short: "Creates or updates a bleve search index",
	Long:  `Creates or updates a bleve search index for all TMs`,
	Run:   executeCreateSearchIndex,
}

func init() {
	RootCmd.AddCommand(createSiCmd)
	createSiCmd.Flags().StringP("repo", "r", "", "name of the remote to pull from")
	createSiCmd.Flags().StringP("directory", "d", "", "TM repository directory to pull from")
}

func executeCreateSearchIndex(cmd *cobra.Command, args []string) {
	spec := RepoSpecFromFlags(cmd)
	err := cli.CreateSearchIndex(context.Background(), spec)
	if err != nil {
		os.Exit(1)
	}
}
