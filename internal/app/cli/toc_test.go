package cli

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes/mocks"
)

func TestUpdateToc(t *testing.T) {
	r := mocks.NewRemote(t)

	t.Run("no remote", func(t *testing.T) {
		remotes.MockRemotesGet(t, func(s model.RepoSpec) (remotes.Remote, error) {
			if reflect.DeepEqual(model.NewRemoteSpec("remoteName"), s) {
				return nil, remotes.ErrRemoteNotFound
			}
			err := fmt.Errorf("unexpected spec in mock: %v", s)
			remotes.MockFail(t, err)
			return nil, err

		})

		err := UpdateToc(model.NewRemoteSpec("remoteName"), nil)
		assert.Error(t, err)
	})

	t.Run("error building toc", func(t *testing.T) {
		remotes.MockRemotesGet(t, func(s model.RepoSpec) (remotes.Remote, error) {
			if reflect.DeepEqual(model.NewDirSpec("somewhere"), s) {
				return r, nil
			}
			err := fmt.Errorf("unexpected spec in mock: %v", s)
			remotes.MockFail(t, err)
			return nil, err

		})

		r.On("UpdateToc").Return(errors.New("something failed")).Once()
		err := UpdateToc(model.NewDirSpec("somewhere"), nil)
		assert.ErrorContains(t, err, "something failed")
	})

	t.Run("ok", func(t *testing.T) {
		remotes.MockRemotesGet(t, func(s model.RepoSpec) (remotes.Remote, error) {
			if reflect.DeepEqual(model.NewDirSpec("somewhere"), s) {
				return r, nil
			}
			err := fmt.Errorf("unexpected spec in mock: %v", s)
			remotes.MockFail(t, err)
			return nil, err

		})
		r.On("UpdateToc").Return(nil).Once()
		err := UpdateToc(model.NewDirSpec("somewhere"), nil)
		assert.NoError(t, err)
	})

	t.Run("ok with ids", func(t *testing.T) {
		remotes.MockRemotesGet(t, func(s model.RepoSpec) (remotes.Remote, error) {
			if reflect.DeepEqual(model.NewDirSpec("somewhere"), s) {
				return r, nil
			}
			err := fmt.Errorf("unexpected spec in mock: %v", s)
			remotes.MockFail(t, err)
			return nil, err

		})
		r.On("UpdateToc", "id1", "id2").Return(nil).Once()
		err := UpdateToc(model.NewDirSpec("somewhere"), []string{"id1", "id2"})
		assert.NoError(t, err)
	})
}
