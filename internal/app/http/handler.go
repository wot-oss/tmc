package http

import (
	"github.com/gorilla/mux"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"net/http"
)

// //go:generate oapi-codegen -package http -generate types -o models.gen.go ../../../api/tm-catalog.openapi.yaml
// //go:generate oapi-codegen -package http -generate gorilla-server -o server.gen.go ../../../api/tm-catalog.openapi.yaml

type TmcHandler struct {
}

func NewTmcHandler() *TmcHandler {
	return &TmcHandler{}
}

func (h *TmcHandler) GetInventory(w http.ResponseWriter, r *http.Request, params GetInventoryParams) {
	remote, err := remotes.Get("")
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}
	//todo: filter
	toc, err := remote.List("")
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	filterParams := toFilterParams(params)
	Filter(&toc, filterParams)

	searchParams := toSearchParams(params)
	Search(&toc, searchParams)

	resp := toInventoryResponse(toc)

	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetInventoryById Get an inventory entry by inventory ID
// (GET /inventory/{inventoryId})
func (h *TmcHandler) GetInventoryById(w http.ResponseWriter, r *http.Request, inventoryId string) {
	//todo: check if iventoryId is valid format

	remote, err := remotes.Get("")
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	toc, err := remote.List("")
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	tocEntry, ok := toc.Contents[inventoryId]
	if !ok {
		HandleErrorResponse(w, r, NewNotFoundError(nil, "Inventory with Id %s not found", inventoryId))
		return
	}

	resp := toInventoryEntryResponse(inventoryId, tocEntry)

	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetInventoryVersionsById Get the versions of an inventory entry
// (GET /inventory/{inventoryId}/versions)
func (h *TmcHandler) GetInventoryVersionsById(w http.ResponseWriter, r *http.Request, inventoryId string) {
	//todo: check if iventoryId is valid format

	remote, err := remotes.Get("")
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	toc, err := remote.List("")
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	tocEntry, ok := toc.Contents[inventoryId]
	if !ok {
		HandleErrorResponse(w, r, NewNotFoundError(nil, "Inventory with Id %s not found", inventoryId))
		return
	}

	resp := toInventoryEntryVersionsResponse(tocEntry.Versions)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetThingModelById Get the content of a Thing Model by it's ID
// (GET /thing-models/{tmId})
func (h *TmcHandler) GetThingModelById(w http.ResponseWriter, r *http.Request, tmId string) {
	remote, err := remotes.Get("")
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	mTmId, err := model.ParseTMID(tmId, false)
	if err == model.ErrInvalidId {
		HandleErrorResponse(w, r, NewBadRequestError(err, "Invalid parameter: %s", tmId))
		return
	} else if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	data, err := remote.Fetch(mTmId)
	if err != nil && err.Error() == "file does not exist" {
		HandleErrorResponse(w, r, NewNotFoundError(err, "File does not exists"))
		return
	} else if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	HandleByteResponse(w, r, http.StatusOK, mimeJSON, data)
}

func handleNoRoute(w http.ResponseWriter, r *http.Request) {
	HandleErrorResponse(w, r, NewNotFoundError(nil, "Path not handled by Thing Model Catalog"))
}

func NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(handleNoRoute)
	return r
}
