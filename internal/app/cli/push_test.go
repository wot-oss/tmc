package cli

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

func TestPushExecutor_Push(t *testing.T) {
	now := func() time.Time {
		return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC)
	}

	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)

	rm.On("Get", "remote").Return(r, nil)
	e := NewPushExecutor(rm, now)
	tmid := model.MustParseTMID("omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123243-1e788769a659.tm.json", false)
	tmid.Name = ""
	r.On("Push", tmid, mock.Anything).Return(nil)
	r.On("CreateToC").Return(nil)

	res, err := e.Push("../../../test/data/push/omnilamp-versioned.json", "remote", "", false)
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, PushOK, res[0].typ)

	res, err = e.Push("does-not-exist.json", "remote", "", false)
	assert.Error(t, err)

	tmid2 := model.MustParseTMID("omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231111123243-1e788769a659.tm.json", false)
	tmid2.Name = ""

	now = func() time.Time {
		return time.Date(2023, time.November, 11, 12, 32, 43, 0, time.UTC)
	}
	e = NewPushExecutor(rm, now)
	r.On("Push", tmid2, mock.Anything).Return(&remotes.ErrTMExists{ExistingId: "omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20231110123243-1e788769a659.tm.json"})
	res, err = e.Push("../../../test/data/push/omnilamp-versioned.json", "remote", "", false)
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, TMExists, res[0].typ)

	tmid3 := model.MustParseTMID("omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20230811123243-1e788769a659.tm.json", false)
	tmid3.Name = ""
	now = func() time.Time {
		return time.Date(2023, time.August, 11, 12, 32, 43, 0, time.UTC)
	}
	e = NewPushExecutor(rm, now)
	r.On("Push", tmid3, mock.Anything).Return(errors.New("unexpected"))
	res, err = e.Push("../../../test/data/push/omnilamp-versioned.json", "remote", "", false)
	assert.Error(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, PushErr, res[0].typ)

	//fixme: test with optpath

}

func TestPushExecutor_Push_Directory(t *testing.T) {
	//fixme: write code here
}
