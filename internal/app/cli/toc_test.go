package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

func TestUpdateToc(t *testing.T) {
	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)

	t.Run("no remote", func(t *testing.T) {
		rm.On("Get", remotes.NewRemoteSpec("remoteName")).Return(nil, remotes.ErrRemoteNotFound)
		err := UpdateToc(rm, remotes.NewRemoteSpec("remoteName"))
		assert.Error(t, err)
	})

	t.Run("error building toc", func(t *testing.T) {
		rm.On("Get", remotes.NewDirSpec("somewhere")).Return(r, nil)
		r.On("CreateToC").Return(errors.New("something failed")).Once()
		err := UpdateToc(rm, remotes.NewDirSpec("somewhere"))
		assert.ErrorContains(t, err, "something failed")
	})

	t.Run("ok", func(t *testing.T) {
		rm.On("Get", remotes.NewDirSpec("somewhere")).Return(r, nil)
		r.On("CreateToC").Return(nil).Once()
		err := UpdateToc(rm, remotes.NewDirSpec("somewhere"))
		assert.NoError(t, err)
	})
}
