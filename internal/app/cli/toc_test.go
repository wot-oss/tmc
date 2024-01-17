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
		err := UpdateToc(rm, remotes.NewRemoteSpec("remoteName"), nil)
		assert.Error(t, err)
	})

	t.Run("error building toc", func(t *testing.T) {
		rm.On("Get", remotes.NewDirSpec("somewhere")).Return(r, nil)
		r.On("UpdateToc").Return(errors.New("something failed")).Once()
		err := UpdateToc(rm, remotes.NewDirSpec("somewhere"), nil)
		assert.ErrorContains(t, err, "something failed")
	})

	t.Run("ok", func(t *testing.T) {
		rm.On("Get", remotes.NewDirSpec("somewhere")).Return(r, nil)
		r.On("UpdateToc").Return(nil).Once()
		err := UpdateToc(rm, remotes.NewDirSpec("somewhere"), nil)
		assert.NoError(t, err)
	})

	t.Run("ok with ids", func(t *testing.T) {
		rm.On("Get", remotes.NewDirSpec("somewhere")).Return(r, nil)
		r.On("UpdateToc", "id1", "id2").Return(nil).Once()
		err := UpdateToc(rm, remotes.NewDirSpec("somewhere"), []string{"id1", "id2"})
		assert.NoError(t, err)
	})
}
