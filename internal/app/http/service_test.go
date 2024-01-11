package http

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"net/http"
	"sort"
	"testing"
)

const remote = "someRemote"

func Test_CheckHealthLive(t *testing.T) {

	t.Run("with set RemoteManager", func(t *testing.T) {
		// given: a service under test where a RemoteManager is set
		rm := remotes.NewMockRemoteManager(t)
		underTest := NewDefaultHandlerService(rm, remote)
		// when: check health live
		err := underTest.CheckHealthLive(nil)
		// then: there is no error
		assert.NoError(t, err)
	})

	t.Run("with unset RemoteManager", func(t *testing.T) {
		// given: a service under test where a RemoteManager is unset
		underTest := NewDefaultHandlerService(nil, remote)
		// when: check health live
		err := underTest.CheckHealthLive(nil)
		// then: there is no error, unset remote manager does not matter
		assert.NoError(t, err)
	})
}

func Test_CheckHealthReady(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)
	underTest := NewDefaultHandlerService(rm, remote)

	t.Run("with valid remote", func(t *testing.T) {
		// given: the remote can be found by the remote manager
		rm.On("Get", remote).Return(r, nil).Once()
		// when check health ready
		err := underTest.CheckHealthReady(nil)
		// then: no error is thrown
		assert.NoError(t, err)
	})

	t.Run("with invalid remote", func(t *testing.T) {
		// given: the remote cannot be found by the remote manager
		rm.On("Get", remote).Return(nil, errors.New("invalid remote name")).Once()
		// when check health ready
		err := underTest.CheckHealthReady(nil)
		// then: an error is thrown
		assert.Error(t, err)
	})

	t.Run("with unset RemoteManager", func(t *testing.T) {
		// given: a service under test where a RemoteManager is unset
		underTest := NewDefaultHandlerService(nil, remote)
		// when: check health ready
		err := underTest.CheckHealthReady(nil)
		// then: an error is thrown
		assert.Error(t, err)
	})
}

func Test_CheckHealthStartup(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)
	underTest := NewDefaultHandlerService(rm, remote)

	t.Run("with valid remote", func(t *testing.T) {
		// given: the remote can be found by the remote manager
		rm.On("Get", remote).Return(r, nil).Once()
		// when check health startup
		err := underTest.CheckHealthStartup(nil)
		// then: no error is thrown
		assert.NoError(t, err)
	})

	t.Run("with invalid remote", func(t *testing.T) {
		// given: the remote cannot be found by the remote manager
		rm.On("Get", remote).Return(nil, errors.New("invalid remote name")).Once()
		// when check health startup
		err := underTest.CheckHealthStartup(nil)
		// then: an error is thrown
		assert.Error(t, err)
	})

	t.Run("with unset RemoteManager", func(t *testing.T) {
		// given: a service under test where a RemoteManager is unset
		underTest := NewDefaultHandlerService(nil, remote)
		// when: check health startup
		err := underTest.CheckHealthStartup(nil)
		// then: an error is thrown
		assert.Error(t, err)
	})
}

func Test_CheckHealth(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)
	underTest := NewDefaultHandlerService(rm, remote)

	t.Run("with valid remote", func(t *testing.T) {
		// given: the remote can be found by the remote manager
		rm.On("Get", remote).Return(r, nil).Once()
		// when check health
		err := underTest.CheckHealth(nil)
		// then: no error is thrown
		assert.NoError(t, err)
	})

	t.Run("with invalid remote", func(t *testing.T) {
		// given: the remote cannot be found by the remote manager
		rm.On("Get", remote).Return(nil, errors.New("invalid remote name")).Once()
		// when check health
		err := underTest.CheckHealth(nil)
		// then: an error is thrown
		assert.Error(t, err)
	})

	t.Run("with unset RemoteManager", func(t *testing.T) {
		// given: a service under test where a RemoteManager is unset
		underTest := NewDefaultHandlerService(nil, remote)
		// when: check health
		err := underTest.CheckHealth(nil)
		// then: an error is thrown
		assert.Error(t, err)
	})
}

func Test_ListInventory(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)
	underTest := NewDefaultHandlerService(rm, remote)

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
						FoundIn: "r1",
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
						FoundIn: "r1",
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
						FoundIn: "r1",
					},
				},
			},
		},
	}

	t.Run("list all", func(t *testing.T) {
		// given: remote having some inventory entries
		r := remotes.NewMockRemote(t)
		r.On("List", &model.SearchParams{}).Return(listResult, nil).Once()
		rm.On("All").Return([]remotes.Remote{r}, nil).Once()
		// when: list all
		res, err := underTest.ListInventory(nil, &model.SearchParams{})
		// then: there is no error
		assert.NoError(t, err)
		// and then: the search result is returned
		assert.Equal(t, &listResult, res)
	})

	t.Run("with unset remote manager", func(t *testing.T) {
		// given: a service under test where a RemoteManager is unset
		underTest := NewDefaultHandlerService(nil, remote)
		// when: list all
		res, err := underTest.ListInventory(nil, nil)
		// then: it returns an error and nil result
		assert.Nil(t, res)
		assert.Error(t, err)
	})
}

func Test_FindInventoryEntry(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)

	t.Run("with unset remote manager", func(t *testing.T) {
		// given: a service under test where a RemoteManager is unset
		underTest := NewDefaultHandlerService(nil, remote)
		// when: finding entry
		res, err := underTest.FindInventoryEntry(nil, "a/b/c")
		// then: it returns an error and nil result
		assert.Nil(t, res)
		assert.Error(t, err)
	})

	t.Run("inventory entry cannot be found", func(t *testing.T) {
		underTest := NewDefaultHandlerService(rm, remote)
		inventoryName := "a/b/c"
		// given: remote returns empty search result
		r := remotes.NewMockRemote(t)
		r.On("List", &model.SearchParams{Name: inventoryName}).Return(model.SearchResult{}, nil).Once()
		rm.On("All").Return([]remotes.Remote{r}, nil).Once()
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

	rm := remotes.NewMockRemoteManager(t)
	underTest := NewDefaultHandlerService(rm, remote)

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
		r := remotes.NewMockRemote(t)
		r.On("List", &model.SearchParams{}).Return(listResult, nil).Once()
		rm.On("All").Return([]remotes.Remote{r}, nil).Once()

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

	t.Run("with unset remote manager", func(t *testing.T) {
		underTest := NewDefaultHandlerService(nil, remote)
		// when: list all
		res, err := underTest.ListAuthors(nil, nil)
		// then: it returns an error and empty list
		assert.Equal(t, []string{}, res)
		assert.Error(t, err)
	})
}

func Test_ListManufacturers(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)
	underTest := NewDefaultHandlerService(rm, remote)

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
		r := remotes.NewMockRemote(t)
		r.On("List", &model.SearchParams{}).Return(listResult, nil).Once()
		rm.On("All").Return([]remotes.Remote{r}, nil).Once()

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

	t.Run("with unset remote manager", func(t *testing.T) {
		underTest := NewDefaultHandlerService(nil, remote)
		// when: list all
		res, err := underTest.ListManufacturers(nil, nil)
		// then: it returns an error and empty list
		assert.Equal(t, []string{}, res)
		assert.Error(t, err)
	})
}

func Test_ListMpns(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)
	underTest := NewDefaultHandlerService(rm, remote)

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
		r := remotes.NewMockRemote(t)
		r.On("List", &model.SearchParams{}).Return(listResult, nil).Once()
		rm.On("All").Return([]remotes.Remote{r}, nil).Once()

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

	t.Run("with unset remote manager", func(t *testing.T) {
		underTest := NewDefaultHandlerService(nil, remote)
		// when: list all
		res, err := underTest.ListMpns(nil, nil)
		// then: it returns an error and empty list
		assert.Equal(t, []string{}, res)
		assert.Error(t, err)
	})
}

func Test_FetchingThingModel(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)
	underTest := NewDefaultHandlerService(rm, remote)
	tmID := "b-corp/eagle/PM20/v1.0.0-20240107123001-234d1b462fff.tm.json"

	t.Run("with invalid tmID", func(t *testing.T) {
		invalidTmID := ""
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(nil, invalidTmID)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is status code 400
		assert.Error(t, err)
		sErr, ok := err.(*BaseHttpError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, sErr.Status)
	})

	t.Run("with unset remote manager", func(t *testing.T) {
		underTest := NewDefaultHandlerService(nil, remote)
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(nil, tmID)
		// then: it returns an error and nil result
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}

func Test_PushingThingModel(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)

	t.Run("with unset remote manager", func(t *testing.T) {
		underTest := NewDefaultHandlerService(nil, remote)
		// when: pushing ThingModel
		res, err := underTest.PushThingModel(nil, nil)
		// then: it returns an error
		assert.Equal(t, "", res)
		assert.Error(t, err)
	})

	t.Run("with unset push remote name", func(t *testing.T) {
		underTest := NewDefaultHandlerService(rm, "")
		// when: pushing ThingModel
		res, err := underTest.PushThingModel(nil, nil)
		// then: it returns an error
		assert.Equal(t, "", res)
		assert.Error(t, err)
	})
}
