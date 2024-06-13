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
		Stderrf("Could not ìnitialize a repo instance for %s: %v\ncheck config", spec, err)
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
		return err
	})

	return results, err

}

func (p *ImportExecutor) importFile(ctx context.Context, filename string, repo repos.Repo, opts repos.ImportOptions) (repos.ImportResult, error) {
	_, raw, err := utils.ReadRequiredFile(filename)
	if err != nil {
		Stderrf("Couldn't read file %s: %v", filename, err)
		return repos.ImportResult{}, fmt.Errorf("error importing file %s: %w", filename, err)
	}
	res, err := commands.NewImportCommand(p.now).ImportFile(ctx, raw, repo, opts)
	if err != nil {
		var errExists *repos.ErrTMIDConflict
		if errors.As(err, &errExists) {
			return repos.ImportResult{Type: repos.ImportResultTMExists, Message: fmt.Sprintf("file %s already exists as %s", filename, errExists.ExistingId), Err: errExists}, nil
		}
		err := fmt.Errorf("error importing file %s: %w", filename, err)
		return res, err
	}
	switch res.Type {
	case repos.ImportResultWarning:
		res.Message = fmt.Sprintf("file %s imported as %s. TM's version and timestamp clash with existing one %s", filename, res.TmID, res.Err.ExistingId)
	case repos.ImportResultOK:
		res.Message = fmt.Sprintf("file %s imported as %s", filename, res.TmID)
	default:
		return res, fmt.Errorf("unexpected ImportResult type: %v", res.Type)
	}
	return res, err
}
