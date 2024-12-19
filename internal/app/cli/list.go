package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/viper"
	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/config"
	"github.com/wot-oss/tmc/internal/model"
)

func List(ctx context.Context, repo model.RepoSpec, search *model.Filters, format string) error {
	if !IsValidOutputFormat(format) {
		Stderrf("%v", ErrInvalidOutputFormat)
		return ErrInvalidOutputFormat
	}
	index, err, errs := commands.List(ctx, repo, search)
	if err != nil {
		Stderrf("Error listing: %v", err)
		return err
	}

	if len(errs) > 0 {
		err = errs[0]
	}

	switch format {
	case OutputFormatJSON:
		resp := toListResults(index)
		printJSON(resp)
	case OutputFormatPlain:
		printIndex(index)
	}
	printErrs("Errors occurred while listing:", errs)
	return err
}

// TODO: use better table writer with eliding etc.
func printIndex(res model.SearchResult) {
	colWidth := columnWidth()
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintf(table, "NAME\tAUTHOR\tMANUFACTURER\tMPN\tREPO\n")
	for _, value := range res.Entries {
		name := value.Name
		man := elideString(value.Manufacturer.Name, colWidth)
		mpn := elideString(value.Mpn, colWidth)
		auth := elideString(value.Author.Name, colWidth)
		repo := elideString(fmt.Sprintf("%v", value.FoundIn), colWidth)
		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", name, auth, man, mpn, repo)
	}
	_ = table.Flush()
}

func toListResults(res model.SearchResult) []ListResultEntry {
	var r []ListResultEntry
	for _, e := range res.Entries {
		r = append(r, ListResultEntry{
			Name:         e.Name,
			Author:       e.Author.Name,
			Manufacturer: e.Manufacturer.Name,
			MPN:          e.Mpn,
			Repo:         e.FoundIn.String(),
		})
	}
	return r
}

type ListResultEntry struct {
	Name         string `json:"name"`
	Author       string `json:"author"`
	Manufacturer string `json:"manufacturer"`
	MPN          string `json:"mpn"`
	Repo         string `json:"repo"`
}

func elideString(value string, colWidth int) string {
	if len(value) < colWidth {
		return value
	}

	var elidedValue string
	for i, rn := range value {
		elidedValue += string(rn)
		if i >= (colWidth - 4) {
			return elidedValue + "..."
		}
	}
	return value + "..."
}

func columnWidth() int {
	return viper.GetInt(config.KeyColumnWidth)
}
