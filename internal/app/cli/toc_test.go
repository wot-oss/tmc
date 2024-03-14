package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes/mocks"
	rMocks "github.com/web-of-things-open-source/tm-catalog-cli/internal/testutils/remotesmocks"
)

func TestUpdateToc(t *testing.T) {
	r := mocks.NewRemote(t)

	t.Run("no remote", func(t *testing.T) {
		rMocks.MockRemotesGet(t, rMocks.CreateMockGetFunction(t, model.NewRemoteSpec("remoteName"), nil, remotes.ErrRemoteNotFound))

		err := UpdateToc(model.NewRemoteSpec("remoteName"), nil)
		assert.Error(t, err)
	})

	t.Run("error building toc", func(t *testing.T) {
		rMocks.MockRemotesGet(t, rMocks.CreateMockGetFunction(t, model.NewDirSpec("somewhere"), r, nil))

		r.On("UpdateToc").Return(errors.New("something failed")).Once()
		err := UpdateToc(model.NewDirSpec("somewhere"), nil)
		assert.ErrorContains(t, err, "something failed")
	})

	t.Run("ok", func(t *testing.T) {
		rMocks.MockRemotesGet(t, rMocks.CreateMockGetFunction(t, model.NewDirSpec("somewhere"), r, nil))
		r.On("UpdateToc").Return(nil).Once()
		err := UpdateToc(model.NewDirSpec("somewhere"), nil)
		assert.NoError(t, err)
	})

	t.Run("ok with ids", func(t *testing.T) {
		rMocks.MockRemotesGet(t, rMocks.CreateMockGetFunction(t, model.NewDirSpec("somewhere"), r, nil))
		r.On("UpdateToc", "id1", "id2").Return(nil).Once()
		err := UpdateToc(model.NewDirSpec("somewhere"), []string{"id1", "id2"})
		assert.NoError(t, err)
	})
}
