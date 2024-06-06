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

var filterFlags = cli.FilterFlags{}

var listCmd = &cobra.Command{
	Use:   "list <NAME PATTERN>",
	Short: "List TMs in catalog",
	Long: `List TMs in catalog by name pattern, filters or search. 
The name can be a full name or a prefix consisting of complete path parts. 
E.g. 'MyCompany/BarTech' will not match 'MyCompany/BarTechCorp', but will match 'MyCompany/BarTech/BazLamp'.

Name pattern, filters and search can be combined to narrow down the result.`,
	Args:              cobra.MaximumNArgs(1),
	Run:               executeList,
	ValidArgsFunction: completion.CompleteTMNames,
}

func init() {
	RootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("repo", "r", "", "Name of the repository to list. Lists all if omitted")
	_ = listCmd.RegisterFlagCompletionFunc("repo", completion.CompleteRepoNames)
	listCmd.Flags().StringP("directory", "d", "", "Use the specified directory as repository. This option allows directly using a directory as a local TM repository, forgoing creating a named repository.")
	_ = listCmd.MarkFlagDirname("directory")
	listCmd.Flags().StringVar(&filterFlags.FilterAuthor, "filter.author", "", "filter TMs by one or more comma-separated authors")
	listCmd.Flags().StringVar(&filterFlags.FilterManufacturer, "filter.manufacturer", "", "filter TMs by one or more comma-separated manufacturers")
	listCmd.Flags().StringVar(&filterFlags.FilterMpn, "filter.mpn", "", "filter TMs by one or more comma-separated mpn (manufacturer part number)")
	listCmd.Flags().StringVarP(&filterFlags.Search, "search", "s", "", "search TMs by their content matching the search term")
}

func executeList(cmd *cobra.Command, args []string) {
	repoName := cmd.Flag("repo").Value.String()
	dirName := cmd.Flag("directory").Value.String()

	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	spec, err := model.NewSpec(repoName, dirName)
	if errors.Is(err, model.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --repo and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}

	search := cli.CreateSearchParamsFromCLI(filterFlags, name)
	err = cli.List(context.Background(), spec, search)
	if err != nil {
		cli.Stderrf("list failed")
		os.Exit(1)
	}
}
