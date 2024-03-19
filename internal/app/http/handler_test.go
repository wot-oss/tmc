package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/wot-oss/tmc/internal/app/http/mocks"
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
		hs.On("CheckHealthLive", nil).Return(nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns 204 status and empty body
		assertHealthyResponse204(t, rec)
	})

	t.Run("with error", func(t *testing.T) {
		hs.On("CheckHealthLive", nil).Return(unknownErr).Once()
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
		hs.On("CheckHealthReady", nil).Return(nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns 204 status and empty body
		assertHealthyResponse204(t, rec)
	})

	t.Run("with error", func(t *testing.T) {
		hs.On("CheckHealthReady", nil).Return(unknownErr).Once()
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
		hs.On("CheckHealthStartup", nil).Return(nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns 204 status and empty body
		assertHealthyResponse204(t, rec)
	})

	t.Run("with error", func(t *testing.T) {
		hs.On("CheckHealthStartup", nil).Return(unknownErr).Once()
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
		hs.On("CheckHealth", nil).Return(nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns 204 status and empty body
		assertHealthyResponse204(t, rec)
	})

	t.Run("with error", func(t *testing.T) {
		hs.On("CheckHealth", nil).Return(unknownErr).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns 503 status and json error as body
		assertResponse503(t, rec, route)
	})
}

func Test_Inventory(t *testing.T) {

	route := "/inventory"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("list all", func(t *testing.T) {

		hs.On("ListInventory", nil, &model.SearchParams{}).Return(&listResult1, nil).Once()

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
		// and then result is ordered ascending by name
		isSorted := sort.SliceIsSorted(response.Data, func(i, j int) bool {
			return response.Data[i].Name < response.Data[j].Name
		})
		assert.True(t, isSorted)
	})

	t.Run("list with filter and search parameter", func(t *testing.T) {
		// given: the route with filter and search parameters
		fAuthors := []string{"a1", "a2"}
		fMan := []string{"man1", "man2"}
		fMpn := []string{"mpn1", "mpn2"}
		search := "foo"

		filterRoute := fmt.Sprintf("%s?filter.author=%s&filter.manufacturer=%s&filter.mpn=%s&search=%s",
			route, strings.Join(fAuthors, ","), strings.Join(fMan, ","), strings.Join(fMpn, ","), search)

		// and given: searchParams, expected to be converted from request query parameters
		expectedSearchParams := &model.SearchParams{
			Author:       fAuthors,
			Manufacturer: fMan,
			Mpn:          fMpn,
			Query:        search,
		}

		hs.On("ListInventory", nil, expectedSearchParams).Return(&listResult1, nil).Once()

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, filterRoute).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
	})

	t.Run("with unknown error", func(t *testing.T) {
		hs.On("ListInventory", nil, &model.SearchParams{}).Return(nil, unknownErr).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, route)
	})

	t.Run("with repository access error", func(t *testing.T) {
		hs.On("ListInventory", nil, &model.SearchParams{}).Return(nil, repos.NewRepoAccessError(model.NewRepoSpec("rem"), errors.New("unexpected"))).Once()
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

	route := "/inventory/" + inventoryName

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("with success", func(t *testing.T) {
		hs.On("FindInventoryEntry", nil, inventoryName).Return(&mockInventoryEntry, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response server.InventoryEntryResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: result has all data set
		assertInventoryEntry(t, mockInventoryEntry, response.Data)
	})

	t.Run("with unknown error", func(t *testing.T) {
		hs.On("FindInventoryEntry", nil, inventoryName).Return(nil, unknownErr).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, route)
	})
}

func Test_InventoryEntryVersionsByName(t *testing.T) {
	mockListResult := listResult2
	mockInventoryEntry := mockListResult.Entries[0]

	inventoryName := mockInventoryEntry.Name

	route := "/inventory/" + inventoryName + "/.versions"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("with success", func(t *testing.T) {
		hs.On("FindInventoryEntry", nil, inventoryName).Return(&mockInventoryEntry, nil).Once()

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response server.InventoryEntryVersionsResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: result has all data set
		assertInventoryEntryVersions(t, mockInventoryEntry.Versions, response.Data)
	})

	t.Run("with unknown error", func(t *testing.T) {
		hs.On("FindInventoryEntry", nil, inventoryName).Return(nil, unknownErr).Once()
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
		hs.On("ListAuthors", nil, &model.SearchParams{}).Return(authors, nil).Once()
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

	t.Run("list with filter and search parameter", func(t *testing.T) {
		// given: the route with filter and search parameters
		fMan := []string{"man1", "man2"}
		fMpn := []string{"mpn1", "mpn2"}
		search := "foo"

		filterRoute := fmt.Sprintf("%s?filter.manufacturer=%s&filter.mpn=%s&search=%s",
			route, strings.Join(fMan, ","), strings.Join(fMpn, ","), search)

		// and given: searchParams, expected to be converted from request query parameters
		expectedSearchParams := &model.SearchParams{
			Manufacturer: fMan,
			Mpn:          fMpn,
			Query:        search,
		}

		hs.On("ListAuthors", nil, expectedSearchParams).Return(authors, nil).Once()

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, filterRoute).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
	})

	t.Run("with unknown error", func(t *testing.T) {
		hs.On("ListAuthors", nil, &model.SearchParams{}).Return(nil, unknownErr).Once()
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
		hs.On("ListManufacturers", nil, &model.SearchParams{}).Return(manufacturers, nil).Once()
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

	t.Run("list with filter and search parameter", func(t *testing.T) {
		// given: the route with filter and search parameters
		fAuthors := []string{"a1", "a2"}
		fMpn := []string{"mpn1", "mpn2"}
		search := "foo"

		filterRoute := fmt.Sprintf("%s?filter.author=%s&filter.mpn=%s&search=%s",
			route, strings.Join(fAuthors, ","), strings.Join(fMpn, ","), search)

		// and given: searchParams, expected to be converted from request query parameters
		expectedSearchParams := &model.SearchParams{
			Author: fAuthors,
			Mpn:    fMpn,
			Query:  search,
		}

		hs.On("ListManufacturers", nil, expectedSearchParams).Return(manufacturers, nil).Once()

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, filterRoute).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
	})

	t.Run("with unknown error", func(t *testing.T) {
		hs.On("ListManufacturers", nil, &model.SearchParams{}).Return(nil, unknownErr).Once()
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
		hs.On("ListMpns", nil, &model.SearchParams{}).Return(mpns, nil).Once()
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

	t.Run("list with filter and search parameter", func(t *testing.T) {
		// given: the route with filter and search parameters
		fAuthors := []string{"a1", "a2"}
		fMan := []string{"man1", "man2"}
		search := "foo"

		filterRoute := fmt.Sprintf("%s?filter.author=%s&filter.manufacturer=%s&search=%s",
			route, strings.Join(fAuthors, ","), strings.Join(fMan, ","), search)

		// and given: searchParams, expected to be converted from request query parameters
		expectedSearchParams := &model.SearchParams{
			Author:       fAuthors,
			Manufacturer: fMan,
			Query:        search,
		}

		hs.On("ListMpns", nil, expectedSearchParams).Return(mpns, nil).Once()

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, filterRoute).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
	})

	t.Run("with unknown error", func(t *testing.T) {
		hs.On("ListMpns", nil, &model.SearchParams{}).Return(nil, unknownErr).Once()
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
		hs.On("FetchThingModel", nil, tmID, false).Return(tmContent, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
		assert.Equal(t, tmContent, rec.Body.Bytes())
	})

	t.Run("with false restoreId", func(t *testing.T) {
		hs.On("FetchThingModel", nil, tmID, false).Return(tmContent, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route+"?restoreId=false").RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
		assert.Equal(t, tmContent, rec.Body.Bytes())
	})
	t.Run("with true restoreId", func(t *testing.T) {
		hs.On("FetchThingModel", nil, tmID, true).Return(tmContent, nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route+"?restoreId=true").RunOnHandler(httpHandler)
		// then: it returns status 200
		assertResponse200(t, rec)
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
		hs.On("FetchThingModel", nil, "some-invalid-tm-id", false).Return(nil, model.ErrInvalidId).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, invalidRoute).RunOnHandler(httpHandler)
		// then: it returns status 400 and json error as body
		assertResponse400(t, rec, invalidRoute)
	})

	t.Run("with not found error", func(t *testing.T) {
		hs.On("FetchThingModel", nil, tmID, false).Return(nil, repos.ErrTmNotFound).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 404 and json error as body
		assertResponse404(t, rec, route)
	})
}

func Test_PushThingModel(t *testing.T) {

	tmID := "a generated TM ID"
	_, tmContent, err := utils.ReadRequiredFile("../../../test/data/push/omnilamp-versioned.json")
	assert.NoError(t, err)

	route := "/thing-models"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("with success", func(t *testing.T) {
		hs.On("PushThingModel", nil, tmContent).Return(tmID, nil).Once()
		// when: calling the route

		rec := testutils.NewRequest(http.MethodPost, route).
			WithHeader(HeaderContentType, MimeJSON).
			WithBody(tmContent).
			RunOnHandler(httpHandler)

		// then: it returns status 201
		assertResponse201(t, rec)
		// and then: the body is of correct type
		var response server.PushThingModelResponse
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

	t.Run("with validation error", func(t *testing.T) {
		// given: some invalid ThingModel
		invalidContent := []byte("some invalid ThingModel")
		hs.On("PushThingModel", nil, invalidContent).Return("", &jsonschema.ValidationError{}).Once()
		// when: calling the route

		rec := testutils.NewRequest(http.MethodPost, route).
			WithHeader(HeaderContentType, MimeJSON).
			WithBody(invalidContent).
			RunOnHandler(httpHandler)

		// then: it returns status 400
		assertResponse400(t, rec, route)
	})

	t.Run("with conflicting id", func(t *testing.T) {
		// given: a thing model file that conflicts with existing id
		cErr := &repos.ErrTMIDConflict{
			Type:       repos.IdConflictSameTimestamp,
			ExistingId: "existing-id",
		}
		hs.On("PushThingModel", nil, tmContent).Return("", cErr).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodPost, route).
			WithHeader(HeaderContentType, MimeJSON).
			WithBody(tmContent).
			RunOnHandler(httpHandler)

		// then: it returns status 409 with appropriate error
		assertResponse409(t, rec, route, cErr)
	})

	t.Run("with unknown error", func(t *testing.T) {
		// and given: some invalid ThingModel
		invalidContent := []byte("some invalid ThingModel")
		hs.On("PushThingModel", nil, invalidContent).Return("", unknownErr).Once()
		// when: calling the route

		rec := testutils.NewRequest(http.MethodPost, route).
			WithHeader(HeaderContentType, MimeJSON).
			WithBody(invalidContent).
			RunOnHandler(httpHandler)

		// then: it returns status 500
		assertResponse500(t, rec, route)
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
		hs.On("DeleteThingModel", mock.Anything, tmID).Return(nil).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodDelete, route).RunOnHandler(httpHandler)
		// then: it returns status 204
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, 0, rec.Body.Len())
	})

	t.Run("with invalid tmID", func(t *testing.T) {
		// given: route with invalid tmID
		route := "/thing-models/some-invalid-tm-id?force=true"
		hs.On("DeleteThingModel", mock.Anything, "some-invalid-tm-id").Return(model.ErrInvalidId).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodDelete, route).RunOnHandler(httpHandler)
		// then: it returns status 400 and json error as body
		assertResponse400(t, rec, route)
	})

	t.Run("with not found error", func(t *testing.T) {
		route := "/thing-models/" + tmID + "?force=true"
		hs.On("DeleteThingModel", mock.Anything, tmID).Return(repos.ErrTmNotFound).Once()
		// when: calling the route
		rec := testutils.NewRequest(http.MethodDelete, route).RunOnHandler(httpHandler)
		// then: it returns status 404 and json error as body
		assertResponse404(t, rec, route)
	})

}

func Test_Completions(t *testing.T) {

	route := "/.completions"

	hs := mocks.NewHandlerService(t)
	httpHandler := setupTestHttpHandler(hs)

	t.Run("no parameters", func(t *testing.T) {
		hs.On("GetCompletions", mock.Anything, "", "").Return(nil, repos.ErrInvalidCompletionParams).Once()

		// when: calling the route
		rec := testutils.NewRequest(http.MethodGet, route).RunOnHandler(httpHandler)
		// then: it returns status 400
		assertResponse400(t, rec, route)
		// and then: the body is of correct type
		var response server.ErrorResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
	})

	t.Run("unknown completion kind", func(t *testing.T) {
		hs.On("GetCompletions", mock.Anything, "something", "").Return(nil, repos.ErrInvalidCompletionParams).Once()

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
		hs.On("GetCompletions", mock.Anything, "names", "").Return([]string{"abc", "def"}, nil).Once()

		// when: calling the route
		rr := fmt.Sprintf("%s?kind=names&toComplete=", route)
		rec := testutils.NewRequest(http.MethodGet, rr).RunOnHandler(httpHandler)
		// then: it returns status 200
		assert.Equal(t, http.StatusOK, rec.Code)
		// and then: the body is of correct type
		assert.Equal(t, MimeText, rec.Header().Get(HeaderContentType))
		assert.Equal(t, []byte("abc\ndef\n"), rec.Body.Bytes())
	})

	t.Run("with unknown error", func(t *testing.T) {
		hs.On("GetCompletions", mock.Anything, "names", "").Return(nil, unknownErr).Once()
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

func assertResponse409(t *testing.T, rec *httptest.ResponseRecorder, route string, idErr *repos.ErrTMIDConflict) {
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
	assert.Equal(t, ref.Name, entry.Name)
	assert.Equal(t, ref.Author.Name, entry.SchemaAuthor.SchemaName)
	assert.Equal(t, ref.Manufacturer.Name, entry.SchemaManufacturer.SchemaName)
	assert.Equal(t, ref.Mpn, entry.SchemaMpn)
	assert.True(t, strings.HasSuffix(entry.Links.Self, "./inventory/"+ref.Name))

	assert.Equal(t, len(ref.Versions), len(entry.Versions))
	assertInventoryEntryVersions(t, ref.Versions, entry.Versions)
}

func assertInventoryEntryVersions(t *testing.T, ref []model.FoundVersion, versions []server.InventoryEntryVersion) {
	for idx, refVer := range ref {
		entryVer := versions[idx]

		assert.Equal(t, refVer.Description, entryVer.Description)
		assert.Equal(t, refVer.Version.Model, entryVer.Version.Model)
		assert.True(t, strings.HasSuffix(entryVer.Links.Content, "/thing-models/"+refVer.TMID))
		assert.Equal(t, refVer.TMID, entryVer.TmID)
		assert.Equal(t, refVer.Digest, entryVer.Digest)
		assert.Equal(t, refVer.TimeStamp, entryVer.Timestamp)
		assert.Equal(t, refVer.ExternalID, entryVer.ExternalID)
	}
}

var (
	listResult1 = model.SearchResult{
		Entries: []model.FoundEntry{
			{
				Name:         "a-corp/eagle/BT2000",
				Author:       model.SchemaAuthor{Name: "a-corp"},
				Manufacturer: model.SchemaManufacturer{Name: "eagle"},
				Mpn:          "BT2000",
				Versions: []model.FoundVersion{
					{
						IndexVersion: model.IndexVersion{
							TMID:        "a-corp/eagle/BT2000/v1.0.0-20240108140117-243d1b462ccc.tm.json",
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
							TMID:        "a-corp/eagle/BT2000/v1.0.0-20231231153548-243d1b462ddd.tm.json",
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
				Name:         "b-corp/frog/BT3000",
				Author:       model.SchemaAuthor{Name: "b-corp"},
				Manufacturer: model.SchemaManufacturer{Name: "frog"},
				Mpn:          "BT3000",
				Versions: []model.FoundVersion{
					{
						IndexVersion: model.IndexVersion{
							TMID:        "b-corp/frog/BT3000/v1.0.0-20240108140117-743d1b462uuu.tm.json",
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
				Versions: []model.FoundVersion{
					{
						IndexVersion: model.IndexVersion{
							TMID:        "b-corp/eagle/PM20/v1.0.0-20240107123001-234d1b462fff.tm.json",
							Description: "desc version v1.0.0",
							Version:     model.Version{Model: "1.0.0"},
							Digest:      "234d1b462fff",
							TimeStamp:   "20240107123001",
							ExternalID:  "ext-4",
						},
						FoundIn: model.FoundSource{RepoName: "r2"},
					},
				},
			},
		},
	}
)
