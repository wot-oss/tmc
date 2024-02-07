package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/server"
)

type TmcHandler struct {
	Service HandlerService
	Options TmcHandlerOptions
}

type TmcHandlerOptions struct {
	UrlContextRoot string
}

func NewTmcHandler(handlerService HandlerService, options TmcHandlerOptions) *TmcHandler {
	return &TmcHandler{
		Service: handlerService,
		Options: options,
	}
}

func (h *TmcHandler) GetInventory(w http.ResponseWriter, r *http.Request, params server.GetInventoryParams) {

	searchParams := convertParams(params)

	inv, err := h.Service.ListInventory(nil, searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	ctx := h.createContext(r)
	resp := toInventoryResponse(ctx, *inv)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetInventoryByName Get an inventory entry by inventory name
// (GET /inventory/{name})
func (h *TmcHandler) GetInventoryByName(w http.ResponseWriter, r *http.Request, name string) {

	entry, err := h.Service.FindInventoryEntry(nil, name)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	ctx := h.createContext(r)
	resp := toInventoryEntryResponse(ctx, *entry)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetInventoryVersionsByName Get the versions of an inventory entry
// (GET /inventory/{inventoryId}/versions)
func (h *TmcHandler) GetInventoryVersionsByName(w http.ResponseWriter, r *http.Request, name string) {

	entry, err := h.Service.FindInventoryEntry(nil, name)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	ctx := h.createContext(r)
	resp := toInventoryEntryVersionsResponse(ctx, entry.Versions)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetThingModelById Get the content of a Thing Model by its ID
// (GET /thing-models/{tmID})
func (h *TmcHandler) GetThingModelById(w http.ResponseWriter, r *http.Request, tmID string) {

	data, err := h.Service.FetchThingModel(nil, tmID)
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

	tmID, err := h.Service.PushThingModel(nil, b)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toPushThingModelResponse(tmID)

	HandleJsonResponse(w, r, http.StatusCreated, resp)
}

func (h *TmcHandler) GetAuthors(w http.ResponseWriter, r *http.Request, params server.GetAuthorsParams) {

	searchParams := convertParams(params)

	authors, err := h.Service.ListAuthors(nil, searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toAuthorsResponse(authors)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

func (h *TmcHandler) GetManufacturers(w http.ResponseWriter, r *http.Request, params server.GetManufacturersParams) {

	searchParams := convertParams(params)

	mans, err := h.Service.ListManufacturers(nil, searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toManufacturersResponse(mans)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

func (h *TmcHandler) GetMpns(w http.ResponseWriter, r *http.Request, params server.GetMpnsParams) {

	searchParams := convertParams(params)

	mpns, err := h.Service.ListMpns(nil, searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toMpnsResponse(mpns)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetHealth Get the overall health of the service
// (GET /healthz)
func (h *TmcHandler) GetHealth(w http.ResponseWriter, r *http.Request) {

	err := h.Service.CheckHealth(nil)
	if err != nil {
		HandleErrorResponse(w, r, NewServiceUnavailableError(err, err.Error()))
		return
	}
	HandleHealthyResponse(w, r)
}

// GetHealthLive Returns the liveness of the service
// (GET /healthz/live)
func (h *TmcHandler) GetHealthLive(w http.ResponseWriter, r *http.Request) {

	err := h.Service.CheckHealthLive(nil)
	if err != nil {
		HandleErrorResponse(w, r, NewServiceUnavailableError(err, err.Error()))
		return
	}
	HandleHealthyResponse(w, r)
}

// GetHealthReady Returns the readiness of the service
// (GET /healthz/ready)
func (h *TmcHandler) GetHealthReady(w http.ResponseWriter, r *http.Request) {

	err := h.Service.CheckHealthReady(nil)
	if err != nil {
		HandleErrorResponse(w, r, NewServiceUnavailableError(err, err.Error()))
		return
	}
	HandleHealthyResponse(w, r)
}

// GetHealthStartup Returns whether the service is initialized
// (GET /healthz/startup)
func (h *TmcHandler) GetHealthStartup(w http.ResponseWriter, r *http.Request) {

	err := h.Service.CheckHealthStartup(nil)
	if err != nil {
		HandleErrorResponse(w, r, NewServiceUnavailableError(err, err.Error()))
		return
	}
	HandleHealthyResponse(w, r)
}

func (h *TmcHandler) GetCompletions(w http.ResponseWriter, r *http.Request, params server.GetCompletionsParams) {
	kind := ""
	if params.Kind != nil {
		kind = string(*params.Kind)
	}
	toComplete := ""
	if params.ToComplete != nil {
		toComplete = *params.ToComplete
	}
	vals, err := h.Service.GetCompletions(context.TODO(), kind, toComplete)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}
	buf := bytes.NewBuffer(nil)
	for _, line := range vals {
		_, err := fmt.Fprintf(buf, "%s\n", line)
		if err != nil {
			HandleErrorResponse(w, r, err)
			return
		}
	}

	HandleByteResponse(w, r, http.StatusOK, mimeText, buf.Bytes())
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
		return -1
	}

	path = path[idx:]
	d := strings.Count(path, "/")
	return d
}
