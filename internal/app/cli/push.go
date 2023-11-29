package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
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
}

func (r PushResult) String() string {
	return fmt.Sprintf("%v\t %s", r.typ, r.text)
}

// Push pushes file or directory to remote repository
// Returns the list of push results up to the first encountered error, and the error
func Push(filename, remoteName, optPath string, optTree bool) ([]PushResult, error) {
	remote, err := remotes.Get(remoteName)
	if err != nil {
		Stderrf("Could not ìnitialize a remote instance for %s: %v\ncheck config", remoteName, err)
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

	if stat.IsDir() {
		return pushDirectory(abs, remote, optPath, optTree)
	} else {
		res, err := pushFile(filename, remote, optPath)
		return []PushResult{res}, err
	}
}

func pushDirectory(absDirname string, remote remotes.Remote, optPath string, optTree bool) ([]PushResult, error) {
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

		res, err := pushFile(path, remote, optPath)
		results = append(results, res)
		return err
	})

	return results, err

}

func pushFile(filename string, remote remotes.Remote, optPath string) (PushResult, error) {
	_, raw, err := utils.ReadRequiredFile(filename)
	if err != nil {
		Stderrf("Couldn't read file %s: %v", filename, err)
		return PushResult{PushErr, fmt.Sprintf("error pushing file %s: %s", filename, err.Error())}, err
	}
	id, err := commands.PushFile(raw, remote, optPath)
	if err != nil {
		var errExists *remotes.ErrTMExists
		if errors.As(err, &errExists) {
			return PushResult{TMExists, fmt.Sprintf("file %s already exists as %s", filename, id.String())}, nil
		}
		return PushResult{PushErr, fmt.Sprintf("error pushing file %s: %s", filename, err.Error())}, err
	}

	return PushResult{PushOK, fmt.Sprintf("file %s pushed as %s", filename, id.String())}, nil
}
