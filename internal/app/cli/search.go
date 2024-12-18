package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
)

func Search(ctx context.Context, repo model.RepoSpec, query, format string) error {
	if !IsValidOutputFormat(format) {
		Stderrf("%v", ErrInvalidOutputFormat)
		return ErrInvalidOutputFormat
	}
	index, err, errs := commands.Search(ctx, repo, query)
	if err != nil {
		Stderrf("Error searching: %v", err)
		return err
	}

	if len(errs) > 0 {
		err = errs[0]
	}

	switch format {
	case OutputFormatJSON:
		resp := toSearchCommandResult(index)
		printJSON(resp)
	case OutputFormatPlain:
		printSearchResult(index)
	}
	printErrs("Errors occurred while listing:", errs)
	return err
}

func printSearchResult(res model.SearchResult) {
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

func toSearchCommandResult(res model.SearchResult) []SearchResultEntry {
	var r []SearchResultEntry
	for _, e := range res.Entries {
		for _, v := range e.Versions {
			r = append(r, SearchResultEntry{
				TMID: v.TMID,
				Repo: v.FoundIn.String(),
			})
		}
	}
	return r
}

type SearchResultEntry struct {
	TMID string `json:"tmid"`
	Repo string `json:"repo"`
}
