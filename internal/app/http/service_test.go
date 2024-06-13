package http

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"testing"

	"github.com/stretchr/testify/mock"
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
		args := []string{"arg0", "arg1"}
		r.On("ListCompletions", mock.Anything, "names", args, "toComplete").Return(names, nil)
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: list all
		res, err := underTest.GetCompletions(context.Background(), "names", args, "toComplete")
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

func TestService_FetchThingModel(t *testing.T) {
	r := mocks.NewRepo(t)
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)

	t.Run("with invalid tmID", func(t *testing.T) {
		invalidTmID := ""
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(nil, invalidTmID, false)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrInvalidId
		assert.ErrorIs(t, err, model.ErrInvalidId)
	})

	t.Run("with tmID not found", func(t *testing.T) {
		tmID := "b-corp/eagle/pm20/v1.0.0-20240107123001-234d1b462fff.tm.json"
		r.On("Fetch", mock.Anything, tmID).Return(tmID, nil, repos.ErrTMNotFound).Once()
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(context.Background(), tmID, false)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrNotFound
		assert.ErrorIs(t, err, repos.ErrTMNotFound)
	})

	t.Run("with tmID found", func(t *testing.T) {
		_, raw, err := utils.ReadRequiredFile("../../../test/data/import/omnilamp.json")
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
func TestService_FetchLatestThingModel(t *testing.T) {
	r := mocks.NewRepo(t)
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)

	t.Run("with invalid fetch name", func(t *testing.T) {
		// when: fetching ThingModel
		res, err := underTest.FetchLatestThingModel(nil, "b-corp\\eagle/PM20", false)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrInvalidFetchName
		assert.ErrorIs(t, err, model.ErrInvalidFetchName)
	})

	t.Run("with invalid semantic version", func(t *testing.T) {
		// when: fetching ThingModel
		res, err := underTest.FetchLatestThingModel(nil, "b-corp/eagle/PM20:v1.", false)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrInvalidIdOrName
		assert.ErrorIs(t, err, model.ErrInvalidFetchName)
	})

	t.Run("with fetch name not found", func(t *testing.T) {
		fn := "b-corp/eagle/pm20"
		r.On("Versions", mock.Anything, fn).Return(nil, nil).Once()
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: fetching ThingModel
		res, err := underTest.FetchLatestThingModel(context.Background(), fn, false)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrTMNameNotFound
		assert.ErrorIs(t, err, repos.ErrTMNameNotFound)
	})

	t.Run("with fetch name found", func(t *testing.T) {
		_, raw, err := utils.ReadRequiredFile("../../../test/data/import/omnilamp.json")
		fn := "b-corp/eagle/pm20"
		tmID := fn + "/v1.0.0-20240107123001-234d1b462fff.tm.json"

		r.On("Versions", mock.Anything, fn).Return([]model.FoundVersion{singleFoundVersion}, nil).Once()
		r.On("Fetch", mock.Anything, tmID).Return(tmID, raw, nil).Once()
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r))
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r2"), r, nil))
		// when: fetching ThingModel
		res, err := underTest.FetchLatestThingModel(context.Background(), fn, false)
		// then: it returns the unchanged ThingModel content
		assert.NotNil(t, res)
		assert.Equal(t, raw, res)
		// and then: there is no error
		assert.NoError(t, err)
	})
}
func TestService_GetLatestTMMetadata(t *testing.T) {
	r := mocks.NewRepo(t)
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)

	t.Run("with invalid fetch name", func(t *testing.T) {
		// when: fetching ThingModel
		res, err := underTest.GetLatestTMMetadata(nil, "b-corp\\eagle/PM20")
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrInvalidFetchName
		assert.ErrorIs(t, err, model.ErrInvalidFetchName)
	})

	t.Run("with invalid semantic version", func(t *testing.T) {
		// when: fetching ThingModel
		res, err := underTest.GetLatestTMMetadata(nil, "b-corp/eagle/PM20:v1.")
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrInvalidFetchName
		assert.ErrorIs(t, err, model.ErrInvalidFetchName)
	})

	t.Run("with fetch name not found", func(t *testing.T) {
		fn := "b-corp/eagle/pm20"
		r.On("Versions", mock.Anything, fn).Return(nil, nil).Once()
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: fetching ThingModel
		res, err := underTest.GetLatestTMMetadata(context.Background(), fn)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrTMNameNotFound
		assert.ErrorIs(t, err, repos.ErrTMNameNotFound)
	})

	t.Run("with fetch name found", func(t *testing.T) {
		fn := "b-corp/eagle/pm20"
		tmID := fn + "/v1.0.0-20240107123001-234d1b462fff.tm.json"
		r.On("Versions", mock.Anything, fn).Return([]model.FoundVersion{singleFoundVersion}, nil).Once()
		r.On("GetTMMetadata", mock.Anything, tmID).Return(&singleFoundVersion, nil).Once()
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r))
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r2"), r, nil))
		// when: fetching ThingModel
		res, err := underTest.GetLatestTMMetadata(context.Background(), fn)
		// then: it returns the unchanged ThingModel content
		assert.NotNil(t, res)
		assert.Equal(t, &singleFoundVersion, res)
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
		// when: deleting ThingModel
		err := underTest.DeleteThingModel(context.Background(), tmid)
		// then: it returns nil result
		assert.NoError(t, err)
	})

	t.Run("with error when deleting", func(t *testing.T) {
		tmid := "some-id2"
		r.On("Delete", mock.Anything, tmid).Return(repos.ErrTMNotFound).Once()
		// when: deleting ThingModel
		err := underTest.DeleteThingModel(context.Background(), tmid)
		// then: it returns error result
		assert.ErrorIs(t, err, repos.ErrTMNotFound)
	})

}

func TestService_ImportThingModel(t *testing.T) {
	r := mocks.NewRepo(t)
	importTarget := model.NewRepoSpec("importRepo")
	underTest, _ := NewDefaultHandlerService(repo, importTarget)

	t.Run("with validation error", func(t *testing.T) {
		// given: some invalid content for a ThingModel
		invalidContent := []byte("invalid content")
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, importTarget, r, nil))
		// when: importing ThingModel
		res, err := underTest.ImportThingModel(nil, invalidContent, repos.ImportOptions{})
		// then: it returns an empty ImportResult
		assert.Equal(t, repos.ImportResult{}, res)
		// and then: there is an error
		assert.ErrorContains(t, err, "invalid character 'i' looking for beginning of value")
	})

	t.Run("with import repo name that cannot be found", func(t *testing.T) {
		// given: invalid importTarget
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, importTarget, nil, repos.ErrRepoNotFound))
		// when: importing ThingModel
		res, err := underTest.ImportThingModel(nil, []byte("some TM content"), repos.ImportOptions{})
		// then: it returns empty import result
		assert.Equal(t, repos.ImportResult{}, res)
		// and then: there is an error
		assert.ErrorContains(t, err, "repo not found")
		// and then: the error says that the repo cannot be found
		assert.Equal(t, repos.ErrRepoNotFound, err)
	})
	t.Run("with content conflict", func(t *testing.T) {
		// given: some valid content for a ThingModel
		_, tmContent, _ := utils.ReadRequiredFile("../../../test/data/import/omnilamp.json")
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, importTarget, r, nil))
		cErr := &repos.ErrTMIDConflict{
			Type:       repos.IdConflictSameContent,
			ExistingId: "existing-id",
		}
		expRes := repos.ImportResult{
			Type:    repos.ImportResultTMExists,
			TmID:    "",
			Message: cErr.Error(),
			Err:     cErr,
		}
		r.On("Import", mock.Anything, mock.Anything, mock.Anything, repos.ImportOptions{}).Return(expRes, nil).Once()
		// when: importing ThingModel
		res, err := underTest.ImportThingModel(nil, tmContent, repos.ImportOptions{})
		// then: it returns empty tmID
		assert.Equal(t, expRes, res)
		// and then: there is no error
		assert.NoError(t, err)
	})
	t.Run("with timestamp conflict", func(t *testing.T) {
		// given: some valid content for a ThingModel
		_, tmContent, _ := utils.ReadRequiredFile("../../../test/data/import/omnilamp.json")
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, importTarget, r, nil))
		cErr := &repos.ErrTMIDConflict{
			Type:       repos.IdConflictSameTimestamp,
			ExistingId: "existing-id",
		}
		expRes := repos.ImportResult{
			Type:    repos.ImportResultWarning,
			TmID:    "new-id",
			Message: cErr.Error(),
			Err:     cErr,
		}

		r.On("Import", mock.Anything, mock.Anything, mock.Anything, repos.ImportOptions{}).Return(expRes, nil).Once()
		r.On("Index", mock.Anything, mock.Anything).Return(nil)
		// when: importing ThingModel
		res, err := underTest.ImportThingModel(nil, tmContent, repos.ImportOptions{})
		// then: it returns expected warning result
		assert.Equal(t, expRes, res)
		// and then: there is no error
		assert.NoError(t, err)
	})
}

func TestService_GetTMMetadata(t *testing.T) {
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)
	tmID := "b-corp/eagle/PM20/v1.0.0-20240107123001-234d1b462fff.tm.json"
	// given: repo returns some attachments
	r := mocks.NewRepo(t)
	r.On("GetTMMetadata", mock.Anything, tmID).Return(&singleFoundVersion, nil).Once()
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repo, r, nil))
	// when: listing attachments
	res, err := underTest.GetTMMetadata(context.Background(), tmID)
	// then: service returns the attachment names
	assert.NoError(t, err)
	assert.Equal(t, &singleFoundVersion, res)
}

func TestService_FetchAttachment(t *testing.T) {
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)
	inventoryName := "a/b/c"
	attContent := []byte("# readme file")
	attName := "README.md"
	// given: repo returns an attachment
	r := mocks.NewRepo(t)
	r.On("FetchAttachment", mock.Anything, model.NewTMNameAttachmentContainerRef(inventoryName), attName).Return(attContent, nil).Once()
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repo, r, nil))
	// when: fetching an attachment
	res, err := underTest.FetchAttachment(context.Background(), model.NewTMNameAttachmentContainerRef(inventoryName), attName)
	// then: service returns the attachment content
	assert.NoError(t, err)
	assert.Equal(t, attContent, res)
}

func TestService_PushAttachment(t *testing.T) {
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)
	inventoryName := "a/b/c"
	attContent := []byte("# readme file")
	attName := "README.md"
	// given: a repo
	r := mocks.NewRepo(t)
	r.On("PushAttachment", mock.Anything, model.NewTMNameAttachmentContainerRef(inventoryName), attName, attContent).Return(nil).Once()
	r.On("Index", mock.Anything, inventoryName).Return(nil).Once()
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repo, r, nil))
	// when: pushing an attachment
	err := underTest.PushAttachment(context.Background(), model.NewTMNameAttachmentContainerRef(inventoryName), attName, attContent)
	// then: service returns no error
	assert.NoError(t, err)
}

func TestService_DeleteAttachment(t *testing.T) {
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, repo)
	inventoryName := "a/b/c"
	attName := "README.md"
	// given: repo returns an attachment
	r := mocks.NewRepo(t)
	r.On("DeleteAttachment", mock.Anything, model.NewTMNameAttachmentContainerRef(inventoryName), attName).Return(nil).Once()
	r.On("Index", mock.Anything, inventoryName).Return(nil).Once()
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repo, r, nil))
	// when: deleting an attachment
	err := underTest.DeleteAttachment(context.Background(), model.NewTMNameAttachmentContainerRef(inventoryName), attName)
	// then: service returns no error
	assert.NoError(t, err)
}

var singleFoundVersion = model.FoundVersion{
	IndexVersion: model.IndexVersion{
		Description: "desc version v1.0.0",
		Version:     model.Version{Model: "1.0.0"},
		TMID:        "b-corp/eagle/pm20/v1.0.0-20240107123001-234d1b462fff.tm.json",
		Digest:      "234d1b462fff",
		TimeStamp:   "20240107123001",
		ExternalID:  "ext-4",
		AttachmentContainer: model.AttachmentContainer{
			Attachments: []model.Attachment{
				{
					Name: "README.md",
				},
				{
					Name: "User Guide.pdf",
				},
			},
		},
	},
	FoundIn: model.FoundSource{RepoName: "r2"},
}
