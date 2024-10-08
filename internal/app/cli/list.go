package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
)

// TODO: figure out how to use viper
const columnWidthName = "TMC_COLUMNWIDTH"
const columnWidthDefault = 40

func List(ctx context.Context, repo model.RepoSpec, search *model.SearchParams) error {
	index, err, errs := commands.List(ctx, repo, search)
	if err != nil {
		Stderrf("Error listing: %v", err)
		return err
	}

	if len(errs) > 0 {
		err = errs[0]
	}

	printIndex(index)
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

		sc := ""
		for i, ver := range value.Versions {
			if i > 0 {
				sc = sc + ", "
			}
			sc = sc + fmt.Sprintf("%2.5f (v%s)", ver.SearchScore, ver.Version.Model)
		}
		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\n", name, auth, man, mpn, repo, sc)

	}
	_ = table.Flush()
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
	cw, err := strconv.Atoi(os.Getenv(columnWidthName))
	if err != nil {
		cw = columnWidthDefault
	}
	return cw
}
