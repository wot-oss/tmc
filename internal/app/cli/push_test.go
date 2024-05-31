package cli

import (
	"context"
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

func TestPushExecutor_Push(t *testing.T) {
	r := mocks.NewRepo(t)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("repo"), r, nil))

	t.Run("push when none exists", func(t *testing.T) {

		now := func() time.Time { return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC) }
		e := NewPushExecutor(now)
		id := "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id)
		r.On("Push", mock.Anything, tmid, mock.Anything, repos.PushOptions{}).Return(repos.PushResult{Type: repos.PushResultOK, TmID: id}, nil)
		r.On("Index", mock.Anything, id).Return(nil)

		res, err := e.Push(context.Background(), "../../../test/data/push/omnilamp-versioned.json", model.NewRepoSpec("repo"), false, repos.PushOptions{})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, repos.PushResultOK, res[0].Type)
	})

	t.Run("push non-existing file", func(t *testing.T) {

		now := func() time.Time { return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC) }
		e := NewPushExecutor(now)
		_, err := e.Push(context.Background(), "does-not-exist.json", model.NewRepoSpec("repo"), false, repos.PushOptions{})
		assert.Error(t, err)
	})

	t.Run("push when repo has the same TM", func(t *testing.T) {

		tmid2 := model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231111123243-98b3fbd291f4.tm.json")

		now := func() time.Time {
			return time.Date(2023, time.November, 11, 12, 32, 43, 0, time.UTC)
		}
		e := NewPushExecutor(now)
		cErr := &repos.ErrTMIDConflict{Type: repos.IdConflictSameContent,
			ExistingId: "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"}
		r.On("Push", mock.Anything, tmid2, mock.Anything, repos.PushOptions{}).Return(repos.PushResult{
			Type:    repos.PushResultTMExists,
			TmID:    "",
			Message: cErr.Error(),
			Err:     cErr,
		}, nil)
		res, err := e.Push(context.Background(), "../../../test/data/push/omnilamp-versioned.json", model.NewRepoSpec("repo"), false, repos.PushOptions{})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, repos.PushResultTMExists, res[0].Type)
	})

	t.Run("push fails", func(t *testing.T) {

		tmid3 := model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20230811123243-98b3fbd291f4.tm.json")
		now := func() time.Time {
			return time.Date(2023, time.August, 11, 12, 32, 43, 0, time.UTC)
		}
		e := NewPushExecutor(now)
		r.On("Push", mock.Anything, tmid3, mock.Anything, repos.PushOptions{}).Return(repos.PushResult{}, errors.New("unexpected"))
		res, err := e.Push(context.Background(), "../../../test/data/push/omnilamp-versioned.json", model.NewRepoSpec("repo"), false, repos.PushOptions{})
		assert.Error(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, repos.PushResult{}, res[0])
	})

	t.Run("push with optPath", func(t *testing.T) {
		now := func() time.Time { return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC) }
		e := NewPushExecutor(now)
		id := "omnicorp-tm-department/omnicorp/omnilamp/a/b/c/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id)
		opts := repos.PushOptions{OptPath: "a/b/c"}
		r.On("Push", mock.Anything, tmid, mock.Anything, opts).Return(repos.PushResult{Type: repos.PushResultOK, TmID: id}, nil)
		r.On("Index", mock.Anything, id).Return(nil)

		res, err := e.Push(context.Background(), "../../../test/data/push/omnilamp-versioned.json", model.NewRepoSpec("repo"), false, opts)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, repos.PushResultOK, res[0].Type)
	})
}

func TestPushExecutor_Push_Directory(t *testing.T) {
	r := mocks.NewRepo(t)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("repo"), r, nil))

	t.Run("push directory", func(t *testing.T) {
		clk := testutils.NewTestClock(time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC), time.Second)
		e := NewPushExecutor(clk.Now)
		opts := repos.PushOptions{}
		tmid := model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json")
		r.On("Push", mock.Anything, tmid, mock.Anything, opts).Return(repos.PushResult{Type: repos.PushResultOK, TmID: tmid.String()}, nil)
		tmid = model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json")
		r.On("Push", mock.Anything, tmid, mock.Anything, opts).Return(repos.PushResult{Type: repos.PushResultOK, TmID: tmid.String()}, nil)
		tmid = model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123245-98b3fbd291f4.tm.json")
		cErr := &repos.ErrTMIDConflict{Type: repos.IdConflictSameContent,
			ExistingId: "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"}
		r.On("Push", mock.Anything, tmid, mock.Anything, opts).Return(
			repos.PushResult{
				Type:    repos.PushResultTMExists,
				TmID:    "",
				Message: cErr.Error(),
				Err:     cErr,
			}, nil)
		tmid = model.MustParseTMID("omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20231110123246-575dfac219e2.tm.json")
		cErr = &repos.ErrTMIDConflict{Type: repos.IdConflictSameContent,
			ExistingId: "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json"}
		r.On("Push", mock.Anything, tmid, mock.Anything, opts).Return(
			repos.PushResult{
				Type:    repos.PushResultTMExists,
				TmID:    "",
				Message: cErr.Error(),
				Err:     cErr,
			}, nil)
		r.On("Index", mock.Anything,
			"omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json",
			"omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json").Return(nil)

		res, err := e.Push(context.Background(), "../../../test/data/push", model.NewRepoSpec("repo"), false, opts)
		assert.NoError(t, err)
		assert.Len(t, res, 4)
		assert.Equalf(t, repos.PushResultOK, res[0].Type, "res[0]: want PushResultOK, got %v", res[0].Type)
		assert.Equalf(t, repos.PushResultOK, res[1].Type, "res[1]: want PushResultOK, got %v", res[1].Type)
		assert.Equalf(t, repos.PushResultTMExists, res[2].Type, "res[2]: want PushResultTMExists, got %v", res[2].Type)
		assert.Equalf(t, repos.PushResultTMExists, res[3].Type, "res[3]: want PushResultTMExists, got %v", res[3].Type)

	})

	t.Run("push directory with optPath", func(t *testing.T) {
		clk := testutils.NewTestClock(time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC), time.Second)
		e := NewPushExecutor(clk.Now)
		opts := repos.PushOptions{OptPath: "opt"}
		id1 := "omnicorp-tm-department/omnicorp/omnilamp/opt/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id1)
		r.On("Push", mock.Anything, tmid, mock.Anything, opts).Return(repos.PushResult{Type: repos.PushResultOK, TmID: tmid.String()}, nil)
		id2 := "omnicorp-tm-department/omnicorp/omnilamp/opt/v0.0.0-20231110123244-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id2)
		r.On("Push", mock.Anything, tmid, mock.Anything, opts).Return(repos.PushResult{Type: repos.PushResultOK, TmID: tmid.String()}, nil)
		id3 := "omnicorp-tm-department/omnicorp/omnilamp/opt/v3.2.1-20231110123245-98b3fbd291f4.tm.json"
		tmid = model.MustParseTMID(id3)
		r.On("Push", mock.Anything, tmid, mock.Anything, opts).Return(repos.PushResult{Type: repos.PushResultOK, TmID: tmid.String()}, nil)
		id4 := "omnicorp-tm-department/omnicorp/omnilamp/opt/v0.0.0-20231110123246-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id4)
		r.On("Push", mock.Anything, tmid, mock.Anything, opts).Return(repos.PushResult{Type: repos.PushResultOK, TmID: tmid.String()}, nil)
		r.On("Index", mock.Anything, id1, id2, id3, id4).Return(nil)

		res, err := e.Push(context.Background(), "../../../test/data/push", model.NewRepoSpec("repo"), false, opts)
		assert.NoError(t, err)
		assert.Len(t, res, 4)
		for i, r := range res {
			assert.Equalf(t, repos.PushResultOK, r.Type, "res[%d]: want PushResultOK, got %v", i, r.Type)
		}

	})

	t.Run("push directory with optTree", func(t *testing.T) {
		clk := testutils.NewTestClock(time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC), time.Second)
		e := NewPushExecutor(clk.Now)
		id1 := "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id1)
		r.On("Push", mock.Anything, tmid, mock.Anything, repos.PushOptions{OptPath: "/"}).Return(repos.PushResult{Type: repos.PushResultOK, TmID: tmid.String()}, nil)
		id2 := "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id2)
		r.On("Push", mock.Anything, tmid, mock.Anything, repos.PushOptions{OptPath: "/"}).Return(repos.PushResult{Type: repos.PushResultOK, TmID: tmid.String()}, nil)
		id3 := "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v3.2.1-20231110123245-98b3fbd291f4.tm.json"
		tmid = model.MustParseTMID(id3)
		r.On("Push", mock.Anything, tmid, mock.Anything, repos.PushOptions{OptPath: "/subfolder"}).Return(repos.PushResult{Type: repos.PushResultOK, TmID: tmid.String()}, nil)
		id4 := "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v0.0.0-20231110123246-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id4)
		r.On("Push", mock.Anything, tmid, mock.Anything, repos.PushOptions{OptPath: "/subfolder"}).Return(repos.PushResult{Type: repos.PushResultOK, TmID: tmid.String()}, nil)
		r.On("Index", mock.Anything, id1, id2, id3, id4).Return(nil)

		res, err := e.Push(context.Background(), "../../../test/data/push", model.NewRepoSpec("repo"), true, repos.PushOptions{})
		assert.NoError(t, err)
		assert.Len(t, res, 4)
		for i, r := range res {
			assert.Equalf(t, repos.PushResultOK, r.Type, "res[%d]: want PushResultOK, got %v", i, r.Type)
		}
	})

}
