package cmd

import (
	"errors"
	"os"
	"strings"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

var (
	flagFilterAuthor       string
	flagFilterManufacturer string
	flagFilterMpn          string
	flagFilterExternalID   string
	flagSearch             string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List TMs in catalog",
	Long:  `List TMs and optionally filter them`,
	Args:  cobra.MaximumNArgs(1),
	Run:   executeList,
}

func init() {
	RootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("remote", "r", "", "name of the remote to list")
	listCmd.Flags().StringP("directory", "d", "", "TM repository directory to list")
	listCmd.Flags().StringVar(&flagFilterAuthor, "filter.author", "", "filter TMs by one or more comma-separated authors")
	listCmd.Flags().StringVar(&flagFilterManufacturer, "filter.manufacturer", "", "filter TMs by one or more comma-separated manufacturers")
	listCmd.Flags().StringVar(&flagFilterMpn, "filter.mpn", "", "filter TMs by one or more comma-separated mpn (manufacturer part number)")
	listCmd.Flags().StringVar(&flagFilterExternalID, "filter.externalID", "", "filter TMs by one or more comma-separated external ID")
	listCmd.Flags().StringVarP(&flagSearch, "search", "s", "", "search TMs by their content matching the search term")
}

func executeList(cmd *cobra.Command, args []string) {
	remoteName := cmd.Flag("remote").Value.String()
	dirName := cmd.Flag("directory").Value.String()

	search := convertSearchParams()
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

func convertSearchParams() *model.SearchParams {
	var search *model.SearchParams

	if hasSearchParamsSet() {
		search = &model.SearchParams{}
		if flagFilterAuthor != "" {
			search.Author = strings.Split(flagFilterAuthor, ",")
		}
		if flagFilterManufacturer != "" {
			search.Manufacturer = strings.Split(flagFilterManufacturer, ",")
		}
		if flagFilterMpn != "" {
			search.Mpn = strings.Split(flagFilterMpn, ",")
		}
		if flagFilterExternalID != "" {
			search.ExternalID = strings.Split(flagFilterExternalID, ",")
		}
		if flagSearch != "" {
			search.Query = flagSearch
		}
	}
	return search
}

func hasSearchParamsSet() bool {
	return flagFilterAuthor != "" || flagFilterManufacturer != "" || flagFilterMpn != "" ||
		flagFilterExternalID != "" || flagSearch != ""
}
