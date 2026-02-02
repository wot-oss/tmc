package repos

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/aws/smithy-go"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

var ErrS3NotExists = errors.New("file does not exist in S3")
var ErrS3Op = errors.New("operational error")
var ErrS3Unknown = errors.New("unknown error")

func createS3RepoConfig(bytes []byte) (ConfigMap, error) {
	rc, err := AsRepoConfig(bytes)
	if err != nil {
		return nil, err
	}
	if rType, found := utils.JsGetString(rc, KeyRepoType); found {
		if rType != RepoTypeS3 {
			return nil, fmt.Errorf("invalid json config. type must be \"s3\" or absent")
		}
	}

	rc[KeyRepoType] = RepoTypeS3

	_, found := utils.JsGetString(rc, KeyRepoAWSBucket)
	if !found {
		return nil, fmt.Errorf("invalid json config. must have string \"aws_bucket\"")
	}

	_, foundAk := utils.JsGetString(rc, KeyRepoAWSAccessKeyId)
	_, foundSk := utils.JsGetString(rc, KeyRepoAWSSecretAccessKey)
	if (!foundAk && foundSk) || (foundAk && !foundSk) {
		return nil, fmt.Errorf("invalid json config. must have string \"aws_access_key_id\" and string \"aws_secret_access_key\" when setting credentials explicit")
	}

	return rc, nil
}

//go:generate mockery --name S3Client --outpkg s3mocks --output ../testutils/s3mocks
type S3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	Options() s3.Options
}

type S3Repo struct {
	region string
	bucket string
	spec   model.RepoSpec

	client S3Client
	// cached index
	idx *model.Index
}

func NewS3Repo(cfg ConfigMap, spec model.RepoSpec) (*S3Repo, error) {

	bucket, found := cfg.GetString(KeyRepoAWSBucket)
	if !found {
		return nil, fmt.Errorf("cannot create a AWS S3 repo from spec %v. Invalid config. aws_bucket is either not found or not a string", spec)
	}

	optFns := []func(*config.LoadOptions) error{}

	region, found := cfg.GetString(KeyRepoAWSRegion)
	if found {
		optFns = append(optFns, config.WithRegion(region))
	}

	ak, foundAk := cfg.GetString(KeyRepoAWSAccessKeyId)
	sk, foundSk := cfg.GetString(KeyRepoAWSSecretAccessKey)
	endpoint, foundEp := cfg.GetString(KeyRepoAWSEndpoint)
	if (!foundAk && foundSk) || (foundAk && !foundSk) {
		return nil, fmt.Errorf("cannot create a AWS S3 repo from spec %v. Invalid config. aws_access_key_id and aws_secret_access_key must be set both as type string when setting credentials explicit", spec)
	}
	if foundAk {
		optFns = append(optFns, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(ak, sk, "")))
	}

	configS3, err := config.LoadDefaultConfig(context.Background(), optFns...)
	if foundEp {
		configS3.BaseEndpoint = aws.String(endpoint)
	}
	if err != nil {
		err := fmt.Errorf("error loading S3 configuration: %w", err)
		return nil, err
	}

	c := s3.NewFromConfig(configS3, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &S3Repo{
		bucket: bucket,
		region: region,
		client: c,
		spec:   spec,
	}, nil
}

func (s *S3Repo) CanonicalRoot() string {
	return s.region + s.bucket
}

func (s *S3Repo) Import(ctx context.Context, id model.TMID, raw []byte, opts ImportOptions) (ImportResult, error) {
	if len(raw) == 0 {
		err := fmt.Errorf("nothing to write for id %v", id)
		return ImportResultFromError(err)
	}
	idS := id.String()

	match, existingId := s.getExistingID(ctx, idS)
	if (match == idMatchDigest || match == idMatchFull) && !opts.Force {
		err := &ErrTMIDConflict{Type: IdConflictSameContent, ExistingId: existingId}
		return ImportResult{Type: ImportResultError, Message: err.Error(), Err: err}, err
	}

	err := s3WriteObject(ctx, s.client, s.bucket, idS, raw)
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

func (s *S3Repo) Delete(ctx context.Context, id string) error {

	_, err := model.ParseTMID(id)
	if err != nil {
		return err
	}
	unlock, err := s.lockIndex(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	index, err := s.readIndex(ctx)
	if err != nil {
		return err
	}
	if index.FindByTMID(id) == nil {
		return model.ErrTMNotFound
	}

	_, err = s3Stat(ctx, s.client, s.bucket, id)
	if err != nil { //ErrS3NotExists
		return fmt.Errorf("couldn't delete TM file %s: %w", id, model.ErrTMNotFound)
	}

	err = s3RemoveObject(ctx, s.client, s.bucket, id)

	attDir, _ := s.getAttachmentsDir(model.NewTMIDAttachmentContainerRef(id))
	h, err := s.listAttachments(ctx, model.NewTMIDAttachmentContainerRef(id))
	if err != nil {
		return err
	}
	err = s3RemoveAll(ctx, s.client, s.bucket, attDir)
	if err != nil {
		return err
	}
	if h.soleVersion { // delete attachments belonging to TM name when deleting the last remaining version of a TM
		_attDir := path.Dir(attDir)
		// make sure there's no mistake, and we're about to delete the correct dir with attachments
		if path.Base(_attDir) != model.AttachmentsDir {
			return fmt.Errorf("internal error while deleting %s: not in .attachments directory: %s", id, attDir)
		}
		err = s3RemoveAll(ctx, s.client, s.bucket, _attDir)
		if err != nil {
			return err
		}
	}

	_, err = s.updateIndex(ctx, s.indexUpdaterForIds(id))
	return err
}

func s3Filenames(id string) (string, string) {
	dir := toS3Dir(path.Dir(id))
	base := path.Base(id)
	return dir, base
}

func (s *S3Repo) getExistingID(ctx context.Context, ids string) (idMatch, string) {
	dir, base := s3Filenames(ids)
	// try full repoName as given
	if _, err := s3Stat(ctx, s.client, s.bucket, ids); err == nil {
		return idMatchFull, ids
	}
	// try without timestamp
	entries, err := s3ListObjects(ctx, s.client, s.bucket, toS3Dir(dir))
	if err != nil {
		return idMatchNone, ""
	}
	version, err := model.ParseTMVersion(strings.TrimSuffix(base, TMExt))
	if err != nil {
		utils.GetLogger(ctx, "S3Repo").Warn("invalid TM version in TM id", "id", ids, "error", err)
		return idMatchNone, ""
	}
	idPrefix := strings.TrimSuffix(ids, base)
	existingTMVersions := findTMS3EntriesByBaseVersion(entries, version)
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

// findTMS3EntriesByBaseVersion finds directory entries that correspond to TM file names, converts those to TMVersions,
// filters out those that have a differing base version from the one given as argument, and sorts the remaining in
// descending order
func findTMS3EntriesByBaseVersion(infos []S3ObjectInfo, version model.TMVersion) []model.TMVersion {
	baseString := version.BaseString()
	var res []model.TMVersion
	for _, info := range infos {
		_, base := s3Filenames(info.Path)
		ver, err := model.ParseTMVersion(strings.TrimSuffix(base, TMExt))
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

func (s *S3Repo) Fetch(ctx context.Context, id string) (string, []byte, error) {

	_, err := model.ParseTMID(id)
	if err != nil {
		return "", nil, err
	}
	match, actualId := s.getExistingID(ctx, id)
	if match != idMatchFull && match != idMatchDigest {
		return "", nil, model.ErrTMNotFound
	}
	b, err := s3ReadObject(ctx, s.client, s.bucket, actualId)
	return actualId, b, err
}

func (s *S3Repo) Index(ctx context.Context, ids ...string) error {

	unlock, err := s.lockIndex(ctx)
	defer unlock()
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		_, err = s.updateIndex(ctx, s.fullIndexRebuild)
		return err
	}
	_, err = s.updateIndex(ctx, s.indexUpdaterForIds(ids...))
	return err
}

func (s *S3Repo) CheckIntegrity(ctx context.Context, filter model.ResourceFilter) (results []model.CheckResult, err error) {

	unlock, err := s.lockIndex(ctx)
	defer unlock()
	if err != nil {
		return nil, err
	}
	idx, err := s.readIndex(ctx)
	if err != nil {
		if errors.Is(err, ErrNoIndex) {
			return nil, nil
		}
		return nil, err
	}
	idx.Sort()

	res, err := s.verifyAllFilesAreIndexed(ctx, idx, filter)
	return res, err
}

func (s *S3Repo) Spec() model.RepoSpec {
	return s.spec
}

func (s *S3Repo) List(ctx context.Context, search *model.Filters) (model.SearchResult, error) {
	unlock, err := s.lockIndex(ctx)
	defer unlock()
	if err != nil {
		return model.SearchResult{}, err
	}

	idx, err := s.readIndex(ctx)
	if err != nil {
		return model.SearchResult{}, err
	}

	idx.Sort() // the index is supposed to be sorted on disk, but we don't trust external storage, hence we'll sort here one more time to be extra sure
	sr := model.NewIndexToFoundMapper(s.Spec().ToFoundSource()).ToSearchResult(*idx)
	filtered := &sr
	err = filtered.Filter(search)
	if err != nil {
		return model.SearchResult{}, err
	}
	return *filtered, nil
}

// readIndex reads the contents of the index file. Must be called after the lock is acquired with lockIndex()
func (s *S3Repo) readIndex(ctx context.Context) (*model.Index, error) {
	if s.idx != nil {
		return s.idx, nil
	}
	data, err := s3ReadObject(ctx, s.client, s.bucket, s.indexFilename())
	if err != nil {
		if errors.Is(err, ErrS3NotExists) {
			err = ErrNoIndex
		}
		return nil, err
	}

	var index model.Index
	err = json.Unmarshal(data, &index)
	if err == nil {
		s.idx = &index
	}
	return &index, err
}

func (s *S3Repo) indexFilename() string {
	return path.Join(RepoConfDir, IndexFilename)
}

func (s *S3Repo) Versions(ctx context.Context, name string) ([]model.FoundVersion, error) {
	name = strings.TrimSpace(name)
	res, err := s.List(ctx, &model.Filters{Name: name})
	if err != nil {
		return nil, err
	}

	if len(res.Entries) != 1 {
		err := fmt.Errorf("%w: %s", model.ErrTMNameNotFound, name)
		return nil, err
	}

	return res.Entries[0].Versions, nil
}

func (s *S3Repo) GetTMMetadata(ctx context.Context, tmID string) ([]model.FoundVersion, error) {
	id, err := model.ParseTMID(tmID)
	if err != nil {
		return nil, err
	}
	versions, err := s.Versions(ctx, id.Name)
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

func (s *S3Repo) ImportAttachment(ctx context.Context, container model.AttachmentContainerRef, attachment model.Attachment, content []byte, force bool) error {
	unlock, err := s.lockIndex(ctx)
	defer unlock()
	if err != nil {
		return err
	}

	attDir, err := s.prepareAttachmentOperation(ctx, container)
	if err != nil {
		return err
	}

	err = s.verifyAttachmentExistsInIndex(ctx, container, attachment.Name)
	if err == nil && !force {
		return ErrAttachmentExists
	}
	if err != nil && !errors.Is(err, model.ErrAttachmentNotFound) {
		return err
	}

	err = s3WriteObject(ctx, s.client, s.bucket, path.Join(attDir, attachment.Name), content)
	if err != nil {
		return err
	}

	_, err = s.updateIndex(ctx, s.indexUpdaterForImportAttachment(container, attachment, content))
	if err != nil {
		return err
	}

	return nil
}

// prepareAttachmentOperation prepares for a CRUD operation on attachments
// Must be called after the index lock has been acquired with lockIndex
func (s *S3Repo) prepareAttachmentOperation(ctx context.Context, ref model.AttachmentContainerRef) (string, error) {
	attDir, err := s.getAttachmentsDir(ref)
	if err != nil {
		return "", err
	}
	// use listAttachments to validate ref
	_, err = s.listAttachments(ctx, ref)
	if err != nil {
		return "", err
	}
	return attDir, nil
}

func (s *S3Repo) FetchAttachment(ctx context.Context, ref model.AttachmentContainerRef, attachmentName string) ([]byte, error) {
	unlock, err := s.lockIndex(ctx)
	defer unlock()
	if err != nil {
		return nil, err
	}

	attDir, err := s.prepareAttachmentOperation(ctx, ref)
	if err != nil {
		return nil, err
	}

	err = s.verifyAttachmentExistsInIndex(ctx, ref, attachmentName)
	if err != nil {
		return nil, err
	}

	file, err := s3ReadObject(ctx, s.client, s.bucket, path.Join(attDir, attachmentName))
	if errors.Is(err, ErrS3NotExists) {
		return nil, model.ErrAttachmentNotFound
	}
	return file, err
}

func (s *S3Repo) verifyAttachmentExistsInIndex(ctx context.Context, ref model.AttachmentContainerRef, attachmentName string) error {
	atts, err := s.listAttachments(ctx, ref)
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

func (s *S3Repo) DeleteAttachment(ctx context.Context, ref model.AttachmentContainerRef, attachmentName string) error {
	unlock, err := s.lockIndex(ctx)
	defer unlock()
	if err != nil {
		return err
	}

	attDir, err := s.prepareAttachmentOperation(ctx, ref)
	if err != nil {
		return err
	}

	err = s.verifyAttachmentExistsInIndex(ctx, ref, attachmentName)
	if err != nil {
		return err
	}

	err = s3RemoveObject(ctx, s.client, s.bucket, path.Join(attDir, attachmentName))
	if err != nil {
		return err
	}

	_, err = s.updateIndex(ctx, s.indexUpdaterForDeleteAttachment(ref, attachmentName))
	if err != nil {
		return err
	}

	return nil
}

// listAttachments returns the attachment list belonging to given tmNameOrId
// Returns ErrTMNotFound or ErrTMNameNotFound if ref is not present in this repository
// Must be called after the index lock has been acquired with lockIndex
func (s *S3Repo) listAttachments(ctx context.Context, ref model.AttachmentContainerRef) (*attachmentsContainer, error) {
	index, err := s.readIndex(ctx)
	if err != nil {
		return nil, err
	}
	return findAttachmentContainer(index, ref)
}

func (s *S3Repo) getS3FileNames(ctx context.Context, dir string) ([]string, error) {
	entries, err := s3ListObjects(ctx, s.client, s.bucket, dir)
	if err != nil && !errors.Is(err, ErrS3NotExists) {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		files = append(files, e.Name)
	}
	return files, nil
}

// getAttachmentsDir returns the directory where the attachments to the given tmNameOrId are stored
func (s *S3Repo) getAttachmentsDir(ref model.AttachmentContainerRef) (string, error) {
	return model.RelAttachmentsDir(ref)
}

func (s *S3Repo) updateIndex(ctx context.Context, updater indexUpdater) (*model.Index, error) {
	// Prepare data collection for logging stats
	start := time.Now()

	oldNames := s.readNamesFile(ctx)
	oldIndex, err := s.readIndex(ctx)
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
	s.idx = newIndex
	duration := time.Now().Sub(start)
	// Ignore error as we are sure our struct does not contain channel,
	// complex or function values that would throw an error.
	newIndexJson, _ := json.MarshalIndent(newIndex, "", "  ")
	err = s3WriteObject(ctx, s.client, s.bucket, s.indexFilename(), newIndexJson)
	if err != nil {
		return nil, err
	}
	err = s.writeNamesFile(ctx, names)
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Updated index with %d records in %s ", fileCount, duration.String())
	utils.GetLogger(ctx, "S3Repo").Debug(msg)

	return newIndex, nil
}

func (s *S3Repo) indexUpdaterForIds(ids ...string) indexUpdater {
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
			info, statErr := s3Stat(ctx, s.client, s.bucket, id)
			upd, id, nameDeleted, err := s.updateIndexWithFile(ctx, newIndex, id, info, statErr)
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
		err := s.reindexAttachments(ctx, updatedAttContainers, oldIndex, newIndex)
		if err != nil {
			return nil, nil, 0, err
		}

		return newIndex, newNames, fileCount, nil
	}

}
func (s *S3Repo) indexUpdaterForDeleteAttachment(ref model.AttachmentContainerRef, attName string) indexUpdater {
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
func (s *S3Repo) indexUpdaterForImportAttachment(ref model.AttachmentContainerRef, att model.Attachment, content []byte) indexUpdater {
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

func (s *S3Repo) fullIndexRebuild(ctx context.Context, oldIndex *model.Index, _ []string) (*model.Index, []string, int, error) {
	fileCount := 0
	updatedAttContainers := make(map[model.AttachmentContainerRef]struct{})
	newIndex := &model.Index{
		Meta: model.IndexMeta{Created: time.Now()},
		Data: []*model.IndexEntry{},
	}
	var names []string

	entries, err := s3ListObjects(ctx, s.client, s.bucket, "")
	for _, e := range entries {
		select {
		case <-ctx.Done():
			return nil, nil, 0, ctx.Err()
		default:
		}

		upd, id, _, err := s.updateIndexWithFile(ctx, newIndex, e.Path, &e, err)
		if err != nil {
			return nil, nil, 0, ctx.Err()
		}
		if upd {
			fileCount++
			names = append(names, id.Name)
			updatedAttContainers[model.NewTMIDAttachmentContainerRef(id.String())] = struct{}{}
			updatedAttContainers[model.NewTMNameAttachmentContainerRef(id.Name)] = struct{}{}
		}
	}

	if err != nil {
		return nil, nil, 0, err
	}
	err = s.reindexAttachments(ctx, updatedAttContainers, oldIndex, newIndex)
	if err != nil {
		return nil, nil, 0, err
	}

	return newIndex, names, fileCount, nil
}

func (s *S3Repo) reindexAttachments(ctx context.Context, containers map[model.AttachmentContainerRef]struct{}, oldIndex *model.Index, newIndex *model.Index) error {
	for ref := range containers {
		dir, _ := s.getAttachmentsDir(ref) // ref is sure to be valid
		nameAttachments, err := s.getS3FileNames(ctx, toS3Dir(dir))
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

func (s *S3Repo) updateIndexWithFile(ctx context.Context, idx *model.Index, id string, info *S3ObjectInfo, err error) (updated bool, indexedId model.TMID, deletedName string, errr error) {
	log := utils.GetLogger(ctx, "S3Repo")
	if errors.Is(err, ErrS3NotExists) {
		upd, name, err := idx.Delete(id)
		if err != nil {
			return false, model.TMID{}, "", err
		}
		return upd, model.TMID{}, name, nil
	}
	if err != nil {
		return false, model.TMID{}, "", err
	}
	if !strings.HasSuffix(info.Path, TMExt) {
		return false, model.TMID{}, "", nil
	}
	thingMeta, err := s.getThingMetadata(ctx, info.Path)
	if err != nil {
		log.Warn(fmt.Sprintf("failed to extract metadata from file %s: %v. The file will be excluded from index", info.Path, err))
		return false, model.TMID{}, "", nil
	}
	err = idx.Insert(&thingMeta.tm)
	if err != nil {
		log.Warn(fmt.Sprintf("failed to insert %s into index: %v. The file will be excluded from index", info.Path, err))
		return false, model.TMID{}, "", nil
	}
	return true, thingMeta.id, "", nil
}

func (s *S3Repo) lockIndex(ctx context.Context) (unlockFunc, error) {
	// todo: locking index not yet implemented for S3, compare with fs_repo.go!
	ctx, cancel := context.WithTimeout(ctx, indexLockTimeout)
	unlock := func() {
		cancel()
		s.idx = nil
	}
	return unlock, nil
}

func (s *S3Repo) readNamesFile(ctx context.Context) []string {
	lines, _ := s3ReadFileLines(ctx, s.client, s.bucket, path.Join(RepoConfDir, TmNamesFile))
	return lines
}

func (s *S3Repo) writeNamesFile(ctx context.Context, names []string) error {
	slices.Sort(names)
	names = slices.Compact(names)
	return s3WriteFileLines(ctx, s.client, s.bucket, path.Join(RepoConfDir, TmNamesFile), names)
}

func (s *S3Repo) readIgnoreFile(ctx context.Context) (*ignore.GitIgnore, error) {
	ignoreFileName := path.Join(RepoConfDir, TmIgnoreFile)
	_, err := s3Stat(ctx, s.client, s.bucket, ignoreFileName)
	if errors.Is(err, ErrS3NotExists) {
		err := s.writeDefaultIgnoreFile(ctx)
		if err != nil {
			return nil, err
		}
	}
	lines, err := s3ReadFileLines(ctx, s.client, s.bucket, ignoreFileName)
	if err != nil {
		return nil, err
	}
	gitIgnore := ignore.CompileIgnoreLines(lines...)
	return gitIgnore, nil
}

func (s *S3Repo) writeDefaultIgnoreFile(ctx context.Context) error {
	return s3WriteFileLines(ctx, s.client, s.bucket, path.Join(RepoConfDir, TmIgnoreFile), repoDefaultIgnore)
}

func (s *S3Repo) getThingMetadata(ctx context.Context, id string) (*thingMetadata, error) {
	data, err := s3ReadObject(ctx, s.client, s.bucket, id)
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

func (s *S3Repo) ListCompletions(ctx context.Context, kind string, args []string, toComplete string) ([]string, error) {
	switch kind {
	case CompletionKindNames:
		unlock, err := s.lockIndex(ctx)
		defer unlock()
		if err != nil {
			return nil, err
		}
		ns := s.readNamesFile(ctx)
		_, seg := longestPath(toComplete)
		names := namesToCompletions(ns, toComplete, seg+1)
		return names, nil
	case CompletionKindFetchNames:
		if strings.Contains(toComplete, "..") {
			err := fmt.Errorf("%w :no completions for name containing '..': %s", ErrInvalidCompletionParams, toComplete)
			return nil, err
		}

		name, _, _ := strings.Cut(toComplete, ":")

		entries, err := s3ListObjects(ctx, s.client, s.bucket, name)
		if err != nil {
			return nil, err
		}
		vm := make(map[string]struct{})
		for _, e := range entries {
			if strings.HasSuffix(e.Path, TMExt) {
				ver, err := model.ParseTMVersion(strings.TrimSuffix(e.Name, TMExt))
				if err != nil {
					utils.GetLogger(ctx, "S3Repo.ListCompletions").Debug(err.Error())
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
		unlock, err := s.lockIndex(ctx)
		defer unlock()
		if err != nil {
			return nil, err
		}
		names := s.readNamesFile(ctx)
		lPath, seg := longestPath(toComplete)
		comps := namesToCompletions(names, toComplete, seg+1)
		if _, found := slices.BinarySearch(names, lPath); found { // current toComplete is a full TM name plus '/'
			// append versions to comps
			idx, err := s.readIndex(ctx)
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
		return getAttachmentCompletions(ctx, args, s)
	default:
		return nil, ErrInvalidCompletionParams
	}
}

func (s *S3Repo) verifyAllFilesAreIndexed(ctx context.Context, idx *model.Index, filter model.ResourceFilter) ([]model.CheckResult, error) {
	if filter == nil {
		filter = func(_ string) bool { return true }
	}
	ignor, err := s.prepareIgnoreFunc(ctx)
	if err != nil {
		return nil, err
	}

	var results []model.CheckResult

	entries, err := s3ListObjects(ctx, s.client, s.bucket, "")
	for _, e := range entries {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		if ignor(e.Path) || !filter(e.Path) {
			continue
		}
		checkResult := s.verifyFileIsIndexed(e.Path, idx)
		results = append(results, checkResult)
	}

	return results, err
}

func (s *S3Repo) verifyFileIsIndexed(file string, idx *model.Index) model.CheckResult {
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

func (s *S3Repo) prepareIgnoreFunc(ctx context.Context) (func(string) bool, error) {
	gitIgnore, err := s.readIgnoreFile(ctx)
	if err != nil {
		return nil, err
	}
	return func(s string) bool {
		return gitIgnore.MatchesPath(s)
	}, nil
}

type S3ObjectInfo struct {
	Path string
	Name string
}

func s3Stat(ctx context.Context, client S3Client, bucket string, objectKey string) (*S3ObjectInfo, error) {
	_, err := client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(bucket), Key: &objectKey})

	if err != nil {
		utils.GetLogger(ctx, "S3Repo").Warn("failed to stat object from S3", "object", objectKey, "bucket", bucket, "error", err.Error())

		var ae smithy.APIError
		var oe *smithy.OperationError

		switch true {
		case errors.As(err, &ae) && ae.ErrorCode() == "NotFound":
			return nil, fmt.Errorf("%w, object: %s error: %s", ErrS3NotExists, objectKey, err.Error())
		case errors.As(err, &oe):
			return nil, fmt.Errorf("%w, object: %s error: %s", ErrS3Op, objectKey, err.Error())
		default:
			return nil, fmt.Errorf("%w %s", ErrS3Unknown, err.Error())
		}
	}

	return &S3ObjectInfo{
		Path: objectKey,
		Name: path.Base(objectKey),
	}, nil
}

func s3WriteObject(ctx context.Context, client S3Client, bucket string, objectKey string, data []byte) error {
	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
		Body:   bytes.NewReader(data),
	})

	if err != nil {
		utils.GetLogger(ctx, "S3Repo").Warn("failed to write object to S3", "object", objectKey, "bucket", bucket, "error", err.Error())

		var oe *smithy.OperationError
		if errors.As(err, &oe) {
			return fmt.Errorf("%w, object: %s error: %s", ErrS3Op, objectKey, err.Error())
		}
		return fmt.Errorf("%w, object: %s error: %s", ErrS3Unknown, objectKey, err.Error())
	}

	msg := fmt.Sprintf("object %s successfully written to S3: bucket %s", objectKey, bucket)
	utils.GetLogger(ctx, "S3Repo").Debug(msg)

	return nil
}

func s3ReadObject(ctx context.Context, client S3Client, bucket string, objectKey string) ([]byte, error) {

	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		utils.GetLogger(ctx, "S3Repo").Warn("failed to read object from S3", "object", objectKey, "bucket", bucket, "error", err.Error())

		var noKey *types.NoSuchKey
		var oe *smithy.OperationError

		switch true {
		case errors.As(err, &noKey):
			return nil, fmt.Errorf("%w, object: %s error: %s", ErrS3NotExists, objectKey, err.Error())
		case errors.As(err, &oe):
			return nil, fmt.Errorf("%w, object: %s error: %s", ErrS3Op, objectKey, err.Error())
		default:
			return nil, fmt.Errorf("%w, object: %s error: %s", ErrS3Unknown, objectKey, err.Error())
		}
	}

	defer result.Body.Close()

	b, err := io.ReadAll(result.Body)

	return b, err
}

func s3ListObjects(ctx context.Context, client S3Client, bucket, objectPrefix string) ([]S3ObjectInfo, error) {

	output, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: &objectPrefix,
	})

	if err != nil {
		utils.GetLogger(ctx, "S3Repo").Warn("failed to list objects from S3", "bucket", bucket, "error", err.Error())

		var opErr *smithy.OperationError
		if errors.As(err, &opErr) {
			return nil, fmt.Errorf("%w %s", ErrS3Op, err.Error())
		}
		return nil, fmt.Errorf("%w %s", ErrS3Unknown, err.Error())
	}

	var infos []S3ObjectInfo
	for _, o := range output.Contents {
		if !strings.HasPrefix(path.Dir(*o.Key), "") {
			continue
		}

		infos = append(infos, S3ObjectInfo{
			Path: *o.Key,
			Name: path.Base(*o.Key),
		})
	}
	return infos, err
}

func s3RemoveObject(ctx context.Context, client S3Client, bucket string, objectKey string) error {
	_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    &objectKey,
	})

	if err != nil {
		utils.GetLogger(ctx, "S3Repo").Warn("failed to remove object from S3", "object", objectKey, "bucket", bucket, "error", err.Error())

		var opErr *smithy.OperationError
		if errors.As(err, &opErr) {
			return fmt.Errorf("%w, object: %s error: %s", ErrS3Op, objectKey, err.Error())
		}
		return fmt.Errorf("%w, object: %s error: %s", ErrS3Unknown, objectKey, err.Error())
	}
	return nil
}

func s3RemoveAll(ctx context.Context, client S3Client, bucket string, objectPrefix string) error {

	files, err := s3ListObjects(ctx, client, bucket, objectPrefix)
	if err != nil {
		return err
	}

	var objectIds []types.ObjectIdentifier
	for _, o := range files {
		objectIds = append(objectIds, types.ObjectIdentifier{Key: &o.Path})
	}

	if len(objectIds) > 0 {
		_, err = client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &types.Delete{Objects: objectIds},
		})

		if err != nil {
			utils.GetLogger(ctx, "S3Repo").Warn("failed to remove objects from S3", "objectPrefix", objectPrefix, "bucket", bucket, "error", err.Error())

			var opErr *smithy.OperationError
			if errors.As(err, &opErr) {
				return fmt.Errorf("%w, objectPrefix: %s error: %s", ErrS3Op, objectPrefix, err.Error())
			}
			return fmt.Errorf("%w, objectPrefix: %s error: %s", ErrS3Unknown, objectPrefix, err.Error())
		}
	}
	return nil
}

func s3WriteFileLines(ctx context.Context, client S3Client, bucket string, fileName string, lines []string) error {
	buf := bytes.NewBuffer(nil)
	for _, line := range lines {
		_, err := fmt.Fprintln(buf, line)
		if err != nil {
			return err
		}
	}
	return s3WriteObject(ctx, client, bucket, fileName, buf.Bytes())
}

func s3ReadFileLines(ctx context.Context, client S3Client, bucket string, fileName string) ([]string, error) {
	b, err := s3ReadObject(ctx, client, bucket, fileName)
	if err != nil {
		return nil, err
	}

	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func toS3Dir(objectKey string) string {
	if strings.HasSuffix(objectKey, "/") {
		return objectKey
	} else {
		return objectKey + "/"
	}
}
