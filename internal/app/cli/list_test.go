package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/repos/mocks"
	"github.com/wot-oss/tmc/internal/testutils"
	rMocks "github.com/wot-oss/tmc/internal/testutils/reposmocks"
)

func TestElideString(t *testing.T) {
	testCases := []struct {
		colWidth int
		input    string
		expected string
	}{
		{
			colWidth: 10,
			input:    "",
			expected: "",
		},
		{
			colWidth: 10,
			input:    "testing",
			expected: "testing",
		},
		{
			colWidth: 4,
			input:    "testing",
			expected: "t...",
		},
	}
	for _, test := range testCases {
		out := elideString(test.input, test.colWidth)
		if test.expected != elideString(test.input, test.colWidth) {
			t.Errorf("failed eliding '%s' to %d characters:", test.input, test.colWidth)
			t.Errorf("expected '%s' got '%s'", test.expected, out)
		}
	}
}

var listRes = model.SearchResult{
	Entries: []model.FoundEntry{
		{
			Name:         "b-corp/frog/bt3000",
			Author:       model.SchemaAuthor{Name: "b-corp"},
			Manufacturer: model.SchemaManufacturer{Name: "frog"},
			Mpn:          "bt3000",
			Versions: []model.FoundVersion{
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
			},
		},
	},
}

func TestList(t *testing.T) {

	t.Run("with ok", func(t *testing.T) {
		restoreStdout, getStdout := testutils.ReplaceStdout()
		restoreStderr, getStderr := testutils.ReplaceStderr()
		defer restoreStdout()
		defer restoreStderr()

		// given: a RepoManager and a repo having 1 ThingModel
		repoSpec := model.NewRepoSpec("r1")
		r := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repoSpec, r, nil))

		var sp *model.Filters

		r.On("Spec").Return(repoSpec).Maybe()
		r.On("List", mock.Anything, sp).Return(listRes, nil).Once()

		// when: list from the repo
		err := List(context.Background(), repoSpec, sp, OutputFormatPlain)
		stdout := getStdout()
		stderr := getStderr()

		// then: there is no error
		assert.NoError(t, err)
		// and then: stdout outputs the listable ThingModel
		assert.Contains(t, stdout, listRes.Entries[0].Name)
		var actual any
		err = json.Unmarshal([]byte(stdout), &actual)
		assert.Error(t, err)
		// and then: stderr has no outputs
		assert.Equal(t, "", stderr)
	})
	t.Run("with ok output json", func(t *testing.T) {
		restoreStdout, getStdout := testutils.ReplaceStdout()
		restoreStderr, getStderr := testutils.ReplaceStderr()
		defer restoreStdout()
		defer restoreStderr()

		// given: a RepoManager and a repo having 1 ThingModel
		repoSpec := model.NewRepoSpec("r1")
		r := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repoSpec, r, nil))

		var sp *model.Filters

		r.On("Spec").Return(repoSpec).Maybe()
		r.On("List", mock.Anything, sp).Return(listRes, nil).Once()

		// when: list from the repo
		err := List(context.Background(), repoSpec, sp, OutputFormatJSON)
		stdout := getStdout()
		stderr := getStderr()

		// then: there is no error
		assert.NoError(t, err)
		// and then: stdout outputs the listable ThingModel
		assert.Contains(t, stdout, fmt.Sprintf("\"name\": \"%s\"", listRes.Entries[0].Name))
		var actual any
		err = json.Unmarshal([]byte(stdout), &actual)
		assert.NoError(t, err)
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

		var sp *model.Filters

		// and given: repo 1 can be listed
		r1.On("Spec").Return(repoSpec1).Maybe()
		r1.On("List", mock.Anything, sp).Return(listRes, nil).Once()

		// and given: repo 2 returns an error when accessing
		accessError := errors.New("some repo access error")
		r2.On("Spec").Return(repoSpec2).Maybe()
		r2.On("List", mock.Anything, sp).Return(model.SearchResult{}, accessError).Once()

		// when: list from both repos
		err := List(context.Background(), model.EmptySpec, sp, OutputFormatPlain)
		stdout := getStdout()
		stderr := getStderr()

		// then: there is a total error
		assert.Error(t, err)
		// and then: stdout outputs the listable ThingModel
		assert.Contains(t, stdout, listRes.Entries[0].Name)
		// and then: stderr outputs error for the repo that could not be accessed
		assert.Contains(t, stderr, accessError.Error())
	})
}
