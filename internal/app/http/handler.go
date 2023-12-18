package http

import (
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

// //go:generate oapi-codegen -package http -generate types -o models.gen.go ../../../api/tm-catalog.openapi.yaml
// //go:generate oapi-codegen -package http -generate gorilla-server -o server.gen.go ../../../api/tm-catalog.openapi.yaml

type TmcHandler struct {
}

func NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(handleNoRoute)
	return r
}

func handleNoRoute(w http.ResponseWriter, r *http.Request) {
	HandleErrorResponse(w, r, NewNotFoundError(nil, "Path not handled by Thing Model Catalog"))
}

func NewTmcHandler() *TmcHandler {
	return &TmcHandler{}
}

func (h *TmcHandler) GetInventory(w http.ResponseWriter, r *http.Request, params GetInventoryParams) {
	searchParams := convertParams(params)

	toc, err := listToc(searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toInventoryResponse(*toc)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetInventoryByName Get an inventory entry by inventory name
// (GET /inventory/{name})
func (h *TmcHandler) GetInventoryByName(w http.ResponseWriter, r *http.Request, name string) {

	tocEntry, err := findTocEntry(name)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toInventoryEntryResponse(*tocEntry)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetInventoryVersionsByName Get the versions of an inventory entry
// (GET /inventory/{inventoryId}/versions)
func (h *TmcHandler) GetInventoryVersionsByName(w http.ResponseWriter, r *http.Request, name string) {
	tocEntry, err := findTocEntry(name)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toInventoryEntryVersionsResponse(tocEntry.Versions)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetThingModelById Get the content of a Thing Model by its ID
// (GET /thing-models/{tmID})
func (h *TmcHandler) GetThingModelById(w http.ResponseWriter, r *http.Request, tmID string) {
	data, err := fetchThingModel(tmID)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	HandleByteResponse(w, r, http.StatusOK, mimeJSON, data)
}

func (h *TmcHandler) PushThingModel(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get(headerContentType)

	if contentType != mimeJSON {
		HandleErrorResponse(w, r, NewBadRequestError(nil, "Invalid Content-Type header: %s", contentType))
		return
	}

	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	err = r.Body.Close()

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}
	tmID, err := pushThingModel(b)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toPushThingModelResponse(tmID)

	HandleJsonResponse(w, r, 201, resp)
}

func (h *TmcHandler) GetAuthors(w http.ResponseWriter, r *http.Request, params GetAuthorsParams) {
	searchParams := convertParams(params)

	toc, err := listToc(searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	authors := listTocAuthors(toc)

	resp := toAuthorsResponse(authors)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

func (h *TmcHandler) GetManufacturers(w http.ResponseWriter, r *http.Request, params GetManufacturersParams) {
	searchParams := convertParams(params)

	toc, err := listToc(searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	mans := listTocManufacturers(toc)

	resp := toManufacturersResponse(mans)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

func (h *TmcHandler) GetMpns(w http.ResponseWriter, r *http.Request, params GetMpnsParams) {
	searchParams := convertParams(params)

	toc, err := listToc(searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	mpns := listTocMpns(toc)

	resp := toMpnsResponse(mpns)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}
