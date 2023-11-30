package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

func ListVersions(remoteName, name string) error {
	tocEntry, err := commands.ListVersions(remoteName, name)
	if err != nil {
		Stderrf("Could not list versions for %s: %v\ncheck config", name, err)
		return err
	}

	printToCThing(name, tocEntry)
	return nil
}

func printToCThing(name string, tocEntry model.FoundEntry) {
	//	colWidth := columnWidth()
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintf(table, "NAME\tDESCRIPTION\tREMOTE\tPATH\n")
	for _, v := range tocEntry.Versions {
		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", name, v.Description, v.FoundIn, v.Links["content"])
	}
	_ = table.Flush()
}
