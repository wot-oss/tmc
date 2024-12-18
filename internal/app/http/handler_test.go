package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/wot-oss/tmc/internal/app/http/mocks"
	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/testutils"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wot-oss/tmc/internal/app/http/server"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/utils"
)

func Test_getRelativeDepth(t *testing.T) {
	type args struct {
		path        string
		siblingPath string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"", args{"/inventory", "/inventory"}, 0},
		{"", args{"/long/path/to/inventory", "/inventory"}, 0},
		{"", args{"/somewhere/inventory/long/way/down", "/inventory"}, 3},
		{"", args{"/inventory/something", "/inventory"}, 1},
		{"", args{"/unrelated/path", "/inventory"}, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRelativeDepth(tt.args.path, tt.args.siblingPath); got != tt.want {
				t.Errorf("getRelativeDepth() = %v, want %v", got, tt.want)
			}
		})
	}
}

var unknownErr = errors.New("an unknown error")

func setupTestHttpHandler(hs HandlerService) http.Handler {

	handler := NewTmcHandler(
		hs,
		TmcHandlerOptions{
			UrlContextRoot: "",
		})

	return NewHttpHandler(handler, nil)
}

func Test_healthLive(t *testing.T) {

	route := "/healthz/live"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("with success", func(t *testing.T) {
		hs.On("CheckHealthLive", mock.Anything).Return(nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns 204 status and empty body
		assertHealthyResponse204(t, rec)
	})

	t.Run("with error", func(t *testing.T) {
		hs.On("CheckHealthLive", mock.Anything).Return(unknownErr).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns 503 status and json error as body
		assertResponse503(t, rec, route)
	})
}

func Test_healthReady(t *testing.T) {

	route := "/healthz/ready"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("with success", func(t *testing.T) {
		hs.On("CheckHealthReady", mock.Anything).Return(nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns 204 status and empty body
		assertHealthyResponse204(t, rec)
	})

	t.Run("with error", func(t *testing.T) {
		hs.On("CheckHealthReady", mock.Anything).Return(unknownErr).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns 503 status and json error as body
		assertResponse503(t, rec, route)
	})
}

func Test_healthStartup(t *testing.T) {

	route := "/healthz/startup"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("with success", func(t *testing.T) {
		hs.On("CheckHealthStartup", mock.Anything).Return(nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns 204 status and empty body
		assertHealthyResponse204(t, rec)
	})

	t.Run("with error", func(t *testing.T) {
		hs.On("CheckHealthStartup", mock.Anything).Return(unknownErr).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns 503 status and json error as body
		assertResponse503(t, rec, route)
	})
}

func Test_health(t *testing.T) {

	route := "/healthz"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("with success", func(t *testing.T) {
		hs.On("CheckHealth", mock.Anything).Return(nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns 204 status and empty body
		assertHealthyResponse204(t, rec)
	})

	t.Run("with error", func(t *testing.T) {
		hs.On("CheckHealth", mock.Anything).Return(unknownErr).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns 503 status and json error as body
		assertResponse503(t, rec, route)
	})
}

func Test_Info(t *testing.T) {

	route := "/info"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("with success", func(t *testing.T) {
		oldTmcVersion := utils.TmcVersion
		utils.TmcVersion = "v0.1.1"
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response server.InfoResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then result contains all data
		assert.Equal(t, server.InfoResponse{
			Name: "tmc",
			Version: server.InfoVersion{
				Implementation: "0.1.1",
			},
			Details: &[]string{},
		}, response)

		utils.TmcVersion = oldTmcVersion
	})
}

func Test_Inventory(t *testing.T) {

	route := "/inventory"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("list all", func(t *testing.T) {
		var search *model.Filters
		hs.On("ListInventory", mock.Anything, "", search).Return(&listResult1, nil).Once()

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response server.InventoryResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: the result contains all data
		assert.Equal(t, 2, len(response.Data))
		assertInventoryEntry(t, listResult1.Entries[0], response.Data[0])
		assertInventoryEntry(t, listResult1.Entries[1], response.Data[1])
		assert.Equal(t, listResult1.LastUpdated.Format(time.RFC3339), response.Meta.LastUpdated)
		// and then result is ordered ascending by name
		isSorted := sort.SliceIsSorted(response.Data, func(i, j int) bool {
			return response.Data[i].TmName < response.Data[j].TmName
		})
		assert.True(t, isSorted)
	})
	t.Run("list all from single repo", func(t *testing.T) {
		var search *model.Filters
		hs.On("ListInventory", mock.Anything, "r1", search).Return(&listResult1, nil).Once()

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route+"?repo=r1").RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response server.InventoryResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: the result contains all data
		assert.Equal(t, 2, len(response.Data))
		assertInventoryEntry(t, listResult1.Entries[0], response.Data[0])
		assertInventoryEntry(t, listResult1.Entries[1], response.Data[1])
		// and then result is ordered ascending by name
		isSorted := sort.SliceIsSorted(response.Data, func(i, j int) bool {
			return response.Data[i].TmName < response.Data[j].TmName
		})
		assert.True(t, isSorted)
	})
	t.Run("list all from invalid repo", func(t *testing.T) {
		var search *model.Filters
		hs.On("ListInventory", mock.Anything, "invalid", search).Return(nil, repos.ErrRepoNotFound).Once()

		// when: calling the route
		rt := route + "?repo=invalid"
		rec := testutils.NewRequest(http.MethodGet, rt).RunOnHandler(httpHandler)
		// then: it returns status 400
		assertResponse400(t, rec, rt)
	})

	t.Run("list with filter and search parameter", func(t *testing.T) {
		// given: the route with filter and search parameters
		fAuthors := "a1,a2"
		fMan := "man1,man2"
		fMpn := "mpn1,mpn2"
		fProtos := "coap,https"
		search := "foo"

		filterRoute := fmt.Sprintf("%s?filter.author=%s&filter.manufacturer=%s&filter.mpn=%s&filter.protocol=%s&search=%s",
			route, fAuthors, fMan, fMpn, fProtos, search)

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, filterRoute).RunOnHandler(httpHandler)
		// then: it returns status 400
		assertResponse400(t, rec, filterRoute)
	})
	t.Run("list with filter parameters", func(t *testing.T) {
		// given: the route with filter and search parameters
		fAuthors := "a1,a2"
		fMan := "man1,man2"
		fMpn := "mpn1,mpn2"
		fProtos := "coap,https"

		filterRoute := fmt.Sprintf("%s?filter.author=%s&filter.manufacturer=%s&filter.mpn=%s&filter.protocol=%s",
			route, fAuthors, fMan, fMpn, fProtos)
		// and given: searchParams, expected to be converted from request query parameters
		expectedFilters := model.ToFilters(&fAuthors, &fMan, &fMpn, &fProtos, nil, &model.FilterOptions{NameFilterType: model.PrefixMatch})

		hs.On("ListInventory", mock.Anything, "", expectedFilters).Return(&listResult1, nil).Once()

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, filterRoute).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
	})
	t.Run("list with search parameter", func(t *testing.T) {
		// given: the route with search parameter
		search := "foo"

		filterRoute := fmt.Sprintf("%s?search=%s",
			route, search)
		hs.On("SearchInventory", mock.Anything, "", search).Return(&listResult1, nil).Once()

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, filterRoute).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
	})

	t.Run("with unknown error", func(t *testing.T) {
		var sp *model.Filters
		hs.On("ListInventory", mock.Anything, "", sp).Return(nil, unknownErr).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, route)
	})

	t.Run("with repository access error", func(t *testing.T) {
		var sp *model.Filters
		hs.On("ListInventory", mock.Anything, "", sp).Return(nil, repos.NewRepoAccessError(model.NewRepoSpec("rem"), errors.New("unexpected"))).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 502 and json error as body
		assertResponse502(t, rec, route)
	})
}

func Test_InventoryByName(t *testing.T) {
	mockListResult := listResult2
	mockInventoryEntry := mockListResult.Entries[0]

	inventoryName := mockInventoryEntry.Name

	route := "/inventory/.tmName/" + inventoryName

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("with success", func(t *testing.T) {
		hs.On("FindInventoryEntries", mock.Anything, "", inventoryName).Return([]model.FoundEntry{mockInventoryEntry}, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response server.InventoryEntryResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: result has all data set
		if assert.Len(t, response.Data, 1) {
			assertInventoryEntry(t, mockInventoryEntry, response.Data[0])
		}
	})

	t.Run("with unknown error", func(t *testing.T) {
		hs.On("FindInventoryEntries", mock.Anything, "", inventoryName).Return(nil, unknownErr).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, route)
	})
}

func Test_Authors(t *testing.T) {

	route := "/authors"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	authors := []string{"author1", "author2", "author3"}

	t.Run("list all", func(t *testing.T) {
		var sp *model.Filters
		hs.On("ListAuthors", mock.Anything, sp).Return(authors, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response server.AuthorsResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then result contains all data
		assert.Equal(t, authors, response.Data)
		// and then result is ordered ascending by name
		isSorted := sort.SliceIsSorted(response.Data, func(i, j int) bool {
			return response.Data[i] < response.Data[j]
		})
		assert.True(t, isSorted)
	})

	t.Run("list with filter parameters", func(t *testing.T) {
		// given: the route with filter and search parameters
		fMan := "man1,man2"
		fMpn := "mpn1,mpn2"

		filterRoute := fmt.Sprintf("%s?filter.manufacturer=%s&filter.mpn=%s",
			route, fMan, fMpn)

		// and given: searchParams, expected to be converted from request query parameters
		expectedSearchParams := model.ToFilters(nil, &fMan, &fMpn, nil, nil, &model.FilterOptions{NameFilterType: model.PrefixMatch})

		hs.On("ListAuthors", mock.Anything, expectedSearchParams).Return(authors, nil).Once()

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, filterRoute).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
	})

	t.Run("with unknown error", func(t *testing.T) {
		var sp *model.Filters
		hs.On("ListAuthors", mock.Anything, sp).Return(nil, unknownErr).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, route)
	})
}

func Test_Manufacturers(t *testing.T) {

	route := "/manufacturers"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	manufacturers := []string{"man1", "man2", "man3"}

	t.Run("list all", func(t *testing.T) {
		var sp *model.Filters
		hs.On("ListManufacturers", mock.Anything, sp).Return(manufacturers, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response server.ManufacturersResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then result contains all data
		assert.Equal(t, manufacturers, response.Data)
		// and then result is ordered ascending by name
		isSorted := sort.SliceIsSorted(response.Data, func(i, j int) bool {
			return response.Data[i] < response.Data[j]
		})
		assert.True(t, isSorted)
	})

	t.Run("list with filter parameters", func(t *testing.T) {
		// given: the route with filter and search parameters
		fAuthors := []string{"a1", "a2"}
		fMpn := []string{"mpn1", "mpn2"}

		filterRoute := fmt.Sprintf("%s?filter.author=%s&filter.mpn=%s",
			route, strings.Join(fAuthors, ","), strings.Join(fMpn, ","))

		// and given: searchParams, expected to be converted from request query parameters
		expectedSearchParams := &model.Filters{
			Author:  fAuthors,
			Mpn:     fMpn,
			Options: model.FilterOptions{NameFilterType: model.PrefixMatch},
		}

		hs.On("ListManufacturers", mock.Anything, expectedSearchParams).Return(manufacturers, nil).Once()

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, filterRoute).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
	})

	t.Run("with unknown error", func(t *testing.T) {
		var sp *model.Filters
		hs.On("ListManufacturers", mock.Anything, sp).Return(nil, unknownErr).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, route)
	})
}

func Test_Mpns(t *testing.T) {

	route := "/mpns"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)
	mpns := []string{"mpn1", "mpn2", "mpn3"}

	t.Run("list all", func(t *testing.T) {
		var sp *model.Filters
		hs.On("ListMpns", mock.Anything, sp).Return(mpns, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response server.MpnsResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: duplicates are removed
		assert.Equal(t, 3, len(response.Data))
		// and then result contains all data
		assert.Equal(t, mpns, response.Data)
		// and then result is ordered ascending by name
		isSorted := sort.SliceIsSorted(response.Data, func(i, j int) bool {
			return response.Data[i] < response.Data[j]
		})
		assert.True(t, isSorted)
	})

	t.Run("list with filter parameters", func(t *testing.T) {
		// given: the route with filter and search parameters
		fAuthors := []string{"a1", "a2"}
		fMan := []string{"man1", "man2"}

		filterRoute := fmt.Sprintf("%s?filter.author=%s&filter.manufacturer=%s",
			route, strings.Join(fAuthors, ","), strings.Join(fMan, ","))

		// and given: searchParams, expected to be converted from request query parameters
		expectedSearchParams := &model.Filters{
			Author:       fAuthors,
			Manufacturer: fMan,
			Options:      model.FilterOptions{NameFilterType: model.PrefixMatch},
		}

		hs.On("ListMpns", mock.Anything, expectedSearchParams).Return(mpns, nil).Once()

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, filterRoute).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
	})

	t.Run("with unknown error", func(t *testing.T) {
		var sp *model.Filters
		hs.On("ListMpns", mock.Anything, sp).Return(nil, unknownErr).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, route)
	})
}
func Test_GetRepos(t *testing.T) {

	route := "/repos"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("with ok", func(t *testing.T) {
		r1d := "r1 description"
		ds := []model.RepoDescription{
			{
				Name:        "r1",
				Type:        "file",
				Description: r1d,
			},
			{
				Name: "r2",
				Type: "file",
			},
		}
		hs.On("ListRepos", mock.Anything).Return(ds, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response server.ReposResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then result contains all data
		assert.Equal(t, []server.RepoDescription{
			{
				Description: &r1d,
				Name:        "r1",
			},
			{
				Name: "r2",
			},
		}, response.Data)
	})

	t.Run("with one repo", func(t *testing.T) {
		hs.On("ListRepos", mock.Anything).Return([]model.RepoDescription{{Name: "r1"}}, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response server.ReposResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then result contains a single repo description
		assert.Equal(t, []server.RepoDescription{{Name: "r1"}}, response.Data)
	})

	t.Run("with nil result", func(t *testing.T) {
		hs.On("ListRepos", mock.Anything).Return(nil, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response server.ReposResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then result contains all data
		assert.Equal(t, []server.RepoDescription{}, response.Data)
	})

	t.Run("with unknown error", func(t *testing.T) {
		// given: service returns an error
		hs.On("ListRepos", mock.Anything).Return(nil, errors.New("unexpected")).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: the body is of correct type
		var response server.ReposResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, route)
	})
}
func Test_GetInventoryByID(t *testing.T) {
	ver := listResult2.Entries[0].Versions[0]
	tmID := ver.TMID
	route := "/inventory/" + tmID

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)
	t.Run("get inventory by tm id", func(t *testing.T) {
		hs.On("GetTMMetadata", mock.Anything, "", tmID).Return([]model.FoundVersion{ver}, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response server.InventoryEntryVersionsResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: result contains all data
		ctx := context.Background()
		ctx = context.WithValue(ctx, ctxRelPathDepth, 4)
		ctx = context.WithValue(ctx, ctxUrlRoot, "")
		assertInventoryEntryVersion(t, ver, response.Data[0])
	})

	t.Run("list with invalid tm id", func(t *testing.T) {
		// given: the route with invalid tm id
		route := "/inventory/invalid-id"

		hs.On("GetTMMetadata", mock.Anything, "", "invalid-id").Return(nil, model.ErrInvalidIdOrName).Once()

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 400
		assertResponse400(t, rec, route)
	})

	t.Run("with unknown error", func(t *testing.T) {
		// given: unknown error calling GetTMMetadata
		hs.On("GetTMMetadata", mock.Anything, "", tmID).Return(nil, unknownErr).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, route)
	})
}

func Test_FetchThingModel(t *testing.T) {
	tmID := listResult2.Entries[0].Versions[0].TMID
	tmContent := []byte("this is the content of a ThingModel")

	route := "/thing-models/" + tmID

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("with valid repo", func(t *testing.T) {
		hs.On("FetchThingModel", mock.Anything, "", tmID, false).Return(tmContent, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponseTM200(t, rec)
		assert.Equal(t, tmContent, rec.Body.Bytes())
	})

	t.Run("with false restoreId", func(t *testing.T) {
		hs.On("FetchThingModel", mock.Anything, "", tmID, false).Return(tmContent, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route+"?restoreId=false").RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponseTM200(t, rec)
		assert.Equal(t, tmContent, rec.Body.Bytes())
	})
	t.Run("with true restoreId", func(t *testing.T) {
		hs.On("FetchThingModel", mock.Anything, "", tmID, true).Return(tmContent, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route+"?restoreId=true").RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponseTM200(t, rec)
		assert.Equal(t, tmContent, rec.Body.Bytes())
	})
	t.Run("with invalid restoreId", func(t *testing.T) {
		// when: calling the route
		rr := route + "?restoreId=value"
		rec := testutils.NewRequest(http.MethodGet, rr).RunOnHandler(httpHandler)
		// then: it returns status 400
		assertResponse400(t, rec, rr)
	})
	t.Run("with invalid tmID", func(t *testing.T) {
		// given: route with invalid tmID
		invalidRoute := "/thing-models/some-invalid-tm-id"
		hs.On("FetchThingModel", mock.Anything, "", "some-invalid-tm-id", false).Return(nil, model.ErrInvalidId).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, invalidRoute).RunOnHandler(httpHandler)
		// then: it returns status 400 and json error as body
		assertResponse400(t, rec, invalidRoute)
	})

	t.Run("with not found error", func(t *testing.T) {
		hs.On("FetchThingModel", mock.Anything, "", tmID, false).Return(nil, model.ErrTMNotFound).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 404 and json error as body
		assertResponse404(t, rec, route)
	})
}
func Test_FetchAttachment(t *testing.T) {
	tmID := listResult2.Entries[0].Versions[0].TMID
	attContent := []byte("this is the content of an attachment")

	route := "/thing-models/" + tmID + "/.attachments/README.txt"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("with valid repo", func(t *testing.T) {
		hs.On("FetchAttachment", mock.Anything, "", model.NewTMIDAttachmentContainerRef(tmID), "README.txt", false).Return(attContent, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 200
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MimeOctetStream, rec.Header().Get(HeaderContentType))
		assert.Equal(t, attContent, rec.Body.Bytes())
	})

	t.Run("with invalid tmID", func(t *testing.T) {
		// given: route with invalid tmID
		invalidRoute := "/thing-models/some-invalid-tm-id/.attachments/README.txt"
		hs.On("FetchAttachment", mock.Anything, "", model.NewTMIDAttachmentContainerRef("some-invalid-tm-id"), "README.txt", false).Return(nil, model.ErrInvalidId).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, invalidRoute).RunOnHandler(httpHandler)
		// then: it returns status 400 and json error as body
		assertResponse400(t, rec, invalidRoute)
	})

	t.Run("with not found error", func(t *testing.T) {
		hs.On("FetchAttachment", mock.Anything, "", model.NewTMIDAttachmentContainerRef(tmID), "README.txt", false).Return(nil, model.ErrTMNotFound).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 404 and json error as body
		assertResponse404(t, rec, route)
		var errResponse server.ErrorResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &errResponse)
		if assert.NotNil(t, errResponse.Code) {
			assert.Equal(t, model.ErrTMNotFound.Subject, *errResponse.Code)
		}
	})
}

func Test_ImportThingModel(t *testing.T) {

	tmID := "a generated TM ID"
	_, tmContent, err := utils.ReadRequiredFile("../../../test/data/import/omnilamp-versioned.json")
	assert.NoError(t, err)

	route := "/thing-models"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("with success", func(t *testing.T) {
		hs.On("ImportThingModel", mock.Anything, "", tmContent, repos.ImportOptions{}).Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmID}, nil).Once()
		// when: calling the route

		rec := testutils.NewRequest(http.MethodPost, route).
			WithHeader(HeaderContentType, MimeJSON).
			WithBody(tmContent).
			RunOnHandler(httpHandler)

		// then: it returns status 201
		assertResponse201(t, rec)
		// and then: the body is of correct type
		var response server.ImportThingModelResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: tmID is set in response
		assert.NotNil(t, response.Data.TmID)
		assert.Equal(t, tmID, response.Data.TmID)
		assert.NoError(t, err)
	})

	t.Run("with missing or wrong Content-Type", func(t *testing.T) {
		contentTypes := []string{"", "application/pdf", "application/xml"}

		for _, c := range contentTypes {

			rec := testutils.NewRequest(http.MethodPost, route).
				WithHeader(HeaderContentType, c).
				WithBody(tmContent).
				RunOnHandler(httpHandler)

			// then: it returns status 400
			assertResponse400(t, rec, route)
		}
	})

	t.Run("with empty request body", func(t *testing.T) {
		// given: some empty ThingModel content
		var emptyContent []byte

		// when: calling the route
		rec := testutils.NewRequest(http.MethodPost, route).
			WithHeader(HeaderContentType, MimeJSON).
			WithBody(emptyContent).
			RunOnHandler(httpHandler)

		// then: it returns status 400
		assertResponse400(t, rec, route)
	})

	t.Run("with validation error", func(t *testing.T) {
		// given: some invalid ThingModel
		invalidContent := []byte("some invalid ThingModel")
		var err2 error = &jsonschema.ValidationError{}
		pr, err := repos.ImportResultFromError(err2)
		hs.On("ImportThingModel", mock.Anything, "", invalidContent, repos.ImportOptions{}).Return(pr, err).Once()
		// when: calling the route

		rec := testutils.NewRequest(http.MethodPost, route).
			WithHeader(HeaderContentType, MimeJSON).
			WithBody(invalidContent).
			RunOnHandler(httpHandler)

		// then: it returns status 400
		assertResponse400(t, rec, route)
	})

	t.Run("with too long name", func(t *testing.T) {
		// given: a thing model with too long name
		pr, err := repos.ImportResultFromError(fmt.Errorf("%w: %s", commands.ErrTMNameTooLong, "this-name-is-too-long"))
		hs.On("ImportThingModel", mock.Anything, "", tmContent, repos.ImportOptions{}).Return(pr, err).Once()
		// when: calling the route

		rec := testutils.NewRequest(http.MethodPost, route).
			WithHeader(HeaderContentType, MimeJSON).
			WithBody(tmContent).
			RunOnHandler(httpHandler)

		// then: it returns status 400
		assertResponse400(t, rec, route)
	})

	t.Run("with timestamp conflict", func(t *testing.T) {
		// given: a thing model file that conflicts with existing id
		cErr := &repos.ErrTMIDConflict{
			Type:       repos.IdConflictSameTimestamp,
			ExistingId: "existing-id",
		}
		hs.On("ImportThingModel", mock.Anything, "", tmContent, repos.ImportOptions{}).Return(repos.ImportResult{
			Type:    repos.ImportResultWarning,
			TmID:    tmID,
			Message: cErr.Error(),
			Err:     cErr,
		}, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodPost, route).
			WithHeader(HeaderContentType, MimeJSON).
			WithBody(tmContent).
			RunOnHandler(httpHandler)

		// then: it returns status 201
		assertResponse201(t, rec)
		// and then: the body is of correct type
		var response server.ImportThingModelResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: tmID is set in response
		assert.NotNil(t, response.Data.TmID)
		assert.Equal(t, tmID, response.Data.TmID)
		assert.NotNil(t, response.Data.Code)
		assert.Contains(t, *response.Data.Code, "existing-id")
	})
	t.Run("with content conflict", func(t *testing.T) {
		// given: a thing model file that conflicts with existing id
		cErr := &repos.ErrTMIDConflict{
			Type:       repos.IdConflictSameContent,
			ExistingId: "existing-id",
		}
		result := repos.ImportResult{
			Type:    repos.ImportResultError,
			TmID:    "",
			Message: cErr.Error(),
			Err:     cErr,
		}
		hs.On("ImportThingModel", mock.Anything, "", tmContent, repos.ImportOptions{}).Return(result, cErr).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodPost, route).
			WithHeader(HeaderContentType, MimeJSON).
			WithBody(tmContent).
			RunOnHandler(httpHandler)

		// then: it returns status 409 with appropriate error
		assertResponse409TMIDConflict(t, rec, route, cErr)
	})

	t.Run("with conflicting id with same timestamp", func(t *testing.T) {
		// given: a thing model file that conflicts with existing id
		cErr := &repos.ErrTMIDConflict{
			Type:       repos.IdConflictSameTimestamp,
			ExistingId: "existing-id",
		}
		result := repos.ImportResult{
			Type:    repos.ImportResultWarning,
			TmID:    tmID,
			Message: cErr.Error(),
			Err:     cErr,
		}
		hs.On("ImportThingModel", mock.Anything, "", tmContent, repos.ImportOptions{}).Return(result, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodPost, route).
			WithHeader(HeaderContentType, MimeJSON).
			WithBody(tmContent).
			RunOnHandler(httpHandler)

		// then: it returns status 201
		assertResponse201(t, rec)
		// and then: the body is of correct type
		var response server.ImportThingModelResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: tmID is set in response
		assert.NotNil(t, response.Data.TmID)
		assert.Equal(t, tmID, response.Data.TmID)
		assert.NotNil(t, response.Data.Code)
		assert.Contains(t, *response.Data.Code, "existing-id")
	})

	t.Run("with unknown error", func(t *testing.T) {
		// and given: some invalid ThingModel
		invalidContent := []byte("some invalid ThingModel")
		result, _ := repos.ImportResultFromError(unknownErr)
		hs.On("ImportThingModel", mock.Anything, "", invalidContent, repos.ImportOptions{}).Return(result, unknownErr).Once()
		// when: calling the route

		rec := testutils.NewRequest(http.MethodPost, route).
			WithHeader(HeaderContentType, MimeJSON).
			WithBody(invalidContent).
			RunOnHandler(httpHandler)

		// then: it returns status 500
		assertResponse500(t, rec, route)
	})
}
func Test_ImportAttachment(t *testing.T) {

	attContent := []byte("# readme.md file")

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("tmid attachment", func(t *testing.T) {
		tmID := "a-corp/eagle/bt2000/v1.0.0-20231231153548-243d1b462ddd.tm.json"
		route := "/thing-models/" + tmID + "/.attachments/README.md"

		t.Run("with success", func(t *testing.T) {
			hs.On("ImportAttachment", mock.Anything, "", model.NewTMIDAttachmentContainerRef(tmID), "README.md", attContent, "text/markdown", true).Return(nil).Once()
			// when: calling the route
			rec := testutils.NewRequest(http.MethodPut, route+"?force=true").
				WithHeader(HeaderContentType, "text/markdown").
				WithBody(attContent).
				RunOnHandler(httpHandler)

			// then: it returns status 204
			assert.Equal(t, http.StatusNoContent, rec.Code)
		})

		t.Run("with invalid force parameter", func(t *testing.T) {
			// when: calling the route
			rec := testutils.NewRequest(http.MethodPut, route+"?force=42").
				WithHeader(HeaderContentType, "text/markdown").
				WithBody(attContent).
				RunOnHandler(httpHandler)

			// then: it returns status 400
			assertResponse400(t, rec, route+"?force=42")
		})

		t.Run("with invalid id", func(t *testing.T) {
			// given: some route with invalid tmID
			route := "/thing-models/not-an-id/.attachments/README.md"
			hs.On("ImportAttachment", mock.Anything, "", model.NewTMIDAttachmentContainerRef("not-an-id"), "README.md", attContent, "text/markdown", false).Return(model.ErrInvalidIdOrName).Once()
			// when: calling the route

			rec := testutils.NewRequest(http.MethodPut, route).
				WithHeader(HeaderContentType, "text/markdown").
				WithBody(attContent).
				RunOnHandler(httpHandler)

			// then: it returns status 400
			assertResponse400(t, rec, route)
		})

		t.Run("with attachment conflict", func(t *testing.T) {
			// given: some route with invalid tmID
			route := "/thing-models/" + tmID + "/.attachments/DONTREADME.md"
			hs.On("ImportAttachment", mock.Anything, "", model.NewTMIDAttachmentContainerRef(tmID), "DONTREADME.md", attContent, "text/markdown", false).Return(repos.ErrAttachmentExists).Once()
			// when: calling the route
			rec := testutils.NewRequest(http.MethodPut, route).
				WithHeader(HeaderContentType, "text/markdown").
				WithBody(attContent).
				RunOnHandler(httpHandler)

			// then: it returns status 409
			assertResponse409(t, rec, route)
		})

		t.Run("with empty request body", func(t *testing.T) {
			// given: some empty attachment content
			var emptyContent []byte

			// when: calling the route
			rec := testutils.NewRequest(http.MethodPut, route).
				WithHeader(HeaderContentType, MimeJSON).
				WithBody(emptyContent).
				RunOnHandler(httpHandler)

			// then: it returns status 400
			assertResponse400(t, rec, route)
		})

		t.Run("with unknown error", func(t *testing.T) {
			// and given: some unknown error
			hs.On("ImportAttachment", mock.Anything, "", model.NewTMIDAttachmentContainerRef(tmID), "README.md", attContent, MimeOctetStream, false).Return(unknownErr).Once()
			// when: calling the route
			rec := testutils.NewRequest(http.MethodPut, route).
				WithHeader(HeaderContentType, MimeOctetStream).
				WithBody(attContent).
				RunOnHandler(httpHandler)

			// then: it returns status 500
			assertResponse500(t, rec, route)
		})
	})
	t.Run("tmname attachment", func(t *testing.T) {
		tmName := "a-corp/eagle/bt2000"
		route := "/thing-models/.tmName/" + tmName + "/.attachments/README.md"

		t.Run("with success", func(t *testing.T) {
			hs.On("ImportAttachment", mock.Anything, "", model.NewTMNameAttachmentContainerRef(tmName), "README.md", attContent, "text/markdown", true).Return(nil).Once()
			// when: calling the route
			rec := testutils.NewRequest(http.MethodPut, route+"?force=true").
				WithHeader(HeaderContentType, "text/markdown").
				WithBody(attContent).
				RunOnHandler(httpHandler)

			// then: it returns status 204
			assert.Equal(t, http.StatusNoContent, rec.Code)
		})

		t.Run("with invalid force parameter", func(t *testing.T) {
			// when: calling the route
			rec := testutils.NewRequest(http.MethodPut, route+"?force=42").
				WithHeader(HeaderContentType, "text/markdown").
				WithBody(attContent).
				RunOnHandler(httpHandler)

			// then: it returns status 400
			assertResponse400(t, rec, route+"?force=42")
		})

		t.Run("with invalid id", func(t *testing.T) {
			// given: some route with invalid tmName
			route := "/thing-models/.tmName/not-an-name/.attachments/README.md"
			hs.On("ImportAttachment", mock.Anything, "", model.NewTMNameAttachmentContainerRef("not-an-name"), "README.md", attContent, "text/markdown", false).Return(model.ErrInvalidIdOrName).Once()
			// when: calling the route

			rec := testutils.NewRequest(http.MethodPut, route).
				WithHeader(HeaderContentType, "text/markdown").
				WithBody(attContent).
				RunOnHandler(httpHandler)

			// then: it returns status 400
			assertResponse400(t, rec, route)
		})

		t.Run("with attachment conflict", func(t *testing.T) {
			// given: some route with invalid tmName
			route := "/thing-models/.tmName/" + tmName + "/.attachments/DONTREADME.md"
			hs.On("ImportAttachment", mock.Anything, "", model.NewTMNameAttachmentContainerRef(tmName), "DONTREADME.md", attContent, "text/markdown", false).Return(repos.ErrAttachmentExists).Once()
			// when: calling the route
			rec := testutils.NewRequest(http.MethodPut, route).
				WithHeader(HeaderContentType, "text/markdown").
				WithBody(attContent).
				RunOnHandler(httpHandler)

			// then: it returns status 409
			assertResponse409(t, rec, route)
		})

		t.Run("with empty request body", func(t *testing.T) {
			// given: some empty attachment content
			var emptyContent []byte

			// when: calling the route
			rec := testutils.NewRequest(http.MethodPut, route).
				WithHeader(HeaderContentType, MimeJSON).
				WithBody(emptyContent).
				RunOnHandler(httpHandler)

			// then: it returns status 400
			assertResponse400(t, rec, route)
		})

		t.Run("with unknown error", func(t *testing.T) {
			// and given: some unknown error
			hs.On("ImportAttachment", mock.Anything, "", model.NewTMNameAttachmentContainerRef(tmName), "README.md", attContent, MimeOctetStream, false).Return(unknownErr).Once()
			// when: calling the route
			rec := testutils.NewRequest(http.MethodPut, route).
				WithHeader(HeaderContentType, MimeOctetStream).
				WithBody(attContent).
				RunOnHandler(httpHandler)

			// then: it returns status 500
			assertResponse500(t, rec, route)
		})
	})
}

func Test_DeleteThingModelById(t *testing.T) {
	tmID := listResult2.Entries[0].Versions[0].TMID

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("without force parameter", func(t *testing.T) {
		route := "/thing-models/" + tmID
		// when: calling the route
		rec := testutils.NewRequest(http.MethodDelete, route).RunOnHandler(httpHandler)
		// then: it returns status 400
		assertResponse400(t, rec, route)
	})

	t.Run("with invalid force parameter", func(t *testing.T) {
		route := "/thing-models/" + tmID + "?force=yes"
		// when: calling the route
		rec := testutils.NewRequest(http.MethodDelete, route).RunOnHandler(httpHandler)
		// then: it returns status 400
		assertResponse400(t, rec, route)
	})

	t.Run("with valid tmID", func(t *testing.T) {
		route := "/thing-models/" + tmID + "?force=true"
		hs.On("DeleteThingModel", mock.Anything, "", tmID).Return(nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodDelete, route).RunOnHandler(httpHandler)
		// then: it returns status 204
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, 0, rec.Body.Len())
	})

	t.Run("with invalid tmID", func(t *testing.T) {
		// given: route with invalid tmID
		route := "/thing-models/some-invalid-tm-id?force=true"
		hs.On("DeleteThingModel", mock.Anything, "", "some-invalid-tm-id").Return(model.ErrInvalidId).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodDelete, route).RunOnHandler(httpHandler)
		// then: it returns status 400 and json error as body
		assertResponse400(t, rec, route)
	})

	t.Run("with not found error", func(t *testing.T) {
		route := "/thing-models/" + tmID + "?force=true"
		hs.On("DeleteThingModel", mock.Anything, "", tmID).Return(model.ErrTMNotFound).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodDelete, route).RunOnHandler(httpHandler)
		// then: it returns status 404 and json error as body
		assertResponse404(t, rec, route)
		var errResponse server.ErrorResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &errResponse)
		if assert.NotNil(t, errResponse.Code) {
			assert.Equal(t, model.ErrTMNotFound.Subject, *errResponse.Code)
		}

	})

}
func Test_DeleteAttachment(t *testing.T) {
	tmID := listResult2.Entries[0].Versions[0].TMID

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	route := "/thing-models/" + tmID + "/.attachments/README.txt"

	t.Run("with valid tmID", func(t *testing.T) {
		hs.On("DeleteAttachment", mock.Anything, "", model.NewTMIDAttachmentContainerRef(tmID), "README.txt").Return(nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodDelete, route).RunOnHandler(httpHandler)
		// then: it returns status 204
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, 0, rec.Body.Len())
	})

	t.Run("with invalid tmID", func(t *testing.T) {
		// given: route with invalid tmID
		route := "/thing-models/some-invalid-tm-id/.attachments/README.txt"
		hs.On("DeleteAttachment", mock.Anything, "", model.NewTMIDAttachmentContainerRef("some-invalid-tm-id"), "README.txt").Return(model.ErrInvalidId).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodDelete, route).RunOnHandler(httpHandler)
		// then: it returns status 400 and json error as body
		assertResponse400(t, rec, route)
	})

	t.Run("with not found error", func(t *testing.T) {
		hs.On("DeleteAttachment", mock.Anything, "", model.NewTMIDAttachmentContainerRef(tmID), "README.txt").Return(model.ErrAttachmentNotFound).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodDelete, route).RunOnHandler(httpHandler)
		// then: it returns status 404 and json error as body
		assertResponse404(t, rec, route)
		var errResponse server.ErrorResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &errResponse)
		if assert.NotNil(t, errResponse.Code) {
			assert.Equal(t, model.ErrAttachmentNotFound.Subject, *errResponse.Code)
		}
	})

}

func Test_Completions(t *testing.T) {

	route := "/.completions"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("no parameters", func(t *testing.T) {
		hs.On("GetCompletions", mock.Anything, "", mock.Anything, "").Return(nil, repos.ErrInvalidCompletionParams).Once()

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 400
		assertResponse400(t, rec, route)
		// and then: the body is of correct type
		var response server.ErrorResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
	})

	t.Run("unknown completion kind", func(t *testing.T) {
		hs.On("GetCompletions", mock.Anything, "something", mock.Anything, "").Return(nil, repos.ErrInvalidCompletionParams).Once()

		// when: calling the route
		rr := fmt.Sprintf("%s?kind=something", route)
		rec := testutils.NewRequest(http.MethodGet, rr).RunOnHandler(httpHandler)
		// then: it returns status 400
		assertResponse400(t, rec, rr)
		// and then: the body is of correct type
		var response server.ErrorResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
	})

	t.Run("known completion kind", func(t *testing.T) {
		hs.On("GetCompletions", mock.Anything, "names", []string{"aut/man/mpn"}, "RE").Return([]string{"README.md", "README.pdf"}, nil).Once()

		// when: calling the route
		rr := fmt.Sprintf("%s?kind=names&toComplete=RE&args=aut%%2Fman%%2Fmpn", route)
		rec := testutils.NewRequest(http.MethodGet, rr).RunOnHandler(httpHandler)
		// then: it returns status 200
		assert.Equal(t, http.StatusOK, rec.Code)
		// and then: the body is of correct type
		assert.Equal(t, MimeText, rec.Header().Get(HeaderContentType))
		assert.Equal(t, []byte("README.md\nREADME.pdf\n"), rec.Body.Bytes())
	})

	t.Run("with unknown error", func(t *testing.T) {
		hs.On("GetCompletions", mock.Anything, "names", mock.Anything, "").Return(nil, unknownErr).Once()
		// when: calling the route
		rr := fmt.Sprintf("%s?kind=names&toComplete=", route)
		rec := testutils.NewRequest(http.MethodGet, rr).RunOnHandler(httpHandler)
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, rr)
	})
}

func assertHealthyResponse204(t *testing.T, rec *httptest.ResponseRecorder) {
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, 0, rec.Body.Len())
	assert.Equal(t, NoCache, rec.Header().Get(HeaderCacheControl))
}

func assertResponse200(t *testing.T, rec *httptest.ResponseRecorder) {
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, MimeJSON, rec.Header().Get(HeaderContentType))
}

func assertResponseTM200(t *testing.T, rec *httptest.ResponseRecorder) {
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, MimeTMJSON, rec.Header().Get(HeaderContentType))
}

func assertResponse201(t *testing.T, rec *httptest.ResponseRecorder) {
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, MimeJSON, rec.Header().Get(HeaderContentType))
}

func assertResponse400(t *testing.T, rec *httptest.ResponseRecorder, route string) {
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var errResponse server.ErrorResponse
	assertUnmarshalResponse(t, rec.Body.Bytes(), &errResponse)
	assert.Equal(t, http.StatusBadRequest, errResponse.Status)
	assert.Equal(t, route, *errResponse.Instance)
	assert.Equal(t, Error400Title, errResponse.Title)

	assert.Equal(t, MimeProblemJSON, rec.Header().Get(HeaderContentType))
	assert.Equal(t, NoSniff, rec.Header().Get(HeaderXContentTypeOptions))
}

func assertResponse409TMIDConflict(t *testing.T, rec *httptest.ResponseRecorder, route string, idErr *repos.ErrTMIDConflict) {
	assert.Equal(t, http.StatusConflict, rec.Code)
	var errResponse server.ErrorResponse
	assertUnmarshalResponse(t, rec.Body.Bytes(), &errResponse)
	assert.Equal(t, http.StatusConflict, errResponse.Status)
	assert.Equal(t, route, *errResponse.Instance)
	assert.Equal(t, Error409Title, errResponse.Title)
	if assert.NotNil(t, errResponse.Code) {
		cErr, err := repos.ParseErrTMIDConflict(*errResponse.Code)
		assert.NoError(t, err)
		assert.Equal(t, idErr, cErr)
	}

	assert.Equal(t, MimeProblemJSON, rec.Header().Get(HeaderContentType))
	assert.Equal(t, NoSniff, rec.Header().Get(HeaderXContentTypeOptions))
}
func assertResponse409(t *testing.T, rec *httptest.ResponseRecorder, route string) {
	assert.Equal(t, http.StatusConflict, rec.Code)
	var errResponse server.ErrorResponse
	assertUnmarshalResponse(t, rec.Body.Bytes(), &errResponse)
	assert.Equal(t, http.StatusConflict, errResponse.Status)
	assert.Equal(t, route, *errResponse.Instance)
	assert.Equal(t, Error409Title, errResponse.Title)

	assert.Equal(t, MimeProblemJSON, rec.Header().Get(HeaderContentType))
	assert.Equal(t, NoSniff, rec.Header().Get(HeaderXContentTypeOptions))
}

func assertResponse404(t *testing.T, rec *httptest.ResponseRecorder, route string) {
	assert.Equal(t, http.StatusNotFound, rec.Code)
	var errResponse server.ErrorResponse
	assertUnmarshalResponse(t, rec.Body.Bytes(), &errResponse)
	assert.Equal(t, http.StatusNotFound, errResponse.Status)
	assert.Equal(t, route, *errResponse.Instance)
	assert.Equal(t, Error404Title, errResponse.Title)

	assert.Equal(t, MimeProblemJSON, rec.Header().Get(HeaderContentType))
	assert.Equal(t, NoSniff, rec.Header().Get(HeaderXContentTypeOptions))
}

func assertResponse500(t *testing.T, rec *httptest.ResponseRecorder, route string) {
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	var errResponse server.ErrorResponse
	assertUnmarshalResponse(t, rec.Body.Bytes(), &errResponse)
	assert.Equal(t, http.StatusInternalServerError, errResponse.Status)
	assert.Equal(t, route, *errResponse.Instance)
	assert.Equal(t, Error500Title, errResponse.Title)
	assert.Equal(t, Error500Detail, *errResponse.Detail)

	assert.Equal(t, MimeProblemJSON, rec.Header().Get(HeaderContentType))
	assert.Equal(t, NoSniff, rec.Header().Get(HeaderXContentTypeOptions))
}

func assertResponse502(t *testing.T, rec *httptest.ResponseRecorder, route string) {
	assert.Equal(t, http.StatusBadGateway, rec.Code)
	var errResponse server.ErrorResponse
	assertUnmarshalResponse(t, rec.Body.Bytes(), &errResponse)
	assert.Equal(t, http.StatusBadGateway, errResponse.Status)
	assert.Equal(t, route, *errResponse.Instance)
	assert.Equal(t, Error502Title, errResponse.Title)
	assert.Equal(t, Error502Detail, *errResponse.Detail)

	assert.Equal(t, MimeProblemJSON, rec.Header().Get(HeaderContentType))
	assert.Equal(t, NoSniff, rec.Header().Get(HeaderXContentTypeOptions))
}

func assertResponse503(t *testing.T, rec *httptest.ResponseRecorder, route string) {
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	var errResponse server.ErrorResponse
	assertUnmarshalResponse(t, rec.Body.Bytes(), &errResponse)
	assert.Equal(t, http.StatusServiceUnavailable, errResponse.Status)
	assert.Equal(t, route, *errResponse.Instance)
	assert.Equal(t, Error503Title, errResponse.Title)

	assert.Equal(t, MimeProblemJSON, rec.Header().Get(HeaderContentType))
	assert.Equal(t, NoSniff, rec.Header().Get(HeaderXContentTypeOptions))
}

func assertUnmarshalResponse(t *testing.T, data []byte, v any) {
	err := json.Unmarshal(data, v)
	assert.NoError(t, err, "error unmarshalling response")
}

func assertInventoryEntry(t *testing.T, ref model.FoundEntry, entry server.InventoryEntry) {
	assert.Equal(t, ref.Name, entry.TmName)
	assert.Equal(t, ref.Author.Name, entry.SchemaAuthor.SchemaName)
	assert.Equal(t, ref.Manufacturer.Name, entry.SchemaManufacturer.SchemaName)
	assert.Equal(t, ref.Mpn, entry.SchemaMpn)
	expSuffix := "/inventory/.tmName/" + ref.Name
	if ref.FoundIn.RepoName != "" {
		expSuffix += "?repo=" + ref.FoundIn.RepoName
	}
	assert.Truef(t, strings.HasSuffix(entry.Links.Self, expSuffix), "%s does not end with %s", entry.Links.Self, expSuffix)
	assertAttachments(t, path.Join(".tmName", ref.Name), ref.Attachments, entry.Attachments)
	assert.Equal(t, len(ref.Versions), len(entry.Versions))
	assertInventoryEntryVersions(t, ref.Versions, entry.Versions)
}

func assertInventoryEntryVersions(t *testing.T, ref []model.FoundVersion, versions []server.InventoryEntryVersion) {
	for idx, refVer := range ref {
		entryVer := versions[idx]

		assertInventoryEntryVersion(t, refVer, entryVer)
	}
}

func assertInventoryEntryVersion(t *testing.T, refVer model.FoundVersion, entryVer server.InventoryEntryVersion) {
	assert.Equal(t, refVer.Description, entryVer.Description)
	assert.Equal(t, refVer.Version.Model, entryVer.Version.Model)
	assert.True(t, strings.HasSuffix(entryVer.Links.Content, "/thing-models/"+refVer.TMID))
	assert.Equal(t, refVer.TMID, entryVer.TmID)
	assert.Equal(t, refVer.Digest, entryVer.Digest)
	assert.Equal(t, refVer.TimeStamp, entryVer.Timestamp)
	assert.Equal(t, refVer.ExternalID, entryVer.ExternalID)
	assertAttachments(t, refVer.TMID, refVer.Attachments, entryVer.Attachments)
}

func assertAttachments(t *testing.T, linkPrefix string, expAtts []model.Attachment, atts *server.AttachmentsList) {
	if atts == nil {
		assert.True(t, len(expAtts) == 0, "empty attachments, but not empty expected attachments")
		return
	}
	assert.Equal(t, len(expAtts), len(*atts))
	for i, a := range *atts {
		expAtt := expAtts[i]
		assert.Equal(t, expAtt.Name, a.Name)
		expPath := path.Join(linkPrefix, ".attachments", url.PathEscape(expAtt.Name))
		assert.Truef(t, strings.Contains(a.Links.Content, expPath), "%s does not contain %s", a.Links.Content, expPath)
	}
}

var (
	listResult1 = model.SearchResult{
		LastUpdated: time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC),
		Entries: []model.FoundEntry{
			{
				Name:         "a-corp/eagle/bt2000",
				Author:       model.SchemaAuthor{Name: "a-corp"},
				Manufacturer: model.SchemaManufacturer{Name: "eagle"},
				Mpn:          "bt2000",
				AttachmentContainer: model.AttachmentContainer{
					Attachments: []model.Attachment{
						{
							Name: "README.md",
						},
					},
				},
				FoundIn: model.FoundSource{RepoName: "r1"},
				Versions: []model.FoundVersion{
					{
						IndexVersion: &model.IndexVersion{
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
						IndexVersion: &model.IndexVersion{
							TMID:        "a-corp/eagle/bt2000/v1.0.0-20231231153548-243d1b462ddd.tm.json",
							Description: "desc version v0.0.0",
							Version:     model.Version{Model: "0.0.0"},
							Digest:      "243d1b462ddd",
							TimeStamp:   "20231231153548",
							ExternalID:  "ext-1",
							AttachmentContainer: model.AttachmentContainer{
								Attachments: []model.Attachment{
									{
										Name: "CHANGELOG.md",
									},
								},
							},
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
				FoundIn:      model.FoundSource{RepoName: "r1"},
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

	listResult2 = model.SearchResult{
		Entries: []model.FoundEntry{
			{
				Name:         "b-corp/eagle/PM20",
				Author:       model.SchemaAuthor{Name: "b-corp"},
				Manufacturer: model.SchemaManufacturer{Name: "eagle"},
				Mpn:          "PM20",
				FoundIn:      model.FoundSource{RepoName: "r2"},
				Versions: []model.FoundVersion{
					{
						IndexVersion: &model.IndexVersion{
							Description: "desc version v1.0.0",
							Version:     model.Version{Model: "1.0.0"},
							TMID:        "b-corp/eagle/PM20/v1.0.0-20240107123001-234d1b462fff.tm.json",
							Digest:      "234d1b462fff",
							TimeStamp:   "20240107123001",
							ExternalID:  "ext-4",
							Protocols:   []string{"coaps", "https"},
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
					},
				},
			},
		},
	}
)
