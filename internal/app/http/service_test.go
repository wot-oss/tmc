package http

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/repos/mocks"
	rMocks "github.com/wot-oss/tmc/internal/testutils/reposmocks"
	"github.com/wot-oss/tmc/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

var repo = model.NewRepoSpec("someRepo")

func Test_CheckHealthLive(t *testing.T) {
	// given: a service under test
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)
	// when: check health live
	err := underTest.CheckHealthLive(nil)
	// then: there is no error
	assert.NoError(t, err)
}

func Test_CheckHealthReady(t *testing.T) {

	r := mocks.NewRepo(t)
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)

	t.Run("with valid repo", func(t *testing.T) {
		// given: a repo
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repo, r, nil))

		// when check health ready
		err := underTest.CheckHealthReady(nil)
		// then: no error is thrown
		assert.NoError(t, err)
	})

	t.Run("with invalid repo", func(t *testing.T) {
		// given: the repo cannot be found
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repo, nil, errors.New("invalid repo name")))
		// when check health ready
		err := underTest.CheckHealthReady(nil)
		// then: an error is thrown
		assert.Error(t, err)
	})
}

func Test_CheckHealthStartup(t *testing.T) {

	r := mocks.NewRepo(t)
	underTest, _ := NewDefaultHandlerService(repo, repo)

	t.Run("with valid repo", func(t *testing.T) {
		// given: the repo can be found
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repo, r, nil))
		// when check health startup
		err := underTest.CheckHealthStartup(nil)
		// then: no error is thrown
		assert.NoError(t, err)
	})

	t.Run("with invalid repo", func(t *testing.T) {
		// given: the repo cannot be found
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repo, nil, errors.New("invalid repo name")))
		// when check health startup
		err := underTest.CheckHealthStartup(nil)
		// then: an error is thrown
		assert.Error(t, err)
	})
}

func Test_CheckHealth(t *testing.T) {

	r := mocks.NewRepo(t)
	underTest, _ := NewDefaultHandlerService(repo, repo)

	t.Run("with valid repo", func(t *testing.T) {
		// given: the repo can be found
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repo, r, nil))

		// when check health
		err := underTest.CheckHealth(nil)
		// then: no error is thrown
		assert.NoError(t, err)
	})

	t.Run("with invalid repo", func(t *testing.T) {
		// given: the repo cannot be found
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repo, nil, errors.New("invalid repo name")))
		// when check health
		err := underTest.CheckHealth(nil)
		// then: an error is thrown
		assert.Error(t, err)
	})
}

func Test_ListInventory(t *testing.T) {

	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)

	listResult := model.SearchResult{
		Entries: []model.FoundEntry{
			{
				Name:         "a-corp/eagle/bt2000",
				Author:       model.SchemaAuthor{Name: "a-corp"},
				Manufacturer: model.SchemaManufacturer{Name: "eagle"},
				Mpn:          "bt2000",
				Versions: []model.FoundVersion{
					{
						IndexVersion: model.IndexVersion{
							TMID:        "a-corp/eagle/bt2000/v1.0.0-20240108140117-243d1b462ccc.tm.json",
							Description: "desc version v1.0.0",
							Version:     model.Version{Model: "1.0.0"},
							Digest:      "243d1b462ccc",
							TimeStamp:   "20240108140117",
							ExternalID:  "ext-2",
						},
						FoundIn: model.FoundSource{RepoName: "r1"},
					},
					{
						IndexVersion: model.IndexVersion{
							TMID:        "a-corp/eagle/bt2000/v1.0.0-20231231153548-243d1b462ddd.tm.json",
							Description: "desc version v0.0.0",
							Version:     model.Version{Model: "0.0.0"},
							Digest:      "243d1b462ddd",
							TimeStamp:   "20231231153548",
							ExternalID:  "ext-1",
						},
						FoundIn: model.FoundSource{RepoName: "r1"},
					},
				},
			},
			{
				Name:         "b-corp/frog/bt3000",
				Author:       model.SchemaAuthor{Name: "b-corp"},
				Manufacturer: model.SchemaManufacturer{Name: "frog"},
				Mpn:          "bt3000",
				Versions: []model.FoundVersion{
					{
						IndexVersion: model.IndexVersion{
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

	t.Run("list all", func(t *testing.T) {
		// given: repo having some inventory entries
		r := mocks.NewRepo(t)
		searchParams := &model.SearchParams{Author: []string{"a-corp", "b-corp"}}
		r.On("List", mock.Anything, searchParams).Return(listResult, nil).Once()
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: list all
		res, err := underTest.ListInventory(context.Background(), searchParams)
		// then: there is no error
		assert.NoError(t, err)
		// and then: the search result is returned
		assert.Equal(t, &listResult, res)
	})
	t.Run("list with one upstream error", func(t *testing.T) {
		// given: repo having some inventory entries
		r := mocks.NewRepo(t)
		r2 := mocks.NewRepo(t)
		var sp *model.SearchParams
		r.On("List", mock.Anything, sp).Return(listResult, nil).Once()
		r2.On("List", mock.Anything, sp).Return(model.SearchResult{}, errors.New("unexpected")).Once()
		r2.On("Spec").Return(model.NewRepoSpec("r2")).Once()
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r, r2))
		// when: list all
		res, err := underTest.ListInventory(context.Background(), sp)
		// then: there is an error of type repos.RepoAccessError
		var aErr *repos.RepoAccessError
		assert.ErrorAs(t, err, &aErr)
		// and then: the search result is returned
		assert.Nil(t, res)
	})
}

func Test_GetCompletions(t *testing.T) {
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)

	t.Run("list names", func(t *testing.T) {
		// given: repo having some inventory entries
		r := mocks.NewRepo(t)
		names := []string{"a/b/c", "d/e/f"}
		r.On("ListCompletions", mock.Anything, "names", "toComplete").Return(names, nil)
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: list all
		res, err := underTest.GetCompletions(context.Background(), "names", "toComplete")
		// then: there is no error
		assert.NoError(t, err)
		// and then: the search result is returned
		assert.Equal(t, names, res)
	})
}

func Test_FindInventoryEntry(t *testing.T) {

	t.Run("inventory entry cannot be found", func(t *testing.T) {
		underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)
		inventoryName := "a/b/c"
		// given: repo returns empty search result
		r := mocks.NewRepo(t)
		r.On("List", mock.Anything, &model.SearchParams{Name: inventoryName}).Return(model.SearchResult{}, nil).Once()
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: finding entry
		res, err := underTest.FindInventoryEntry(context.Background(), inventoryName)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is status code 404
		sErr, ok := err.(*BaseHttpError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusNotFound, sErr.Status)
	})
}

func Test_ListAuthors(t *testing.T) {

	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)

	// given: some inventory entries with unordered and duplicated authors
	listResult := model.SearchResult{
		Entries: []model.FoundEntry{
			{
				Name:   "z-corp/eagle/bt2000",
				Author: model.SchemaAuthor{Name: "z-corp"},
			},
			{
				Name:   "a-corp/frog/bt4000",
				Author: model.SchemaAuthor{Name: "a-corp"},
			},
			{
				Name:   "a-corp/frog/bt7000",
				Author: model.SchemaAuthor{Name: "a-corp"},
			},
		},
	}

	t.Run("list all", func(t *testing.T) {
		// given: repo returning the inventory entries
		r := mocks.NewRepo(t)
		r.On("List", mock.Anything, &model.SearchParams{}).Return(listResult, nil).Once()
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r))

		// when: list all authors
		res, err := underTest.ListAuthors(context.Background(), &model.SearchParams{})
		// then: there is no error
		assert.NoError(t, err)
		// and then: the result is sorted asc by name
		isSorted := sort.SliceIsSorted(res, func(i, j int) bool {
			return res[i] < res[j]
		})
		assert.True(t, isSorted)
		// and then: the result contains no duplicates
		assert.Equal(t, []string{"a-corp", "z-corp"}, res)
	})
}

func Test_ListManufacturers(t *testing.T) {

	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)

	// given: some inventory entries with unordered and duplicated manufacturers
	listResult := model.SearchResult{
		Entries: []model.FoundEntry{
			{
				Name:         "a-corp/frog/bt4000",
				Manufacturer: model.SchemaManufacturer{Name: "frog"},
			},
			{
				Name:         "z-corp/eagle/bt2000",
				Manufacturer: model.SchemaManufacturer{Name: "eagle"},
			},
			{
				Name:         "a-corp/frog/bt7000",
				Manufacturer: model.SchemaManufacturer{Name: "frog"},
			},
		},
	}

	t.Run("list all", func(t *testing.T) {
		// given: repo returning the inventory entries
		r := mocks.NewRepo(t)
		r.On("List", mock.Anything, &model.SearchParams{}).Return(listResult, nil).Once()
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r))

		// when: list all manufacturers
		res, err := underTest.ListManufacturers(context.Background(), &model.SearchParams{})
		// then: there is no error
		assert.NoError(t, err)
		// and then: the result is sorted asc by name
		isSorted := sort.SliceIsSorted(res, func(i, j int) bool {
			return res[i] < res[j]
		})
		assert.True(t, isSorted)
		// and then: the result contains no duplicates
		assert.Equal(t, []string{"eagle", "frog"}, res)
	})
}

func Test_ListMpns(t *testing.T) {

	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)

	// given: some inventory entries with unordered and duplicated mpns
	listResult := model.SearchResult{
		Entries: []model.FoundEntry{
			{
				Name: "a-corp/frog/bt4000",
				Mpn:  "bt4000",
			},
			{
				Name: "z-corp/eagle/bt2000",
				Mpn:  "bt2000",
			},
			{
				Name: "a-corp/frog/bt4000",
				Mpn:  "bt4000",
			},
		},
	}

	t.Run("list all", func(t *testing.T) {
		// given: repo returning the inventory entries
		r := mocks.NewRepo(t)
		r.On("List", mock.Anything, &model.SearchParams{}).Return(listResult, nil).Once()
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r))

		// when: list all
		res, err := underTest.ListMpns(context.Background(), &model.SearchParams{})
		// then: there is no error
		assert.NoError(t, err)
		// and then: the result is sorted asc by name
		isSorted := sort.SliceIsSorted(res, func(i, j int) bool {
			return res[i] < res[j]
		})
		assert.True(t, isSorted)
		// and then: the result contains no duplicates
		assert.Equal(t, []string{"bt2000", "bt4000"}, res)
	})
}

func Test_FetchingThingModel(t *testing.T) {

	r := mocks.NewRepo(t)
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)

	t.Run("with invalid tmID", func(t *testing.T) {
		invalidTmID := ""
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(nil, invalidTmID, false)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrInvalidFetchName
		assert.ErrorIs(t, err, commands.ErrInvalidFetchName)
	})

	t.Run("with invalid fetch name", func(t *testing.T) {
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(nil, "b-corp\\eagle/PM20", false)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrInvalidFetchName
		assert.ErrorIs(t, err, commands.ErrInvalidFetchName)
	})

	t.Run("with invalid semantic version", func(t *testing.T) {
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(nil, "b-corp/eagle/PM20:v1.", false)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrInvalidFetchName
		assert.ErrorIs(t, err, commands.ErrInvalidFetchName)
	})

	t.Run("with tmID not found", func(t *testing.T) {
		tmID := "b-corp/eagle/pm20/v1.0.0-20240107123001-234d1b462fff.tm.json"
		r.On("Fetch", mock.Anything, tmID).Return(tmID, nil, repos.ErrTmNotFound).Once()
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(context.Background(), tmID, false)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrTmNotFound
		assert.ErrorIs(t, err, repos.ErrTmNotFound)
	})

	t.Run("with fetch name not found", func(t *testing.T) {
		fn := "b-corp/eagle/pm20"
		r.On("Versions", mock.Anything, fn).Return(nil, nil).Once()
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(context.Background(), fn, false)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrTmNotFound
		assert.ErrorIs(t, err, repos.ErrTmNotFound)
	})

	t.Run("with tmID found", func(t *testing.T) {
		_, raw, err := utils.ReadRequiredFile("../../../test/data/push/omnilamp.json")
		tmID := "b-corp/eagle/pm20/v1.0.0-20240107123001-234d1b462fff.tm.json"
		r.On("Fetch", mock.Anything, tmID).Return(tmID, raw, nil).Once()
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(context.Background(), tmID, false)
		// then: it returns the unchanged ThingModel content
		assert.NotNil(t, res)
		assert.Equal(t, raw, res)
		// and then: there is no error
		assert.NoError(t, err)
	})
}
func Test_DeleteThingModel(t *testing.T) {

	r := mocks.NewRepo(t)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repo, r, nil))
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)

	t.Run("without errors", func(t *testing.T) {
		tmid := "some-id"
		r.On("Delete", mock.Anything, tmid).Return(nil).Once()
		r.On("Index", mock.Anything, tmid).Return(nil).Once()
		// when: deleting ThingModel
		err := underTest.DeleteThingModel(context.Background(), tmid)
		// then: it returns nil result
		assert.NoError(t, err)
	})

	t.Run("with error when deleting", func(t *testing.T) {
		tmid := "some-id2"
		r.On("Delete", mock.Anything, tmid).Return(repos.ErrTmNotFound).Once()
		// when: deleting ThingModel
		err := underTest.DeleteThingModel(context.Background(), tmid)
		// then: it returns error result
		assert.ErrorIs(t, err, repos.ErrTmNotFound)
	})

	t.Run("with error when indexing", func(t *testing.T) {
		tmid := "some-id3"
		r.On("Delete", mock.Anything, tmid).Return(nil).Once()
		r.On("Index", mock.Anything, tmid).Return(errors.New("could not update index")).Once()
		// when: deleting ThingModel
		err := underTest.DeleteThingModel(context.Background(), tmid)
		// then: it returns error result
		assert.ErrorContains(t, err, "could not update index")
	})
}

func Test_HandlerService_PushThingModel(t *testing.T) {
	r := mocks.NewRepo(t)
	pushTarget := model.NewRepoSpec("pushRepo")
	underTest, _ := NewDefaultHandlerService(repo, pushTarget)

	t.Run("with validation error", func(t *testing.T) {
		// given: some invalid content for a ThingModel
		invalidContent := []byte("invalid content")
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, pushTarget, r, nil))
		// when: pushing ThingModel
		res, err := underTest.PushThingModel(nil, invalidContent, repos.PushOptions{})
		// then: it returns an error PushResult
		assert.Equal(t, repos.PushResult{
			Type: repos.PushResultError,
			Text: "invalid character 'i' looking for beginning of value",
			TmID: "",
		}, res)
		// and then: there is an error
		assert.Error(t, err)
	})

	t.Run("with push repo name that cannot be found", func(t *testing.T) {
		// given: invalid pushTarget
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, pushTarget, nil, repos.ErrRepoNotFound))
		// when: pushing ThingModel
		res, err := underTest.PushThingModel(nil, []byte("some TM content"), repos.PushOptions{})
		// then: it returns empty tmID
		assert.Equal(t, repos.PushResult{
			Type: repos.PushResultError,
			Text: "repo not found",
			TmID: "",
		}, res)
		// and then: there is an error
		assert.Error(t, err)
		// and then: the error says that the repo cannot be found
		assert.Equal(t, repos.ErrRepoNotFound, err)
	})
	t.Run("with content conflict", func(t *testing.T) {
		// given: some valid content for a ThingModel
		_, tmContent, _ := utils.ReadRequiredFile("../../../test/data/push/omnilamp.json")
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, pushTarget, r, nil))
		cErr := &repos.ErrTMIDConflict{
			Type:       repos.IdConflictSameContent,
			ExistingId: "existing-id",
		}
		expRes := repos.PushResult{repos.PushResultTMExists, "", "existing-id"}
		r.On("Push", mock.Anything, mock.Anything, mock.Anything, repos.PushOptions{}).Return(expRes, cErr).Once()
		// when: pushing ThingModel
		res, err := underTest.PushThingModel(nil, tmContent, repos.PushOptions{})
		// then: it returns empty tmID
		assert.Equal(t, expRes, res)
		// and then: there is an error
		assert.Equal(t, cErr, err)
	})
	t.Run("with timestamp conflict", func(t *testing.T) {
		// given: some valid content for a ThingModel
		_, tmContent, _ := utils.ReadRequiredFile("../../../test/data/push/omnilamp.json")
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, pushTarget, r, nil))
		cErr := &repos.ErrTMIDConflict{
			Type:       repos.IdConflictSameTimestamp,
			ExistingId: "existing-id",
		}
		expRes := repos.PushResult{repos.PushResultWarning, cErr.Error(), "existing-id"}
		r.On("Push", mock.Anything, mock.Anything, mock.Anything, repos.PushOptions{}).Return(expRes, nil).Once()
		r.On("Index", mock.Anything, mock.Anything).Return(nil)
		// when: pushing ThingModel
		res, err := underTest.PushThingModel(nil, tmContent, repos.PushOptions{})
		// then: it returns expected warning result
		assert.Equal(t, expRes, res)
		// and then: there is no error
		assert.NoError(t, err)
	})
}
