package cmd

import (
	"errors"
	"os"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

var filterFlags = cli.FilterFlags{}

var listCmd = &cobra.Command{
	Use:   "list <NAME PATTERN>",
	Short: "List TMs in catalog",
	Long: `List TMs in catalog by name pattern, filters or search. 
The pattern can be a full name or just a prefix the names shall start with. 
Name pattern, filters and search can be combined to narrow down the result.`,
	Args: cobra.MaximumNArgs(1),
	Run:  executeList,
}

func init() {
	RootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("remote", "r", "", "name of the remote to list")
	listCmd.Flags().StringP("directory", "d", "", "TM repository directory to list")
	listCmd.Flags().StringVar(&filterFlags.FilterAuthor, "filter.author", "", "filter TMs by one or more comma-separated authors")
	listCmd.Flags().StringVar(&filterFlags.FilterManufacturer, "filter.manufacturer", "", "filter TMs by one or more comma-separated manufacturers")
	listCmd.Flags().StringVar(&filterFlags.FilterMpn, "filter.mpn", "", "filter TMs by one or more comma-separated mpn (manufacturer part number)")
	listCmd.Flags().StringVar(&filterFlags.FilterExternalID, "filter.externalID", "", "filter TMs by one or more comma-separated external ID")
	listCmd.Flags().StringVarP(&filterFlags.Search, "search", "s", "", "search TMs by their content matching the search term")
}

func executeList(cmd *cobra.Command, args []string) {
	remoteName := cmd.Flag("remote").Value.String()
	dirName := cmd.Flag("directory").Value.String()

	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	search := cli.CreateSearchParamsFromCLI(filterFlags, name)
	if search != nil {
		search.Options = &model.SearchOptions{NameFilterType: model.PrefixMatch}
	}

	spec, err := remotes.NewSpec(remoteName, dirName)
	if errors.Is(err, remotes.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --remote and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}

	err = cli.List(spec, search)
	if err != nil {
		os.Exit(1)
	}
}
