package http

import (
	"errors"
	"net/http"
	"sort"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes/mocks"
	rMocks "github.com/web-of-things-open-source/tm-catalog-cli/internal/testutils/remotesmocks"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

var remote = model.NewRemoteSpec("someRemote")

func Test_CheckHealthLive(t *testing.T) {
	// given: a service under test
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, remote)
	// when: check health live
	err := underTest.CheckHealthLive(nil)
	// then: there is no error
	assert.NoError(t, err)
}

func Test_CheckHealthReady(t *testing.T) {

	r := mocks.NewRemote(t)
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, remote)

	t.Run("with valid remote", func(t *testing.T) {
		// given: a remote
		rMocks.MockRemotesGet(t, rMocks.CreateMockGetFunction(t, remote, r, nil))

		// when check health ready
		err := underTest.CheckHealthReady(nil)
		// then: no error is thrown
		assert.NoError(t, err)
	})

	t.Run("with invalid remote", func(t *testing.T) {
		// given: the remote cannot be found
		rMocks.MockRemotesGet(t, rMocks.CreateMockGetFunction(t, remote, nil, errors.New("invalid remote name")))
		// when check health ready
		err := underTest.CheckHealthReady(nil)
		// then: an error is thrown
		assert.Error(t, err)
	})
}

func Test_CheckHealthStartup(t *testing.T) {

	r := mocks.NewRemote(t)
	underTest, _ := NewDefaultHandlerService(remote, remote)

	t.Run("with valid remote", func(t *testing.T) {
		// given: the remote can be found
		rMocks.MockRemotesGet(t, rMocks.CreateMockGetFunction(t, remote, r, nil))
		// when check health startup
		err := underTest.CheckHealthStartup(nil)
		// then: no error is thrown
		assert.NoError(t, err)
	})

	t.Run("with invalid remote", func(t *testing.T) {
		// given: the remote cannot be found
		rMocks.MockRemotesGet(t, rMocks.CreateMockGetFunction(t, remote, nil, errors.New("invalid remote name")))
		// when check health startup
		err := underTest.CheckHealthStartup(nil)
		// then: an error is thrown
		assert.Error(t, err)
	})
}

func Test_CheckHealth(t *testing.T) {

	r := mocks.NewRemote(t)
	underTest, _ := NewDefaultHandlerService(remote, remote)

	t.Run("with valid remote", func(t *testing.T) {
		// given: the remote can be found
		rMocks.MockRemotesGet(t, rMocks.CreateMockGetFunction(t, remote, r, nil))

		// when check health
		err := underTest.CheckHealth(nil)
		// then: no error is thrown
		assert.NoError(t, err)
	})

	t.Run("with invalid remote", func(t *testing.T) {
		// given: the remote cannot be found
		rMocks.MockRemotesGet(t, rMocks.CreateMockGetFunction(t, remote, nil, errors.New("invalid remote name")))
		// when check health
		err := underTest.CheckHealth(nil)
		// then: an error is thrown
		assert.Error(t, err)
	})
}

func Test_ListInventory(t *testing.T) {

	underTest, _ := NewDefaultHandlerService(model.EmptySpec, remote)

	listResult := model.SearchResult{
		Entries: []model.FoundEntry{
			{
				Name:         "a-corp/eagle/BT2000",
				Author:       model.SchemaAuthor{Name: "a-corp"},
				Manufacturer: model.SchemaManufacturer{Name: "eagle"},
				Mpn:          "BT2000",
				Versions: []model.FoundVersion{
					{
						TOCVersion: model.TOCVersion{
							TMID:        "a-corp/eagle/BT2000/v1.0.0-20240108140117-243d1b462ccc.tm.json",
							Description: "desc version v1.0.0",
							Version:     model.Version{Model: "1.0.0"},
							Digest:      "243d1b462ccc",
							TimeStamp:   "20240108140117",
							ExternalID:  "ext-2",
						},
						FoundIn: model.FoundSource{RemoteName: "r1"},
					},
					{
						TOCVersion: model.TOCVersion{
							TMID:        "a-corp/eagle/BT2000/v1.0.0-20231231153548-243d1b462ddd.tm.json",
							Description: "desc version v0.0.0",
							Version:     model.Version{Model: "0.0.0"},
							Digest:      "243d1b462ddd",
							TimeStamp:   "20231231153548",
							ExternalID:  "ext-1",
						},
						FoundIn: model.FoundSource{RemoteName: "r1"},
					},
				},
			},
			{
				Name:         "b-corp/frog/BT3000",
				Author:       model.SchemaAuthor{Name: "b-corp"},
				Manufacturer: model.SchemaManufacturer{Name: "frog"},
				Mpn:          "BT3000",
				Versions: []model.FoundVersion{
					{
						TOCVersion: model.TOCVersion{
							TMID:        "b-corp/frog/BT3000/v1.0.0-20240108140117-743d1b462uuu.tm.json",
							Description: "desc version v1.0.0",
							Version:     model.Version{Model: "1.0.0"},
							Digest:      "743d1b462uuu",
							TimeStamp:   "20240108140117",
							ExternalID:  "ext-3",
						},
						FoundIn: model.FoundSource{RemoteName: "r1"},
					},
				},
			},
		},
	}

	t.Run("list all", func(t *testing.T) {
		// given: remote having some inventory entries
		r := mocks.NewRemote(t)
		r.On("List", &model.SearchParams{Author: []string{"a-corp", "b-corp"}}).Return(listResult, nil).Once()
		rMocks.MockRemotesAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: list all
		res, err := underTest.ListInventory(nil, &model.SearchParams{Author: []string{"a-corp", "b-corp"}})
		// then: there is no error
		assert.NoError(t, err)
		// and then: the search result is returned
		assert.Equal(t, &listResult, res)
	})
	t.Run("list with one upstream error", func(t *testing.T) {
		// given: remote having some inventory entries
		r := mocks.NewRemote(t)
		r2 := mocks.NewRemote(t)
		r.On("List", &model.SearchParams{}).Return(listResult, nil).Once()
		r2.On("List", &model.SearchParams{}).Return(model.SearchResult{}, errors.New("unexpected")).Once()
		r2.On("Spec").Return(model.NewRemoteSpec("r2")).Once()
		rMocks.MockRemotesAll(t, rMocks.CreateMockAllFunction(nil, r, r2))
		// when: list all
		res, err := underTest.ListInventory(nil, &model.SearchParams{})
		// then: there is an error of type remotes.RepoAccessError
		var aErr *remotes.RepoAccessError
		assert.ErrorAs(t, err, &aErr)
		// and then: the search result is returned
		assert.Nil(t, res)
	})
}

func Test_GetCompletions(t *testing.T) {
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, remote)

	t.Run("list names", func(t *testing.T) {
		// given: remote having some inventory entries
		r := mocks.NewRemote(t)
		names := []string{"a/b/c", "d/e/f"}
		r.On("ListCompletions", "names", "toComplete").Return(names, nil)
		rMocks.MockRemotesAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: list all
		res, err := underTest.GetCompletions(nil, "names", "toComplete")
		// then: there is no error
		assert.NoError(t, err)
		// and then: the search result is returned
		assert.Equal(t, names, res)
	})
}

func Test_FindInventoryEntry(t *testing.T) {

	t.Run("inventory entry cannot be found", func(t *testing.T) {
		underTest, _ := NewDefaultHandlerService(model.EmptySpec, remote)
		inventoryName := "a/b/c"
		// given: remote returns empty search result
		r := mocks.NewRemote(t)
		r.On("List", &model.SearchParams{Name: inventoryName}).Return(model.SearchResult{}, nil).Once()
		rMocks.MockRemotesAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: finding entry
		res, err := underTest.FindInventoryEntry(nil, inventoryName)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is status code 404
		sErr, ok := err.(*BaseHttpError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusNotFound, sErr.Status)
	})
}

func Test_ListAuthors(t *testing.T) {

	underTest, _ := NewDefaultHandlerService(model.EmptySpec, remote)

	// given: some inventory entries with unordered and duplicated authors
	listResult := model.SearchResult{
		Entries: []model.FoundEntry{
			{
				Name:   "z-corp/eagle/BT2000",
				Author: model.SchemaAuthor{Name: "z-corp"},
			},
			{
				Name:   "a-corp/frog/BT4000",
				Author: model.SchemaAuthor{Name: "a-corp"},
			},
			{
				Name:   "a-corp/frog/BT7000",
				Author: model.SchemaAuthor{Name: "a-corp"},
			},
		},
	}

	t.Run("list all", func(t *testing.T) {
		// given: remote returning the inventory entries
		r := mocks.NewRemote(t)
		r.On("List", &model.SearchParams{}).Return(listResult, nil).Once()
		rMocks.MockRemotesAll(t, rMocks.CreateMockAllFunction(nil, r))

		// when: list all authors
		res, err := underTest.ListAuthors(nil, &model.SearchParams{})
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

	underTest, _ := NewDefaultHandlerService(model.EmptySpec, remote)

	// given: some inventory entries with unordered and duplicated manufacturers
	listResult := model.SearchResult{
		Entries: []model.FoundEntry{
			{
				Name:         "a-corp/frog/BT4000",
				Manufacturer: model.SchemaManufacturer{Name: "frog"},
			},
			{
				Name:         "z-corp/eagle/BT2000",
				Manufacturer: model.SchemaManufacturer{Name: "eagle"},
			},
			{
				Name:         "a-corp/frog/BT7000",
				Manufacturer: model.SchemaManufacturer{Name: "frog"},
			},
		},
	}

	t.Run("list all", func(t *testing.T) {
		// given: remote returning the inventory entries
		r := mocks.NewRemote(t)
		r.On("List", &model.SearchParams{}).Return(listResult, nil).Once()
		rMocks.MockRemotesAll(t, rMocks.CreateMockAllFunction(nil, r))

		// when: list all manufacturers
		res, err := underTest.ListManufacturers(nil, &model.SearchParams{})
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

	underTest, _ := NewDefaultHandlerService(model.EmptySpec, remote)

	// given: some inventory entries with unordered and duplicated mpns
	listResult := model.SearchResult{
		Entries: []model.FoundEntry{
			{
				Name: "a-corp/frog/BT4000",
				Mpn:  "BT4000",
			},
			{
				Name: "z-corp/eagle/BT2000",
				Mpn:  "BT2000",
			},
			{
				Name: "a-corp/frog/BT4000",
				Mpn:  "BT4000",
			},
		},
	}

	t.Run("list all", func(t *testing.T) {
		// given: remote returning the inventory entries
		r := mocks.NewRemote(t)
		r.On("List", &model.SearchParams{}).Return(listResult, nil).Once()
		rMocks.MockRemotesAll(t, rMocks.CreateMockAllFunction(nil, r))

		// when: list all
		res, err := underTest.ListMpns(nil, &model.SearchParams{})
		// then: there is no error
		assert.NoError(t, err)
		// and then: the result is sorted asc by name
		isSorted := sort.SliceIsSorted(res, func(i, j int) bool {
			return res[i] < res[j]
		})
		assert.True(t, isSorted)
		// and then: the result contains no duplicates
		assert.Equal(t, []string{"BT2000", "BT4000"}, res)
	})
}

func Test_FetchingThingModel(t *testing.T) {

	r := mocks.NewRemote(t)
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, remote)

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
		tmID := "b-corp/eagle/PM20/v1.0.0-20240107123001-234d1b462fff.tm.json"
		r.On("Fetch", tmID).Return(tmID, nil, remotes.ErrTmNotFound).Once()
		rMocks.MockRemotesAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(nil, tmID, false)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrTmNotFound
		assert.ErrorIs(t, err, remotes.ErrTmNotFound)
	})

	t.Run("with fetch name not found", func(t *testing.T) {
		fn := "b-corp/eagle/PM20"
		r.On("Versions", fn).Return(nil, nil).Once()
		rMocks.MockRemotesAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(nil, fn, false)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrTmNotFound
		assert.ErrorIs(t, err, remotes.ErrTmNotFound)
	})

	t.Run("with tmID found", func(t *testing.T) {
		_, raw, err := utils.ReadRequiredFile("../../../test/data/push/omnilamp.json")
		tmID := "b-corp/eagle/PM20/v1.0.0-20240107123001-234d1b462fff.tm.json"
		r.On("Fetch", tmID).Return(tmID, raw, nil).Once()
		rMocks.MockRemotesAll(t, rMocks.CreateMockAllFunction(nil, r))
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(nil, tmID, false)
		// then: it returns the unchanged ThingModel content
		assert.NotNil(t, res)
		assert.Equal(t, raw, res)
		// and then: there is no error
		assert.NoError(t, err)
	})
}
func Test_DeleteThingModel(t *testing.T) {

	r := mocks.NewRemote(t)
	rMocks.MockRemotesGet(t, rMocks.CreateMockGetFunction(t, remote, r, nil))
	underTest, _ := NewDefaultHandlerService(model.EmptySpec, remote)

	t.Run("without errors", func(t *testing.T) {
		tmid := "some-id"
		r.On("Delete", tmid).Return(nil).Once()
		r.On("UpdateToc", tmid).Return(nil).Once()
		// when: deleting ThingModel
		err := underTest.DeleteThingModel(nil, tmid)
		// then: it returns nil result
		assert.NoError(t, err)
	})

	t.Run("with error when deleting", func(t *testing.T) {
		tmid := "some-id2"
		r.On("Delete", tmid).Return(remotes.ErrTmNotFound).Once()
		// when: deleting ThingModel
		err := underTest.DeleteThingModel(nil, tmid)
		// then: it returns error result
		assert.ErrorIs(t, err, remotes.ErrTmNotFound)
	})

	t.Run("with error when updating toc", func(t *testing.T) {
		tmid := "some-id3"
		r.On("Delete", tmid).Return(nil).Once()
		r.On("UpdateToc", tmid).Return(errors.New("could not update toc")).Once()
		// when: deleting ThingModel
		err := underTest.DeleteThingModel(nil, tmid)
		// then: it returns error result
		assert.ErrorContains(t, err, "could not update toc")
	})
}

func Test_PushingThingModel(t *testing.T) {
	r := mocks.NewRemote(t)
	pushTarget := model.NewRemoteSpec("pushRemote")
	underTest, _ := NewDefaultHandlerService(remote, pushTarget)

	t.Run("with validation error", func(t *testing.T) {
		// given: some invalid content for a ThingModel
		invalidContent := []byte("invalid content")
		rMocks.MockRemotesGet(t, rMocks.CreateMockGetFunction(t, pushTarget, r, nil))
		// when: pushing ThingModel
		res, err := underTest.PushThingModel(nil, invalidContent)
		// then: it returns empty tmID
		assert.Equal(t, "", res)
		// and then: there is an error
		assert.Error(t, err)
	})

	t.Run("with push remote name that cannot be found", func(t *testing.T) {
		// given: invalid pushTarget
		rMocks.MockRemotesGet(t, rMocks.CreateMockGetFunction(t, pushTarget, nil, remotes.ErrRemoteNotFound))
		// when: pushing ThingModel
		res, err := underTest.PushThingModel(nil, []byte("some TM content"))
		// then: it returns empty tmID
		assert.Equal(t, "", res)
		// and then: there is an error
		assert.Error(t, err)
		// and then: the error says that the remote cannot be found
		assert.Equal(t, remotes.ErrRemoteNotFound, err)
	})
	t.Run("with content conflict", func(t *testing.T) {
		// given: some valid content for a ThingModel
		_, tmContent, _ := utils.ReadRequiredFile("../../../test/data/push/omnilamp.json")
		rMocks.MockRemotesGet(t, rMocks.CreateMockGetFunction(t, pushTarget, r, nil))
		cErr := &remotes.ErrTMIDConflict{
			Type:       remotes.IdConflictSameContent,
			ExistingId: "existing-id",
		}
		r.On("Push", mock.Anything, mock.Anything).Return(cErr).Once()
		// when: pushing ThingModel
		res, err := underTest.PushThingModel(nil, tmContent)
		// then: it returns empty tmID
		assert.Equal(t, "", res)
		// and then: there is an error
		assert.Equal(t, cErr, err)
	})
	t.Run("with timestamp conflict", func(t *testing.T) {
		// given: some valid content for a ThingModel
		_, tmContent, _ := utils.ReadRequiredFile("../../../test/data/push/omnilamp.json")
		rMocks.MockRemotesGet(t, rMocks.CreateMockGetFunction(t, pushTarget, r, nil))
		cErr := &remotes.ErrTMIDConflict{
			Type:       remotes.IdConflictSameTimestamp,
			ExistingId: "existing-id",
		}
		r.On("Push", mock.Anything, mock.Anything).Return(cErr).Once()
		r.On("Push", mock.Anything, mock.Anything).Return(nil).Once() // expect a second push attempt
		r.On("UpdateToc", mock.Anything).Return(nil)
		// when: pushing ThingModel
		res, err := underTest.PushThingModel(nil, tmContent)
		// then: it returns non-empty tmID
		assert.NotEmpty(t, res)
		// and then: there is an error
		assert.NoError(t, err)
	})
}
