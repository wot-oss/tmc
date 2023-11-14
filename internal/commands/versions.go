package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

func PrintToCThing(name string, tocThing model.TocThing) {
	//	colWidth := columnWidth()
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(table, "NAME\tVERSION\tTIME\tDESCRIPTION\tPATH\n")
	for _, v := range tocThing.Versions {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", name, v.Version.Model, v.TimeStamp, v.Description, v.Path)
	}
	table.Flush()

}
