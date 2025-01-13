package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var searchCmd = &cobra.Command{
	Use:   "search [<search-term> ...]",
	Short: "Search full text of TMs in catalog using bleve search engine",
	Long: `Search full text of TMs in catalog using bleve search engine. For each repository to be searched,
a local search index has to be created once using 'create-si' command.

The accepted search query syntax is described at https://blevesearch.com/docs/Query-String-Query/`,
	Args:              cobra.MinimumNArgs(1),
	Run:               executeSearch,
	ValidArgsFunction: completion.CompleteTMNames,
}

func init() {
	RootCmd.AddCommand(searchCmd)
	AddRepoConstraintFlags(searchCmd)
	AddOutputFormatFlag(searchCmd)
}

func executeSearch(cmd *cobra.Command, args []string) {
	spec := RepoSpecFromFlags(cmd)
	format := cmd.Flag("format").Value.String()

	searchQuery := strings.Join(args, " ")
	err := cli.Search(context.Background(), spec, searchQuery, format)
	if err != nil {
		cli.Stderrf("search failed")
		os.Exit(1)
	}
}
