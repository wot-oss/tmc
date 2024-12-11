package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
)

func ListVersions(ctx context.Context, spec model.RepoSpec, name, format string) error {
	if !IsValidOutputFormat(format) {
		Stderrf("%v", ErrInvalidOutputFormat)
		return ErrInvalidOutputFormat
	}
	indexVersions, err, errs := commands.NewVersionsCommand().ListVersions(ctx, spec, name)
	if err != nil {
		Stderrf("Could not list versions of %s: %v", name, err)
		return err
	}

	if len(errs) > 0 {
		err = errs[0]
	}
	switch format {
	case OutputFormatJSON:
		res := toVersionResults(name, indexVersions)
		printJSON(res)
	case OutputFormatPlain:
		printIndexThing(name, indexVersions)
	}

	printErrs("Errors occurred while listing versions:", errs)
	return err
}

type VersionResultEntry struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Repo        string `json:"repo"`
	ID          string `json:"id"`
}

func toVersionResults(name string, vers []model.FoundVersion) []VersionResultEntry {
	var r []VersionResultEntry
	for _, e := range vers {
		r = append(r, VersionResultEntry{
			Name:        name,
			Version:     e.Version.Model,
			Description: e.Description,
			Repo:        e.FoundIn.String(),
			ID:          e.TMID,
		})
	}
	return r

}

func printIndexThing(name string, versions []model.FoundVersion) {
	//	colWidth := columnWidth()
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintf(table, "NAME\tVERSION\tDESCRIPTION\tREPO\tID\n")
	for _, v := range versions {
		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", name, v.Version.Model, v.Description, v.FoundIn, v.Links["content"])
	}
	_ = table.Flush()
}
