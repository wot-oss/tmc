package commands

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos/mocks"
	rMocks "github.com/wot-oss/tmc/internal/testutils/reposmocks"
)

func TestVersionsCommand_ListVersions(t *testing.T) {

	t.Run("merged", func(t *testing.T) {

		r1 := mocks.NewRepo(t)
		r2 := mocks.NewRepo(t)
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r1, r2))
		r1.On("Versions", "senseall").Return(
			[]model.FoundVersion{
				{
					IndexVersion: model.IndexVersion{
						TMID: "omnicorp/senseall/v0.36.0-20231231153548-243d1b462ccc.tm.json",
					},
					FoundIn: model.FoundSource{RepoName: "r1"},
				},
				{
					IndexVersion: model.IndexVersion{
						TMID: "omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb.tm.json",
					},
					FoundIn: model.FoundSource{RepoName: "r1"},
				},
			}, nil)
		r2.On("Versions", "senseall").Return([]model.FoundVersion{
			{
				IndexVersion: model.IndexVersion{
					TMID: "omnicorp/senseall/v0.34.0-20231130153548-243d1b462aaa.tm.json",
				},
				FoundIn: model.FoundSource{RepoName: "r2"},
			},
			{
				IndexVersion: model.IndexVersion{
					TMID: "omnicorp/senseall/v0.35.0-20231230173548-243d1b462bbb.tm.json",
				},
				FoundIn: model.FoundSource{RepoName: "r2"},
			},
		}, nil)
		c := NewVersionsCommand()
		res, err, errs := c.ListVersions(model.EmptySpec, "senseall")

		assert.NoError(t, err)
		assert.Len(t, errs, 0)
		assert.Len(t, res, 3)
		assert.Equal(t, []model.FoundVersion{
			{
				IndexVersion: model.IndexVersion{TMID: "omnicorp/senseall/v0.34.0-20231130153548-243d1b462aaa.tm.json"},
				FoundIn:      model.FoundSource{RepoName: "r2"},
			},
			{
				IndexVersion: model.IndexVersion{TMID: "omnicorp/senseall/v0.35.0-20231230173548-243d1b462bbb.tm.json"},
				FoundIn:      model.FoundSource{RepoName: "r2"},
			},
			{
				IndexVersion: model.IndexVersion{TMID: "omnicorp/senseall/v0.36.0-20231231153548-243d1b462ccc.tm.json"},
				FoundIn:      model.FoundSource{RepoName: "r1"},
			},
		}, res)

	})

	t.Run("one error", func(t *testing.T) {

		r1 := mocks.NewRepo(t)
		r2 := mocks.NewRepo(t)
		r2.On("Spec").Return(model.NewRepoSpec("r2"))
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r1, r2))
		r1.On("Versions", "senseall").Return(
			[]model.FoundVersion{
				{
					IndexVersion: model.IndexVersion{
						TMID: "omnicorp/senseall/v0.36.0-20231231153548-243d1b462ccc.tm.json",
					},
					FoundIn: model.FoundSource{RepoName: "r1"},
				},
				{
					IndexVersion: model.IndexVersion{
						TMID: "omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb.tm.json",
					},
					FoundIn: model.FoundSource{RepoName: "r1"},
				},
			}, nil)
		r2.On("Versions", "senseall").Return(nil, errors.New("unexpected error"))
		c := NewVersionsCommand()
		res, err, errs := c.ListVersions(model.EmptySpec, "senseall")
		if assert.Len(t, errs, 1) {
			assert.ErrorContains(t, errs[0], "unexpected error")
		}
		assert.NoError(t, err)
		assert.Len(t, res, 2)
		assert.Equal(t, []model.FoundVersion{
			{
				IndexVersion: model.IndexVersion{TMID: "omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb.tm.json"},
				FoundIn:      model.FoundSource{RepoName: "r1"},
			},
			{
				IndexVersion: model.IndexVersion{TMID: "omnicorp/senseall/v0.36.0-20231231153548-243d1b462ccc.tm.json"},
				FoundIn:      model.FoundSource{RepoName: "r1"},
			},
		}, res)

	})

}
