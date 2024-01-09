package remotes

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

const defaultDirPermissions = 0775
const defaultFilePermissions = 0664

type FileRemote struct {
	root string
	spec RepoSpec
}

type ErrTMExists struct {
	ExistingId string
}

var ErrRootInvalid = errors.New("root is not a directory")

func (e *ErrTMExists) Error() string {
	return fmt.Sprintf("Thing Model already exists under id: %v", e.ExistingId)
}

func NewFileRemote(config map[string]any, spec RepoSpec) (*FileRemote, error) {
	loc := utils.JsGetString(config, KeyRemoteLoc)
	if loc == nil {
		return nil, fmt.Errorf("invalid file remote config. loc is either not found or not a string")
	}
	rootPath, err := utils.ExpandHome(*loc)
	if err != nil {
		return nil, err
	}
	return &FileRemote{
		root: rootPath,
		spec: spec,
	}, nil
}

func (f *FileRemote) Push(id model.TMID, raw []byte) error {
	if len(raw) == 0 {
		return errors.New("nothing to write")
	}
	fullPath, dir, _ := f.filenames(id.String())
	err := os.MkdirAll(dir, defaultDirPermissions)
	if err != nil {
		return fmt.Errorf("could not create directory %s: %w", dir, err)
	}

	if found, existingId := f.getExistingID(id.String()); found {
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

func (f *FileRemote) filenames(id string) (string, string, string) {
	fullPath := filepath.Join(f.root, id)
	dir := filepath.Dir(fullPath)
	base := filepath.Base(fullPath)
	return fullPath, dir, base
}

func (f *FileRemote) getExistingID(ids string) (bool, string) {
	fullName, dir, base := f.filenames(ids)
	// try full remoteName as given
	if _, err := os.Stat(fullName); err == nil {
		return true, ids
	}
	// try without timestamp
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, ""
	}
	version, err := model.ParseTMVersion(strings.TrimSuffix(base, model.TMFileExtension))
	if err != nil {
		slog.Default().Error("invalid TM version in TM id", "id", ids, "error", err)
		return false, ""
	}
	existingTMVersions := findTMFileEntriesByBaseVersion(entries, version)
	if len(existingTMVersions) > 0 && existingTMVersions[0].Hash == version.Hash {
		return true,
			strings.TrimSuffix(ids, base) + existingTMVersions[0].String() + model.TMFileExtension
	}

	return false, ""
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

func (f *FileRemote) Fetch(id string) (string, []byte, error) {
	err := f.checkRootValid()
	if err != nil {
		return "", nil, err
	}
	exists, actualId := f.getExistingID(id)
	if !exists {
		return "", nil, os.ErrNotExist
	}
	actualFilename, _, _ := f.filenames(actualId)
	b, err := os.ReadFile(actualFilename)
	return actualId, b, err
}

func (f *FileRemote) CreateToC(ids ...string) error {
	err := f.checkRootValid()
	if err != nil {
		return err
	}
	return f.createTOC(ids)
}

func (f *FileRemote) Spec() RepoSpec {
	return f.spec
}

func (f *FileRemote) List(search *model.SearchParams) (model.SearchResult, error) {
	log := slog.Default()
	log.Debug(fmt.Sprintf("Creating list with filter '%v'", search))

	err := f.checkRootValid()
	if err != nil {
		return model.SearchResult{}, err
	}

	toc, err := f.readTOC()
	if err != nil {
		return model.SearchResult{}, err
	}
	toc.Filter(search)
	return model.NewSearchResultFromTOC(toc, f.Spec().ToFoundSource()), nil
}

func (f *FileRemote) readTOC() (model.TOC, error) {
	data, err := os.ReadFile(filepath.Join(f.root, TOCFilename))
	if err != nil {
		return model.TOC{}, errors.New("no table of contents found. Run `create-toc` for this remote")
	}

	var toc model.TOC
	err = json.Unmarshal(data, &toc)
	return toc, err
}

func (f *FileRemote) Versions(name string) (model.FoundEntry, error) {
	log := slog.Default()
	if len(name) == 0 {
		log.Error("Please specify a remoteName to show the TM.")
		return model.FoundEntry{}, errors.New("please specify a remoteName to show the TM")
	}
	name = strings.TrimSpace(name)
	toc, err := f.List(&model.SearchParams{Name: name})
	if err != nil {
		return model.FoundEntry{}, err
	}

	if len(toc.Entries) != 1 {
		log.Error(fmt.Sprintf("No thing model found for remoteName: %s", name))
		return model.FoundEntry{}, ErrEntryNotFound
	}

	return toc.Entries[0], nil
}

func (f *FileRemote) checkRootValid() error {
	stat, err := os.Stat(f.root)
	if err != nil || !stat.IsDir() {
		return fmt.Errorf("%s: %w", f.Spec(), ErrRootInvalid)
	}
	return nil
}

func createFileRemoteConfig(dirName string, bytes []byte) (map[string]any, error) {
	if dirName != "" {
		absDir, err := makeAbs(dirName)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			KeyRemoteType: RemoteTypeFile,
			KeyRemoteLoc:  absDir,
		}, nil
	} else {
		rc, err := AsRemoteConfig(bytes)
		if err != nil {
			return nil, err
		}
		if rType := utils.JsGetString(rc, KeyRemoteType); rType != nil {
			if *rType != RemoteTypeFile {
				return nil, fmt.Errorf("invalid json config. type must be \"file\" or absent")
			}
		}
		rc[KeyRemoteType] = RemoteTypeFile
		l := utils.JsGetString(rc, KeyRemoteLoc)
		if l == nil {
			return nil, fmt.Errorf("invalid json config. must have string \"loc\"")
		}
		la, err := makeAbs(*l)
		if err != nil {
			return nil, err
		}
		rc[KeyRemoteLoc] = la
		return rc, nil
	}
}

func makeAbs(dir string) (string, error) {
	if filepath.IsAbs(dir) {
		return dir, nil
	} else {
		if !strings.HasPrefix(dir, "~") {
			var err error
			dir, err = filepath.Abs(dir)
			if err != nil {
				return "", err
			}
		}
		return dir, nil
	}
}

const TMExt = ".tm.json"
const TOCFilename = "tm-catalog.toc.json"

func (f *FileRemote) createTOC(ids []string) error {
	// Prepare data collection for logging stats
	var log = slog.Default()
	fileCount := 0
	start := time.Now()

	var newTOC *model.TOC

	if len(ids) == 0 { // full rebuild
		newTOC = &model.TOC{
			Meta: model.TOCMeta{Created: time.Now()},
			Data: []*model.TOCEntry{},
		}
		err := filepath.Walk(f.root, func(path string, info os.FileInfo, err error) error {
			upd, err := updateTocWithFile(newTOC, path, info, log, err)
			if err != nil {
				return err
			}
			if upd {
				fileCount++
			}
			return nil
		})
		if err != nil {
			return err
		}

	} else {
		toc, err := f.readTOC()
		if err != nil {
			newTOC = &model.TOC{
				Meta: model.TOCMeta{Created: time.Now()},
				Data: []*model.TOCEntry{},
			}
		} else {
			newTOC = &toc
		}
		for _, id := range ids {
			path := filepath.Join(f.root, id)
			info, err := os.Stat(path)
			upd, _ := updateTocWithFile(newTOC, path, info, log, err)
			if upd {
				fileCount++
			}
		}
	}
	duration := time.Now().Sub(start)
	// Ignore error as we are sure our struct does not contain channel,
	// complex or function values that would throw an error.
	newTOCJson, _ := json.MarshalIndent(newTOC, "", "  ")
	err := os.WriteFile(filepath.Join(f.root, TOCFilename), newTOCJson, defaultFilePermissions)
	if err != nil {
		return err
	}
	msg := "Created table of content with %d entries in %s "
	msg = fmt.Sprintf(msg, fileCount, duration.String())
	log.Info(msg)
	return nil
}

func updateTocWithFile(newTOC *model.TOC, path string, info os.FileInfo, log *slog.Logger, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	if info.IsDir() || !strings.HasSuffix(info.Name(), TMExt) {
		return false, nil
	}
	thingMeta, err := getThingMetadata(path)
	if err != nil {
		msg := "Failed to extract metadata from file %s with error:"
		msg = fmt.Sprintf(msg, path)
		log.Error(msg)
		log.Error(err.Error())
		log.Error("The file will be excluded from the table of contents.")
		return false, nil
	}
	err = newTOC.Insert(&thingMeta)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to insert %s into toc:", path))
		log.Error(err.Error())
		log.Error("The file will be excluded from the table of contents.")
		return false, nil
	}
	return true, nil
}

func getThingMetadata(path string) (model.CatalogThingModel, error) {
	// TODO: should internal.ReadRequiredFiles be used here?
	data, err := os.ReadFile(path)
	if err != nil {
		return model.CatalogThingModel{}, err
	}

	var ctm model.CatalogThingModel
	err = json.Unmarshal(data, &ctm)
	if err != nil {
		return model.CatalogThingModel{}, err
	}

	return ctm, nil
}
