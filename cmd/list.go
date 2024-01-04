package cmd

import (
	"os"
	"strings"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

var (
	flagFilterAuthor       string
	flagFilterManufacturer string
	flagFilterMpn          string
	flagFilterExternalID   string
	flagSearchContent      string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List TMs in catalog",
	Long:  `List TMs and optionally filter them. --remote is optional if there's only one remote configured`,
	Args:  cobra.MaximumNArgs(1),
	Run:   executeList,
}

func init() {
	RootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("remote", "r", "", "name of the remote to list")
	listCmd.Flags().StringVar(&flagFilterAuthor, "filter.author", "", "filter TMs by one or more comma-separated authors")
	listCmd.Flags().StringVar(&flagFilterManufacturer, "filter.manufacturer", "", "filter TMs by one or more comma-separated manufacturers")
	listCmd.Flags().StringVar(&flagFilterMpn, "filter.mpn", "", "filter TMs by one or more comma-separated mpn (manufacturer part number)")
	listCmd.Flags().StringVar(&flagFilterExternalID, "filter.externalID", "", "filter TMs by one or more comma-separated external ID")
	listCmd.Flags().StringVar(&flagSearchContent, "search.content", "", "search TMs by their content matching the search term")
}

func executeList(cmd *cobra.Command, args []string) {
	remoteName := cmd.Flag("remote").Value.String()

	search := convertSearchParams()

	err := cli.List(remoteName, search)
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
		if flagSearchContent != "" {
			search.Query = flagSearchContent
		}
	}
	return search
}

func hasSearchParamsSet() bool {
	return flagFilterAuthor != "" || flagFilterManufacturer != "" || flagFilterMpn != "" ||
		flagFilterExternalID != "" || flagSearchContent != ""
}
