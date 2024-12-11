package repos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gofrs/flock"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

const (
	defaultDirPermissions  = 0775
	defaultFilePermissions = 0664
	indexLockTimeout       = 5 * time.Second
	indexLocRetryDelay     = 13 * time.Millisecond
	TMExt                  = ".tm.json"
)

var ErrRootInvalid = errors.New("root is not a directory")
var osStat = os.Stat         // mockable for testing
var osReadFile = os.ReadFile // mockable for testing

// FileRepo implements a Repo TM repository backed by a file system
type FileRepo struct {
	root string
	spec model.RepoSpec

	// cached index
	idx *model.Index
}

func (f *FileRepo) CanonicalRoot() string {
	abs, _ := filepath.Abs(f.root)
	return abs
}

func NewFileRepo(config ConfigMap, spec model.RepoSpec) (*FileRepo, error) {
	loc, found := config.GetString(KeyRepoLoc)
	if !found {
		return nil, fmt.Errorf("cannot create a file repo from spec %v. Invalid config. loc is either not found or not a string", spec)
	}
	rootPath, err := utils.ExpandHome(loc)
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
		err := fmt.Errorf("nothing to write for id %v", id)
		return ImportResultFromError(err)
	}
	idS := id.String()
	fullPath, dir, _ := f.filenames(idS)
	err := os.MkdirAll(dir, defaultDirPermissions)
	if err != nil {
		err := fmt.Errorf("could not create directory %s: %w", dir, err)
		return ImportResultFromError(err)
	}

	match, existingId := f.getExistingID(ctx, idS)
	if (match == idMatchDigest || match == idMatchFull) && !opts.Force {
		err := &ErrTMIDConflict{Type: IdConflictSameContent, ExistingId: existingId}
		return ImportResult{Type: ImportResultError, Message: err.Error(), Err: err}, err
	}

	err = utils.AtomicWriteFile(fullPath, raw, defaultFilePermissions)
	if err != nil {
		err := fmt.Errorf("could not write TM to catalog: %v", err)
		return ImportResultFromError(err)
	}

	if match == idMatchTimestamp && !opts.Force {
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
	unlock, err := f.lockIndex(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	index, err := f.readIndex()
	if err != nil {
		return err
	}
	if index.FindByTMID(id) == nil {
		return model.ErrTMNotFound
	}
	fullFilename, dir, _ := f.filenames(id)
	err = os.Remove(fullFilename)
	if os.IsNotExist(err) {
		return fmt.Errorf("couldn't delete TM file %s: %w", id, model.ErrTMNotFound)
	}
	attDir, _ := f.getAttachmentsDir(model.NewTMIDAttachmentContainerRef(id))
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
			return fmt.Errorf("internal error while deleting %s: not in .attachments directory: %s", id, attDir)
		}
		err = os.RemoveAll(_attDir)
		if err != nil {
			return err
		}
	}
	_ = rmEmptyDirs(dir, f.root)

	_, err = f.updateIndex(ctx, f.indexUpdaterForIds(id))
	return err
}

func rmEmptyDirs(from string, upTo string) error {
	from, errF := filepath.Abs(from)
	upTo, errU := filepath.Abs(upTo)
	if errF != nil {
		errF = fmt.Errorf("from path %s cannot be converted to absolute path: %w", from, errF)
		return errF
	} else if errU != nil {
		errU = fmt.Errorf("upTo path %s cannot be converted to absolute path: %w", upTo, errU)
		return errU
	} else if !strings.HasPrefix(from, upTo) {
		err := fmt.Errorf("cannot remove empty dirs: from path %s is not below upTo %s", from, upTo)
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

func (f *FileRepo) getExistingID(ctx context.Context, ids string) (idMatch, string) {
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
		utils.GetLogger(ctx, "FileRepo").Warn("invalid TM version in TM id", "id", ids, "error", err)
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
	slices.SortStableFunc(res, func(a, b model.TMVersion) int {
		return a.Compare(b)
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
	match, actualId := f.getExistingID(ctx, id)
	if match != idMatchFull && match != idMatchDigest {
		return "", nil, model.ErrTMNotFound
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

	if len(ids) == 0 {
		_, err = f.updateIndex(ctx, f.fullIndexRebuild)
		return err
	}
	_, err = f.updateIndex(ctx, f.indexUpdaterForIds(ids...))
	return err
}

func (f *FileRepo) CheckIntegrity(ctx context.Context, filter model.ResourceFilter) (results []model.CheckResult, err error) {
	err = f.checkRootValid()
	if err != nil {
		return nil, err
	}

	unlock, err := f.lockIndex(ctx)
	defer unlock()
	if err != nil {
		return nil, err
	}
	idx, err := f.readIndex()
	if err != nil {
		if errors.Is(err, ErrNoIndex) {
			return nil, nil
		}
		return nil, err
	}
	idx.Sort()

	r, err := f.verifyAllFilesAreIndexed(ctx, idx, filter)
	return r, err
}

func (f *FileRepo) Spec() model.RepoSpec {
	return f.spec
}

func (f *FileRepo) List(ctx context.Context, search *model.SearchParams) (model.SearchResult, error) {
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
	idx.Sort() // the index is supposed to be sorted on disk, but we don't trust external storage, hence we'll sort here one more time to be extra sure
	sr := model.NewIndexToFoundMapper(f.Spec().ToFoundSource()).ToSearchResult(*idx)
	filtered := &sr
	err = filtered.Filter(search)
	return *filtered, err
}

// readIndex reads the contents of the index file. Must be called after the lock is acquired with lockIndex()
func (f *FileRepo) readIndex() (*model.Index, error) {
	if f.idx != nil {
		return f.idx, nil
	}
	data, err := os.ReadFile(f.indexFilename())
	if err != nil {
		if os.IsNotExist(err) {
			err = ErrNoIndex
		}
		return nil, err
	}

	var index model.Index
	err = json.Unmarshal(data, &index)
	if err == nil {
		f.idx = &index
	}
	return &index, err
}

func (f *FileRepo) indexFilename() string {
	return filepath.Join(f.root, RepoConfDir, IndexFilename)
}

func (f *FileRepo) Versions(ctx context.Context, name string) ([]model.FoundVersion, error) {
	name = strings.TrimSpace(name)
	res, err := f.List(ctx, &model.SearchParams{Name: name})
	if err != nil {
		return nil, err
	}

	if len(res.Entries) != 1 {
		err := fmt.Errorf("%w: %s", model.ErrTMNameNotFound, name)
		return nil, err
	}

	return res.Entries[0].Versions, nil
}

func (f *FileRepo) GetTMMetadata(ctx context.Context, tmID string) ([]model.FoundVersion, error) {
	id, err := model.ParseTMID(tmID)
	if err != nil {
		return nil, err
	}
	versions, err := f.Versions(ctx, id.Name)
	if err != nil {
		return nil, err
	}
	for _, v := range versions {
		if v.TMID == tmID {
			return []model.FoundVersion{v}, nil
		}
	}
	return nil, model.ErrTMNotFound
}

func (f *FileRepo) ImportAttachment(ctx context.Context, container model.AttachmentContainerRef, attachment model.Attachment, content []byte, force bool) error {
	err := f.checkRootValid()
	if err != nil {
		return err
	}

	unlock, err := f.lockIndex(ctx)
	defer unlock()
	if err != nil {
		return err
	}

	attDir, err := f.prepareAttachmentOperation(container)
	if err != nil {
		return err
	}

	err = f.verifyAttachmentExistsInIndex(container, attachment.Name)
	if err == nil && !force {
		return ErrAttachmentExists
	}
	if err != nil && !errors.Is(err, model.ErrAttachmentNotFound) {
		return err
	}
	err = os.MkdirAll(attDir, defaultDirPermissions)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(attDir, attachment.Name), content, defaultFilePermissions)
	if err != nil {
		return err
	}

	_, err = f.updateIndex(ctx, f.indexUpdaterForImportAttachment(container, attachment, content))
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
		return nil, model.ErrAttachmentNotFound
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
		return model.ErrAttachmentNotFound
	}
	return nil
}
func (f *FileRepo) DeleteAttachment(ctx context.Context, ref model.AttachmentContainerRef, attachmentName string) error {
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

	_, err = f.updateIndex(ctx, f.indexUpdaterForDeleteAttachment(ref, attachmentName))
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
	c, e, err := index.FindAttachmentContainer(ref)
	if err != nil {
		return nil, err
	}

	k := ref.Kind()
	if k == model.AttachmentContainerKindTMID {
		return &attachmentsContainer{
			attachments: c.Attachments,
			soleVersion: len(e.Versions) == 1,
		}, nil
	}
	// k==model.AttachmentContainerKindTMName -> return inventory entry's attachments
	return &attachmentsContainer{
		attachments: e.Attachments,
		soleVersion: false,
	}, nil
}

func getFileNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	return files, nil
}

// getAttachmentsDir returns the directory where the attachments to the given tmNameOrId are stored
func (f *FileRepo) getAttachmentsDir(ref model.AttachmentContainerRef) (string, error) {
	relDir, err := model.RelAttachmentsDir(ref)
	attDir := filepath.Join(f.root, relDir)
	return attDir, err
}

func (f *FileRepo) checkRootValid() error {
	stat, err := os.Stat(f.root)
	if err != nil || !stat.IsDir() {
		err := fmt.Errorf("%s: %w", f.Spec(), ErrRootInvalid)
		return err
	}
	return nil
}

func createFileRepoConfig(bytes []byte) (ConfigMap, error) {
	rc, err := AsRepoConfig(bytes)
	if err != nil {
		return nil, err
	}
	if rType, found := utils.JsGetString(rc, KeyRepoType); found {
		if rType != RepoTypeFile {
			return nil, fmt.Errorf("invalid json config. type must be \"file\" or absent")
		}
	}
	rc[KeyRepoType] = RepoTypeFile
	l, found := utils.JsGetString(rc, KeyRepoLoc)
	if !found {
		return nil, fmt.Errorf("invalid json config. must have string \"loc\"")
	}
	la, err := makeAbs(l)
	if err != nil {
		return nil, err
	}
	rc[KeyRepoLoc] = la
	return rc, nil
}

func makeAbs(dir string) (string, error) {
	if isEnvReference(dir) {
		return dir, nil
	}
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

func (f *FileRepo) updateIndex(ctx context.Context, updater indexUpdater) (*model.Index, error) {
	// Prepare data collection for logging stats
	start := time.Now()

	oldNames := f.readNamesFile()
	oldIndex, err := f.readIndex()
	if err != nil {
		oldIndex = &model.Index{
			Meta: model.IndexMeta{Created: time.Now()},
			Data: []*model.IndexEntry{},
		}
	}

	newIndex, names, fileCount, err := updater(ctx, oldIndex, oldNames)
	if err != nil {
		return nil, err
	}

	newIndex.Sort()
	newIndex.Meta.Created = time.Now()
	f.idx = newIndex
	duration := time.Now().Sub(start)
	// Ignore error as we are sure our struct does not contain channel,
	// complex or function values that would throw an error.
	newIndexJson, _ := json.MarshalIndent(newIndex, "", "  ")
	err = utils.AtomicWriteFile(f.indexFilename(), newIndexJson, defaultFilePermissions)
	if err != nil {
		return nil, err
	}
	err = f.writeNamesFile(names)
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Updated index with %d records in %s ", fileCount, duration.String())
	utils.GetLogger(ctx, "FileRepo").Debug(msg)

	return newIndex, nil
}

type indexUpdater func(ctx context.Context, oldIndex *model.Index, oldNames []string) (newIndex *model.Index, newNames []string, updatedFileCount int, err error)

func (f *FileRepo) indexUpdaterForIds(ids ...string) indexUpdater {
	return func(ctx context.Context, oldIndex *model.Index, oldNames []string) (*model.Index, []string, int, error) {
		fileCount := 0
		newNames := oldNames
		newIndex := oldIndex
		updatedAttContainers := make(map[model.AttachmentContainerRef]struct{})
		for _, id := range ids {
			select {
			case <-ctx.Done():
				return nil, nil, 0, ctx.Err()
			default:
			}
			pth := filepath.Join(f.root, id)
			info, statErr := osStat(pth)
			upd, id, nameDeleted, err := f.updateIndexWithFile(ctx, newIndex, pth, info, statErr)
			if err != nil {
				return nil, nil, 0, err
			}
			if upd {
				fileCount++
				if nameDeleted != "" {
					newNames = slices.DeleteFunc(newNames, func(s string) bool {
						return s == nameDeleted
					})
				} else if id.Name != "" {
					newNames = append(newNames, id.Name)
					updatedAttContainers[model.NewTMIDAttachmentContainerRef(id.String())] = struct{}{}
					updatedAttContainers[model.NewTMNameAttachmentContainerRef(id.Name)] = struct{}{}
				}
			}
		}
		err := f.reindexAttachments(updatedAttContainers, oldIndex, newIndex)
		if err != nil {
			return nil, nil, 0, err
		}

		return newIndex, newNames, fileCount, nil
	}

}
func (f *FileRepo) indexUpdaterForDeleteAttachment(ref model.AttachmentContainerRef, attName string) indexUpdater {
	return func(ctx context.Context, oldIndex *model.Index, oldNames []string) (*model.Index, []string, int, error) {
		select {
		case <-ctx.Done():
			return nil, nil, 0, ctx.Err()
		default:
		}
		itemCount := 0
		cont, _, _ := oldIndex.FindAttachmentContainer(ref)
		if cont != nil {
			oldCnt := len(cont.Attachments)
			cont.Attachments = slices.DeleteFunc(cont.Attachments, func(attachment model.Attachment) bool {
				return attachment.Name == attName
			})
			if len(cont.Attachments) != oldCnt {
				itemCount = 1
			}
		}
		return oldIndex, oldNames, itemCount, nil

	}
}
func (f *FileRepo) indexUpdaterForImportAttachment(ref model.AttachmentContainerRef, att model.Attachment, content []byte) indexUpdater {
	return func(ctx context.Context, oldIndex *model.Index, oldNames []string) (*model.Index, []string, int, error) {
		select {
		case <-ctx.Done():
			return nil, nil, 0, ctx.Err()
		default:
		}
		mt := att.MediaType
		if mt == "" {
			cont, _, _ := oldIndex.FindAttachmentContainer(ref)
			if cont != nil {
				oldAtt, _ := cont.FindAttachment(att.Name)
				mt = oldAtt.MediaType
			}
		}
		mediaType := utils.DetectMediaType(mt, att.Name, utils.ReadCloserGetterFromBytes(content))
		a := model.Attachment{Name: att.Name, MediaType: mediaType}
		err := oldIndex.InsertAttachments(ref, a)
		return oldIndex, oldNames, 1, err
	}
}

func (f *FileRepo) fullIndexRebuild(ctx context.Context, oldIndex *model.Index, _ []string) (*model.Index, []string, int, error) {
	fileCount := 0
	updatedAttContainers := make(map[model.AttachmentContainerRef]struct{})
	newIndex := &model.Index{
		Meta: model.IndexMeta{Created: time.Now()},
		Data: []*model.IndexEntry{},
	}
	var names []string
	err := filepath.Walk(f.root, func(path string, info os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		upd, id, _, err := f.updateIndexWithFile(ctx, newIndex, path, info, err)
		if err != nil {
			return err
		}
		if upd {
			fileCount++
			names = append(names, id.Name)
			updatedAttContainers[model.NewTMIDAttachmentContainerRef(id.String())] = struct{}{}
			updatedAttContainers[model.NewTMNameAttachmentContainerRef(id.Name)] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, nil, 0, err
	}
	err = f.reindexAttachments(updatedAttContainers, oldIndex, newIndex)
	if err != nil {
		return nil, nil, 0, err
	}

	return newIndex, names, fileCount, nil
}

func (f *FileRepo) reindexAttachments(containers map[model.AttachmentContainerRef]struct{}, oldIndex *model.Index, newIndex *model.Index) error {
	for ref := range containers {
		dir, _ := f.getAttachmentsDir(ref) // ref is sure to be valid
		nameAttachments, err := getFileNames(dir)
		if err != nil {
			return err
		}
		container, _, _ := oldIndex.FindAttachmentContainer(ref)
		var atts []model.Attachment
		for _, na := range nameAttachments {
			att, _ := container.FindAttachment(na)
			oldMt := att.MediaType
			mediaType := utils.DetectMediaType(oldMt, na, utils.ReadCloserGetterFromFilename(na))
			a := model.Attachment{Name: na, MediaType: mediaType}
			atts = append(atts, a)
		}
		err = newIndex.InsertAttachments(ref, atts...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *FileRepo) updateIndexWithFile(ctx context.Context, idx *model.Index, path string, info os.FileInfo, err error) (updated bool, indexedId model.TMID, deletedName string, errr error) {
	log := utils.GetLogger(ctx, "FileRepo")
	idS, _ := strings.CutPrefix(filepath.ToSlash(filepath.Clean(path)), filepath.ToSlash(filepath.Clean(f.root)))
	idS, _ = strings.CutPrefix(idS, "/")
	if os.IsNotExist(err) {
		upd, name, err := idx.Delete(idS)
		if err != nil {
			return false, model.TMID{}, "", err
		}
		return upd, model.TMID{}, name, nil
	}
	if err != nil {
		return false, model.TMID{}, "", err
	}
	if info.IsDir() {
		return false, model.TMID{}, "", nil
	}
	if !strings.HasSuffix(info.Name(), TMExt) {
		return false, model.TMID{}, "", nil
	}
	thingMeta, err := f.getThingMetadata(path)
	if err != nil {
		log.Warn(fmt.Sprintf("failed to extract metadata from file %s: %v. The file will be excluded from index", path, err))
		return false, model.TMID{}, "", nil
	}
	err = idx.Insert(&thingMeta.tm)
	if err != nil {
		log.Warn(fmt.Sprintf("failed to insert %s into index: %v. The file will be excluded from index", path, err))
		return false, model.TMID{}, "", nil
	}
	return true, thingMeta.id, "", nil
}

type unlockFunc func()

func (f *FileRepo) lockIndex(ctx context.Context) (unlockFunc, error) {
	rd := filepath.Join(f.root, RepoConfDir)
	stat, err := os.Stat(rd)
	if err != nil || !stat.IsDir() {
		err := os.MkdirAll(rd, defaultDirPermissions)
		if err != nil {
			err := fmt.Errorf("couldn't create repo config dir %s: %w", rd, err)
			return func() {}, err
		}
	}
	idxFile := f.indexFilename()

	fl := flock.New(idxFile + ".lock")
	ctx, cancel := context.WithTimeout(ctx, indexLockTimeout)
	unlock := func() {
		cancel()
		_ = fl.Unlock()
		f.idx = nil
	}
	locked, err := fl.TryLockContext(ctx, indexLocRetryDelay)
	if err != nil || !locked {
		err = fmt.Errorf("failed to lock index file %s: %w", idxFile, err)
		return unlock, err
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
func (f *FileRepo) readIgnoreFile() (*ignore.GitIgnore, error) {
	ignoreFileName := filepath.Join(f.root, RepoConfDir, TmIgnoreFile)
	_, err := os.Stat(ignoreFileName)
	if os.IsNotExist(err) {
		err := f.writeDefaultIgnoreFile()
		if err != nil {
			return nil, err
		}
	}
	lines, err := utils.ReadFileLines(ignoreFileName)
	if err != nil {
		return nil, err
	}
	gitIgnore := ignore.CompileIgnoreLines(lines...)
	return gitIgnore, nil
}
func (f *FileRepo) writeDefaultIgnoreFile() error {
	ignoreDefaults := []string{
		"# ignore any top-level files",
		"/*",
		"!/*/",
		"",
		"# ignore any top-level directories starting with a dot",
		"/.*/",
	}
	return utils.WriteFileLines(ignoreDefaults, filepath.Join(f.root, RepoConfDir, TmIgnoreFile), defaultFilePermissions)
}

type thingMetadata struct {
	tm model.ThingModel
	id model.TMID
}

func (f *FileRepo) getThingMetadata(path string) (*thingMetadata, error) {
	data, err := osReadFile(path)
	if err != nil {
		return nil, err
	}

	ctm, err := model.ParseThingModel(data)
	if err != nil {
		return nil, err
	}

	tmid, err := model.ParseTMID(ctm.ID)
	if err != nil {
		return nil, err
	}

	return &thingMetadata{
		tm: *ctm,
		id: tmid,
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
			err := fmt.Errorf("%w :no completions for name containing '..': %s", ErrInvalidCompletionParams, toComplete)
			return nil, err
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
					utils.GetLogger(ctx, "FileRepo.ListCompletions").Debug(err.Error())
					continue
				}
				vm[ver.BaseString()] = struct{}{}
			}
		}
		var vs []string
		for v := range vm {
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

func (f *FileRepo) verifyAllFilesAreIndexed(ctx context.Context, idx *model.Index, filter model.ResourceFilter) ([]model.CheckResult, error) {
	if filter == nil {
		filter = func(_ string) bool { return true }
	}
	ignor, err := f.prepareIgnoreFunc()
	if err != nil {
		return nil, err
	}

	var results []model.CheckResult

	err = filepath.Walk(f.root, func(path string, info os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		resourceName, err := filepath.Rel(f.root, path)
		if err != nil {
			return err
		}
		resourceName = filepath.ToSlash(resourceName)
		if ignor(resourceName) || !filter(resourceName) {
			return nil
		}
		checkResult := f.verifyFileIsIndexed(resourceName, idx)
		results = append(results, checkResult)

		return nil
	})

	return results, err

}

func (f *FileRepo) verifyFileIsIndexed(file string, idx *model.Index) model.CheckResult {
	if isTmcConfigFile(file) {
		return model.CheckResult{model.CheckOK, file, "OK"}
	}
	if isAtt, ref, attName := isAttachmentFile(file); isAtt {
		container, _, err := idx.FindAttachmentContainer(ref)
		if err != nil {
			var nfErr *model.ErrNotFound
			if errors.As(err, &nfErr) {
				return model.CheckResult{model.CheckErr, file, "appears to be an attachment file to a TM name or TM ID which does not exist. Make sure you import it using TMC CLI"}
			}
		}
		_, found := container.FindAttachment(attName)
		if !found {
			return model.CheckResult{model.CheckErr, file, "appears to be an attachment file which is not known to the repository. Make sure you import it using TMC CLI"}
		}
		return model.CheckResult{model.CheckOK, file, "OK"}
	}
	if isTMFile(file) {
		ver := idx.FindByTMID(file)
		if ver == nil {
			return model.CheckResult{model.CheckErr, file, "appears to be a TM file which is not known to the repository. Make sure you import it using TMC CLI"}
		}
		return model.CheckResult{model.CheckOK, file, "OK"}
	}
	return model.CheckResult{model.CheckErr, file, "file unknown"}
}

func (f *FileRepo) prepareIgnoreFunc() (func(string) bool, error) {
	gitIgnore, err := f.readIgnoreFile()
	if err != nil {
		return nil, err
	}
	return func(s string) bool {
		return gitIgnore.MatchesPath(s)
	}, nil
}

func isTMFile(file string) bool {
	_, err := model.ParseTMID(file)
	return err == nil
}

func isAttachmentFile(file string) (bool, model.AttachmentContainerRef, string) {
	before, after, found := strings.Cut(file, "/"+model.AttachmentsDir+"/")
	if !found {
		return false, model.AttachmentContainerRef{}, ""
	}
	tmName, err := model.ParseFetchName(before)
	if err != nil || tmName.Semver != "" {
		return false, model.AttachmentContainerRef{}, ""
	}
	attName := path.Base(after)
	if tmVer := path.Dir(after); tmVer != "." {
		_, err := model.ParseTMVersion(tmVer)
		if err != nil {
			return false, model.AttachmentContainerRef{}, ""
		}
		return true, model.NewTMIDAttachmentContainerRef(tmName.Name + "/" + tmVer + ".tm.json"), attName
	}
	return true, model.NewTMNameAttachmentContainerRef(tmName.Name), attName
}

func isTmcConfigFile(p string) bool {
	return p == path.Join(RepoConfDir, IndexFilename) ||
		p == path.Join(RepoConfDir, IndexFilename+".lock") ||
		p == path.Join(RepoConfDir, TmIgnoreFile) ||
		p == path.Join(RepoConfDir, TmNamesFile)

}

func getAttachmentCompletions(ctx context.Context, args []string, f Repo) ([]string, error) {
	if len(args) > 0 {
		_, err := model.ParseTMID(args[0])
		if err == nil {
			metas, err := f.GetTMMetadata(ctx, args[0])
			if err != nil {
				return nil, err
			}
			var attNames []string
			for _, m := range metas {
				for _, a := range m.Attachments {
					attNames = append(attNames, a.Name)
				}
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
}
