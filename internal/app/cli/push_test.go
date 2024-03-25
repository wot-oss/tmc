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
		id := "omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id, false)
		r.On("Push", mock.Anything, tmid, mock.Anything).Return(nil)
		r.On("Index", mock.Anything, id).Return(nil)

		res, err := e.Push(context.Background(), "../../../test/data/push/omnilamp-versioned.json", model.NewRepoSpec("repo"), "", false)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, PushOK, res[0].typ)
	})

	t.Run("push non-existing file", func(t *testing.T) {

		now := func() time.Time { return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC) }
		e := NewPushExecutor(now)
		_, err := e.Push(context.Background(), "does-not-exist.json", model.NewRepoSpec("repo"), "", false)
		assert.Error(t, err)
	})

	t.Run("push when repo has the same TM", func(t *testing.T) {

		tmid2 := model.MustParseTMID("omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231111123243-98b3fbd291f4.tm.json", false)

		now := func() time.Time {
			return time.Date(2023, time.November, 11, 12, 32, 43, 0, time.UTC)
		}
		e := NewPushExecutor(now)
		r.On("Push", mock.Anything, tmid2, mock.Anything).Return(&repos.ErrTMIDConflict{Type: repos.IdConflictSameContent,
			ExistingId: "omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"})
		res, err := e.Push(context.Background(), "../../../test/data/push/omnilamp-versioned.json", model.NewRepoSpec("repo"), "", false)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, TMExists, res[0].typ)
	})

	t.Run("push fails", func(t *testing.T) {

		tmid3 := model.MustParseTMID("omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20230811123243-98b3fbd291f4.tm.json", false)
		now := func() time.Time {
			return time.Date(2023, time.August, 11, 12, 32, 43, 0, time.UTC)
		}
		e := NewPushExecutor(now)
		r.On("Push", mock.Anything, tmid3, mock.Anything).Return(errors.New("unexpected"))
		res, err := e.Push(context.Background(), "../../../test/data/push/omnilamp-versioned.json", model.NewRepoSpec("repo"), "", false)
		assert.Error(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, PushErr, res[0].typ)
	})

	t.Run("push with optPath", func(t *testing.T) {
		now := func() time.Time { return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC) }
		e := NewPushExecutor(now)
		id := "omnicorp-TM-department/omnicorp/omnilamp/a/b/c/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id, false)
		r.On("Push", mock.Anything, tmid, mock.Anything).Return(nil)
		r.On("Index", mock.Anything, id).Return(nil)

		res, err := e.Push(context.Background(), "../../../test/data/push/omnilamp-versioned.json", model.NewRepoSpec("repo"), "a/b/c", false)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, PushOK, res[0].typ)
	})
}

func TestPushExecutor_Push_Directory(t *testing.T) {
	r := mocks.NewRepo(t)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("repo"), r, nil))

	t.Run("push directory", func(t *testing.T) {
		clk := testutils.NewTestClock(time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC), time.Second)
		e := NewPushExecutor(clk.Now)
		tmid := model.MustParseTMID("omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json", false)
		r.On("Push", mock.Anything, tmid, mock.Anything).Return(nil)
		tmid = model.MustParseTMID("omnicorp-TM-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json", false)
		r.On("Push", mock.Anything, tmid, mock.Anything).Return(nil)
		tmid = model.MustParseTMID("omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123245-98b3fbd291f4.tm.json", false)
		r.On("Push", mock.Anything, tmid, mock.Anything).Return(&repos.ErrTMIDConflict{Type: repos.IdConflictSameContent,
			ExistingId: "omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"})
		tmid = model.MustParseTMID("omnicorp-TM-department/omnicorp/omnilamp/v0.0.0-20231110123246-575dfac219e2.tm.json", false)
		r.On("Push", mock.Anything, tmid, mock.Anything).Return(&repos.ErrTMIDConflict{Type: repos.IdConflictSameContent,
			ExistingId: "omnicorp-TM-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json"})
		r.On("Index", mock.Anything,
			"omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json",
			"omnicorp-TM-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json").Return(nil)

		res, err := e.Push(context.Background(), "../../../test/data/push", model.NewRepoSpec("repo"), "", false)
		assert.NoError(t, err)
		assert.Len(t, res, 4)
		assert.Equalf(t, PushOK, res[0].typ, "res[0]: want PushOK, got %v", res[0].typ)
		assert.Equalf(t, PushOK, res[1].typ, "res[1]: want PushOK, got %v", res[1].typ)
		assert.Equalf(t, TMExists, res[2].typ, "res[2]: want TMExists, got %v", res[2].typ)
		assert.Equalf(t, TMExists, res[3].typ, "res[3]: want TMExists, got %v", res[3].typ)

	})

	t.Run("push directory with optPath", func(t *testing.T) {
		clk := testutils.NewTestClock(time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC), time.Second)
		e := NewPushExecutor(clk.Now)
		id1 := "omnicorp-TM-department/omnicorp/omnilamp/opt/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id1, false)
		r.On("Push", mock.Anything, tmid, mock.Anything).Return(nil)
		id2 := "omnicorp-TM-department/omnicorp/omnilamp/opt/v0.0.0-20231110123244-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id2, false)
		r.On("Push", mock.Anything, tmid, mock.Anything).Return(nil)
		id3 := "omnicorp-TM-department/omnicorp/omnilamp/opt/v3.2.1-20231110123245-98b3fbd291f4.tm.json"
		tmid = model.MustParseTMID(id3, false)
		r.On("Push", mock.Anything, tmid, mock.Anything).Return(nil)
		id4 := "omnicorp-TM-department/omnicorp/omnilamp/opt/v0.0.0-20231110123246-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id4, false)
		r.On("Push", mock.Anything, tmid, mock.Anything).Return(nil)
		r.On("Index", mock.Anything, id1, id2, id3, id4).Return(nil)

		res, err := e.Push(context.Background(), "../../../test/data/push", model.NewRepoSpec("repo"), "opt", false)
		assert.NoError(t, err)
		assert.Len(t, res, 4)
		for i, r := range res {
			assert.Equalf(t, PushOK, r.typ, "res[%d]: want PushOK, got %v", i, r.typ)
		}

	})

	t.Run("push directory with optTree", func(t *testing.T) {
		clk := testutils.NewTestClock(time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC), time.Second)
		e := NewPushExecutor(clk.Now)
		id1 := "omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id1, false)
		r.On("Push", mock.Anything, tmid, mock.Anything).Return(nil)
		id2 := "omnicorp-TM-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id2, false)
		r.On("Push", mock.Anything, tmid, mock.Anything).Return(nil)
		id3 := "omnicorp-TM-department/omnicorp/omnilamp/subfolder/v3.2.1-20231110123245-98b3fbd291f4.tm.json"
		tmid = model.MustParseTMID(id3, false)
		r.On("Push", mock.Anything, tmid, mock.Anything).Return(nil)
		id4 := "omnicorp-TM-department/omnicorp/omnilamp/subfolder/v0.0.0-20231110123246-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id4, false)
		r.On("Push", mock.Anything, tmid, mock.Anything).Return(nil)
		r.On("Index", mock.Anything, id1, id2, id3, id4).Return(nil)

		res, err := e.Push(context.Background(), "../../../test/data/push", model.NewRepoSpec("repo"), "", true)
		assert.NoError(t, err)
		assert.Len(t, res, 4)
		for i, r := range res {
			assert.Equalf(t, PushOK, r.typ, "res[%d]: want PushOK, got %v", i, r.typ)
		}
	})

}
