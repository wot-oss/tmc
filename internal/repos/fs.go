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

func (f *FileRepo) Import(ctx context.Context, id model.TMID, raw []byte, opts ImportOptions) (ImportResult, error) {
	if len(raw) == 0 {
		err := errors.New("nothing to write")
		return ImportResultFromError(err)
	}
	idS := id.String()
	fullPath, dir, _ := f.filenames(idS)
	err := os.MkdirAll(dir, defaultDirPermissions)
	if err != nil {
		err := fmt.Errorf("could not create directory %s: %w", dir, err)
		return ImportResultFromError(err)
	}

	match, existingId := f.getExistingID(idS)
	log := slog.Default()
	log.Debug(fmt.Sprintf("match: %v, existingId: %v", match, existingId))
	if (match == idMatchDigest || match == idMatchFull) && !opts.Force {
		log.Info(fmt.Sprintf("Same TM content already exists under ID %v", existingId))
		err := &ErrTMIDConflict{Type: IdConflictSameContent, ExistingId: existingId}
		return ImportResult{Type: ImportResultError, Message: err.Error(), Err: err}, err
	}

	err = utils.AtomicWriteFile(fullPath, raw, defaultFilePermissions)
	if err != nil {
		err := fmt.Errorf("could not write TM to catalog: %v", err)
		return ImportResultFromError(err)
	}
	log.Info("saved Thing Model file", "filename", fullPath)

	if match == idMatchTimestamp && !opts.Force {
		msg := fmt.Sprintf("Version and timestamp clash with existing %v", existingId)
		log.Info(msg)
		err := &ErrTMIDConflict{Type: IdConflictSameTimestamp, ExistingId: existingId}
		return ImportResult{Type: ImportResultWarning, TmID: idS, Message: err.Error(), Err: err}, nil
	}
	return ImportResult{Type: ImportResultOK, TmID: idS, Message: "OK"}, nil
}

func (f *FileRepo) Delete(ctx context.Context, id string) error {
	err := f.checkRootValid()
	if err != nil {
		return err
	}
	_, err = model.ParseTMID(id)
	if err != nil {
		return err
	}
	match, _ := f.getExistingID(id)
	if match != idMatchFull {
		return ErrTMNotFound
	}
	fullFilename, dir, _ := f.filenames(id)
	err = os.Remove(fullFilename)
	if os.IsNotExist(err) {
		return ErrTMNotFound
	}
	attDir, _ := f.getAttachmentsDir(model.NewTMIDAttachmentContainerRef(id))
	unlock, err := f.lockIndex(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	h, err := f.listAttachments(model.NewTMIDAttachmentContainerRef(id))
	if err != nil {
		return err
	}
	err = os.RemoveAll(attDir)
	if err != nil {
		return err
	}
	if h.soleVersion { // delete attachments belonging to TM name when deleting the last remaining version of a TM
		_attDir := filepath.Dir(attDir)
		// make sure there's no mistake, and we're about to delete the correct dir with attachments
		if filepath.Base(_attDir) != model.AttachmentsDir {
			return fmt.Errorf("internal error: not in .attachments directory: %s", attDir)
		}
		err = os.RemoveAll(_attDir)
		if err != nil {
			return err
		}
	}
	_ = rmEmptyDirs(dir, f.root)

	_, err = f.updateIndex(ctx, []string{id}, true)
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
	idPrefix := strings.TrimSuffix(ids, base)
	existingTMVersions := findTMFileEntriesByBaseVersion(entries, version)
	if idx := slices.IndexFunc(existingTMVersions, func(v model.TMVersion) bool {
		return v.Hash == version.Hash
	}); idx != -1 {
		return idMatchDigest, idPrefix + existingTMVersions[idx].String() + TMExt
	}
	if idx := slices.IndexFunc(existingTMVersions, func(v model.TMVersion) bool {
		return v.Timestamp == version.Timestamp
	}); idx != -1 {
		return idMatchTimestamp, idPrefix + existingTMVersions[idx].String() + TMExt
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
	_, err = model.ParseTMID(id)
	if err != nil {
		return "", nil, err
	}
	match, actualId := f.getExistingID(id)
	if match != idMatchFull && match != idMatchDigest {
		return "", nil, ErrTMNotFound
	}
	actualFilename, _, _ := f.filenames(actualId)
	b, err := os.ReadFile(actualFilename)
	return actualId, b, err
}

func (f *FileRepo) Index(ctx context.Context, ids ...string) error {
	err := f.checkRootValid()
	if err != nil {
		return err
	}
	unlock, err := f.lockIndex(ctx)
	defer unlock()
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

	idxOld, errIdxOld := f.readIndex()
	idxNew, err := f.updateIndex(ctx, []string{}, false)
	if err != nil {
		return err
	}

	// if there are no TMs and there is an empty or missing index file, it's considered to be valid
	if idxNew.IsEmpty() && (errIdxOld == ErrNoIndex || idxOld.IsEmpty()) {
		return nil
	} else if errIdxOld != nil {
		return errIdxOld
	}

	idxNew.Sort()
	idxOld.Sort()

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
	return model.NewIndexToFoundMapper(f.Spec().ToFoundSource()).ToSearchResult(*idx), nil
}

// readIndex reads the contents of the index file. Must be called after the lock is acquired with lockIndex()
func (f *FileRepo) readIndex() (*model.Index, error) {
	data, err := os.ReadFile(f.indexFilename())
	if err != nil {
		if os.IsNotExist(err) {
			err = ErrNoIndex
		}
		return nil, err
	}

	var index model.Index
	err = json.Unmarshal(data, &index)
	return &index, err
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
		err := fmt.Errorf("%w: %s", ErrTMNameNotFound, name)
		log.Error(err.Error())
		return nil, err
	}

	return res.Entries[0].Versions, nil
}

func (f *FileRepo) GetTMMetadata(ctx context.Context, tmID string) (*model.FoundVersion, error) {
	id, err := model.ParseTMID(tmID)
	if err != nil {
		return nil, err
	}
	match, actualId := f.getExistingID(tmID)
	if match != idMatchFull && match != idMatchDigest {
		return nil, ErrTMNotFound
	}

	versions, err := f.Versions(ctx, id.Name)
	if err != nil {
		return nil, err
	}
	for _, v := range versions {
		if v.TMID == actualId {
			return &v, nil
		}
	}
	return nil, ErrTMNotFound
}

func (f *FileRepo) PushAttachment(ctx context.Context, ref model.AttachmentContainerRef, attachmentName string, content []byte) error {
	log := slog.Default()
	log.Debug(fmt.Sprintf("Pushing attachment %s for '%v'", attachmentName, ref))

	err := f.checkRootValid()
	if err != nil {
		return err
	}
	log.Debug("root dir validated")

	unlock, err := f.lockIndex(ctx)
	defer unlock()
	if err != nil {
		return err
	}
	log.Debug("index locked")

	attDir, err := f.prepareAttachmentOperation(ref)
	if err != nil {
		return err
	}

	err = os.MkdirAll(attDir, defaultDirPermissions)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(attDir, attachmentName), content, defaultFilePermissions)
	if err != nil {
		return err
	}
	return nil
}

// prepareAttachmentOperation prepares for a CRUD operation on attachments
// Must be called after the index lock has been acquired with lockIndex
func (f *FileRepo) prepareAttachmentOperation(ref model.AttachmentContainerRef) (string, error) {
	attDir, err := f.getAttachmentsDir(ref)
	if err != nil {
		return "", err
	}
	// use listAttachments to validate ref
	_, err = f.listAttachments(ref)
	if err != nil {
		return "", err
	}
	return attDir, nil
}

func (f *FileRepo) FetchAttachment(ctx context.Context, ref model.AttachmentContainerRef, attachmentName string) ([]byte, error) {
	log := slog.Default()
	log.Debug(fmt.Sprintf("Fetching attachment %s for '%v'", attachmentName, ref))

	err := f.checkRootValid()
	if err != nil {
		return nil, err
	}

	unlock, err := f.lockIndex(ctx)
	defer unlock()
	if err != nil {
		return nil, err
	}

	attDir, err := f.prepareAttachmentOperation(ref)
	if err != nil {
		return nil, err
	}

	err = f.verifyAttachmentExistsInIndex(ref, attachmentName)
	if err != nil {
		return nil, err
	}

	file, err := os.ReadFile(filepath.Join(attDir, attachmentName))
	if os.IsNotExist(err) {
		return nil, ErrAttachmentNotFound
	}
	return file, err
}

func (f *FileRepo) verifyAttachmentExistsInIndex(ref model.AttachmentContainerRef, attachmentName string) error {
	atts, err := f.listAttachments(ref)
	if err != nil {
		return err
	}

	if !slices.ContainsFunc(atts.attachments, func(attachment model.Attachment) bool {
		return attachment.Name == attachmentName
	}) {
		return ErrAttachmentNotFound
	}
	return nil
}
func (f *FileRepo) DeleteAttachment(ctx context.Context, ref model.AttachmentContainerRef, attachmentName string) error {
	log := slog.Default()
	log.Debug(fmt.Sprintf("Deleting attachment %s for '%v'", attachmentName, ref))

	err := f.checkRootValid()
	if err != nil {
		return err
	}

	unlock, err := f.lockIndex(ctx)
	defer unlock()
	if err != nil {
		return err
	}

	attDir, err := f.prepareAttachmentOperation(ref)
	if err != nil {
		return err
	}

	err = f.verifyAttachmentExistsInIndex(ref, attachmentName)
	if err != nil {
		return err
	}

	err = os.Remove(filepath.Join(attDir, attachmentName))
	if err != nil {
		return err
	}

	err = removeDirIfEmpty(attDir)
	if err != nil {
		return err
	}

	if filepath.Base(filepath.Dir(attDir)) == model.AttachmentsDir {
		return removeDirIfEmpty(filepath.Dir(attDir))
	}
	return nil
}

func removeDirIfEmpty(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return os.Remove(dir)
	}
	return nil
}

type attachmentsContainer struct {
	attachments []model.Attachment
	soleVersion bool
}

// listAttachments returns the attachment list belonging to given tmNameOrId
// Returns ErrTMNotFound or ErrTMNameNotFound if ref is not present in this repository
// Must be called after the index lock has been acquired with lockIndex
func (f *FileRepo) listAttachments(ref model.AttachmentContainerRef) (*attachmentsContainer, error) {
	index, err := f.readIndex()
	if err != nil {
		return nil, err
	}
	return findAttachmentContainer(index, ref)
}

func findAttachmentContainer(index *model.Index, ref model.AttachmentContainerRef) (*attachmentsContainer, error) {
	k := ref.Kind()
	var tmName string
	switch k {
	case model.AttachmentContainerKindInvalid:
		return nil, model.ErrInvalidIdOrName
	case model.AttachmentContainerKindTMID:
		id, err := model.ParseTMID(ref.TMID)
		if err != nil {
			return nil, err
		}
		tmName = id.Name
	case model.AttachmentContainerKindTMName:
		fn, err := model.ParseFetchName(ref.TMName)
		if err != nil || fn.Semver != "" {
			return nil, model.ErrInvalidIdOrName
		}
		tmName = ref.TMName
	}

	indexEntry := index.FindByName(tmName)
	if indexEntry == nil {
		if ref.Kind() == model.AttachmentContainerKindTMID {
			return nil, ErrTMNotFound
		} else {
			return nil, ErrTMNameNotFound
		}
	}
	versions := indexEntry.Versions
	if k == model.AttachmentContainerKindTMID {
		for _, v := range versions {
			if v.TMID == ref.TMID {
				return &attachmentsContainer{
					attachments: v.Attachments,
					soleVersion: len(versions) == 1,
				}, nil
			}
		}
		return nil, ErrTMNotFound
	}
	// k==model.AttachmentContainerKindTMName -> return inventory entry's attachments
	return &attachmentsContainer{
		attachments: indexEntry.Attachments,
		soleVersion: false,
	}, nil
}
func readFileNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var attachments []string
	for _, e := range entries {
		if !e.IsDir() {
			attachments = append(attachments, e.Name())
		}
	}
	return attachments, nil
}

// getAttachmentsDir returns the directory where the attachments to the given tmNameOrId are stored
func (f *FileRepo) getAttachmentsDir(ref model.AttachmentContainerRef) (string, error) {
	relDir, err := model.RelAttachmentsDir(ref)
	attDir := filepath.Join(f.root, relDir)
	slog.Default().Debug("attachments dir calculated", "container", ref, "attDir", attDir)
	return attDir, err
}

func (f *FileRepo) checkRootValid() error {
	stat, err := os.Stat(f.root)
	if err != nil || !stat.IsDir() {
		err := fmt.Errorf("%s: %w", f.Spec(), ErrRootInvalid)
		slog.Default().Debug(err.Error())
		return err
	}
	slog.Default().Debug(fmt.Sprintf("%s: root dir check ok", f.Spec()))
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

	var newIndex *model.Index
	names := f.readNamesFile()

	namesToReindexAttachments := map[string]struct{}{}
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
				namesToReindexAttachments[name] = struct{}{}
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
			newIndex = index
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
					namesToReindexAttachments[name] = struct{}{}
					names = append(names, name)
				}
			}
		}
	}

	for name, _ := range namesToReindexAttachments {
		dir, _ := f.getAttachmentsDir(model.NewTMNameAttachmentContainerRef(name)) // name is sure to be valid
		nameAttachments, err := readFileNames(dir)
		if err != nil {
			return nil, err
		}
		newIndex.SetEntryAttachments(name, nameAttachments)
	}

	newIndex.Sort()
	duration := time.Now().Sub(start)
	// Ignore error as we are sure our struct does not contain channel,
	// complex or function values that would throw an error.
	newIndexJson, _ := json.MarshalIndent(newIndex, "", "  ")
	if persist {
		err := utils.AtomicWriteFile(f.indexFilename(), newIndexJson, defaultFilePermissions)
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
	idOrName, _ := strings.CutPrefix(filepath.ToSlash(filepath.Clean(path)), filepath.ToSlash(filepath.Clean(f.root)))
	idOrName, _ = strings.CutPrefix(idOrName, "/")
	if os.IsNotExist(err) {
		upd, name, err := idx.Delete(idOrName)
		if err != nil {
			return false, "", "", err
		}
		return upd, "", name, nil
	}
	if err != nil {
		return false, "", "", err
	}
	if info.IsDir() {
		if idx.FindByName(idOrName) != nil { // is a valid TM name
			return true, idOrName, "", nil // force reindexing attachments for idOrName
		}
		return false, "", "", nil
	}
	if !strings.HasSuffix(info.Name(), TMExt) {
		return false, "", "", nil
	}
	thingMeta, err := f.getThingMetadata(path)
	if err != nil {
		err = fmt.Errorf("failed to extract metadata from file %s with error: %w", path, err)
		log.Error(err.Error())
		log.Error("The file will be excluded from the table of contents.")
		return false, "", "", nil
	}
	err = idx.Insert(&thingMeta.tm, thingMeta.tmAttachments)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to insert %s into index:", path))
		log.Error(err.Error())
		log.Error("The file will be excluded from index")
		return false, "", "", nil
	}
	return true, thingMeta.id.Name, "", nil
}

type unlockFunc func()

func (f *FileRepo) lockIndex(ctx context.Context) (unlockFunc, error) {
	log := slog.Default()
	log.Debug(fmt.Sprintf("%s: attempting to lock index", f.Spec()))
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
		log.Debug(fmt.Sprintf("unlocked index file: %s", idxFile))
	}
	locked, err := fl.TryLockContext(ctx, indexLocRetryDelay)
	if err != nil {
		//log.Debug("failed to lock index file: %s", idxFile)
		return unlock, err
	}
	if !locked {
		//log.Debug("failed to lock index file: %s", idxFile)
		return unlock, ErrIndexLocked
	}

	f.moveOldIndex(idxFile)

	log.Debug(fmt.Sprintf("locked index file: %s", idxFile))
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

type thingMetadata struct {
	tm            model.ThingModel
	id            model.TMID
	tmAttachments []string
}

func (f *FileRepo) getThingMetadata(path string) (*thingMetadata, error) {
	data, err := osReadFile(path)
	if err != nil {
		return nil, err
	}

	var ctm model.ThingModel
	err = json.Unmarshal(data, &ctm)
	if err != nil {
		return nil, err
	}

	tmid, err := model.ParseTMID(ctm.ID)
	if err != nil {
		return nil, err
	}

	tmAttDir, _ := f.getAttachmentsDir(model.NewTMIDAttachmentContainerRef(ctm.ID)) // there cannot be any error parsing the id we just parsed
	tmAttachments, err := readFileNames(tmAttDir)
	if err != nil {
		return nil, err
	}

	return &thingMetadata{
		tm:            ctm,
		id:            tmid,
		tmAttachments: tmAttachments,
	}, nil
}

func (f *FileRepo) ListCompletions(ctx context.Context, kind string, args []string, toComplete string) ([]string, error) {
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
	case CompletionKindNamesOrIds:
		unlock, err := f.lockIndex(ctx)
		defer unlock()
		if err != nil {
			return nil, err
		}
		names := f.readNamesFile()
		lPath, seg := longestPath(toComplete)
		comps := namesToCompletions(names, toComplete, seg+1)
		if _, found := slices.BinarySearch(names, lPath); found { // current toComplete is a full TM name plus '/'
			// append versions to comps
			idx, err := f.readIndex()
			if err != nil {
				return nil, err
			}
			entry := idx.FindByName(lPath)
			if entry != nil { // shouldn't ever be nil, if index is in sync with names file, but paranoia never sleeps
				for _, v := range entry.Versions {
					comps = append(comps, v.TMID)
				}
			}
		}
		return comps, nil
	case CompletionKindAttachments:
		return getAttachmentCompletions(ctx, args, f)
	default:
		return nil, ErrInvalidCompletionParams
	}
}

func getAttachmentCompletions(ctx context.Context, args []string, f Repo) ([]string, error) {
	if len(args) > 0 {
		_, err := model.ParseTMID(args[0])
		if err == nil {
			metadata, err := f.GetTMMetadata(ctx, args[0])
			if err != nil {
				return nil, err
			}
			var attNames []string
			for _, a := range metadata.Attachments {
				attNames = append(attNames, a.Name)
			}
			return attNames, nil
		}
		sp := &model.SearchParams{Name: args[0]}
		sr, err := f.List(ctx, sp)
		if err != nil {
			return nil, err
		}
		if len(sr.Entries) == 0 {
			return nil, nil
		}
		var attNames []string
		for _, a := range sr.Entries[0].Attachments {
			attNames = append(attNames, a.Name)
		}
		return attNames, nil
	} else {
		return nil, ErrInvalidCompletionParams
	}
	return nil, nil
}
