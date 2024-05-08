package repos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

const (
	defaultDirPermissions  = 0775
	defaultFilePermissions = 0664
	indexLockTimeout       = 5 * time.Second
	indexLocRetryDelay     = 13 * time.Millisecond
	errTmExistsPrefix      = "Thing Model already exists under id: "
	TMExt                  = ".tm.json"
)

var ErrRootInvalid = errors.New("root is not a directory")
var ErrIndexLocked = errors.New("could not acquire lock on index file")
var osStat = os.Stat         // mockable for testing
var osReadFile = os.ReadFile // mockable for testing

// FileRepo implements a Repo TM repository backed by a file system
type FileRepo struct {
	root string
	spec model.RepoSpec
}

func NewFileRepo(config map[string]any, spec model.RepoSpec) (*FileRepo, error) {
	loc := utils.JsGetString(config, KeyRepoLoc)
	if loc == nil {
		return nil, fmt.Errorf("invalid file repo config. loc is either not found or not a string")
	}
	rootPath, err := utils.ExpandHome(*loc)
	if err != nil {
		return nil, err
	}
	return &FileRepo{
		root: rootPath,
		spec: spec,
	}, nil
}

func (f *FileRepo) Push(ctx context.Context, id model.TMID, raw []byte) error {
	if len(raw) == 0 {
		return errors.New("nothing to write")
	}
	idS := id.String()
	fullPath, dir, _ := f.filenames(idS)
	err := os.MkdirAll(dir, defaultDirPermissions)
	if err != nil {
		return fmt.Errorf("could not create directory %s: %w", dir, err)
	}

	match, existingId := f.getExistingID(idS)
	switch match {
	case idMatchDigest:
		slog.Default().Info(fmt.Sprintf("Same TM content already exists under ID %v", existingId))
		return &ErrTMIDConflict{Type: IdConflictSameContent, ExistingId: existingId}
	case idMatchTimestamp:
		slog.Default().Info(fmt.Sprintf("Version and timestamp clash with existing %v", existingId))
		return &ErrTMIDConflict{Type: IdConflictSameTimestamp, ExistingId: existingId}
	}

	err = utils.AtomicWriteFile(fullPath, raw, defaultFilePermissions)
	if err != nil {
		return fmt.Errorf("could not write TM to catalog: %v", err)
	}
	slog.Default().Info("saved Thing Model file", "filename", fullPath)

	return nil
}

func (f *FileRepo) Delete(ctx context.Context, id string) error {
	err := f.checkRootValid()
	if err != nil {
		return err
	}
	err = checkIdValid(id)
	if err != nil {
		return err
	}
	match, _ := f.getExistingID(id)
	if match != idMatchFull {
		return ErrTmNotFound
	}
	fullFilename, dir, _ := f.filenames(id)
	err = os.Remove(fullFilename)
	if os.IsNotExist(err) {
		return ErrTmNotFound
	}
	_ = rmEmptyDirs(dir, f.root)
	return err
}

func rmEmptyDirs(from string, upTo string) error {
	from, errF := filepath.Abs(from)
	upTo, errU := filepath.Abs(upTo)
	if errF != nil {
		slog.Default().Error("from path cannot be converted to absolute path", "error", errF)
		return errF
	} else if errU != nil {
		slog.Default().Error("upTo path cannot be converted to absolute path", "error", errU)
		return errU
	} else if !strings.HasPrefix(from, upTo) {
		err := errors.New("from path is not below upTo")
		slog.Default().Error("error removing empty dirs", "error", err)
		return err
	}

	for len(from) > len(upTo) {
		entries, err := os.ReadDir(from)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			err = os.Remove(from)
			if err != nil {
				return err
			}
		}
		from = filepath.Dir(from)
	}
	return nil
}

func (f *FileRepo) filenames(id string) (string, string, string) {
	fullPath := filepath.Join(f.root, id)
	dir := filepath.Dir(fullPath)
	base := filepath.Base(fullPath)
	return fullPath, dir, base
}

type idMatch int

const (
	idMatchNone = iota
	idMatchFull
	idMatchDigest    // semver and digest match
	idMatchTimestamp // semver and timestamp match
)

func (f *FileRepo) getExistingID(ids string) (idMatch, string) {
	fullName, dir, base := f.filenames(ids)
	// try full repoName as given
	if _, err := os.Stat(fullName); err == nil {
		return idMatchFull, ids
	}
	// try without timestamp
	entries, err := os.ReadDir(dir)
	if err != nil {
		return idMatchNone, ""
	}
	version, err := model.ParseTMVersion(strings.TrimSuffix(base, TMExt))
	if err != nil {
		slog.Default().Error("invalid TM version in TM id", "id", ids, "error", err)
		return idMatchNone, ""
	}
	existingTMVersions := findTMFileEntriesByBaseVersion(entries, version)
	if len(existingTMVersions) > 0 {
		if existingTMVersions[0].Hash == version.Hash {
			return idMatchDigest,
				strings.TrimSuffix(ids, base) + existingTMVersions[0].String() + model.TMFileExtension
		} else {
			for _, v := range existingTMVersions {
				if version.Timestamp == v.Timestamp {
					return idMatchTimestamp, strings.TrimSuffix(ids, base) + v.String() + TMExt
				}
			}
		}
	}

	return idMatchNone, ""
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

func (f *FileRepo) Fetch(ctx context.Context, id string) (string, []byte, error) {
	err := f.checkRootValid()
	if err != nil {
		return "", nil, err
	}
	err = checkIdValid(id)
	if err != nil {
		return "", nil, err
	}
	match, actualId := f.getExistingID(id)
	if match != idMatchFull && match != idMatchDigest {
		return "", nil, ErrTmNotFound
	}
	actualFilename, _, _ := f.filenames(actualId)
	b, err := os.ReadFile(actualFilename)
	return actualId, b, err
}

func checkIdValid(id string) error {
	_, err := model.ParseTMID(id)
	return err
}

func (f *FileRepo) Index(ctx context.Context, ids ...string) error {
	err := f.checkRootValid()
	if err != nil {
		return err
	}
	_, err = f.updateIndex(ctx, ids, true)
	return err
}

func (f *FileRepo) AnalyzeIndex(ctx context.Context) error {
	err := f.checkRootValid()
	if err != nil {
		return err
	}

	idxOld, err := f.readIndex()
	if err != nil {
		return err
	}

	idxNew, err := f.updateIndex(ctx, []string{}, false)
	if err != nil {
		return err
	}

	slices.SortFunc(idxOld.Data, func(a *model.IndexEntry, b *model.IndexEntry) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(idxNew.Data, func(a *model.IndexEntry, b *model.IndexEntry) int {
		return strings.Compare(a.Name, b.Name)
	})

	idxOk := reflect.DeepEqual(idxOld.Data, idxNew.Data)
	if !idxOk {
		return ErrIndexMismatch
	}
	return nil
}

func (f *FileRepo) Spec() model.RepoSpec {
	return f.spec
}

func (f *FileRepo) List(ctx context.Context, search *model.SearchParams) (model.SearchResult, error) {
	log := slog.Default()
	log.Debug(fmt.Sprintf("Creating list with filter '%v'", search))

	err := f.checkRootValid()
	if err != nil {
		return model.SearchResult{}, err
	}

	unlock, err := f.lockIndex(ctx)
	defer unlock()
	if err != nil {
		return model.SearchResult{}, err
	}

	idx, err := f.readIndex()
	if err != nil {
		return model.SearchResult{}, err
	}
	idx.Filter(search)
	return model.NewIndexToFoundMapper(f.Spec().ToFoundSource()).ToSearchResult(idx), nil
}

// readIndex reads the contents of the index file. Must be called after the lock is acquired with lockIndex()
func (f *FileRepo) readIndex() (model.Index, error) {
	data, err := os.ReadFile(f.indexFilename())
	if err != nil {
		return model.Index{}, errors.New("no table of contents found. Run `index` for this repo")
	}

	var index model.Index
	err = json.Unmarshal(data, &index)
	return index, err
}

func (f *FileRepo) indexFilename() string {
	return filepath.Join(f.root, RepoConfDir, IndexFilename)
}

func (f *FileRepo) Versions(ctx context.Context, name string) ([]model.FoundVersion, error) {
	log := slog.Default()
	name = strings.TrimSpace(name)
	res, err := f.List(ctx, &model.SearchParams{Name: name})
	if err != nil {
		return nil, err
	}

	if len(res.Entries) != 1 {
		err := fmt.Errorf("%w: %s", ErrTmNotFound, name)
		log.Error(err.Error())
		return nil, err
	}

	return res.Entries[0].Versions, nil
}

func (f *FileRepo) checkRootValid() error {
	stat, err := os.Stat(f.root)
	if err != nil || !stat.IsDir() {
		return fmt.Errorf("%s: %w", f.Spec(), ErrRootInvalid)
	}
	return nil
}

func createFileRepoConfig(dirName string, bytes []byte) (map[string]any, error) {
	if dirName != "" {
		absDir, err := makeAbs(dirName)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			KeyRepoType: RepoTypeFile,
			KeyRepoLoc:  absDir,
		}, nil
	} else {
		rc, err := AsRepoConfig(bytes)
		if err != nil {
			return nil, err
		}
		if rType := utils.JsGetString(rc, KeyRepoType); rType != nil {
			if *rType != RepoTypeFile {
				return nil, fmt.Errorf("invalid json config. type must be \"file\" or absent")
			}
		}
		rc[KeyRepoType] = RepoTypeFile
		l := utils.JsGetString(rc, KeyRepoLoc)
		if l == nil {
			return nil, fmt.Errorf("invalid json config. must have string \"loc\"")
		}
		la, err := makeAbs(*l)
		if err != nil {
			return nil, err
		}
		rc[KeyRepoLoc] = la
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

func (f *FileRepo) updateIndex(ctx context.Context, ids []string, persist bool) (*model.Index, error) {
	// Prepare data collection for logging stats
	var log = slog.Default()
	fileCount := 0
	start := time.Now()

	cancel, err := f.lockIndex(ctx)
	defer cancel()
	if err != nil {
		return nil, err
	}

	var newIndex *model.Index
	names := f.readNamesFile()

	if len(ids) == 0 { // full rebuild
		newIndex = &model.Index{
			Meta: model.IndexMeta{Created: time.Now()},
			Data: []*model.IndexEntry{},
		}
		names = nil
		err := filepath.Walk(f.root, func(path string, info os.FileInfo, err error) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			upd, name, _, err := f.updateIndexWithFile(newIndex, path, info, log, err)
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
			return nil, err
		}

	} else { // partial update
		index, err := f.readIndex()
		if err != nil {
			newIndex = &model.Index{
				Meta: model.IndexMeta{Created: time.Now()},
				Data: []*model.IndexEntry{},
			}
		} else {
			newIndex = &index
		}
		for _, id := range ids {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			path := filepath.Join(f.root, id)
			info, statErr := osStat(path)
			upd, name, nameDeleted, err := f.updateIndexWithFile(newIndex, path, info, log, statErr)
			if err != nil {
				return nil, err
			}
			if upd {
				fileCount++
				if nameDeleted != "" {
					names = slices.DeleteFunc(names, func(s string) bool {
						return s == nameDeleted
					})
				} else if name != "" {
					names = append(names, name)
				}
			}
		}
	}
	duration := time.Now().Sub(start)
	// Ignore error as we are sure our struct does not contain channel,
	// complex or function values that would throw an error.
	newIndexJson, _ := json.MarshalIndent(newIndex, "", "  ")
	if persist {
		err = utils.AtomicWriteFile(f.indexFilename(), newIndexJson, defaultFilePermissions)
		if err != nil {
			return nil, err
		}
		err = f.writeNamesFile(names)
		if err != nil {
			return nil, err
		}
	}

	msg := "Updated index with %d entries in %s "
	msg = fmt.Sprintf(msg, fileCount, duration.String())
	log.Info(msg)
	return newIndex, nil
}

func (f *FileRepo) updateIndexWithFile(idx *model.Index, path string, info os.FileInfo, log *slog.Logger, err error) (updated bool, addedName string, deletedName string, errr error) {
	if os.IsNotExist(err) {
		id, _ := strings.CutPrefix(filepath.ToSlash(filepath.Clean(path)), filepath.ToSlash(filepath.Clean(f.root)))
		id, _ = strings.CutPrefix(id, "/")
		upd, name, err := idx.Delete(id)
		if err != nil {
			return false, "", "", err
		}
		return upd, "", name, nil
	}
	if err != nil {
		return false, "", "", err
	}
	if info.IsDir() || !strings.HasSuffix(info.Name(), TMExt) {
		return false, "", "", nil
	}
	thingMeta, err := getThingMetadata(path)
	if err != nil {
		msg := "Failed to extract metadata from file %s with error:"
		msg = fmt.Sprintf(msg, path)
		log.Error(msg)
		log.Error(err.Error())
		log.Error("The file will be excluded from the table of contents.")
		return false, "", "", nil
	}
	tmid, err := idx.Insert(&thingMeta)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to insert %s into index:", path))
		log.Error(err.Error())
		log.Error("The file will be excluded from index")
		return false, "", "", nil
	}
	return true, tmid.Name, "", nil
}

type unlockFunc func()

func (f *FileRepo) lockIndex(ctx context.Context) (unlockFunc, error) {
	rd := filepath.Join(f.root, RepoConfDir)
	stat, err := os.Stat(rd)
	if err != nil || !stat.IsDir() {
		err := os.MkdirAll(rd, defaultDirPermissions)
		if err != nil {
			return func() {}, err
		}
	}
	idxFile := f.indexFilename()

	fl := flock.New(idxFile + ".lock")
	ctx, cancel := context.WithTimeout(ctx, indexLockTimeout)
	unlock := func() {
		cancel()
		_ = fl.Unlock()
	}
	locked, err := fl.TryLockContext(ctx, indexLocRetryDelay)
	if err != nil {
		return unlock, err
	}
	if !locked {
		return unlock, ErrIndexLocked
	}

	f.moveOldIndex(idxFile)

	return unlock, nil
}

// moveOldIndex attempts to move the index file at root to .tmc folder and remove old lock file
// ignores all errors
func (f *FileRepo) moveOldIndex(idxFile string) {
	oldIdx := filepath.Join(f.root, IndexFilename)
	_, errOld := os.Stat(oldIdx)
	_, errNew := os.Stat(idxFile)
	if errOld == nil && errNew != nil {
		_ = os.Rename(oldIdx, idxFile)
	}
	_ = os.Remove(oldIdx + ".lock")
}

func (f *FileRepo) readNamesFile() []string {
	lines, _ := utils.ReadFileLines(filepath.Join(f.root, RepoConfDir, TmNamesFile))
	return lines
}
func (f *FileRepo) writeNamesFile(names []string) error {
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

func (f *FileRepo) ListCompletions(ctx context.Context, kind string, toComplete string) ([]string, error) {
	switch kind {
	case CompletionKindNames:
		unlock, err := f.lockIndex(ctx)
		defer unlock()
		if err != nil {
			return nil, err
		}
		ns := f.readNamesFile()
		_, seg := longestPath(toComplete)
		names := namesToCompletions(ns, toComplete, seg+1)
		return names, nil
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
