package remotes

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/toc"
)

const defaultFilePermissions = os.ModePerm //fixme: review permissions

type FileRemote struct {
	root string
}

type ErrTMExists struct {
	ExistingId model.TMID
}

func (e *ErrTMExists) Error() string {
	return fmt.Sprintf("Thing Model already exists under id: %v", e.ExistingId)
}

var winExtraLeadingSlashRegex = regexp.MustCompile("/[a-zA-Z]:.*")

func NewFileRemote(config map[string]any) (*FileRemote, error) {
	urlString := config["url"].(string)
	rootUrl, err := url.Parse(urlString)
	if err != nil {
		slog.Default().Error("could not parse root URL for file remote", "url", urlString, "error", err)
		return nil, fmt.Errorf("could not parse root URL %s for file remote: %w", urlString, err)
	}
	if rootUrl.Scheme != "file" {
		slog.Default().Error("root URL for file remote must begin with 'file:'", "url", urlString)
		return nil, fmt.Errorf("root URL for file remote must begin with file: %s", urlString)
	}
	rootPath := rootUrl.Path
	if rootPath == "" {
		rootPath = rootUrl.Opaque // maybe the user just forgot a slash in the url and it's been parsed as opaque
	}
	rootPath, err = internal.ExpandHome(rootPath)
	if err != nil {
		return nil, err
	}
	//err = os.MkdirAll(rootPath, defaultFilePermissions)
	//if err != nil {
	//	return nil, err
	//}
	if winExtraLeadingSlashRegex.MatchString(rootPath) {
		rootPath = strings.TrimPrefix(rootPath, "/")
	}
	slog.Default().Info("created FileRemote", "root", rootPath)
	return &FileRemote{
		root: rootPath,
	}, nil
}

func (f *FileRemote) Push(id model.TMID, raw []byte) error {
	if len(raw) == 0 {
		return errors.New("nothing to write")
	}
	fullPath, dir := f.filenames(id)
	err := os.MkdirAll(dir, defaultFilePermissions)
	if err != nil {
		return fmt.Errorf("could not create directory %s: %w", dir, err)
	}

	if found, existingId := f.getExistingID(id); found {
		slog.Default().Info(fmt.Sprintf("TM already exists under ID %v", existingId))
		return &ErrTMExists{ExistingId: existingId}
	}

	err = os.WriteFile(fullPath, raw, defaultFilePermissions)
	if err != nil {
		return fmt.Errorf("could not write TM to catalog: %v", err)
	}
	slog.Default().Info("saved Thing Model file", "filename", fullPath)

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
	return toc.Create(f.root)
}

func (f *FileRemote) List(filter string) (model.TOC, error) {
	log := slog.Default()
	if len(filter) == 0 {
		log.Debug("Creating list")
	} else {
		log.Debug(fmt.Sprintf("Creating list with filter '%s'", filter))
	}

	data, err := os.ReadFile(filepath.Join(f.root, toc.TOCFilename))
	if err != nil {
		return model.TOC{}, errors.New("No toc found. Run `create-toc` for this remote.")
	}

	var toc model.TOC
	err = json.Unmarshal(data, &toc)
	toc.Filter(filter)
	if err != nil {
		return model.TOC{}, err
	}
	return toc, nil
}

func (f *FileRemote) Versions(name string) (model.TOCEntry, error) {
	log := slog.Default()
	if len(name) == 0 {
		log.Error("Please specify a name to show the TM.")
		return model.TOCEntry{}, errors.New("Please specify a name to show the TM")
	}
	toc, err := f.List("")
	if err != nil {
		return model.TOCEntry{}, err
	}
	name = strings.TrimSpace(name)

	tocThing := toc.FindByName(name)
	if tocThing == nil {
		msg := fmt.Sprintf("No thing model found for name: %s", name)
		log.Error(msg)
		return model.TOCEntry{}, errors.New(msg)
	}

	return *tocThing, nil
}
