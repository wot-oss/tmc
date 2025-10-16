package cli

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/repos/mocks"
	"github.com/wot-oss/tmc/internal/testutils"
	rMocks "github.com/wot-oss/tmc/internal/testutils/reposmocks"
)

func TestImportExecutor_Import(t *testing.T) {
	r := mocks.NewRepo(t)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("repo"), r, nil))

	t.Run("import when none exists", func(t *testing.T) {

		now := func() time.Time { return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC) }
		e := NewImportExecutor(now)
		id := "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id)
		r.On("Import", mock.Anything, tmid, mock.Anything, repos.ImportOptions{}).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: id}, nil)
		r.On("Index", mock.Anything, id).Return(nil)

		res, err := e.Import(context.Background(), "../../../test/data/import/omnilamp-versioned.json", model.NewRepoSpec("repo"), false, repos.ImportOptions{}, OutputFormatPlain)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, repos.ImportResultOK, res[0].Type)
	})
	t.Run("import with ok with json output", func(t *testing.T) {
		restore, getStdout := testutils.ReplaceStdout()
		defer restore()
		now := func() time.Time { return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC) }
		e := NewImportExecutor(now)
		id := "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id)
		r.On("Import", mock.Anything, tmid, mock.Anything, repos.ImportOptions{}).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: id}, nil)
		r.On("Index", mock.Anything, id).Return(nil)

		res, err := e.Import(context.Background(), "../../../test/data/import/omnilamp-versioned.json", model.NewRepoSpec("repo"), false, repos.ImportOptions{}, OutputFormatJSON)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, repos.ImportResultOK, res[0].Type)
		stdout := getStdout()
		var actual any
		err = json.Unmarshal([]byte(stdout), &actual)
		assert.NoError(t, err)
		expected := []any{map[string]any{"message": "file ../../../test/data/import/omnilamp-versioned.json imported as omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json", "type": "OK"}}
		assert.Equal(t, expected, actual)
	})

	t.Run("import non-existing file", func(t *testing.T) {

		now := func() time.Time { return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC) }
		e := NewImportExecutor(now)
		_, err := e.Import(context.Background(), "does-not-exist.json", model.NewRepoSpec("repo"), false, repos.ImportOptions{}, OutputFormatPlain)
		assert.Error(t, err)
	})

	t.Run("import when repo has the same TM", func(t *testing.T) {

		tmid2 := model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231111123243-98b3fbd291f4.tm.json")

		now := func() time.Time {
			return time.Date(2023, time.November, 11, 12, 32, 43, 0, time.UTC)
		}
		e := NewImportExecutor(now)
		cErr := &repos.ErrTMIDConflict{Type: repos.IdConflictSameContent,
			ExistingId: "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"}
		r.On("Import", mock.Anything, tmid2, mock.Anything, repos.ImportOptions{}).Return(repos.ImportResult{
			Type:    repos.ImportResultError,
			TmID:    "",
			Message: cErr.Error(),
			Err:     cErr,
		}, cErr)
		res, err := e.Import(context.Background(), "../../../test/data/import/omnilamp-versioned.json", model.NewRepoSpec("repo"), false, repos.ImportOptions{}, OutputFormatPlain)
		assert.Error(t, err)
		if assert.Len(t, res, 1) {
			assert.Equal(t, repos.ImportResultError, res[0].Type)
			assert.Equal(t, cErr, res[0].Err)
		}
	})

	t.Run("import fails", func(t *testing.T) {

		tmid3 := model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20230811123243-98b3fbd291f4.tm.json")
		now := func() time.Time {
			return time.Date(2023, time.August, 11, 12, 32, 43, 0, time.UTC)
		}
		e := NewImportExecutor(now)
		ret, resErr := repos.ImportResultFromError(errors.New("unexpected"))
		r.On("Import", mock.Anything, tmid3, mock.Anything, repos.ImportOptions{}).Return(ret, resErr)
		res, err := e.Import(context.Background(), "../../../test/data/import/omnilamp-versioned.json", model.NewRepoSpec("repo"), false, repos.ImportOptions{}, OutputFormatPlain)
		assert.Error(t, err)
		assert.ErrorIs(t, err, resErr)
		if assert.Len(t, res, 1) {
			assert.ErrorIs(t, res[0].Err, resErr)
		}
	})

	t.Run("import with optPath", func(t *testing.T) {
		now := func() time.Time { return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC) }
		e := NewImportExecutor(now)
		id := "omnicorp-tm-department/omnicorp/omnilamp/a/b/c/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id)
		opts := repos.ImportOptions{OptPath: "a/b/c"}
		r.On("Import", mock.Anything, tmid, mock.Anything, opts).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: id}, nil)
		r.On("Index", mock.Anything, id).Return(nil)

		res, err := e.Import(context.Background(), "../../../test/data/import/omnilamp-versioned.json", model.NewRepoSpec("repo"), false, opts, OutputFormatPlain)
		assert.NoError(t, err)
		if assert.Len(t, res, 1) {
			assert.Equal(t, repos.ImportResultOK, res[0].Type)
		}
	})
}

func TestImportExecutor_Import_Directory(t *testing.T) {
	r := mocks.NewRepo(t)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("repo"), r, nil))

	t.Run("import directory", func(t *testing.T) {
		clk := testutils.NewTestClock(time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC), time.Second)
		e := NewImportExecutor(clk.Now)
		opts := repos.ImportOptions{}
		tmid := model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json")
		r.On("Import", mock.Anything, tmid, mock.Anything, opts).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid.String()}, nil)
		tmid = model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json")
		r.On("Import", mock.Anything, tmid, mock.Anything, opts).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid.String()}, nil)
		tmid = model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123245-98b3fbd291f4.tm.json")
		cErr := &repos.ErrTMIDConflict{Type: repos.IdConflictSameContent,
			ExistingId: "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"}
		r.On("Import", mock.Anything, tmid, mock.Anything, opts).Return(
			repos.ImportResult{
				Type:    repos.ImportResultError,
				TmID:    "",
				Message: cErr.Error(),
				Err:     cErr,
			}, cErr)
		tmid = model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20231110123246-575dfac219e2.tm.json")
		cErr = &repos.ErrTMIDConflict{Type: repos.IdConflictSameContent,
			ExistingId: "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json"}
		r.On("Import", mock.Anything, tmid, mock.Anything, opts).Return(
			repos.ImportResult{
				Type:    repos.ImportResultError,
				TmID:    "",
				Message: cErr.Error(),
				Err:     cErr,
			}, cErr)
		r.On("Index", mock.Anything,
			"omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json",
			"omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json").Return(nil)

		res, err := e.Import(context.Background(), "../../../test/data/import", model.NewRepoSpec("repo"), false, opts, OutputFormatPlain)
		assert.Error(t, err)
		if assert.Len(t, res, 4) {
			assert.Equalf(t, repos.ImportResultOK, res[0].Type, "res[0]: want ImportResultOK, got %v", res[0].Type)
			assert.Equalf(t, repos.ImportResultOK, res[1].Type, "res[1]: want ImportResultOK, got %v", res[1].Type)
			assert.Equalf(t, repos.ImportResultError, res[2].Type, "res[2]: want ImportResultError, got %v", res[2].Type)
			assert.Equalf(t, repos.ImportResultError, res[3].Type, "res[3]: want ImportResultError, got %v", res[3].Type)
		}
	})
	t.Run("import directory with ignore-existing", func(t *testing.T) {
		clk := testutils.NewTestClock(time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC), time.Second)
		e := NewImportExecutor(clk.Now)
		opts := repos.ImportOptions{
			IgnoreExisting: true,
		}
		tmid := model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json")
		r.On("Import", mock.Anything, tmid, mock.Anything, opts).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid.String()}, nil)
		tmid = model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json")
		r.On("Import", mock.Anything, tmid, mock.Anything, opts).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid.String()}, nil)
		tmid = model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123245-98b3fbd291f4.tm.json")
		cErr := &repos.ErrTMIDConflict{Type: repos.IdConflictSameContent,
			ExistingId: "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"}
		r.On("Import", mock.Anything, tmid, mock.Anything, opts).Return(
			repos.ImportResult{
				Type:    repos.ImportResultError,
				TmID:    "",
				Message: cErr.Error(),
				Err:     cErr,
			}, cErr)
		tmid = model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20231110123246-575dfac219e2.tm.json")
		cErr = &repos.ErrTMIDConflict{Type: repos.IdConflictSameContent,
			ExistingId: "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json"}
		r.On("Import", mock.Anything, tmid, mock.Anything, opts).Return(
			repos.ImportResult{
				Type:    repos.ImportResultError,
				TmID:    "",
				Message: cErr.Error(),
				Err:     cErr,
			}, cErr)
		r.On("Index", mock.Anything,
			"omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json",
			"omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json").Return(nil)

		res, err := e.Import(context.Background(), "../../../test/data/import", model.NewRepoSpec("repo"), false, opts, OutputFormatPlain)
		assert.NoError(t, err)
		if assert.Len(t, res, 4) {
			assert.Equalf(t, repos.ImportResultOK, res[0].Type, "res[0]: want ImportResultOK, got %v", res[0].Type)
			assert.Equalf(t, repos.ImportResultOK, res[1].Type, "res[1]: want ImportResultOK, got %v", res[1].Type)
			assert.Equalf(t, repos.ImportResultError, res[2].Type, "res[2]: want ImportResultError, got %v", res[2].Type)
			assert.Equalf(t, repos.ImportResultError, res[3].Type, "res[3]: want ImportResultError, got %v", res[3].Type)
		}
	})

	t.Run("import directory with optPath", func(t *testing.T) {
		clk := testutils.NewTestClock(time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC), time.Second)
		e := NewImportExecutor(clk.Now)
		opts := repos.ImportOptions{OptPath: "opt"}
		id1 := "omnicorp-tm-department/omnicorp/omnilamp/opt/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id1)
		r.On("Import", mock.Anything, tmid, mock.Anything, opts).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid.String()}, nil)
		id2 := "omnicorp-tm-department/omnicorp/omnilamp/opt/v0.0.0-20231110123244-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id2)
		r.On("Import", mock.Anything, tmid, mock.Anything, opts).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid.String()}, nil)
		id3 := "omnicorp-tm-department/omnicorp/omnilamp/opt/v3.2.1-20231110123245-98b3fbd291f4.tm.json"
		tmid = model.MustParseTMID(id3)
		r.On("Import", mock.Anything, tmid, mock.Anything, opts).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid.String()}, nil)
		id4 := "omnicorp-tm-department/omnicorp/omnilamp/opt/v0.0.0-20231110123246-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id4)
		r.On("Import", mock.Anything, tmid, mock.Anything, opts).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid.String()}, nil)
		r.On("Index", mock.Anything, id1, id2, id3, id4).Return(nil)

		res, err := e.Import(context.Background(), "../../../test/data/import", model.NewRepoSpec("repo"), false, opts, OutputFormatPlain)
		assert.NoError(t, err)
		if assert.Len(t, res, 4) {
			for i, r := range res {
				assert.Equalf(t, repos.ImportResultOK, r.Type, "res[%d]: want ImportResultOK, got %v", i, r.Type)
			}
		}
	})

	t.Run("import directory with optTree", func(t *testing.T) {
		clk := testutils.NewTestClock(time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC), time.Second)
		e := NewImportExecutor(clk.Now)
		id1 := "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id1)
		r.On("Import", mock.Anything, tmid, mock.Anything, repos.ImportOptions{OptPath: "/"}).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid.String()}, nil)
		id2 := "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id2)
		r.On("Import", mock.Anything, tmid, mock.Anything, repos.ImportOptions{OptPath: "/"}).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid.String()}, nil)
		id3 := "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v3.2.1-20231110123245-98b3fbd291f4.tm.json"
		tmid = model.MustParseTMID(id3)
		r.On("Import", mock.Anything, tmid, mock.Anything, repos.ImportOptions{OptPath: "/subfolder"}).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid.String()}, nil)
		id4 := "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v0.0.0-20231110123246-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id4)
		r.On("Import", mock.Anything, tmid, mock.Anything, repos.ImportOptions{OptPath: "/subfolder"}).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid.String()}, nil)
		r.On("Index", mock.Anything, id1, id2, id3, id4).Return(nil)

		res, err := e.Import(context.Background(), "../../../test/data/import", model.NewRepoSpec("repo"), true, repos.ImportOptions{}, OutputFormatPlain)
		assert.NoError(t, err)
		assert.Len(t, res, 4)
		for i, r := range res {
			assert.Equalf(t, repos.ImportResultOK, r.Type, "res[%d]: want ImportResultOK, got %v", i, r.Type)
		}
	})

	t.Run("import directory with with-attachments", func(t *testing.T) {
		clk := testutils.NewTestClock(time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC), time.Second)
		e := NewImportExecutor(clk.Now)
		opts := repos.ImportOptions{
			WithAttachments: true,
		}
		repoSpec := model.NewRepoSpec("repo")
		ctx := context.Background()
		r.On("Spec").Return(repoSpec)
		tmid1 := model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123244-98b3fbd291f4.tm.json")
		r.On("Import", mock.Anything, tmid1, mock.Anything, opts).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid1.String()}, nil)
		tmid2 := model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20231110123245-575dfac219e2.tm.json")
		r.On("Import", mock.Anything, tmid2, mock.Anything, opts).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid2.String()}, nil)
		r.On("Index", mock.Anything).Return(nil)
		r.On("Index", mock.Anything, tmid1.String(), tmid2.String()).Return(nil)
		v1 := model.IndexVersion{
			ExternalID: "omnilamp.json",
		}
		v2 := model.IndexVersion{
			ExternalID: "omnilamp-versioned.json",
		}
		entry1 := model.FoundEntry{
			Name: tmid1.Name,
			Manufacturer: model.SchemaManufacturer{
				Name: "omnicorp",
			},
			Mpn: "123",
			Author: model.SchemaAuthor{
				Name: "author",
			},
			Versions: []model.FoundVersion{
				{
					IndexVersion: &v1,
				},
			},
		}
		entry2 := model.FoundEntry{
			Name: tmid2.Name,
			Manufacturer: model.SchemaManufacturer{
				Name: "omnicorp",
			},
			Mpn: "000",
			Author: model.SchemaAuthor{
				Name: "author",
			},
			Versions: []model.FoundVersion{
				{
					IndexVersion: &v2,
				},
			},
		}
		r.On("List", mock.Anything, mock.Anything).Return(model.SearchResult{LastUpdated: clk.Now(), Entries: []model.FoundEntry{entry1, entry2}}, nil)
		r.On("ImportAttachment", ctx, model.NewTMNameAttachmentContainerRef(tmid1.Name), model.Attachment{Name: "test.svg", MediaType: "image/svg+xml"}, mock.Anything, mock.Anything).Return(nil)
		r.On("ImportAttachment", ctx, model.NewTMNameAttachmentContainerRef(tmid2.Name), model.Attachment{Name: "test.svg", MediaType: "image/svg+xml"}, mock.Anything, mock.Anything).Return(nil)
		r.On("ImportAttachment", ctx, model.NewTMNameAttachmentContainerRef(tmid2.Name), model.Attachment{Name: "test.txt", MediaType: "text/plain; charset=utf-8"}, mock.Anything, mock.Anything).Return(nil)
		res, err := e.Import(context.Background(), "../../../test/data/import_attachments/subfolder_with_attachments", repoSpec, false, opts, OutputFormatPlain)
		assert.NoError(t, err)
		assert.Len(t, res, 2)
		for i, r := range res {
			assert.Equalf(t, repos.ImportResultOK, r.Type, "res[%d]: want ImportResultOK, got %v", i, r.Type)
		}
	})

}
