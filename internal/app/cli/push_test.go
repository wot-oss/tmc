package cli

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/testutils"
)

func TestPushExecutor_Push(t *testing.T) {
	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)
	rm.On("Get", remotes.NewRemoteSpec("remote")).Return(r, nil)

	t.Run("push when none exists", func(t *testing.T) {

		now := func() time.Time { return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC) }
		e := NewPushExecutor(rm, now)
		id := "omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id, false)
		r.On("Push", tmid, mock.Anything).Return(nil)
		r.On("UpdateToc", id).Return(nil)

		res, err := e.Push("../../../test/data/push/omnilamp-versioned.json", remotes.NewRemoteSpec("remote"), "", false)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, PushOK, res[0].typ)
	})

	t.Run("push non-existing file", func(t *testing.T) {

		now := func() time.Time { return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC) }
		e := NewPushExecutor(rm, now)
		_, err := e.Push("does-not-exist.json", remotes.NewRemoteSpec("remote"), "", false)
		assert.Error(t, err)
	})

	t.Run("push when remote has the same TM", func(t *testing.T) {

		tmid2 := model.MustParseTMID("omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231111123243-98b3fbd291f4.tm.json", false)

		now := func() time.Time {
			return time.Date(2023, time.November, 11, 12, 32, 43, 0, time.UTC)
		}
		e := NewPushExecutor(rm, now)
		r.On("Push", tmid2, mock.Anything).Return(&remotes.ErrTMExists{ExistingId: "omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"})
		res, err := e.Push("../../../test/data/push/omnilamp-versioned.json", remotes.NewRemoteSpec("remote"), "", false)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, TMExists, res[0].typ)
	})

	t.Run("push fails", func(t *testing.T) {

		tmid3 := model.MustParseTMID("omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20230811123243-98b3fbd291f4.tm.json", false)
		now := func() time.Time {
			return time.Date(2023, time.August, 11, 12, 32, 43, 0, time.UTC)
		}
		e := NewPushExecutor(rm, now)
		r.On("Push", tmid3, mock.Anything).Return(errors.New("unexpected"))
		res, err := e.Push("../../../test/data/push/omnilamp-versioned.json", remotes.NewRemoteSpec("remote"), "", false)
		assert.Error(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, PushErr, res[0].typ)
	})

	t.Run("push with optPath", func(t *testing.T) {
		now := func() time.Time { return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC) }
		e := NewPushExecutor(rm, now)
		id := "omnicorp-TM-department/omnicorp/omnilamp/a/b/c/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id, false)
		r.On("Push", tmid, mock.Anything).Return(nil)
		r.On("UpdateToc", id).Return(nil)

		res, err := e.Push("../../../test/data/push/omnilamp-versioned.json", remotes.NewRemoteSpec("remote"), "a/b/c", false)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, PushOK, res[0].typ)
	})
}

func TestPushExecutor_Push_Directory(t *testing.T) {
	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)
	rm.On("Get", remotes.NewRemoteSpec("remote")).Return(r, nil)

	t.Run("push directory", func(t *testing.T) {
		clk := testutils.NewTestClock(time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC), time.Second)
		e := NewPushExecutor(rm, clk.Now)
		tmid := model.MustParseTMID("omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json", false)
		r.On("Push", tmid, mock.Anything).Return(nil)
		tmid = model.MustParseTMID("omnicorp-TM-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json", false)
		r.On("Push", tmid, mock.Anything).Return(nil)
		tmid = model.MustParseTMID("omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123245-98b3fbd291f4.tm.json", false)
		r.On("Push", tmid, mock.Anything).Return(&remotes.ErrTMExists{ExistingId: "omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"})
		tmid = model.MustParseTMID("omnicorp-TM-department/omnicorp/omnilamp/v0.0.0-20231110123246-575dfac219e2.tm.json", false)
		r.On("Push", tmid, mock.Anything).Return(&remotes.ErrTMExists{ExistingId: "omnicorp-TM-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json"})
		r.On("UpdateToc",
			"omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json",
			"omnicorp-TM-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json").Return(nil)

		res, err := e.Push("../../../test/data/push", remotes.NewRemoteSpec("remote"), "", false)
		assert.NoError(t, err)
		assert.Len(t, res, 4)
		assert.Equalf(t, PushOK, res[0].typ, "res[0]: want PushOK, got %v", res[0].typ)
		assert.Equalf(t, PushOK, res[1].typ, "res[1]: want PushOK, got %v", res[1].typ)
		assert.Equalf(t, TMExists, res[2].typ, "res[2]: want TMExists, got %v", res[2].typ)
		assert.Equalf(t, TMExists, res[3].typ, "res[3]: want TMExists, got %v", res[3].typ)

	})

	t.Run("push directory with optPath", func(t *testing.T) {
		clk := testutils.NewTestClock(time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC), time.Second)
		e := NewPushExecutor(rm, clk.Now)
		id1 := "omnicorp-TM-department/omnicorp/omnilamp/opt/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id1, false)
		r.On("Push", tmid, mock.Anything).Return(nil)
		id2 := "omnicorp-TM-department/omnicorp/omnilamp/opt/v0.0.0-20231110123244-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id2, false)
		r.On("Push", tmid, mock.Anything).Return(nil)
		id3 := "omnicorp-TM-department/omnicorp/omnilamp/opt/v3.2.1-20231110123245-98b3fbd291f4.tm.json"
		tmid = model.MustParseTMID(id3, false)
		r.On("Push", tmid, mock.Anything).Return(nil)
		id4 := "omnicorp-TM-department/omnicorp/omnilamp/opt/v0.0.0-20231110123246-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id4, false)
		r.On("Push", tmid, mock.Anything).Return(nil)
		r.On("UpdateToc", id1, id2, id3, id4).Return(nil)

		res, err := e.Push("../../../test/data/push", remotes.NewRemoteSpec("remote"), "opt", false)
		assert.NoError(t, err)
		assert.Len(t, res, 4)
		for i, r := range res {
			assert.Equalf(t, PushOK, r.typ, "res[%d]: want PushOK, got %v", i, r.typ)
		}

	})

	t.Run("push directory with optTree", func(t *testing.T) {
		clk := testutils.NewTestClock(time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC), time.Second)
		e := NewPushExecutor(rm, clk.Now)
		id1 := "omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123243-98b3fbd291f4.tm.json"
		tmid := model.MustParseTMID(id1, false)
		r.On("Push", tmid, mock.Anything).Return(nil)
		id2 := "omnicorp-TM-department/omnicorp/omnilamp/v0.0.0-20231110123244-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id2, false)
		r.On("Push", tmid, mock.Anything).Return(nil)
		id3 := "omnicorp-TM-department/omnicorp/omnilamp/subfolder/v3.2.1-20231110123245-98b3fbd291f4.tm.json"
		tmid = model.MustParseTMID(id3, false)
		r.On("Push", tmid, mock.Anything).Return(nil)
		id4 := "omnicorp-TM-department/omnicorp/omnilamp/subfolder/v0.0.0-20231110123246-575dfac219e2.tm.json"
		tmid = model.MustParseTMID(id4, false)
		r.On("Push", tmid, mock.Anything).Return(nil)
		r.On("UpdateToc", id1, id2, id3, id4).Return(nil)

		res, err := e.Push("../../../test/data/push", remotes.NewRemoteSpec("remote"), "", true)
		assert.NoError(t, err)
		assert.Len(t, res, 4)
		for i, r := range res {
			assert.Equalf(t, PushOK, r.typ, "res[%d]: want PushOK, got %v", i, r.typ)
		}
	})

}
