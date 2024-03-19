package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
)

func ListVersions(spec model.RepoSpec, name string) error {
	indexVersions, err, errs := commands.NewVersionsCommand().ListVersions(spec, name)
	if err != nil {
		Stderrf("Could not list versions of %s: %v", name, err)
		return err
	}
	printIndexThing(name, indexVersions)
	printErrs("Errors occurred while listing versions:", errs)
	return nil
}

func printIndexThing(name string, versions []model.FoundVersion) {
	//	colWidth := columnWidth()
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintf(table, "NAME\tVERSION\tDESCRIPTION\tREPOSITORY\tID\n")
	for _, v := range versions {
		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", name, v.Version.Model, v.Description, v.FoundIn, v.Links["content"])
	}
	_ = table.Flush()
}
