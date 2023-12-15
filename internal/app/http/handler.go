package http

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// //go:generate oapi-codegen -package http -generate types -o models.gen.go ../../../api/tm-catalog.openapi.yaml
// //go:generate oapi-codegen -package http -generate gorilla-server -o server.gen.go ../../../api/tm-catalog.openapi.yaml

type TmcHandler struct {
	Options TmcHandlerOptions
}

type TmcHandlerOptions struct {
	UrlContextRoot string
}

func NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(handleNoRoute)
	return r
}

func handleNoRoute(w http.ResponseWriter, r *http.Request) {
	HandleErrorResponse(w, r, NewNotFoundError(nil, "Path not handled by Thing Model Catalog"))
}

func NewTmcHandler(options TmcHandlerOptions) *TmcHandler {
	return &TmcHandler{
		Options: options,
	}
}

func (h *TmcHandler) GetInventory(w http.ResponseWriter, r *http.Request, params GetInventoryParams) {
	ctx := h.createContext(r)

	filterParams, searchParams := convertParams(params)

	toc, err := listToc(filterParams, searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toInventoryResponse(ctx, *toc)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetInventoryByName Get an inventory entry by inventory name
// (GET /inventory/{name})
func (h *TmcHandler) GetInventoryByName(w http.ResponseWriter, r *http.Request, name string) {
	ctx := h.createContext(r)

	tocEntry, err := findTocEntry(name)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toInventoryEntryResponse(ctx, *tocEntry)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetInventoryVersionsById Get the versions of an inventory entry
// (GET /inventory/{inventoryId}/versions)
func (h *TmcHandler) GetInventoryVersionsByName(w http.ResponseWriter, r *http.Request, name string) {
	ctx := h.createContext(r)

	tocEntry, err := findTocEntry(name)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toInventoryEntryVersionsResponse(ctx, tocEntry.Versions)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetThingModelById Get the content of a Thing Model by it's ID
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

	resp := toPushThingModelResponse(*tmID)

	HandleJsonResponse(w, r, 201, resp)
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

func (h *TmcHandler) createContext(r *http.Request) context.Context {
	relPathDepth := getRelativeDepth(r.URL.Path, basePathInventory)

	ctx := r.Context()
	ctx = context.WithValue(ctx, ctxRelPathDepth, relPathDepth)
	ctx = context.WithValue(ctx, ctxUrlRoot, h.Options.UrlContextRoot)

	return ctx
}

func getRelativeDepth(path, siblingPath string) int {
	path = strings.TrimPrefix(path, "/")
	siblingPath = strings.TrimPrefix(siblingPath, "/")

	idx := strings.Index(path, siblingPath)
	if idx < 0 {
		return 0
	}

	path = path[idx:]
	d := strings.Count(path, "/") + 1
	return d
}
