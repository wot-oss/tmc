package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

const (
	PullOK = PullResultType(iota)
	PullErr
)

type PullResultType int

func (t PullResultType) String() string {
	switch t {
	case PullOK:
		return "OK"
	case PullErr:
		return "error"
	default:
		return "unknown"
	}
}

type PullResult struct {
	typ  PullResultType
	tmid string
	text string
}

func (r PullResult) String() string {
	return fmt.Sprintf("%v\t %s %s", r.typ, r.tmid, r.text)
}

func Pull(ctx context.Context, repo model.RepoSpec, search *model.SearchParams, outputPath string, restoreId bool) error {
	if len(outputPath) == 0 {
		Stderrf("requires output target folder --output")
		return errors.New("--output not provided")
	}

	f, err := os.Stat(outputPath)
	if f != nil && !f.IsDir() {
		Stderrf("output target folder --output is not a folder")
		return errors.New("output target folder --output is not a folder")
	}

	searchResult, err, errs := commands.List(ctx, repo, search)
	if err != nil {
		Stderrf("Error listing: %v", err)
		return err
	}

	vc := 0
	for _, m := range searchResult.Entries {
		vc += len(m.Versions)
	}

	fmt.Printf("Pulling %d ThingModels with %d versions...\n", len(searchResult.Entries), vc)

	var totalRes []PullResult
	for _, entry := range searchResult.Entries {
		for _, version := range entry.Versions {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			res, pErr := pullThingModel(ctx, outputPath, version, restoreId)
			if err == nil && pErr != nil {
				err = pErr
			}
			totalRes = append(totalRes, res)
		}
	}

	if err == nil && len(errs) > 0 {
		err = errs[0]
	}

	for _, res := range totalRes {
		fmt.Println(res)
	}
	printErrs("Errors occurred while listing TMs for pull:", errs)

	return err
}

func pullThingModel(ctx context.Context, outputPath string, version model.FoundVersion, restoreId bool) (PullResult, error) {
	spec := model.NewSpecFromFoundSource(version.FoundIn)
	id, thing, err, errs := commands.FetchByTMID(ctx, spec, version.TMID, restoreId)
	if err == nil && len(errs) > 0 { // spec cannot be empty, therefore, there can be at most one RepoAccessError
		err = errs[0]
	}
	if err != nil {
		Stderrf("Error fetch %s: %v", version.TMID, err)
		return PullResult{PullErr, version.TMID, fmt.Sprintf("(cannot fetch from repo %s)", version.FoundIn)}, err
	}
	thing = utils.ConvertToNativeLineEndings(thing)

	finalOutput := filepath.Join(outputPath, id)

	err = os.MkdirAll(filepath.Dir(finalOutput), 0770)
	if err != nil {
		Stderrf("Could not write ThingModel to file %s: %v", finalOutput, err)
		return PullResult{PullErr, version.TMID, fmt.Sprintf("(cannot write to ouput directory %s)", outputPath)}, err
	}

	err = os.WriteFile(finalOutput, thing, 0660)
	if err != nil {
		Stderrf("Could not write ThingModel to file %s: %v", finalOutput, err)
		return PullResult{PullErr, version.TMID, fmt.Sprintf("(cannot write to ouput directory %s)", outputPath)}, err
	}

	return PullResult{PullOK, version.TMID, ""}, err
}
