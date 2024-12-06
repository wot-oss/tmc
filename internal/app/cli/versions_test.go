package cli

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/repos/mocks"
	"github.com/wot-oss/tmc/internal/testutils"
	rMocks "github.com/wot-oss/tmc/internal/testutils/reposmocks"
)

var versionsRes = []model.FoundVersion{
	{
		IndexVersion: &model.IndexVersion{
			TMID:        "b-corp/frog/bt3000/v1.0.0-20240108140117-743d1b462uuu.tm.json",
			Description: "desc version v1.0.0",
			Version:     model.Version{Model: "1.0.0"},
			Digest:      "743d1b462uuu",
			TimeStamp:   "20240108140117",
			ExternalID:  "ext-3",
		},
		FoundIn: model.FoundSource{RepoName: "r1"},
	},
}

func TestListVersions(t *testing.T) {
	tmName := "b-corp/frog/bt3000"

	t.Run("with ok", func(t *testing.T) {
		restoreStdout, getStdout := testutils.ReplaceStdout()
		restoreStderr, getStderr := testutils.ReplaceStderr()
		defer restoreStdout()
		defer restoreStderr()

		// given: a RepoManager and a repo
		repoSpec := model.NewRepoSpec("r1")
		r := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repoSpec, r, nil))

		r.On("Spec").Return(repoSpec).Maybe()
		r.On("Versions", mock.Anything, tmName).Return(versionsRes, nil).Once()

		// when: list versions from the repo for given ThingModel name
		err := ListVersions(context.Background(), repoSpec, tmName, OutputFormatPlain)
		stdout := getStdout()
		stderr := getStderr()

		// then: there is no error
		assert.NoError(t, err)
		// and then: stdout outputs the versions of the ThingModel
		assert.Contains(t, stdout, tmName)
		// and then: stderr has no outputs
		assert.Equal(t, "", stderr)
	})
	t.Run("with ok json output", func(t *testing.T) {
		restoreStdout, getStdout := testutils.ReplaceStdout()
		restoreStderr, getStderr := testutils.ReplaceStderr()
		defer restoreStdout()
		defer restoreStderr()

		// given: a RepoManager and a repo
		repoSpec := model.NewRepoSpec("r1")
		r := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repoSpec, r, nil))

		r.On("Spec").Return(repoSpec).Maybe()
		r.On("Versions", mock.Anything, tmName).Return(versionsRes, nil).Once()

		// when: list versions from the repo for given ThingModel name
		err := ListVersions(context.Background(), repoSpec, tmName, OutputFormatJSON)
		stdout := getStdout()
		stderr := getStderr()

		// then: there is no error
		assert.NoError(t, err)
		// and then: stdout outputs the versions of the ThingModel
		assert.Contains(t, stdout, tmName)
		var actual any
		err = json.Unmarshal([]byte(stdout), &actual)
		assert.NoError(t, err)
		expected := []any{map[string]any{"description": "desc version v1.0.0", "id": "b-corp/frog/bt3000/v1.0.0-20240108140117-743d1b462uuu.tm.json", "name": "b-corp/frog/bt3000", "repo": "r1", "version": "1.0.0"}}
		assert.Equal(t, expected, actual)
		// and then: stderr has no outputs
		assert.Equal(t, "", stderr)
	})

	t.Run("with error accessing a repo", func(t *testing.T) {
		restoreStdout, getStdout := testutils.ReplaceStdout()
		restoreStderr, getStderr := testutils.ReplaceStderr()
		defer restoreStdout()
		defer restoreStderr()

		// given: a RepoManager and 2 repos
		repoSpec1 := model.NewRepoSpec("r1")
		repoSpec2 := model.NewRepoSpec("r2")
		r1 := mocks.NewRepo(t)
		r2 := mocks.NewRepo(t)

		repoMap := map[string]repos.Repo{repoSpec1.RepoName(): r1, repoSpec2.RepoName(): r2}
		rMocks.MockReposGet(t, func(s model.RepoSpec) (repos.Repo, error) {
			return repoMap[s.RepoName()], nil
		})
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r1, r2))

		// and given: repo 1 returns a version for the given ThingModel name
		r1.On("Spec").Return(repoSpec1).Maybe()
		r1.On("Versions", mock.Anything, tmName).Return(versionsRes, nil).Once()

		// and given: repo 2 returns an error when accessing
		accessError := errors.New("some repo access error")
		r2.On("Spec").Return(repoSpec2).Maybe()
		r2.On("Versions", mock.Anything, tmName).Return([]model.FoundVersion{}, accessError).Once()

		// when: list versions from both repos
		err := ListVersions(context.Background(), model.EmptySpec, tmName, OutputFormatPlain)
		stdout := getStdout()
		stderr := getStderr()

		// then: there is a total error
		assert.Error(t, err)
		// and then: stdout outputs the versions of the ThingModel
		assert.Contains(t, stdout, tmName)
		// and then: stderr outputs error for the repo that could not be accessed
		assert.Contains(t, stderr, accessError.Error())
	})
}
