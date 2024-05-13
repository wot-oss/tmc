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

type PushExecutor struct {
	now commands.Now
}

func NewPushExecutor(now commands.Now) *PushExecutor {
	return &PushExecutor{
		now: now,
	}
}

// Push pushes file or directory to the specified repository
// Returns the list of push results up to the first encountered error, and the error
func (p *PushExecutor) Push(ctx context.Context, filename string, spec model.RepoSpec, optTree bool, opts repos.PushOptions) ([]repos.PushResult, error) {
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

	var res []repos.PushResult
	if stat.IsDir() {
		res, err = p.pushDirectory(ctx, abs, repo, optTree, opts)
	} else {
		singleRes, pushErr := p.pushFile(ctx, filename, repo, opts)
		res = []repos.PushResult{singleRes}
		err = pushErr
	}

	okIds := getOkIds(res)
	if len(okIds) > 0 {
		indexErr := repo.Index(ctx, okIds...)
		if indexErr != nil {
			Stderrf("Cannot create index: %v", indexErr)
			return res, indexErr
		}
	}
	return res, err
}

func getOkIds(res []repos.PushResult) []string {
	var r []string
	for _, pr := range res {
		if pr.Type == repos.PushResultOK {
			r = append(r, pr.TmID)
		}
	}
	return r
}

func (p *PushExecutor) pushDirectory(ctx context.Context, absDirname string, repo repos.Repo, optTree bool, opts repos.PushOptions) ([]repos.PushResult, error) {
	var results []repos.PushResult
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

		res, err := p.pushFile(ctx, path, repo, fileOpts)
		results = append(results, res)
		return err
	})

	return results, err

}

func (p *PushExecutor) pushFile(ctx context.Context, filename string, repo repos.Repo, opts repos.PushOptions) (repos.PushResult, error) {
	_, raw, err := utils.ReadRequiredFile(filename)
	if err != nil {
		Stderrf("Couldn't read file %s: %v", filename, err)
		return repos.NewErrorPushResult(fmt.Errorf("error pushing file %s: %w", filename, err))
	}
	res, err := commands.NewPushCommand(p.now).PushFile(ctx, raw, repo, opts)
	if err != nil {
		var errExists *repos.ErrTMIDConflict
		if errors.As(err, &errExists) {
			return repos.PushResult{repos.PushResultTMExists, fmt.Sprintf("file %s already exists as %s", filename, errExists.ExistingId), errExists.ExistingId}, nil
		}
		err := fmt.Errorf("error pushing file %s: %w", filename, err)
		return repos.NewErrorPushResult(err)
	}

	return repos.PushResult{repos.PushResultOK, fmt.Sprintf("file %s pushed as %s", filename, res.TmID), res.TmID}, nil
}
