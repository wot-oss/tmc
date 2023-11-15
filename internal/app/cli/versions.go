package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

func ListVersions(remoteName, name string) error {
	remote, err := remotes.Get(remoteName)
	if err != nil {
		Stderrf("Could not ìnitialize a remote instance for %s: %v\ncheck config", remoteName, err)
		return err
	}

	tocThing, err := remote.Versions(name)
	if err != nil {
		Stderrf("Could not list versions for %s: %v\ncheck config", name, err)
		return err
	}

	printToCThing(name, tocThing)
	return nil
}
func printToCThing(name string, tocThing model.TocThing) {
	//	colWidth := columnWidth()
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintf(table, "NAME\tVERSION\tTIME\tDESCRIPTION\tPATH\n")
	for _, v := range tocThing.Versions {
		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", name, v.Version.Model, v.TimeStamp, v.Description, v.Path)
	}
	_ = table.Flush()
}
