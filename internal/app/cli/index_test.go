package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/repos/mocks"
	rMocks "github.com/wot-oss/tmc/internal/testutils/reposmocks"
)

func TestIndex(t *testing.T) {
	r := mocks.NewRepo(t)

	t.Run("no repo", func(t *testing.T) {
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("repoName"), nil, repos.ErrRepoNotFound))

		err := Index(context.Background(), model.NewRepoSpec("repoName"), nil)
		assert.Error(t, err)
	})

	t.Run("error building index", func(t *testing.T) {
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewDirSpec("somewhere"), r, nil))

		r.On("Index", mock.Anything).Return(errors.New("something failed")).Once()
		err := Index(context.Background(), model.NewDirSpec("somewhere"), nil)
		assert.ErrorContains(t, err, "something failed")
	})

	t.Run("ok", func(t *testing.T) {
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewDirSpec("somewhere"), r, nil))
		r.On("Index", mock.Anything).Return(nil).Once()
		err := Index(context.Background(), model.NewDirSpec("somewhere"), nil)
		assert.NoError(t, err)
	})

	t.Run("ok with ids", func(t *testing.T) {
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewDirSpec("somewhere"), r, nil))
		r.On("Index", mock.Anything, "id1", "id2").Return(nil).Once()
		err := Index(context.Background(), model.NewDirSpec("somewhere"), []string{"id1", "id2"})
		assert.NoError(t, err)
	})
}
