package http

import (
	"github.com/gorilla/mux"
	"net/http"
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
	filterParams, searchParams := convertParams(params)

	toc, err := listToc(filterParams, searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toInventoryResponse(*toc)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetInventoryById Get an inventory entry by inventory ID
// (GET /inventory/{inventoryId})
func (h *TmcHandler) GetInventoryById(w http.ResponseWriter, r *http.Request, inventoryId string) {

	tocEntry, err := findTocEntry(inventoryId)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toInventoryEntryResponse(inventoryId, *tocEntry)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetInventoryVersionsById Get the versions of an inventory entry
// (GET /inventory/{inventoryId}/versions)
func (h *TmcHandler) GetInventoryVersionsById(w http.ResponseWriter, r *http.Request, inventoryId string) {
	tocEntry, err := findTocEntry(inventoryId)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toInventoryEntryVersionsResponse(tocEntry.Versions)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetThingModelById Get the content of a Thing Model by it's ID
// (GET /thing-models/{tmId})
func (h *TmcHandler) GetThingModelById(w http.ResponseWriter, r *http.Request, tmId string) {
	data, err := fetchThingModel(tmId)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	HandleByteResponse(w, r, http.StatusOK, mimeJSON, data)
}

func (h *TmcHandler) GetAuthors(w http.ResponseWriter, r *http.Request, params GetAuthorsParams) {
	filterParams, searchParams := convertParams(params)

	toc, err := listToc(filterParams, searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	authors := listTocAuthors(toc)

	resp := toAuthorsResponse(authors)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

func (h *TmcHandler) GetManufacturers(w http.ResponseWriter, r *http.Request, params GetManufacturersParams) {
	filterParams, searchParams := convertParams(params)

	toc, err := listToc(filterParams, searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	mans := listTocManufacturers(toc)

	resp := toManufacturersResponse(mans)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

func (h *TmcHandler) GetMpns(w http.ResponseWriter, r *http.Request, params GetMpnsParams) {
	filterParams, searchParams := convertParams(params)

	toc, err := listToc(filterParams, searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	mpns := listTocMpns(toc)

	resp := toMpnsResponse(mpns)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}
