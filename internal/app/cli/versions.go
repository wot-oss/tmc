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
		res := toVersionResults(indexVersions)
		printJSON(res)
	case OutputFormatPlain:
		printFoundVersions(indexVersions)
	}

	printErrs("Errors occurred while listing versions:", errs)
	return err
}

type VersionResultEntry struct {
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Repo        string `json:"repo"`
	ID          string `json:"id"`
}

func toVersionResults(vers []model.FoundVersion) []VersionResultEntry {
	var r []VersionResultEntry
	for _, e := range vers {
		r = append(r, VersionResultEntry{
			Version:     e.Version.Model,
			Description: e.Description,
			Repo:        e.FoundIn.String(),
			ID:          e.TMID,
		})
	}
	return r

}

func printFoundVersions(versions []model.FoundVersion) {
	//	colWidth := columnWidth()
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintf(table, "VERSION\tID\tREPO\tDESCRIPTION\n")
	for _, v := range versions {
		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", v.Version.Model, v.TMID, v.FoundIn, v.Description)
	}
	_ = table.Flush()
}
