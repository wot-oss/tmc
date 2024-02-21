package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

func ListVersions(spec remotes.RepoSpec, name string) error {
	tocVersions, err, errs := commands.NewVersionsCommand(remotes.DefaultManager()).ListVersions(spec, name)
	if err != nil {
		Stderrf("Could not list versions of %s: %v", name, err)
		return err
	}
	printToCThing(name, tocVersions)
	printErrs("Errors occurred while listing versions:", errs)
	return nil
}

func printToCThing(name string, versions []model.FoundVersion) {
	//	colWidth := columnWidth()
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintf(table, "NAME\tVERSION\tDESCRIPTION\tREPOSITORY\tID\n")
	for _, v := range versions {
		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", name, v.Version.Model, v.Description, v.FoundIn, v.Links["content"])
	}
	_ = table.Flush()
}
