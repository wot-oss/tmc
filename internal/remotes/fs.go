package remotes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

const (
	defaultDirPermissions  = 0775
	defaultFilePermissions = 0664
	tocLockTimeout         = 5 * time.Second
	tocLocRetryDelay       = 13 * time.Millisecond
	errTmExistsPrefix      = "Thing Model already exists under id: "

	TMExt = ".tm.json"
)

var ErrRootInvalid = errors.New("root is not a directory")
var ErrTocLocked = errors.New("could not acquire lock on TOC file")

var osStat = os.Stat         // mockable for testing
var osReadFile = os.ReadFile // mockable for testing

// FileRemote implements a Remote TM repository backed by a file system
type FileRemote struct {
	root string
	spec RepoSpec
}

type ErrTMExists struct {
	ExistingId string
}

func (e *ErrTMExists) Error() string {
	return fmt.Sprintf(errTmExistsPrefix+"%v", e.ExistingId)
}

func (e *ErrTMExists) FromString(s string) {
	id, _ := strings.CutPrefix(s, errTmExistsPrefix)
	e.ExistingId = id
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

	err = utils.AtomicWriteFile(fullPath, raw, defaultFilePermissions)
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
	version, err := model.ParseTMVersion(strings.TrimSuffix(base, TMExt))
	if err != nil {
		slog.Default().Error("invalid TM version in TM id", "id", ids, "error", err)
		return false, ""
	}
	existingTMVersions := findTMFileEntriesByBaseVersion(entries, version)
	if len(existingTMVersions) > 0 && existingTMVersions[0].Hash == version.Hash {
		return true,
			strings.TrimSuffix(ids, base) + existingTMVersions[0].String() + TMExt
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

		ver, err := model.ParseTMVersion(strings.TrimSuffix(e.Name(), TMExt))
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
		return "", nil, ErrTmNotFound
	}
	actualFilename, _, _ := f.filenames(actualId)
	b, err := os.ReadFile(actualFilename)
	return actualId, b, err
}

func (f *FileRemote) UpdateToc(ids ...string) error {
	err := f.checkRootValid()
	if err != nil {
		return err
	}
	return f.updateToc(ids)
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

	unlock, err := f.lockTOC()
	defer unlock()
	if err != nil {
		return model.SearchResult{}, err
	}

	toc, err := f.readTOC()
	if err != nil {
		return model.SearchResult{}, err
	}
	toc.Filter(search)
	return model.NewTOCToFoundMapper(f.Spec().ToFoundSource()).ToSearchResult(toc), nil
}

// readToc reads the contents of the TOC file. Must be called after the lock is acquired with lockToc()
func (f *FileRemote) readTOC() (model.TOC, error) {
	data, err := os.ReadFile(f.tocFilename())
	if err != nil {
		return model.TOC{}, errors.New("no table of contents found. Run `update-toc` for this remote")
	}

	var toc model.TOC
	err = json.Unmarshal(data, &toc)
	return toc, err
}

func (f *FileRemote) tocFilename() string {
	return filepath.Join(f.root, RepoConfDir, TOCFilename)
}

func (f *FileRemote) Versions(name string) ([]model.FoundVersion, error) {
	log := slog.Default()
	name = strings.TrimSpace(name)
	toc, err := f.List(&model.SearchParams{Name: name})
	if err != nil {
		return nil, err
	}

	if len(toc.Entries) != 1 {
		err := fmt.Errorf("%w: %s", ErrTmNotFound, name)
		log.Error(err.Error())
		return nil, err
	}

	return toc.Entries[0].Versions, nil
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

func (f *FileRemote) updateToc(ids []string) error {
	// Prepare data collection for logging stats
	var log = slog.Default()
	fileCount := 0
	start := time.Now()

	cancel, err := f.lockTOC()
	defer cancel()
	if err != nil {
		return err
	}

	var newTOC *model.TOC
	names := f.readNamesFile()

	if len(ids) == 0 { // full rebuild
		newTOC = &model.TOC{
			Meta: model.TOCMeta{Created: time.Now()},
			Data: []*model.TOCEntry{},
		}
		err := filepath.Walk(f.root, func(path string, info os.FileInfo, err error) error {
			upd, name, err := f.updateTocWithFile(newTOC, path, info, log, err)
			if err != nil {
				return err
			}
			if upd {
				fileCount++
				names = append(names, name)
			}
			return nil
		})
		if err != nil {
			return err
		}

	} else { // partial update
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
			info, err := osStat(path)
			upd, name, _ := f.updateTocWithFile(newTOC, path, info, log, err)
			if upd {
				fileCount++
				names = append(names, name)
			}
		}
	}
	duration := time.Now().Sub(start)
	// Ignore error as we are sure our struct does not contain channel,
	// complex or function values that would throw an error.
	newTOCJson, _ := json.MarshalIndent(newTOC, "", "  ")
	err = utils.AtomicWriteFile(f.tocFilename(), newTOCJson, defaultFilePermissions)
	if err != nil {
		return err
	}
	err = f.writeNamesFile(names)
	if err != nil {
		return err
	}
	msg := "Created table of content with %d entries in %s "
	msg = fmt.Sprintf(msg, fileCount, duration.String())
	log.Info(msg)
	return nil
}

func (f *FileRemote) updateTocWithFile(newTOC *model.TOC, path string, info os.FileInfo, log *slog.Logger, err error) (bool, string, error) {
	if err != nil {
		return false, "", err
	}
	if info.IsDir() || !strings.HasSuffix(info.Name(), TMExt) {
		return false, "", nil
	}
	thingMeta, err := getThingMetadata(path)
	if err != nil {
		msg := "Failed to extract metadata from file %s with error:"
		msg = fmt.Sprintf(msg, path)
		log.Error(msg)
		log.Error(err.Error())
		log.Error("The file will be excluded from the table of contents.")
		return false, "", nil
	}
	tmid, err := newTOC.Insert(&thingMeta)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to insert %s into toc:", path))
		log.Error(err.Error())
		log.Error("The file will be excluded from the table of contents.")
		return false, "", nil
	}
	return true, tmid.Name, nil
}

type unlockFunc func()

func (f *FileRemote) lockTOC() (unlockFunc, error) {
	rd := filepath.Join(f.root, RepoConfDir)
	stat, err := os.Stat(rd)
	if err != nil || !stat.IsDir() {
		err := os.MkdirAll(rd, defaultDirPermissions)
		if err != nil {
			return func() {}, err
		}
	}
	tocFile := f.tocFilename()

	fl := flock.New(tocFile + ".lock")
	ctx, cancel := context.WithTimeout(context.Background(), tocLockTimeout)
	unlock := func() {
		cancel()
		_ = fl.Unlock()
	}
	locked, err := fl.TryLockContext(ctx, tocLocRetryDelay)
	if err != nil {
		return unlock, err
	}
	if !locked {
		return unlock, ErrTocLocked
	}

	f.moveOldToc(tocFile)

	return unlock, nil
}

// moveOldToc attempts to move the TOC file at root to .tmc folder and remove old lock file
// ignores all errors
func (f *FileRemote) moveOldToc(tocFile string) {
	oldToc := filepath.Join(f.root, TOCFilename)
	_, errOld := os.Stat(oldToc)
	_, errNew := os.Stat(tocFile)
	if errOld == nil && errNew != nil {
		_ = os.Rename(oldToc, tocFile)
	}
	_ = os.Remove(oldToc + ".lock")
}

func (f *FileRemote) readNamesFile() []string {
	lines, _ := utils.ReadFileLines(filepath.Join(f.root, RepoConfDir, TmNamesFile))
	return lines
}
func (f *FileRemote) writeNamesFile(names []string) error {
	slices.Sort(names)
	names = slices.Compact(names)
	return utils.WriteFileLines(names, filepath.Join(f.root, RepoConfDir, TmNamesFile), defaultFilePermissions)
}

func getThingMetadata(path string) (model.ThingModel, error) {
	data, err := osReadFile(path)
	if err != nil {
		return model.ThingModel{}, err
	}

	var ctm model.ThingModel
	err = json.Unmarshal(data, &ctm)
	if err != nil {
		return model.ThingModel{}, err
	}

	return ctm, nil
}

func (f *FileRemote) ListCompletions(kind string, toComplete string) ([]string, error) {
	switch kind {
	case CompletionKindNames:
		unlock, err := f.lockTOC()
		defer unlock()
		if err != nil {
			return nil, err
		}
		return f.readNamesFile(), nil
	case CompletionKindFetchNames:
		if strings.Contains(toComplete, "..") {
			return nil, fmt.Errorf("%w :no completions for name containing '..'", ErrInvalidCompletionParams)
		}

		name, _, _ := strings.Cut(toComplete, ":")

		dir := filepath.Join(f.root, name)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		vm := make(map[string]struct{})
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), TMExt) {
				ver, err := model.ParseTMVersion(strings.TrimSuffix(e.Name(), TMExt))
				if err != nil {
					continue
				}
				vm[ver.BaseString()] = struct{}{}
			}
		}
		var vs []string
		for v, _ := range vm {
			vs = append(vs, fmt.Sprintf("%s:%s", name, v))
		}
		slices.Sort(vs)
		return vs, nil

	default:
		return nil, ErrInvalidCompletionParams
	}
}
