package http

import (
	"errors"
	"net/http"
	"sort"
	"testing"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

var remote = remotes.NewRemoteSpec("someRemote")

func Test_NewDefaultRemote(t *testing.T) {

	t.Run("with set remote manager", func(t *testing.T) {
		rm := remotes.NewMockRemoteManager(t)
		res, err := NewDefaultHandlerService(rm, remote, remote)
		assert.NotNil(t, res)
		assert.NoError(t, err)
	})

	t.Run("with unset remote manager", func(t *testing.T) {
		res, err := NewDefaultHandlerService(nil, remote, remote)
		assert.Nil(t, res)
		assert.Error(t, err)
	})
}

func Test_CheckHealthLive(t *testing.T) {

	t.Run("with set RemoteManager", func(t *testing.T) {
		// given: a service under test where a RemoteManager is set
		rm := remotes.NewMockRemoteManager(t)
		underTest, _ := NewDefaultHandlerService(rm, remotes.EmptySpec, remote)
		// when: check health live
		err := underTest.CheckHealthLive(nil)
		// then: there is no error
		assert.NoError(t, err)
	})
}

func Test_CheckHealthReady(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)
	underTest, _ := NewDefaultHandlerService(rm, remotes.EmptySpec, remote)

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
}

func Test_CheckHealthStartup(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)
	underTest, _ := NewDefaultHandlerService(rm, remote, remote)

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
}

func Test_CheckHealth(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)
	underTest, _ := NewDefaultHandlerService(rm, remote, remote)

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
}

func Test_ListInventory(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)
	underTest, _ := NewDefaultHandlerService(rm, remotes.EmptySpec, remote)

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
		r := remotes.NewMockRemote(t)
		r.On("List", &model.SearchParams{Author: []string{"a-corp", "b-corp"}}).Return(listResult, nil).Once()
		rm.On("All").Return([]remotes.Remote{r}, nil).Once()
		// when: list all
		res, err := underTest.ListInventory(nil, &model.SearchParams{Author: []string{"a-corp", "b-corp"}})
		// then: there is no error
		assert.NoError(t, err)
		// and then: the search result is returned
		assert.Equal(t, &listResult, res)
	})
}

func Test_FindInventoryEntry(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)

	t.Run("inventory entry cannot be found", func(t *testing.T) {
		underTest, _ := NewDefaultHandlerService(rm, remotes.EmptySpec, remote)
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
	underTest, _ := NewDefaultHandlerService(rm, remotes.EmptySpec, remote)

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
}

func Test_ListManufacturers(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)
	underTest, _ := NewDefaultHandlerService(rm, remotes.EmptySpec, remote)

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
}

func Test_ListMpns(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)
	underTest, _ := NewDefaultHandlerService(rm, remotes.EmptySpec, remote)

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
}

func Test_FetchingThingModel(t *testing.T) {

	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)
	underTest, _ := NewDefaultHandlerService(rm, remotes.EmptySpec, remote)

	t.Run("with invalid tmID", func(t *testing.T) {
		invalidTmID := ""
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(nil, invalidTmID)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrInvalidFetchName
		assert.ErrorIs(t, err, commands.ErrInvalidFetchName)
	})

	t.Run("with invalid fetch name", func(t *testing.T) {
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(nil, "b-corp\\eagle/PM20")
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrInvalidFetchName
		assert.ErrorIs(t, err, commands.ErrInvalidFetchName)
	})

	t.Run("with invalid semantic version", func(t *testing.T) {
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(nil, "b-corp/eagle/PM20:v1.")
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrInvalidFetchName
		assert.ErrorIs(t, err, commands.ErrInvalidFetchName)
	})

	t.Run("with tmID not found", func(t *testing.T) {
		tmID := "b-corp/eagle/PM20/v1.0.0-20240107123001-234d1b462fff.tm.json"
		r.On("Fetch", tmID).Return(tmID, nil, remotes.ErrTmNotFound).Once()
		rm.On("All").Return([]remotes.Remote{r}, nil).Once()
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(nil, tmID)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrTmNotFound
		assert.ErrorIs(t, err, remotes.ErrTmNotFound)
	})

	t.Run("with fetch name not found", func(t *testing.T) {
		fn := "b-corp/eagle/PM20"
		r.On("Versions", fn).Return(nil, nil).Once()
		rm.On("All").Return([]remotes.Remote{r}, nil).Once()
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(nil, fn)
		// then: it returns nil result
		assert.Nil(t, res)
		// and then: error is ErrTmNotFound
		assert.ErrorIs(t, err, remotes.ErrTmNotFound)
	})

	t.Run("with tmID found", func(t *testing.T) {
		_, raw, err := utils.ReadRequiredFile("../../../test/data/push/omnilamp.json")
		tmID := "b-corp/eagle/PM20/v1.0.0-20240107123001-234d1b462fff.tm.json"
		r.On("Fetch", tmID).Return(tmID, raw, nil).Once()
		rm.On("All").Return([]remotes.Remote{r}, nil).Once()
		// when: fetching ThingModel
		res, err := underTest.FetchThingModel(nil, tmID)
		// then: it returns the unchanged ThingModel content
		assert.NotNil(t, res)
		assert.Equal(t, raw, res)
		// and then: there is no error
		assert.NoError(t, err)
	})
}

func Test_PushingThingModel(t *testing.T) {
	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)
	pushTarget := remotes.NewRemoteSpec("pushRemote")
	underTest, _ := NewDefaultHandlerService(rm, remote, pushTarget)

	t.Run("with validation error", func(t *testing.T) {
		// given: some invalid content for a ThingModel
		invalidContent := []byte("invalid content")
		rm.On("Get", pushTarget).Return(r, nil).Once()
		// when: pushing ThingModel
		res, err := underTest.PushThingModel(nil, invalidContent)
		// then: it returns empty tmID
		assert.Equal(t, "", res)
		// and then: there is an error
		assert.Error(t, err)
	})

	t.Run("with push remote name that cannot be found", func(t *testing.T) {
		// given: some invalid content for a ThingModel
		rm.On("Get", pushTarget).Return(nil, remotes.ErrRemoteNotFound).Once()
		// when: pushing ThingModel
		res, err := underTest.PushThingModel(nil, []byte("some TM content"))
		// then: it returns empty tmID
		assert.Equal(t, "", res)
		// and then: there is an error
		assert.Error(t, err)
		// and then: the error says that the remote cannot be found
		assert.Equal(t, remotes.ErrRemoteNotFound, err)
	})
}
