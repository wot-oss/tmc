package remotes

import (
	"errors"
	"fmt"
	"log/slog"
	url2 "net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/src/toc"
)

type FileRemote struct {
	root string
}

func NewFileRemote(config map[string]any) (*FileRemote, error) {
	urlString := config["url"].(string)
	rootUrl, err := url2.Parse(urlString)
	if err != nil {
		slog.Default().Error("could not parse root URL for file remote", "url", urlString, "error", err)
		return nil, fmt.Errorf("could not parse root URL %s for file remote: %w", urlString, err)
	}
	if rootUrl.Scheme != "file" {
		slog.Default().Error("root URL for file remote must begin with file:", "url", urlString)
		return nil, fmt.Errorf("root URL for file remote must begin with file: %s", urlString)
	}
	rootPath := rootUrl.Opaque
	if strings.HasPrefix(rootPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			slog.Default().Error("cannot expand user home directory", "error", err)
			return nil, fmt.Errorf("cannot expand user home directory: %w", err)
		}
		rootPath = strings.Replace(rootPath, "~", home, 1)
	}
	return &FileRemote{
		root: rootPath,
	}, nil
}

func (f *FileRemote) Push(_ *model.ThingModel, id model.TMID, raw []byte) error {
	if len(raw) == 0 {
		return errors.New("nothing to write")
	}
	fullPath, dir := f.filenames(id)
	err := os.MkdirAll(dir, os.ModePerm) //fixme: review permissions
	if err != nil {
		return fmt.Errorf("could not create directory %s: %w", dir, err)
	}

	if found, existingId := f.getExistingID(id); found {
		slog.Default().Info(fmt.Sprintf("TM already exists under ID %v", existingId))
		return nil
	}

	err = os.WriteFile(fullPath, raw, os.ModePerm) //fixme: review permissions
	if err != nil {
		return fmt.Errorf("could not write TM to catalog: %v", err)
	}

	return nil
}

func (f *FileRemote) filenames(id model.TMID) (string, string) {
	fullPath := filepath.Join(f.root, id.String())
	dir := filepath.Dir(fullPath)
	return fullPath, dir
}

func (f *FileRemote) getExistingID(id model.TMID) (bool, model.TMID) {
	fullName, dir := f.filenames(id)
	// try full name as given
	if _, err := os.Stat(fullName); err == nil {
		return true, id
	}
	// try without timestamp
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, model.TMID{}
	}
	existingTMVersions := findTMFileEntriesByBaseVersion(entries, id.Version)
	if len(existingTMVersions) > 0 && existingTMVersions[0].Hash == id.Version.Hash {
		ret := id
		ret.Version = existingTMVersions[0]
		return true, ret
	}

	return false, model.TMID{}
}

// findTMFileEntriesByBaseVersion finds directory entries that correspond to TM file names, converts those to TMVersions,
// filters out those that have a differing base version from the one given as argument, and sorts the remaining in
// descending order
func findTMFileEntriesByBaseVersion(entries []os.DirEntry, version model.TMVersion) []model.TMVersion {
	baseString := version.BaseString()
	var res []model.TMVersion
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		ver, err := model.ParseTMVersion(strings.TrimSuffix(e.Name(), model.TMFileExtension))
		if err != nil {
			continue
		}

		if baseString == ver.BaseString() {
			res = append(res, ver)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return strings.Compare(res[i].String(), res[j].String()) > 0
	})
	return res
}

func (f *FileRemote) Fetch(id model.TMID) ([]byte, error) {
	exists, actualId := f.getExistingID(id)
	if !exists {
		return nil, os.ErrNotExist
	}
	actualFilename, _ := f.filenames(actualId)
	return os.ReadFile(actualFilename)
}

func (f *FileRemote) CreateToC() error {
	toc.Create(f.root)
	return nil
}
