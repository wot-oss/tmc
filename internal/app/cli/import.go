package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/utils"
)

type ImportExecutor struct {
	now commands.Now
}

func NewImportExecutor(now commands.Now) *ImportExecutor {
	return &ImportExecutor{
		now: now,
	}
}

// Import imports file or directory into the specified repository
// Returns the list of import results up to the first encountered error, and the error
func (p *ImportExecutor) Import(ctx context.Context, filename string, spec model.RepoSpec, optTree bool, opts repos.ImportOptions) ([]repos.ImportResult, error) {
	repo, err := repos.Get(spec)
	if err != nil {
		Stderrf("Could not initialize a repo instance for %s: %v\ncheck config", spec, err)
		return nil, err
	}

	abs, err := filepath.Abs(filename)
	if err != nil {
		Stderrf("Error expanding file name %s: %v", filename, err)
		return nil, err
	}

	stat, err := os.Stat(abs)
	if err != nil {
		Stderrf("Cannot read file or directory %s: %v", filename, err)
		return nil, err
	}

	var res []repos.ImportResult
	if stat.IsDir() {
		res, err = p.importDirectory(ctx, abs, repo, optTree, opts)
	} else {
		singleRes, impErr := p.importFile(ctx, filename, repo, opts)
		res = []repos.ImportResult{singleRes}
		err = impErr
	}
	defer func() {
		for _, r := range res {
			fmt.Println(r)
		}
	}()
	successfulIds := getSuccessfulIds(res)
	if len(successfulIds) > 0 {
		indexErr := repo.Index(ctx, successfulIds...)
		if indexErr != nil {
			Stderrf("Cannot create index: %v", indexErr)
			return res, indexErr
		}
	}
	return res, err
}

func getSuccessfulIds(res []repos.ImportResult) []string {
	var r []string
	for _, pr := range res {
		if pr.IsSuccessful() {
			r = append(r, pr.TmID)
		}
	}
	return r
}

func (p *ImportExecutor) importDirectory(ctx context.Context, absDirname string, repo repos.Repo, optTree bool, opts repos.ImportOptions) ([]repos.ImportResult, error) {
	var results []repos.ImportResult
	var tErr error
	err := filepath.WalkDir(absDirname, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		if err != nil {
			return err
		}

		fileOpts := opts
		if optTree {
			fileOpts.OptPath = filepath.ToSlash(filepath.Dir(strings.TrimPrefix(path, absDirname)))
		}

		res, err := p.importFile(ctx, path, repo, fileOpts)
		results = append(results, res)
		if tErr == nil {
			tErr = err
		}
		return nil
	})
	if err != nil {
		return results, err
	}
	return results, tErr

}

func (p *ImportExecutor) importFile(ctx context.Context, filename string, repo repos.Repo, opts repos.ImportOptions) (repos.ImportResult, error) {
	_, raw, err := utils.ReadRequiredFile(filename)
	if err != nil {
		err := fmt.Errorf("error reading file %s for import: %w", filename, err)
		Stderrf("%v", err.Error())
		return repos.ImportResultFromError(err)
	}
	res, err := commands.NewImportCommand(p.now).ImportFile(ctx, raw, repo, opts)
	if err != nil {
		var errExists *repos.ErrTMIDConflict
		if errors.As(err, &errExists) {
			res.Message = fmt.Sprintf("file %s already exists as %s", filename, errExists.ExistingId)
			if opts.IgnoreExisting {
				return res, nil
			}
			return res, err
		}
		err := fmt.Errorf("error importing file %s: %w", filename, err)
		Stderrf("%v", err.Error())
		return repos.ImportResultFromError(err)
	}
	switch res.Type {
	case repos.ImportResultWarning:
		warn := res.Message
		var cErr *repos.ErrTMIDConflict
		if errors.As(res.Err, &cErr) {
			warn = fmt.Sprintf("TM's version and timestamp clash with existing one %s", cErr.ExistingId)
		}
		msg := fmt.Sprintf("file %s imported as %s with warning: %s", filename, res.TmID, warn)
		res.Message = msg
	case repos.ImportResultOK:
		res.Message = fmt.Sprintf("file %s imported as %s", filename, res.TmID)
	default:
		err := fmt.Errorf("unexpected ImportResult type %v when importing file %s", res.Type, filename)
		Stderrf("%v", err.Error())
		return repos.ImportResultFromError(err)
	}
	return res, err
}
