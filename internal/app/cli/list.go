package cli

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

// TODO: figure out how to use viper
const columnWidthName = "TMC_COLUMNWIDTH"
const columnWidthDefault = 40

func List(remote model.RepoSpec, search *model.SearchParams) error {
	toc, err, errs := commands.List(remote, search)
	if err != nil {
		Stderrf("Error listing: %v", err)
		return err
	}

	printToC(toc)
	printErrs("Errors occurred while listing:", errs)
	return nil
}

// TODO: use better table writer with eliding etc.
func printToC(toc model.SearchResult) {
	colWidth := columnWidth()
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintf(table, "NAME\tAUTHOR\tMANUFACTURER\tMPN\n")
	for _, value := range toc.Entries {
		name := value.Name
		man := elideString(value.Manufacturer.Name, colWidth)
		mpn := elideString(value.Mpn, colWidth)
		auth := elideString(value.Author.Name, colWidth)
		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", name, auth, man, mpn)
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
