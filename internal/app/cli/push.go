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

type PushResultType int

const (
	PushOK = PushResultType(iota)
	TMExists
	PushErr
)

func (t PushResultType) String() string {
	switch t {
	case PushOK:
		return "OK"
	case TMExists:
		return "exists"
	case PushErr:
		return "error"
	default:
		return "unknown"
	}
}

type PushResult struct {
	typ  PushResultType
	text string
	tmid string
}

func (r PushResult) String() string {
	return fmt.Sprintf("%v\t %s", r.typ, r.text)
}

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
func (p *PushExecutor) Push(ctx context.Context, filename string, spec model.RepoSpec, optPath string, optTree bool) ([]PushResult, error) {
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

	var res []PushResult
	if stat.IsDir() {
		res, err = p.pushDirectory(ctx, abs, repo, optPath, optTree)
	} else {
		singleRes, pushErr := p.pushFile(ctx, filename, repo, optPath)
		res = []PushResult{singleRes}
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

func getOkIds(res []PushResult) []string {
	var r []string
	for _, pr := range res {
		if pr.typ == PushOK {
			r = append(r, pr.tmid)
		}
	}
	return r
}

func (p *PushExecutor) pushDirectory(ctx context.Context, absDirname string, repo repos.Repo, optPath string, optTree bool) ([]PushResult, error) {
	var results []PushResult
	err := filepath.WalkDir(absDirname, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		if err != nil {
			return err
		}

		if optTree {
			optPath = filepath.Dir(strings.TrimPrefix(path, absDirname))
		}

		res, err := p.pushFile(ctx, path, repo, optPath)
		results = append(results, res)
		return err
	})

	return results, err

}

func (p *PushExecutor) pushFile(ctx context.Context, filename string, repo repos.Repo, optPath string) (PushResult, error) {
	_, raw, err := utils.ReadRequiredFile(filename)
	if err != nil {
		Stderrf("Couldn't read file %s: %v", filename, err)
		return PushResult{PushErr, fmt.Sprintf("error pushing file %s: %s", filename, err.Error()), ""}, err
	}
	id, err := commands.NewPushCommand(p.now).PushFile(ctx, raw, repo, optPath)
	if err != nil {
		var errExists *repos.ErrTMIDConflict
		if errors.As(err, &errExists) {
			return PushResult{TMExists, fmt.Sprintf("file %s already exists as %s", filename, errExists.ExistingId), errExists.ExistingId}, nil
		}
		return PushResult{PushErr, fmt.Sprintf("error pushing file %s: %s", filename, err.Error()), id}, err
	}

	return PushResult{PushOK, fmt.Sprintf("file %s pushed as %s", filename, id), id}, nil
}
