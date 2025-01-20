package repos

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/testutils"
	"github.com/wot-oss/tmc/internal/testutils/s3mocks"
	"github.com/wot-oss/tmc/internal/utils"
)

var bucket = "abucket"

func TestCreateS3RepoConfig(t *testing.T) {

	tests := []struct {
		fileConf string
		expErr   bool
		expDescr string
	}{
		{`{"type":"s33", "aws_bucket":"abucket"}`, true, ""},
		{`{"aws_bucket":"abucket"}`, false, ""},
		{`{"aws_bucket":"abucket", "description":"some description"}`, false, "some description"},
		{`{"aws_bucket":"abucket", "aws_access_key_id":"some access key", "aws_secret_access_key":"some secret"}`, false, ""},
		{`{"aws_bucket":"abucket", "aws_access_key_id":"some access key"}`, true, ""},
		{`{"aws_bucket":"abucket", "aws_secret_access_key":"some secret"}`, true, ""},

		{``, true, ""},
		{`[]`, true, ""},
		{`{}`, true, ""},
	}

	for i, test := range tests {
		cf, err := createS3RepoConfig([]byte(test.fileConf))
		if test.expErr {
			assert.Error(t, err, "error expected in test %d for %s", i, test.fileConf)
			continue
		} else {
			assert.NoError(t, err, "no error expected in test %d for %s", i, test.fileConf)
		}
		assert.Equalf(t, "s3", cf[KeyRepoType], "in test %d for %s", i, test.fileConf)
		if test.expDescr != "" {
			assert.Equal(t, test.expDescr, cf[KeyRepoDescription], "in test %d for %s", i, test.fileConf)
		} else {
			assert.Nil(t, cf[KeyRepoDescription], "in test %d for %s", i, test.fileConf)
		}
	}
}

func TestNewS3Repo(t *testing.T) {
	awsBucket := "abucket"
	awsRegion := "eu-central-1"
	awsAk := "some_access_key"
	awsSk := "some_secret_key"
	spec := model.NewRepoSpec("mys3")

	t.Run(" with minimal config (e.g. if AWS IAM role is provided in environment)", func(t *testing.T) {
		repo, err := NewS3Repo(map[string]any{
			KeyRepoType:      RepoTypeS3,
			KeyRepoAWSBucket: awsBucket,
		}, spec)
		assert.NoError(t, err)
		assert.Equal(t, awsBucket, repo.bucket)
		assert.Equal(t, spec, repo.Spec())
	})
	t.Run("with full config", func(t *testing.T) {
		repo, err := NewS3Repo(map[string]any{
			KeyRepoType:               RepoTypeS3,
			KeyRepoAWSBucket:          awsBucket,
			KeyRepoAWSRegion:          awsRegion,
			KeyRepoAWSAccessKeyId:     awsAk,
			KeyRepoAWSSecretAccessKey: awsSk,
		}, spec)
		assert.NoError(t, err)
		assert.Equal(t, awsBucket, repo.bucket)
		assert.Equal(t, awsRegion, repo.region)
		assert.Equal(t, awsRegion, repo.client.Options().Region)
		cred, err := repo.client.Options().Credentials.Retrieve(context.TODO())
		assert.NoError(t, err)
		assert.Equal(t, awsAk, cred.AccessKeyID)
		assert.Equal(t, awsSk, cred.SecretAccessKey)
		assert.Equal(t, spec, repo.Spec())
	})
	t.Run("with full config via environment variables", func(t *testing.T) {
		os.Setenv("MY_AWS_BUCKET", awsBucket)
		os.Setenv("MY_AWS_REGION", awsRegion)
		os.Setenv("MY_AWS_AK", awsAk)
		os.Setenv("MY_AWS_SK", awsSk)

		defer func() {
			os.Unsetenv("MY_AWS_BUCKET")
			os.Unsetenv("MY_AWS_REGION")
			os.Unsetenv("MY_AWS_AK")
			os.Unsetenv("MY_AWS_SK")
		}()

		repo, err := NewS3Repo(map[string]any{
			KeyRepoType:               RepoTypeS3,
			KeyRepoAWSBucket:          "$MY_AWS_BUCKET",
			KeyRepoAWSRegion:          "$MY_AWS_REGION",
			KeyRepoAWSAccessKeyId:     "$MY_AWS_AK",
			KeyRepoAWSSecretAccessKey: "$MY_AWS_SK",
		}, spec)
		assert.NoError(t, err)
		assert.Equal(t, awsBucket, repo.bucket)
		assert.Equal(t, awsRegion, repo.region)
		assert.Equal(t, awsRegion, repo.client.Options().Region)
		cred, err := repo.client.Options().Credentials.Retrieve(context.TODO())
		assert.NoError(t, err)
		assert.Equal(t, awsAk, cred.AccessKeyID)
		assert.Equal(t, awsSk, cred.SecretAccessKey)
		assert.Equal(t, spec, repo.Spec())
	})
	t.Run("with invalid config", func(t *testing.T) {
		_, err := NewS3Repo(map[string]any{
			KeyRepoType: RepoTypeS3,
		}, spec)
		assert.Error(t, err)
		assert.ErrorContains(t, err, KeyRepoAWSBucket)
	})
	t.Run("with invalid credentials pair", func(t *testing.T) {
		_, err := NewS3Repo(map[string]any{
			KeyRepoType:           RepoTypeS3,
			KeyRepoAWSBucket:      awsBucket,
			KeyRepoAWSAccessKeyId: awsAk,
		}, spec)
		assert.Error(t, err)
		assert.ErrorContains(t, err, KeyRepoAWSAccessKeyId)
		assert.ErrorContains(t, err, KeyRepoAWSSecretAccessKey)

		_, err = NewS3Repo(map[string]any{
			KeyRepoType:               RepoTypeS3,
			KeyRepoAWSBucket:          awsBucket,
			KeyRepoAWSSecretAccessKey: awsSk,
		}, spec)
		assert.Error(t, err)
		assert.ErrorContains(t, err, KeyRepoAWSAccessKeyId)
		assert.ErrorContains(t, err, KeyRepoAWSSecretAccessKey)
	})
}

func TestS3Repo_CanonicalRoot(t *testing.T) {
	tests := []struct {
		repo    S3Repo
		expRoot string
	}{
		{S3Repo{region: "eu-central-1", bucket: "abucket"}, "eu-central-1abucket"},
		{S3Repo{region: "", bucket: "abucket"}, "abucket"},
	}

	for i, test := range tests {
		cr := test.repo.CanonicalRoot()
		assert.Equal(t, test.expRoot, cr, "in test %d", i)
	}
}

func TestS3Repo_ListByName(t *testing.T) {
	const tmName = "omnicorp-tm-department/omnicorp/omnilamp"
	_, idx, err := utils.ReadRequiredFile("../../test/data/repos/file/attachments/.tmc/tm-catalog.toc.json")
	assert.NoError(t, err)

	c := s3mocks.NewS3Client(t)
	c.On("GetObject", mock.Anything, mock.Anything).Return(&s3.GetObjectOutput{Body: io.NopCloser(bytes.NewBuffer(idx))}, nil)
	r := S3Repo{bucket: bucket, client: c}

	res, err := r.List(context.Background(), &model.Filters{Name: tmName})
	assert.NoError(t, err)
	if assert.Len(t, res.Entries, 1) {
		if assert.Len(t, res.Entries[0].Attachments, 1) {
			assert.Equal(t, "README.md", res.Entries[0].Attachments[0].Name)
		}
	}
}

func TestS3Repo_Versions(t *testing.T) {
	const tmName = "omnicorp-tm-department/omnicorp/omnilamp"
	_, idx, err := utils.ReadRequiredFile("../../test/data/list/tm-catalog.toc.json")
	assert.NoError(t, err)

	c := s3mocks.NewS3Client(t)
	r := S3Repo{bucket: bucket, client: c}
	ctx := context.Background()

	c.On("GetObject", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewBuffer(idx))}, nil
		})

	vers, err := r.Versions(ctx, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/a/b")
	assert.NoError(t, err)
	assert.Len(t, vers, 1)

	vers, err = r.Versions(ctx, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/subpath")
	assert.NoError(t, err)
	assert.Len(t, vers, 1)

	vers, err = r.Versions(ctx, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall")
	assert.NoError(t, err)
	assert.Len(t, vers, 1)

	vers, err = r.Versions(ctx, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/nothing-here")
	assert.ErrorIs(t, err, model.ErrTMNameNotFound)

	vers, err = r.Versions(ctx, "")
	assert.ErrorIs(t, err, model.ErrTMNameNotFound)
}

func TestS3Repo_GetTMMetadata(t *testing.T) {
	const tmID = "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20240409155220-3f779458e453.tm.json"

	_, idx, err := utils.ReadRequiredFile("../../test/data/repos/file/attachments/.tmc/tm-catalog.toc.json")
	assert.NoError(t, err)

	c := s3mocks.NewS3Client(t)
	c.On("GetObject", mock.Anything, mock.Anything).Return(&s3.GetObjectOutput{Body: io.NopCloser(bytes.NewBuffer(idx))}, nil)

	r := S3Repo{bucket: bucket, client: c}

	res, err := r.GetTMMetadata(context.Background(), tmID)
	assert.NoError(t, err)
	if assert.Len(t, res, 1) {
		if assert.Len(t, res[0].Attachments, 1) {
			assert.Equal(t, "cfg.json", res[0].Attachments[0].Name)
		}
	}
}

func TestS3Repo_Fetch(t *testing.T) {
	ctx := context.Background()

	tmName := "omnicorp-tm-department/omnicorp/omnilamp/"
	fileA := []byte("{\"ver\":\"a\"}")
	fileB := []byte("{\"ver\":\"b\"}")

	idA := tmName + "v1.0.0-20231208142856-a49617d2e4fc.tm.json"
	idAfail := tmName + "v1.0.0-20231212142856-a49617d2e4fc.tm.json"
	idB := tmName + "v1.0.0-20231207142856-b49617d2e4fc.tm.json"
	idC := tmName + "v1.2.1-20231209142856-c49617d2e4fc.tm.json"
	idD := tmName + "v0.0.1-20231208142856-d49617d2e4fc.tm.json"
	idEfail := tmName + "v1.0.0-20231212142856-e49617d2e4fc.tm.json"

	s3ListResponse := s3.ListObjectsV2Output{Contents: []types.Object{
		{Key: &idA}, {Key: &idB}, {Key: &idC}, {Key: &idD},
	}}
	s3NotFoundErr := smithy.GenericAPIError{Code: "NotFound", Message: "Object not found"}

	t.Run("with concrete ThingModels found", func(t *testing.T) {
		c := s3mocks.NewS3Client(t)
		c.On("GetObject", mock.Anything, &s3.GetObjectInput{Bucket: &bucket, Key: &idA}).
			Return(&s3.GetObjectOutput{Body: io.NopCloser(bytes.NewBuffer(fileA))}, nil)
		c.On("GetObject", mock.Anything, &s3.GetObjectInput{Bucket: &bucket, Key: &idB}).
			Return(&s3.GetObjectOutput{Body: io.NopCloser(bytes.NewBuffer(fileB))}, nil)
		c.On("HeadObject", mock.Anything, &s3.HeadObjectInput{Bucket: &bucket, Key: &idA}).Return(nil, nil)
		c.On("HeadObject", mock.Anything, &s3.HeadObjectInput{Bucket: &bucket, Key: &idB}).Return(nil, nil)

		r := S3Repo{bucket: bucket, client: c}

		actId, content, err := r.Fetch(ctx, idA)
		assert.NoError(t, err)
		assert.Equal(t, idA, actId)
		assert.Equal(t, fileA, content)

		actId, content, err = r.Fetch(ctx, idB)
		assert.NoError(t, err)
		assert.Equal(t, idB, actId)
		assert.Equal(t, fileB, content)
	})

	t.Run("with fallback to older timestamp of ThingModel", func(t *testing.T) {

		c := s3mocks.NewS3Client(t)

		c.On("HeadObject", mock.Anything, &s3.HeadObjectInput{Bucket: &bucket, Key: &idAfail}).
			Return(nil, &s3NotFoundErr)
		c.On("ListObjectsV2", mock.Anything, &s3.ListObjectsV2Input{Bucket: &bucket, Prefix: &tmName}).
			Return(&s3ListResponse, nil)
		c.On("GetObject", mock.Anything, &s3.GetObjectInput{Bucket: &bucket, Key: &idA}).
			Return(&s3.GetObjectOutput{Body: io.NopCloser(bytes.NewBuffer(fileA))}, nil)

		r := S3Repo{bucket: bucket, client: c}
		actId, content, err := r.Fetch(ctx, idAfail)
		assert.NoError(t, err)
		assert.Equal(t, idA, actId)
		assert.Equal(t, fileA, content)
	})

	t.Run("with ThingModel not found", func(t *testing.T) {
		c := s3mocks.NewS3Client(t)

		c.On("HeadObject", mock.Anything, &s3.HeadObjectInput{Bucket: &bucket, Key: &idEfail}).
			Return(nil, &s3NotFoundErr)
		c.On("ListObjectsV2", mock.Anything, &s3.ListObjectsV2Input{Bucket: &bucket, Prefix: &tmName}).
			Return(&s3ListResponse, nil)

		r := S3Repo{bucket: bucket, client: c}
		actId, _, err := r.Fetch(ctx, idEfail)
		assert.ErrorIs(t, err, model.ErrTMNotFound)
		assert.Equal(t, "", actId)
	})
}

func TestS3Repo_Import(t *testing.T) {
	temp, _ := os.MkdirTemp("", "s3r")
	defer os.RemoveAll(temp)

	// given: S3 repo with ThingModels
	c := getS3Mock(t, temp)
	r := S3Repo{bucket: bucket, client: c}
	ctx := context.Background()

	tmName := "omnicorp-tm-department/omnicorp/omnilamp"
	id := tmName + "/v0.0.0-20231208142856-c49617d2e4fc.tm.json"
	_, err := r.Import(ctx, model.MustParseTMID(id), []byte{}, ImportOptions{})
	assert.Error(t, err)
	_, err = r.Import(ctx, model.MustParseTMID(id), []byte("{}"), ImportOptions{})
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(temp, toBucketObject(id)))

	_ = os.WriteFile(filepath.Join(temp, toBucketObject(tmName, "v1.0.0-20231208142856-a49617d2e4fc.tm.json")), []byte("{}"), defaultFilePermissions)
	_ = os.WriteFile(filepath.Join(temp, toBucketObject(tmName, "v1.0.0-20231207142856-b49617d2e4fc.tm.json")), []byte("{}"), defaultFilePermissions)
	_ = os.WriteFile(filepath.Join(temp, toBucketObject(tmName, "v1.2.1-20231209142856-c49617d2e4fc.tm.json")), []byte("{}"), defaultFilePermissions)
	_ = os.WriteFile(filepath.Join(temp, toBucketObject(tmName, "v0.0.1-20231208142856-d49617d2e4fc.tm.json")), []byte("{}"), defaultFilePermissions)

	id2 := "omnicorp-tm-department/omnicorp/omnilamp/v1.0.0-20231219123456-a49617d2e4fc.tm.json"
	res, err := r.Import(ctx, model.MustParseTMID(id2), []byte("{}"), ImportOptions{})
	expCErr := &ErrTMIDConflict{Type: IdConflictSameContent, ExistingId: "omnicorp-tm-department/omnicorp/omnilamp/v1.0.0-20231208142856-a49617d2e4fc.tm.json"}
	assert.Equal(t, expCErr, err)
	assert.Equal(t, ImportResult{Type: ImportResultError, Message: expCErr.Error(), Err: expCErr}, res)

	id3 := "omnicorp-tm-department/omnicorp/omnilamp/v1.0.0-20231219123456-f49617d2e4fc.tm.json"
	_, err = r.Import(ctx, model.MustParseTMID(id3), []byte("{}"), ImportOptions{})
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(temp, toBucketObject(id3)))

	id4 := "omnicorp-tm-department/omnicorp/omnilamp/v1.0.0-20231219123456-049617d2e4fc.tm.json"
	res, err = r.Import(ctx, model.MustParseTMID(id4), []byte("{\"val\":1}"), ImportOptions{})
	assert.NoError(t, err)
	expCErr = &ErrTMIDConflict{Type: IdConflictSameTimestamp, ExistingId: id3}
	assert.Equal(t, ImportResult{Type: ImportResultWarning, TmID: id4, Message: expCErr.Error(), Err: expCErr}, res)
}

func TestS3Repo_Delete(t *testing.T) {
	temp, _ := os.MkdirTemp("", "s3r")
	defer os.RemoveAll(temp)
	assert.NoError(t, prepareS3MockBucket("../../test/data/index", temp))

	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	_ = os.Chdir(temp)

	// given: S3 repo with ThingModels
	c := getS3Mock(t, temp)
	r := S3Repo{bucket: bucket, client: c}
	ctx := context.Background()
	// and given: the repo has an index
	assert.NoError(t, r.Index(ctx))

	t.Run("invalid id", func(t *testing.T) {
		err := r.Delete(ctx, "invalid-id")
		assert.ErrorIs(t, err, model.ErrInvalidId)
	})
	t.Run("non-existent id", func(t *testing.T) {
		err := r.Delete(ctx, "auth/man/mpn/v1.0.1-20231024121314-abcd12345679.tm.json")
		assert.ErrorIs(t, err, model.ErrTMNotFound)
	})
	t.Run("hash matching id", func(t *testing.T) {
		err := r.Delete(ctx, "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20230101125023-e414b33a9edf.tm.json")
		assert.ErrorIs(t, err, model.ErrTMNotFound)
	})
	t.Run("existing id with attachment", func(t *testing.T) {
		id1 := "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20240409155220-e414b33a9edf.tm.json"
		id1Attachment := "omnicorp-tm-department/omnicorp/omnilamp/.attachments/v0.0.0-20240409155220-e414b33a9edf/manual.txt"
		id2 := "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v0.0.0-20240409155220-80424c65e4e6.tm.json"
		id3 := "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v3.2.1-20240409155220-3f779458e453.tm.json"
		id4 := "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20240409155220-3f779458e453.tm.json"
		id5 := "omnicorp-tm-department/omnicorp/omnilamp/v3.11.1-20240409155220-da7dbd7ed830.tm.json"

		// verify id1 and attachment really exists
		_, err := os.Stat(filepath.Join(temp, toBucketObject(id1)))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(temp, toBucketObject(id1Attachment)))
		assert.NoError(t, err)

		// when: deleting TM id1
		err = r.Delete(ctx, id1)
		// then: there is no error
		assert.NoError(t, err)
		// and then: TM id1 is really deleted
		_, err = os.Stat(filepath.Join(temp, toBucketObject(id1)))
		assert.True(t, os.IsNotExist(err))
		// and then: attachment of TM id1 is really deleted
		_, err = os.Stat(filepath.Join(temp, toBucketObject(id1Attachment)))
		assert.True(t, os.IsNotExist(err))

		// and then: remaining TMS are not deleted
		_, err = os.Stat(filepath.Join(temp, toBucketObject(id2)))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(temp, toBucketObject(id3)))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(temp, toBucketObject(id4)))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(temp, toBucketObject(id5)))
		assert.NoError(t, err)
	})
}

func TestS3Repo_Index(t *testing.T) {
	temp, _ := os.MkdirTemp("", "s3r")
	defer os.RemoveAll(temp)
	assert.NoError(t, prepareS3MockBucket("../../test/data/index", temp))

	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	_ = os.Chdir(temp)

	// given: S3 repo with ThingModels and no index
	c := getS3Mock(t, temp)
	r := S3Repo{bucket: bucket, client: c}
	ctx := context.Background()

	t.Run("single id/no index file", func(t *testing.T) {
		// when: index the repo with a single id
		err := r.Index(ctx, "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v0.0.0-20240409155220-80424c65e4e6.tm.json")
		// then: there is no error
		assert.NoError(t, err)
		// and then: the index is created
		idx, err := r.readIndex(ctx)
		assert.NoError(t, err)
		zeroTime := time.Time{}
		assert.True(t, idx.Meta.Created.After(zeroTime))
		// and then: index contains one ThingModel
		assert.Equal(t, 1, len(idx.Data))
		assert.Equal(t, "omnicorp-tm-department/omnicorp/omnilamp/subfolder", idx.Data[0].Name)
		// and then: the ThingModel contains one version
		assert.Equal(t, 1, len(idx.Data[0].Versions))
		assert.Equal(t, "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v0.0.0-20240409155220-80424c65e4e6.tm.json", idx.Data[0].Versions[0].TMID)
		// and then: the names file is created
		names := r.readNamesFile(ctx)
		assert.Equal(t, []string{"omnicorp-tm-department/omnicorp/omnilamp/subfolder"}, names)
	})

	t.Run("single id/existing index file", func(t *testing.T) {
		// when: index the repo with another version
		err := r.Index(ctx, "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v3.2.1-20240409155220-3f779458e453.tm.json")
		// then: there is no error
		assert.NoError(t, err)
		// and then: the index is readable
		idx, err := r.readIndex(ctx)
		assert.NoError(t, err)
		// and then: index contains one ThingModel
		assert.Equal(t, 1, len(idx.Data))
		assert.Equal(t, "omnicorp-tm-department/omnicorp/omnilamp/subfolder", idx.Data[0].Name)
		// and then: the ThingModel contains two versions
		assert.Equal(t, 2, len(idx.Data[0].Versions))
		// and then: the names file is readable
		names := r.readNamesFile(ctx)
		assert.Equal(t, []string{"omnicorp-tm-department/omnicorp/omnilamp/subfolder"}, names)
	})

	t.Run("full update/existing index file", func(t *testing.T) {
		// when: full index the repo with all found ThingModels in the repo
		err := r.Index(ctx)
		// then: there is no error
		assert.NoError(t, err)
		// and then: the index is readable
		idx, err := r.readIndex(ctx)
		assert.NoError(t, err)
		// and then: index contains now two ThingModels
		assert.Equal(t, 2, len(idx.Data))
		// and then: the names file is readable
		names := r.readNamesFile(ctx)
		assert.Equal(t, []string{
			"omnicorp-tm-department/omnicorp/omnilamp",
			"omnicorp-tm-department/omnicorp/omnilamp/subfolder",
		}, names)
		// and then: the new-found ThingModel has 3 versions and an attachment
		entry := idx.FindByName("omnicorp-tm-department/omnicorp/omnilamp")
		assert.NotNil(t, entry)
		if assert.Len(t, entry.Versions, 3) {
			assert.Equal(t, []model.Attachment{{Name: "manual.txt", MediaType: "text/plain; charset=utf-8"}}, entry.Versions[2].Attachments)
			assert.Equal(t, []string{"coaps", "https"}, entry.Versions[2].Protocols)
		}
	})

	t.Run("full update/no index file", func(t *testing.T) {
		// given: repo has no index file and names file
		err := os.Remove(toBucketObject(r.indexFilename()))
		assert.NoError(t, err)
		assert.NoError(t, r.writeNamesFile(ctx, nil))

		// when: full index the repo with all found ThingModels in the repo
		err = r.Index(ctx)
		// then: there is no error
		assert.NoError(t, err)
		// and then: the index is readable
		idx, err := r.readIndex(ctx)
		assert.NoError(t, err)
		// and then: index contains now two ThingModels
		assert.Equal(t, 2, len(idx.Data))
		names := r.readNamesFile(ctx)
		assert.Equal(t, []string{
			"omnicorp-tm-department/omnicorp/omnilamp",
			"omnicorp-tm-department/omnicorp/omnilamp/subfolder",
		}, names)
	})

	t.Run("single tm id indexes tm name attachments", func(t *testing.T) {
		tmName := "omnicorp-tm-department/omnicorp/omnilamp/subfolder"
		attPath := filepath.Join(temp, toBucketObject(tmName, model.AttachmentsDir, "README.txt"))
		assert.NoError(t, os.WriteFile(attPath, []byte("Read This, or Else"), defaultFilePermissions))

		// when: index the repo with a single id
		err := r.Index(ctx, "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v3.2.1-20240409155220-3f779458e453.tm.json")
		// then: there is no error
		assert.NoError(t, err)
		// and then: the index is readable
		idx, err := r.readIndex(ctx)
		assert.NoError(t, err)
		// and then: the attachment for the TM name can be found in the index
		entry := idx.FindByName(tmName)
		assert.NotNil(t, entry)
		assert.Equal(t, []model.Attachment{{Name: "README.txt", MediaType: "text/plain; charset=utf-8"}}, entry.Attachments)
	})

	t.Run("single id's/index must be sorted", func(t *testing.T) {
		err := os.Remove(toBucketObject(r.indexFilename()))
		assert.NoError(t, err)
		assert.NoError(t, r.writeNamesFile(ctx, nil))

		tmName1 := "omnicorp-tm-department/omnicorp/omnilamp"
		tmId11 := "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20240409155220-e414b33a9edf.tm.json"
		tmId12 := "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20240409155220-3f779458e453.tm.json"
		tmId13 := "omnicorp-tm-department/omnicorp/omnilamp/v3.11.1-20240409155220-da7dbd7ed830.tm.json"

		tmName2 := "omnicorp-tm-department/omnicorp/omnilamp/subfolder"
		tmId21 := "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v0.0.0-20240409155220-80424c65e4e6.tm.json"
		tmId22 := "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v3.2.1-20240409155220-3f779458e453.tm.json"

		// update index with unordered ID's
		err = r.Index(ctx, tmId21, tmId12, tmId22, tmId13, tmId11)
		assert.NoError(t, err)

		idx, err := r.readIndex(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(idx.Data))

		assert.Equal(t, tmName1, idx.Data[0].Name)
		assert.Equal(t, tmId13, idx.Data[0].Versions[0].TMID)
		assert.Equal(t, tmId12, idx.Data[0].Versions[1].TMID)
		assert.Equal(t, tmId11, idx.Data[0].Versions[2].TMID)
		assert.Equal(t, tmName2, idx.Data[1].Name)
		assert.Equal(t, tmId22, idx.Data[1].Versions[0].TMID)
		assert.Equal(t, tmId21, idx.Data[1].Versions[1].TMID)
	})
}

func TestS3Repo_FetchAttachment(t *testing.T) {
	temp, _ := os.MkdirTemp("", "s3r")
	defer os.RemoveAll(temp)
	assert.NoError(t, prepareS3MockBucket("../../test/data/repos/file/attachments", temp))

	c := getS3Mock(t, temp)
	r := S3Repo{bucket: bucket, client: c}
	ctx := context.Background()

	tmName := "omnicorp-tm-department/omnicorp/omnilamp"
	fileA, _ := os.ReadFile("../../test/data/repos/file/attachments/omnicorp-tm-department/omnicorp/omnilamp/.attachments/README.md")
	fileB, _ := os.ReadFile("../../test/data/repos/file/attachments/omnicorp-tm-department/omnicorp/omnilamp/.attachments/v3.2.1-20240409155220-3f779458e453/cfg.json")
	idA := tmName + "/v3.2.1-20240409155220-3f779458e453.tm.json"
	baseNameA := "README.md"
	baseNameB := "cfg.json"

	t.Run("tm name attachment", func(t *testing.T) {
		content, err := r.FetchAttachment(ctx, model.NewTMNameAttachmentContainerRef(tmName), baseNameA)
		assert.NoError(t, err)
		assert.Equal(t, fileA, content)
	})
	t.Run("tm id attachment", func(t *testing.T) {
		content, err := r.FetchAttachment(ctx, model.NewTMIDAttachmentContainerRef(idA), baseNameB)
		assert.NoError(t, err)
		assert.Equal(t, fileB, content)
	})
	t.Run("non existent attachment", func(t *testing.T) {
		_, err := r.FetchAttachment(ctx, model.NewTMNameAttachmentContainerRef(tmName), "nothing-here")
		assert.ErrorIs(t, err, model.ErrAttachmentNotFound)
	})
	t.Run("non existent tm name", func(t *testing.T) {
		_, err := r.FetchAttachment(ctx, model.NewTMNameAttachmentContainerRef("omnicorp-tm-department/omnicorp/omnidarkness"), baseNameA)
		assert.ErrorIs(t, err, model.ErrTMNameNotFound)
	})
	t.Run("non existent tm id", func(t *testing.T) {
		_, err := r.FetchAttachment(ctx, model.NewTMIDAttachmentContainerRef(tmName+"/v1.2.3-20240409155220-3f779458e453.tm.json"), baseNameA)
		assert.ErrorIs(t, err, model.ErrTMNotFound)
	})
	t.Run("invalid tm name", func(t *testing.T) {
		_, err := r.FetchAttachment(ctx, model.NewTMNameAttachmentContainerRef("omnicorp-tm-departmentomnicorp/omnilamp"), baseNameA)
		assert.ErrorIs(t, err, model.ErrInvalidIdOrName)
	})
	t.Run("invalid tm id", func(t *testing.T) {
		_, err := r.FetchAttachment(ctx, model.NewTMIDAttachmentContainerRef(tmName+"/v1.2.3-20240409155220-3f779458e453"), baseNameA)
		assert.ErrorIs(t, err, model.ErrInvalidId)
	})
}

func TestS3Repo_ImportAttachment(t *testing.T) {
	temp, _ := os.MkdirTemp("", "s3r")
	defer os.RemoveAll(temp)
	assert.NoError(t, prepareS3MockBucket("../../test/data/repos/file/attachments", temp))

	c := getS3Mock(t, temp)
	r := S3Repo{bucket: bucket, client: c}
	ctx := context.Background()

	tmName := "omnicorp-tm-department/omnicorp/omnilamp"
	ver := "v3.2.1-20240409155220-3f779458e453"
	id := tmName + "/" + ver + TMExt
	r2Name := "README2.md"
	r2Content := []byte("# read this, too")

	t.Run("tm name attachment without media type provided", func(t *testing.T) {
		ref := model.NewTMNameAttachmentContainerRef(tmName)
		err := r.ImportAttachment(ctx, ref, model.Attachment{Name: r2Name}, r2Content, false)
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(temp, toBucketObject(tmName, model.AttachmentsDir, r2Name)))
		index, err := r.readIndex(ctx)
		if assert.NoError(t, err) {
			c, _, _ := index.FindAttachmentContainer(ref)
			att, found := c.FindAttachment(r2Name)
			assert.True(t, found)
			assert.NotEmpty(t, att.MediaType)
		}
	})
	t.Run("tm name attachment with media type", func(t *testing.T) {
		ref := model.NewTMNameAttachmentContainerRef(tmName)
		err := r.ImportAttachment(ctx, ref, model.Attachment{Name: r2Name, MediaType: "text/html"}, r2Content, true)
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(temp, toBucketObject(tmName, model.AttachmentsDir, r2Name)))
		index, err := r.readIndex(ctx)
		if assert.NoError(t, err) {
			c, _, _ := index.FindAttachmentContainer(ref)
			att, found := c.FindAttachment(r2Name)
			assert.True(t, found)
			assert.Equal(t, "text/html", att.MediaType)
		}
	})
	t.Run("tm id attachment with media type provided by user", func(t *testing.T) {
		ref := model.NewTMIDAttachmentContainerRef(id)
		err := r.ImportAttachment(ctx, ref, model.Attachment{Name: r2Name, MediaType: "text/markdown"}, r2Content, false)
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(temp, toBucketObject(tmName, model.AttachmentsDir, ver, r2Name)))
		index, err := r.readIndex(ctx)
		if assert.NoError(t, err) {
			c, _, _ := index.FindAttachmentContainer(ref)
			att, found := c.FindAttachment(r2Name)
			assert.True(t, found)
			assert.Equal(t, "text/markdown", att.MediaType)
		}
	})
	t.Run("tm id attachment without media type", func(t *testing.T) {
		ref := model.NewTMIDAttachmentContainerRef(id)
		err := r.ImportAttachment(ctx, ref, model.Attachment{Name: r2Name, MediaType: "text/markdown"}, r2Content, true)
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(temp, toBucketObject(tmName, model.AttachmentsDir, ver, r2Name)))
		index, err := r.readIndex(ctx)
		if assert.NoError(t, err) {
			c, _, _ := index.FindAttachmentContainer(ref)
			att, found := c.FindAttachment(r2Name)
			assert.True(t, found)
			assert.Equal(t, "text/markdown", att.MediaType)
		}
	})
	t.Run("tm id attachment conflict", func(t *testing.T) {
		err := r.ImportAttachment(ctx, model.NewTMIDAttachmentContainerRef(id), model.Attachment{Name: r2Name}, r2Content, false)
		assert.ErrorIs(t, err, ErrAttachmentExists)
	})
	t.Run("non existent tm name", func(t *testing.T) {
		err := r.ImportAttachment(ctx, model.NewTMNameAttachmentContainerRef("omnicorp-tm-department/omnicorp/omnidarkness"), model.Attachment{Name: r2Name}, r2Content, false)
		assert.ErrorIs(t, err, model.ErrTMNameNotFound)
	})
	t.Run("non existent tm id", func(t *testing.T) {
		err := r.ImportAttachment(ctx, model.NewTMIDAttachmentContainerRef(tmName+"/v1.2.3-20240409155220-3f779458e453.tm.json"), model.Attachment{Name: r2Name}, r2Content, false)
		assert.ErrorIs(t, err, model.ErrTMNotFound)
	})
	t.Run("invalid tm name", func(t *testing.T) {
		err := r.ImportAttachment(ctx, model.NewTMNameAttachmentContainerRef("omnicorp-tm-departmentomnicorp/omnilamp"), model.Attachment{Name: r2Name}, r2Content, false)
		assert.ErrorIs(t, err, model.ErrInvalidIdOrName)
	})
	t.Run("invalid tm id", func(t *testing.T) {
		err := r.ImportAttachment(ctx, model.NewTMIDAttachmentContainerRef(tmName+"/v1.2.3-20240409155220-3f779458e453"), model.Attachment{Name: r2Name}, r2Content, false)
		assert.ErrorIs(t, err, model.ErrInvalidId)
	})
}

func TestS3Repo_DeleteAttachment(t *testing.T) {
	temp, _ := os.MkdirTemp("", "s3r")
	defer os.RemoveAll(temp)
	assert.NoError(t, prepareS3MockBucket("../../test/data/repos/file/attachments", temp))

	c := getS3Mock(t, temp)
	r := S3Repo{bucket: bucket, client: c}
	ctx := context.Background()

	tmName := "omnicorp-tm-department/omnicorp/omnilamp"
	ver := "v3.2.1-20240409155220-3f779458e453"
	idA := tmName + "/" + ver + TMExt
	attNameA := "README.md"
	attNameB := "cfg.json"

	t.Run("non existent attachment", func(t *testing.T) {
		err := r.DeleteAttachment(ctx, model.NewTMNameAttachmentContainerRef(tmName), "nothing-here")
		assert.ErrorIs(t, err, model.ErrAttachmentNotFound)
	})
	t.Run("non existent tm name", func(t *testing.T) {
		err := r.DeleteAttachment(ctx, model.NewTMNameAttachmentContainerRef("omnicorp-tm-department/omnicorp/omnidarkness"), attNameA)
		assert.ErrorIs(t, err, model.ErrTMNameNotFound)
	})
	t.Run("non existent tm id", func(t *testing.T) {
		err := r.DeleteAttachment(ctx, model.NewTMIDAttachmentContainerRef(tmName+"/v1.2.3-20240409155220-3f779458e453.tm.json"), attNameA)
		assert.ErrorIs(t, err, model.ErrTMNotFound)
	})
	t.Run("invalid tm name", func(t *testing.T) {
		err := r.DeleteAttachment(ctx, model.NewTMNameAttachmentContainerRef("omnicorp-tm-departmentomnicorp/omnilamp"), attNameA)
		assert.ErrorIs(t, err, model.ErrInvalidIdOrName)
	})
	t.Run("invalid tm id", func(t *testing.T) {
		err := r.DeleteAttachment(ctx, model.NewTMIDAttachmentContainerRef(tmName+"/v1.2.3-20240409155220-3f779458e453"), attNameA)
		assert.ErrorIs(t, err, model.ErrInvalidId)
	})
	t.Run("tm id attachment", func(t *testing.T) {
		err := r.DeleteAttachment(ctx, model.NewTMIDAttachmentContainerRef(idA), attNameB)
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(temp, toBucketObject(tmName, model.AttachmentsDir, ver, attNameA)))
		assert.True(t, os.IsNotExist(err))
	})
	t.Run("tm name attachment", func(t *testing.T) {
		err := r.DeleteAttachment(ctx, model.NewTMNameAttachmentContainerRef(tmName), attNameA)
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(temp, toBucketObject(tmName, model.AttachmentsDir, attNameA)))
		assert.True(t, os.IsNotExist(err))
	})
}

func TestS3Repo_CheckIntegrity(t *testing.T) {

	ctx := context.Background()

	t.Run("with only valid ThingModels", func(t *testing.T) {
		temp, _ := os.MkdirTemp("", "s3r")
		defer os.RemoveAll(temp)
		assert.NoError(t, prepareS3MockBucket("../../test/data/index", temp))

		c := getS3Mock(t, temp)

		// given: a clean repository with index
		r := S3Repo{bucket: bucket, client: c}
		_ = r.Index(ctx)

		// when checking the integrity
		res, err := r.CheckIntegrity(ctx, nil)

		// then: there is no total error
		assert.NoError(t, err)
		// and then: result list only contains OK results
		for _, result := range res {
			assert.Equal(t, model.CheckOK, result.Typ)
		}
	})

	t.Run("with a directory without index", func(t *testing.T) {
		temp, _ := os.MkdirTemp("", "s3r")
		defer os.RemoveAll(temp)
		assert.NoError(t, prepareS3MockBucket("../../test/data/index", temp))

		c := getS3Mock(t, temp)
		// given: a clean repository with index
		r := S3Repo{bucket: bucket, client: c}

		// when checking the integrity
		res, err := r.CheckIntegrity(ctx, nil)

		// then: there is no total error
		assert.NoError(t, err)
		// and then: result list is empty
		assert.Empty(t, res)
	})

	t.Run("with some invalid files", func(t *testing.T) {
		temp, _ := os.MkdirTemp("", "s3r")
		defer os.RemoveAll(temp)
		assert.NoError(t, prepareS3MockBucket("../../test/data/integrity/faulty", temp))

		c := getS3Mock(t, temp)
		// given: a clean repository with index
		r := S3Repo{bucket: bucket, client: c}

		// when: checking the given ThingModels
		res, err := r.CheckIntegrity(ctx, nil)
		// then: there is no total error
		assert.NoError(t, err)
		// then: results include errors for invalid files
		assert.Contains(t, res, model.CheckResult{Typ: model.CheckErr, ResourceName: "omnicorp/mistake.md", Message: "file unknown"})
		assert.Contains(t, res, model.CheckResult{Typ: model.CheckErr, ResourceName: "omnicorp/omnicorp/lightall/.attachments/mistake.md", Message: "appears to be an attachment file which is not known to the repository. Make sure you import it using TMC CLI"})
		assert.Contains(t, res, model.CheckResult{Typ: model.CheckErr, ResourceName: "omnicorp/omnicorp/lightall/.attachments/v1.0.1-20240807094932-5a3840060b05/mistake.md", Message: "appears to be an attachment file which is not known to the repository. Make sure you import it using TMC CLI"})
		assert.Contains(t, res, model.CheckResult{Typ: model.CheckErr, ResourceName: "omnicorp/omnicorp/lightall/v1.0.2-20240819094932-5a3840060b05.tm.json", Message: "appears to be a TM file which is not known to the repository. Make sure you import it using TMC CLI"})
		assert.Contains(t, res, model.CheckResult{Typ: model.CheckErr, ResourceName: "vomnicorp/vomnicorp/mistake.md", Message: "file unknown"})
		// and then: results do not include errors for ignored files
		assert.NotContains(t, res, model.CheckResult{Typ: model.CheckErr, ResourceName: ".dotDir/ignored.txt", Message: "file unknown"})
		assert.NotContains(t, res, model.CheckResult{Typ: model.CheckErr, ResourceName: "ignoredDir/ignored.txt", Message: "file unknown"})
		assert.NotContains(t, res, model.CheckResult{Typ: model.CheckErr, ResourceName: ".dotFile", Message: "file unknown"})
	})
}

func TestS3Repo_ListCompletions(t *testing.T) {
	temp, _ := os.MkdirTemp("", "s3r")
	defer os.RemoveAll(temp)

	c := getS3Mock(t, temp)
	r := S3Repo{bucket: bucket, client: c}
	ctx := context.Background()

	t.Run("invalid", func(t *testing.T) {
		_, err := r.ListCompletions(ctx, "invalid", nil, "")
		assert.ErrorIs(t, err, ErrInvalidCompletionParams)
	})

	t.Run("no names file", func(t *testing.T) {
		names, err := r.ListCompletions(ctx, CompletionKindNames, nil, "")
		assert.NoError(t, err)
		var exp []string
		assert.Equal(t, exp, names)
	})

	t.Run("names", func(t *testing.T) {
		_ = os.WriteFile(filepath.Join(temp, toBucketObject(".tmc", TmNamesFile)), []byte("omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall\n"+
			"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/a/b\n"+
			"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/subpath\n"), defaultFilePermissions)
		t.Run("empty", func(t *testing.T) {
			completions, err := r.ListCompletions(ctx, CompletionKindNames, nil, "")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/"}, completions)
		})
		t.Run("some letters", func(t *testing.T) {
			completions, err := r.ListCompletions(ctx, CompletionKindNames, nil, "om")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/"}, completions)
		})
		t.Run("some letters non existing", func(t *testing.T) {
			completions, err := r.ListCompletions(ctx, CompletionKindNames, nil, "aaa")
			assert.NoError(t, err)
			var expRes []string
			assert.Equal(t, expRes, completions)
		})
		t.Run("full first name part", func(t *testing.T) {
			completions, err := r.ListCompletions(ctx, CompletionKindNames, nil, "omnicorp-r-d-research/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/"}, completions)
		})
		t.Run("some letters second part", func(t *testing.T) {
			completions, err := r.ListCompletions(ctx, CompletionKindNames, nil, "omnicorp-r-d-research/omnicorp")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/"}, completions)
		})
		t.Run("full second part", func(t *testing.T) {
			completions, err := r.ListCompletions(ctx, CompletionKindNames, nil, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall", "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/"}, completions)
		})
		t.Run("full third part", func(t *testing.T) {
			completions, err := r.ListCompletions(ctx, CompletionKindNames, nil, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/a/", "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/subpath"}, completions)
		})
		t.Run("full fourth part", func(t *testing.T) {
			completions, err := r.ListCompletions(ctx, CompletionKindNames, nil, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/a/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/a/b"}, completions)
		})
		t.Run("full name", func(t *testing.T) {
			completions, err := r.ListCompletions(ctx, CompletionKindNames, nil, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/subpath")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/subpath"}, completions)
		})
	})

	t.Run("fetchNames", func(t *testing.T) {
		tmName := "omnicorp-tm-department/omnicorp/omnilamp"
		_ = os.WriteFile(filepath.Join(temp, toBucketObject(tmName, "v1.0.0-20231208142856-a49617d2e4fc.tm.json")), []byte("{}"), defaultFilePermissions)
		_ = os.WriteFile(filepath.Join(temp, toBucketObject(tmName, "v1.0.0-20231207142856-b49617d2e4fc.tm.json")), []byte("{}"), defaultFilePermissions)
		_ = os.WriteFile(filepath.Join(temp, toBucketObject(tmName, "v1.2.1-20231209142856-c49617d2e4fc.tm.json")), []byte("{}"), defaultFilePermissions)
		_ = os.WriteFile(filepath.Join(temp, toBucketObject(tmName, "v0.0.1-20231208142856-d49617d2e4fc.tm.json")), []byte("{}"), defaultFilePermissions)
		fNames, err := r.ListCompletions(ctx, CompletionKindFetchNames, nil, tmName)
		assert.NoError(t, err)
		assert.Equal(t, []string{"omnicorp-tm-department/omnicorp/omnilamp:v0.0.1", "omnicorp-tm-department/omnicorp/omnilamp:v1.0.0", "omnicorp-tm-department/omnicorp/omnilamp:v1.2.1"}, fNames)
	})
}

func getS3Mock(t *testing.T, filePath string) *s3mocks.S3Client {
	c := s3mocks.NewS3Client(t)

	c.On("HeadObject", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
			fName := filepath.Join(filePath, toBucketObject(*params.Key))

			_, err := os.Stat(fName)
			if err != nil {
				return nil, &smithy.GenericAPIError{Code: "NotFound"}
			} else {
				return &s3.HeadObjectOutput{}, nil
			}
		}).Maybe()

	c.On("ListObjectsV2", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
			var s3Objects []types.Object
			d, err := os.ReadDir(filePath)
			assert.NoError(t, err)

			for _, f := range d {
				fName := fromBucketObject(f.Name())
				if !strings.HasPrefix(fName, *params.Prefix) {
					continue
				}

				if f.IsDir() {
					fName = toS3Dir(fName)
				}

				s3Objects = append(s3Objects, types.Object{Key: &fName})
			}
			return &s3.ListObjectsV2Output{Contents: s3Objects}, nil
		}).Maybe()

	c.On("PutObject", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			fName := toBucketObject(*params.Key)

			buf := new(bytes.Buffer)
			buf.ReadFrom(params.Body)
			testutils.CreateFile(filePath, fName, buf.Bytes())
			return &s3.PutObjectOutput{}, nil
		}).Maybe()

	c.On("GetObject", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			fName := filepath.Join(filePath, toBucketObject(*params.Key))

			_, b, err := utils.ReadRequiredFile(fName)
			if err != nil {
				msg := "Object not found"
				return nil, &types.NoSuchKey{Message: &msg}
			}
			return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewBuffer(b))}, nil
		}).Maybe()

	c.On("DeleteObject", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
			fName := filepath.Join(filePath, toBucketObject(*params.Key))

			err := os.Remove(fName)
			if err != nil {
				return nil, &smithy.OperationError{}
			}
			return &s3.DeleteObjectOutput{}, nil
		}).Maybe()

	c.On("DeleteObjects", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
			objects := params.Delete.Objects
			for _, o := range objects {
				fName := filepath.Join(filePath, toBucketObject(*o.Key))

				err := os.Remove(fName)
				if err != nil {
					fmt.Println("Failed to delete:", *o.Key)
					return &s3.DeleteObjectsOutput{}, nil
				}
			}
			return &s3.DeleteObjectsOutput{}, nil
		}).Maybe()

	return c
}

func prepareS3MockBucket(fromDir, toDir string) error {
	err := filepath.Walk(fromDir, func(p string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		srcFile := strings.TrimPrefix(filepath.ToSlash(p), filepath.ToSlash(fromDir))
		srcFile = strings.TrimPrefix(srcFile, "/")
		tgtFile := toBucketObject(srcFile)
		err = testutils.CopyFile(p, path.Join(toDir, tgtFile))
		return err
	})
	return err
}

func toBucketObject(p ...string) string {
	o := filepath.Join(p...)
	o = filepath.ToSlash(o)
	return strings.ReplaceAll(o, "/", "#")
}

func fromBucketObject(o string) string {
	return strings.ReplaceAll(o, "#", "/")
}
