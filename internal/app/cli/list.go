package cli

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

// TODO: figure out how to use viper
const columnWidthName = "TMC_COLUMNWIDTH"
const columnWidthDefault = 40

func ListRemote(remoteName, filter string) error {
	remote, err := remotes.Get(remoteName)
	if err != nil {
		Stderrf("Could not ìnitialize a remote instance for %s: %v\ncheck config", remoteName, err)
		return err
	}
	toc, err := remote.List(filter)
	if err != nil {
		Stderrf("could not list %s: %v", remoteName, err)
		return err
	}
	printToC(toc, filter)
	return nil
}

// TODO: use better table writer with eliding etc.
func printToC(toc model.Toc, filter string) {
	colWidth := columnWidth()
	contents := toc.Contents
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintf(table, "NAME\tMANUFACTURER\tMODEL\n")
	for name, value := range contents {
		name := elideString(name, colWidth)
		man := elideString(value.Manufacturer.Name, colWidth)
		mdl := elideString(value.Mpn, colWidth)
		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\n", name, man, mdl)
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
