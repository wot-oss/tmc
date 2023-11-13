package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

// TODO: figure out how to use viper
const columnWidthName = "TMC_COLUMNWIDTH"
const columnWidthDefault = 40

// TODO: use better table writer with eliding etc.
func PrintToC(toc model.Toc, filter string) {
	filter = prep(filter)
	colWidth := columnWidth()
	contents := toc.Contents
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(table, "NAME\tMANUFACTURER\tMODEL\n")
	for name, value := range contents {
		if !matchFilter(name, value, filter) {
			continue
		}
		name := elideString(name, colWidth)
		man := elideString(value.Manufacturer.Name, colWidth)
		model := elideString(value.Mpn, colWidth)
		fmt.Fprintf(table, "%s\t%s\t%s\n", name, man, model)
	}
	table.Flush()
}

func elideString(value string, colWidth int) string {
	if len(value) < colWidth {
		return value
	}

	var elidedValue string
	for i, rune := range value {
		elidedValue += string(rune)
		if i >= (colWidth - 4) {
			return elidedValue + "..."
		}
	}
	return value + "..."
}

func matchFilter(name string, thing model.TocThing, filter string) bool {
	if strings.Contains(prep(thing.Manufacturer.Name), filter) {
		return true
	}
	if strings.Contains(prep(thing.Mpn), filter) {
		return true
	}
	for _, version := range thing.Versions {
		if strings.Contains(prep(version.Description), filter) {
			return true
		}
	}
	return false
}

func prep(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	return s
}

func columnWidth() int {
	cw, err := strconv.Atoi(os.Getenv(columnWidthName))
	if err != nil {
		cw = columnWidthDefault
	}
	return cw
}
