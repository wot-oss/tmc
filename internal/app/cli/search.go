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

	_, _ = fmt.Fprintf(table, "ID\tREPO\tSCORE\tMATCHES\n")
	for _, entry := range res.Entries {
		for _, v := range entry.Versions {
			repo := elideString(fmt.Sprintf("%v", v.FoundIn), colWidth)
			sm := v.SearchMatch
			_, _ = fmt.Fprintf(table, "%s\t%s\t%v\t%s\n", v.TMID, repo, sm.Score, sm.Locations[0])
			if len(sm.Locations) > 1 {
				for _, l := range sm.Locations[1:] {
					_, _ = fmt.Fprintf(table, "%s\t%s\t%v\t%s\n", "", "", "", l)
				}
			}
		}
	}
	_ = table.Flush()
}

func toSearchCommandResult(res model.SearchResult) []SearchResultEntry {
	var r []SearchResultEntry
	for _, e := range res.Entries {
		for _, v := range e.Versions {
			r = append(r, SearchResultEntry{
				TMID:        v.TMID,
				Repo:        v.FoundIn.String(),
				SearchMatch: v.SearchMatch,
			})
		}
	}
	return r
}

type SearchResultEntry struct {
	TMID        string             `json:"tmid"`
	Repo        string             `json:"repo"`
	SearchMatch *model.SearchMatch `json:"searchMatch,omitempty"`
}
