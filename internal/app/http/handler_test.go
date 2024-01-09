package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"

	"github.com/gorilla/mux"
	"github.com/oapi-codegen/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
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

const pushRemote = "someRemote"

func setupTestRouter(rm remotes.RemoteManager, pushRemote string) *mux.Router {

	handler := NewTmcHandler(
		TmcHandlerOptions{
			UrlContextRoot: "",
			RemoteManager:  rm,
			PushRemote:     pushRemote,
		})

	r := NewRouter()
	options := GorillaServerOptions{
		BaseRouter:       r,
		ErrorHandlerFunc: HandleErrorResponse,
	}

	HandlerWithOptions(handler, options)

	return r
}

func Test_healthLive(t *testing.T) {

	route := "/healthz/live"

	t.Run("with set RemoteManager", func(t *testing.T) {
		// given: a RemoteManager is set
		rm := remotes.NewMockRemoteManager(t)
		router := setupTestRouter(rm, pushRemote)
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns 204 status and empty body
		assertHealthyResponse204(t, rec)
	})

	t.Run("with unset RemoteManager", func(t *testing.T) {
		// given: a RemoteManager is unset
		router := setupTestRouter(nil, pushRemote)
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: unset RemoteManager does not matter, it returns 204 status and empty body
		assertHealthyResponse204(t, rec)
	})
}

func Test_healthReady(t *testing.T) {

	route := "/healthz/ready"

	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)

	router := setupTestRouter(rm, pushRemote)

	t.Run("with valid remote", func(t *testing.T) {
		// given: the remote can be found by the remote manager
		rm.On("Get", pushRemote).Return(r, nil).Once()
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns 204 status and empty body
		assertHealthyResponse204(t, rec)
	})

	t.Run("with invalid remote", func(t *testing.T) {
		// given: the remote cannot be found by the remote manager
		rm.On("Get", pushRemote).Return(nil, errors.New("invalid remote name")).Once()
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns 503 status and json error as body
		assertResponse503(t, rec, route)
	})
}

func Test_healthStartup(t *testing.T) {

	route := "/healthz/startup"

	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)

	router := setupTestRouter(rm, pushRemote)

	t.Run("with valid remote", func(t *testing.T) {
		// given: the remote can be found by the remote manager
		rm.On("Get", pushRemote).Return(r, nil).Once()
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns 204 status and empty body
		assertHealthyResponse204(t, rec)
	})

	t.Run("with invalid remote", func(t *testing.T) {
		// given: the remote cannot be found by the remote manager
		rm.On("Get", pushRemote).Return(nil, errors.New("invalid remote name")).Once()
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns 503 status and json error as body
		assertResponse503(t, rec, route)
	})
}

func Test_health(t *testing.T) {

	route := "/healthz"

	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)

	router := setupTestRouter(rm, pushRemote)

	t.Run("with valid remote", func(t *testing.T) {
		// given: the remote can be found by the remote manager
		rm.On("Get", pushRemote).Return(r, nil).Once()
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns 204 status and empty body
		assertHealthyResponse204(t, rec)
	})

	t.Run("with invalid remote", func(t *testing.T) {
		// given: the remote cannot be found by the remote manager
		rm.On("Get", pushRemote).Return(nil, errors.New("invalid remote name")).Once()
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns 503 status and json error as body
		assertResponse503(t, rec, route)
	})
}

func Test_Inventory(t *testing.T) {

	route := "/inventory"

	rm := remotes.NewMockRemoteManager(t)
	router := setupTestRouter(rm, pushRemote)

	t.Run("list all", func(t *testing.T) {
		// given: 2 remotes having some inventory entries
		r1 := remotes.NewMockRemote(t)
		r1.On("List", &model.SearchParams{}).Return(listResult1, nil).Once()
		r2 := remotes.NewMockRemote(t)
		r2.On("List", &model.SearchParams{}).Return(listResult2, nil).Once()

		rm.On("All").Return([]remotes.Remote{r1, r2}, nil).Once()

		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response InventoryResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: the entries are ordered ascending by name and have all data set
		assert.Equal(t, 3, len(response.Data))
		assertInventoryEntry(t, listResult1.Entries[0], response.Data[0])
		assertInventoryEntry(t, listResult2.Entries[0], response.Data[1])
		assertInventoryEntry(t, listResult1.Entries[1], response.Data[2])
	})

	t.Run("list with filter and search parameter", func(t *testing.T) {
		// given: the route with filter and search parameters
		fAuthors := []string{"a1", "a2"}
		fMan := []string{"man1", "man2"}
		fMpn := []string{"mpn1", "mpn2"}
		fExtID := []string{"ext1", "ext2"}
		search := "foo"

		filterRoute := fmt.Sprintf("%s?filter.author=%s&filter.manufacturer=%s&filter.mpn=%s&filter.externalID=%s&search=%s",
			route, strings.Join(fAuthors, ","), strings.Join(fMan, ","), strings.Join(fMpn, ","), strings.Join(fExtID, ","), search)

		// and given: remotes where the list command is expected to be called with the converted search parameters
		expectedSearchParams := &model.SearchParams{
			Author:       fAuthors,
			Manufacturer: fMan,
			Mpn:          fMpn,
			ExternalID:   fExtID,
			Query:        search,
		}

		r1 := remotes.NewMockRemote(t)
		r1.On("List", expectedSearchParams).Return(listResult1, nil).Once()
		r2 := remotes.NewMockRemote(t)
		r2.On("List", expectedSearchParams).Return(listResult2, nil).Once()
		rm.On("All").Return([]remotes.Remote{r1, r2}, nil).Once()

		// when: calling the route
		rec := testutil.NewRequest().Get(filterRoute).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 200
		assertResponse200(t, rec)
	})

	t.Run("with unknown error", func(t *testing.T) {
		// given: remote manager that returns an error
		rm.On("All").Return(nil, errors.New("an error of unknown type")).Once()
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, route)
	})
}

func Test_InventoryByName(t *testing.T) {
	mockListResult := listResult2
	mockInventoryEntry := mockListResult.Entries[0]

	inventoryName := mockInventoryEntry.Name

	route := "/inventory/" + inventoryName

	rm := remotes.NewMockRemoteManager(t)
	router := setupTestRouter(rm, pushRemote)

	t.Run("with valid remotes", func(t *testing.T) {
		// given: remote having some inventory entries
		r := remotes.NewMockRemote(t)
		r.On("List", &model.SearchParams{Name: inventoryName}).Return(mockListResult, nil).Once()
		rm.On("All").Return([]remotes.Remote{r}, nil).Once()

		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response InventoryEntryResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: result has all data set
		assertInventoryEntry(t, mockInventoryEntry, response.Data)
	})

	t.Run("with unknown error", func(t *testing.T) {
		// given: remote manager that returns an error
		rm.On("All").Return(nil, errors.New("an error of unknown type")).Once()
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, route)
	})
}

func Test_InventoryEntryVersionsByName(t *testing.T) {
	mockListResult := listResult2
	mockInventoryEntry := mockListResult.Entries[0]

	inventoryName := mockInventoryEntry.Name

	route := "/inventory/" + inventoryName + "/versions"

	rm := remotes.NewMockRemoteManager(t)
	router := setupTestRouter(rm, pushRemote)

	t.Run("with valid remotes", func(t *testing.T) {
		// given: remote having some inventory entries
		r := remotes.NewMockRemote(t)
		r.On("List", &model.SearchParams{Name: inventoryName}).Return(mockListResult, nil).Once()
		rm.On("All").Return([]remotes.Remote{r}, nil).Once()

		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response InventoryEntryVersionsResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: result has all data set
		assertInventoryEntryVersions(t, mockInventoryEntry.Versions, response.Data)
	})

	t.Run("with unknown error", func(t *testing.T) {
		// given: remote manager that returns an error
		rm.On("All").Return(nil, errors.New("an error of unknown type")).Once()
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, route)
	})
}

func Test_Authors(t *testing.T) {

	route := "/authors"

	rm := remotes.NewMockRemoteManager(t)
	router := setupTestRouter(rm, pushRemote)

	t.Run("with valid remotes", func(t *testing.T) {
		// given: 2 remotes having some inventory entries
		r1 := remotes.NewMockRemote(t)
		r1.On("List", &model.SearchParams{}).Return(listResult1, nil).Once()
		r2 := remotes.NewMockRemote(t)
		r2.On("List", &model.SearchParams{}).Return(listResult2, nil).Once()

		rm.On("All").Return([]remotes.Remote{r1, r2}, nil).Once()

		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response AuthorsResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: duplicates are removed
		assert.Equal(t, 2, len(response.Data))
		// and then result are ordered ascending by name
		assert.Equal(t, []string{"a-corp", "b-corp"}, response.Data)
	})

	t.Run("with unknown error", func(t *testing.T) {
		// given: remote manager that returns an error
		rm.On("All").Return(nil, errors.New("an error of unknown type")).Once()
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, route)
	})

	t.Run("with filter and search parameter", func(t *testing.T) {
		// given: the route with filter and search parameters
		fMan := []string{"man1", "man2"}
		fMpn := []string{"mpn1", "mpn2"}
		fExtID := []string{"ext1", "ext2"}
		search := "foo"

		filterRoute := fmt.Sprintf("%s?filter.manufacturer=%s&filter.mpn=%s&filter.externalID=%s&search=%s",
			route, strings.Join(fMan, ","), strings.Join(fMpn, ","), strings.Join(fExtID, ","), search)

		// and given: remotes where the list command is expected to be called with the converted search parameters
		expectedSearchParams := &model.SearchParams{
			Manufacturer: fMan,
			Mpn:          fMpn,
			ExternalID:   fExtID,
			Query:        search,
		}

		r1 := remotes.NewMockRemote(t)
		r1.On("List", expectedSearchParams).Return(listResult1, nil).Once()
		r2 := remotes.NewMockRemote(t)
		r2.On("List", expectedSearchParams).Return(listResult2, nil).Once()

		rm.On("All").Return([]remotes.Remote{r1, r2}, nil).Once()

		// when: calling the route
		rec := testutil.NewRequest().Get(filterRoute).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 200
		assertResponse200(t, rec)
	})
}

func Test_Manufacturers(t *testing.T) {

	route := "/manufacturers"

	rm := remotes.NewMockRemoteManager(t)
	router := setupTestRouter(rm, pushRemote)

	t.Run("with valid remotes", func(t *testing.T) {
		// given: 2 remotes having some inventory entries
		r1 := remotes.NewMockRemote(t)
		r1.On("List", &model.SearchParams{}).Return(listResult1, nil).Once()
		r2 := remotes.NewMockRemote(t)
		r2.On("List", &model.SearchParams{}).Return(listResult2, nil).Once()

		rm.On("All").Return([]remotes.Remote{r1, r2}, nil).Once()

		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response ManufacturersResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: duplicates are removed
		assert.Equal(t, 2, len(response.Data))
		// and then result are ordered ascending by name
		assert.Equal(t, []string{"eagle", "frog"}, response.Data)
	})

	t.Run("with unknown error", func(t *testing.T) {
		// given: remote manager that returns an error
		rm.On("All").Return(nil, errors.New("an error of unknown type")).Once()
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, route)
	})

	t.Run("with filter and search parameter", func(t *testing.T) {
		// given: the route with filter and search parameters
		fAuthors := []string{"a1", "a2"}
		fMpn := []string{"mpn1", "mpn2"}
		fExtID := []string{"ext1", "ext2"}
		search := "foo"

		filterRoute := fmt.Sprintf("%s?filter.author=%s&filter.mpn=%s&filter.externalID=%s&search=%s",
			route, strings.Join(fAuthors, ","), strings.Join(fMpn, ","), strings.Join(fExtID, ","), search)

		// and given: remotes where the list command is expected to be called with the converted search parameters
		expectedSearchParams := &model.SearchParams{
			Author:     fAuthors,
			Mpn:        fMpn,
			ExternalID: fExtID,
			Query:      search,
		}

		r1 := remotes.NewMockRemote(t)
		r1.On("List", expectedSearchParams).Return(listResult1, nil).Once()
		r2 := remotes.NewMockRemote(t)
		r2.On("List", expectedSearchParams).Return(listResult2, nil).Once()

		rm.On("All").Return([]remotes.Remote{r1, r2}, nil).Once()

		// when: calling the route
		rec := testutil.NewRequest().Get(filterRoute).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 200
		assertResponse200(t, rec)
	})
}

func Test_Mpns(t *testing.T) {

	route := "/mpns"

	rm := remotes.NewMockRemoteManager(t)
	router := setupTestRouter(rm, pushRemote)

	t.Run("with valid remotes", func(t *testing.T) {
		// given: 2 remotes having some inventory entries
		r1 := remotes.NewMockRemote(t)
		r1.On("List", &model.SearchParams{}).Return(listResult1, nil).Once()
		r2 := remotes.NewMockRemote(t)
		r2.On("List", &model.SearchParams{}).Return(listResult2, nil).Once()

		rm.On("All").Return([]remotes.Remote{r1, r2}, nil).Once()

		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 200
		assertResponse200(t, rec)
		// and then: the body is of correct type
		var response MpnsResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: duplicates are removed
		assert.Equal(t, 3, len(response.Data))
		// and then result are ordered ascending by name
		assert.Equal(t, []string{"BT2000", "BT3000", "PM20"}, response.Data)
	})

	t.Run("with unknown error", func(t *testing.T) {
		// given: remote manager that returns an error
		rm.On("All").Return(nil, errors.New("an error of unknown type")).Once()
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 500 and json error as body
		assertResponse500(t, rec, route)
	})

	t.Run("with filter and search parameter", func(t *testing.T) {
		// given: the route with filter and search parameters
		fAuthors := []string{"a1", "a2"}
		fMan := []string{"man1", "man2"}
		fExtID := []string{"ext1", "ext2"}
		search := "foo"

		filterRoute := fmt.Sprintf("%s?filter.author=%s&filter.manufacturer=%s&filter.externalID=%s&search=%s",
			route, strings.Join(fAuthors, ","), strings.Join(fMan, ","), strings.Join(fExtID, ","), search)

		// and given: remotes where the list command is expected to be called with the converted search parameters
		expectedSearchParams := &model.SearchParams{
			Author:       fAuthors,
			Manufacturer: fMan,
			ExternalID:   fExtID,
			Query:        search,
		}

		r1 := remotes.NewMockRemote(t)
		r1.On("List", expectedSearchParams).Return(listResult1, nil).Once()
		r2 := remotes.NewMockRemote(t)
		r2.On("List", expectedSearchParams).Return(listResult2, nil).Once()

		rm.On("All").Return([]remotes.Remote{r1, r2}, nil).Once()

		// when: calling the route
		rec := testutil.NewRequest().Get(filterRoute).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 200
		assertResponse200(t, rec)
	})
}

func Test_ThingModels(t *testing.T) {
	tmID := listResult2.Entries[0].Versions[0].TMID
	tmContent := []byte("this is the content of a ThingModel")

	route := "/thing-models/" + tmID

	rm := remotes.NewMockRemoteManager(t)
	router := setupTestRouter(rm, pushRemote)
	r := remotes.NewMockRemote(t)

	t.Run("with valid remotes", func(t *testing.T) {
		// given: remote having some inventory entries
		rm.On("Get", "").Return(r, nil).Once()
		r.On("Fetch", tmID).Return(tmID, tmContent, nil).Once()
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 200
		assertResponse200(t, rec)
		assert.Equal(t, tmContent, rec.Body.Bytes())
	})

	t.Run("with invalid tmID", func(t *testing.T) {
		// given: route with invalid tmID
		invalidRoute := "/thing-models/some-invalid-tm-id"
		// when: calling the route
		rec := testutil.NewRequest().Get(invalidRoute).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 400 and json error as body
		assertResponse400(t, rec, invalidRoute)
	})

	t.Run("with error", func(t *testing.T) {
		// given: remote that returns an error
		rm.On("Get", "").Return(r, nil).Once()
		r.On("Fetch", tmID).Return(tmID, nil, errors.New("an error of unknown type")).Once()
		// when: calling the route
		rec := testutil.NewRequest().Get(route).GoWithHTTPHandler(t, router).Recorder
		// then: it returns status 404 and json error as body
		assertResponse404(t, rec, route)
	})
}

func Test_PushThingModel(t *testing.T) {

	_, tmContent, err := utils.ReadRequiredFile("../../../test/data/push/omnilamp-versioned.json")
	assert.NoError(t, err)

	route := "/thing-models"

	rm := remotes.NewMockRemoteManager(t)
	router := setupTestRouter(rm, pushRemote)
	r := remotes.NewMockRemote(t)

	t.Run("with success", func(t *testing.T) {
		// given: remote where to push
		rm.On("Get", pushRemote).Return(r, nil).Once()
		r.On("CreateToC").Return(nil).Once()
		r.On("Push", mock.AnythingOfType("model.TMID"), mock.AnythingOfType("[]uint8")).Return(nil).Once()
		// when: calling the route
		rec := testutil.NewRequest().Post(route).
			WithHeader(headerContentType, mimeJSON).
			WithBody(tmContent).GoWithHTTPHandler(t, router).
			Recorder
		// then: it returns status 201
		assertResponse201(t, rec)
		// and then: the body is of correct type
		var response PushThingModelResponse
		assertUnmarshalResponse(t, rec.Body.Bytes(), &response)
		// and then: tmID is set in response
		assert.NotNil(t, response.Data.TmID)
		_, err := model.ParseTMID(response.Data.TmID, true)
		assert.NoError(t, err)
	})

	t.Run("with missing or wrong Content-Type", func(t *testing.T) {
		contentTypes := []string{"", "application/pdf", "application/xml"}

		for _, c := range contentTypes {
			rec := testutil.NewRequest().Post(route).
				WithHeader(headerContentType, c).
				WithBody(tmContent).GoWithHTTPHandler(t, router).
				Recorder
			// then: it returns status 400
			assertResponse400(t, rec, route)
		}
	})

	t.Run("with validation error", func(t *testing.T) {
		// given: remote where to push
		rm.On("Get", pushRemote).Return(r, nil).Once()
		// and given: some invalid ThingModel
		invalidContent := []byte("some invalid ThingModel")
		// when: calling the route
		rec := testutil.NewRequest().Post(route).
			WithHeader(headerContentType, mimeJSON).
			WithBody(invalidContent).GoWithHTTPHandler(t, router).
			Recorder
		// then: it returns status 400
		assertResponse400(t, rec, route)
	})

	t.Run("with unknown error", func(t *testing.T) {
		// given: remote where to push
		rm.On("Get", pushRemote).Return(nil, errors.New("an error of unknown type")).Once()
		// and given: some invalid ThingModel
		invalidContent := []byte("some invalid ThingModel")
		// when: calling the route
		rec := testutil.NewRequest().Post(route).
			WithHeader(headerContentType, mimeJSON).
			WithBody(invalidContent).GoWithHTTPHandler(t, router).
			Recorder
		// then: it returns status 500
		assertResponse500(t, rec, route)
	})
}

func assertHealthyResponse204(t *testing.T, rec *httptest.ResponseRecorder) {
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, 0, rec.Body.Len())
	assert.Equal(t, noCache, rec.Header().Get(headerCacheControl))
}

func assertResponse200(t *testing.T, rec *httptest.ResponseRecorder) {
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, mimeJSON, rec.Header().Get(headerContentType))
}

func assertResponse201(t *testing.T, rec *httptest.ResponseRecorder) {
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, mimeJSON, rec.Header().Get(headerContentType))
}

func assertResponse400(t *testing.T, rec *httptest.ResponseRecorder, route string) {
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var errResponse ErrorResponse
	assertUnmarshalResponse(t, rec.Body.Bytes(), &errResponse)
	assert.Equal(t, http.StatusBadRequest, errResponse.Status)
	assert.Equal(t, route, *errResponse.Instance)
	assert.Equal(t, error400Title, errResponse.Title)

	assert.Equal(t, mimeProblemJSON, rec.Header().Get(headerContentType))
	assert.Equal(t, noSniff, rec.Header().Get(headerXContentTypeOptions))
}

func assertResponse404(t *testing.T, rec *httptest.ResponseRecorder, route string) {
	assert.Equal(t, http.StatusNotFound, rec.Code)
	var errResponse ErrorResponse
	assertUnmarshalResponse(t, rec.Body.Bytes(), &errResponse)
	assert.Equal(t, http.StatusNotFound, errResponse.Status)
	assert.Equal(t, route, *errResponse.Instance)
	assert.Equal(t, error404Title, errResponse.Title)

	assert.Equal(t, mimeProblemJSON, rec.Header().Get(headerContentType))
	assert.Equal(t, noSniff, rec.Header().Get(headerXContentTypeOptions))
}

func assertResponse500(t *testing.T, rec *httptest.ResponseRecorder, route string) {
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	var errResponse ErrorResponse
	assertUnmarshalResponse(t, rec.Body.Bytes(), &errResponse)
	assert.Equal(t, http.StatusInternalServerError, errResponse.Status)
	assert.Equal(t, route, *errResponse.Instance)
	assert.Equal(t, error500Title, errResponse.Title)
	assert.Equal(t, error500Detail, *errResponse.Detail)

	assert.Equal(t, mimeProblemJSON, rec.Header().Get(headerContentType))
	assert.Equal(t, noSniff, rec.Header().Get(headerXContentTypeOptions))
}

func assertResponse503(t *testing.T, rec *httptest.ResponseRecorder, route string) {
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	var errResponse ErrorResponse
	assertUnmarshalResponse(t, rec.Body.Bytes(), &errResponse)
	assert.Equal(t, http.StatusServiceUnavailable, errResponse.Status)
	assert.Equal(t, route, *errResponse.Instance)
	assert.Equal(t, error503Title, errResponse.Title)

	assert.Equal(t, mimeProblemJSON, rec.Header().Get(headerContentType))
	assert.Equal(t, noSniff, rec.Header().Get(headerXContentTypeOptions))
}

func assertUnmarshalResponse(t *testing.T, data []byte, v any) {
	err := json.Unmarshal(data, v)
	assert.NoError(t, err, "error unmarshalling response")
}

func assertInventoryEntry(t *testing.T, ref model.FoundEntry, entry InventoryEntry) {
	assert.Equal(t, ref.Name, entry.Name)
	assert.Equal(t, ref.Author.Name, entry.SchemaAuthor.SchemaName)
	assert.Equal(t, ref.Manufacturer.Name, entry.SchemaManufacturer.SchemaName)
	assert.Equal(t, ref.Mpn, entry.SchemaMpn)
	assert.True(t, strings.HasSuffix(entry.Links.Self, "./inventory/"+ref.Name))

	assert.Equal(t, len(ref.Versions), len(entry.Versions))
	assertInventoryEntryVersions(t, ref.Versions, entry.Versions)
}

func assertInventoryEntryVersions(t *testing.T, ref []model.FoundVersion, versions []InventoryEntryVersion) {
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

	listResult2 = model.SearchResult{
		Entries: []model.FoundEntry{
			{
				Name:         "b-corp/eagle/PM20",
				Author:       model.SchemaAuthor{Name: "b-corp"},
				Manufacturer: model.SchemaManufacturer{Name: "eagle"},
				Mpn:          "PM20",
				Versions: []model.FoundVersion{
					{
						TOCVersion: model.TOCVersion{
							TMID:        "b-corp/eagle/PM20/v1.0.0-20240107123001-234d1b462fff.tm.json",
							Description: "desc version v1.0.0",
							Version:     model.Version{Model: "1.0.0"},
							Digest:      "234d1b462fff",
							TimeStamp:   "20240107123001",
							ExternalID:  "ext-4",
						},
						FoundIn: "r2",
					},
				},
			},
		},
	}
)
