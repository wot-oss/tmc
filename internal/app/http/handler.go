package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/wot-oss/tmc/internal/app/http/server"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
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

// Get the inventory of the catalog
// (GET /inventory)
func (h *TmcHandler) GetInventory(w http.ResponseWriter, r *http.Request, params server.GetInventoryParams) {

	searchParams := convertParams(params)

	inv, err := h.Service.ListInventory(r.Context(), searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	ctx := h.createContext(r)
	resp := toInventoryResponse(ctx, *inv)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetInventoryByName Get an inventory entry by inventory name
// (GET /inventory/.tmName/{tmName})
func (h *TmcHandler) GetInventoryByName(w http.ResponseWriter, r *http.Request, tmName string) {
	entry, err := h.Service.FindInventoryEntry(r.Context(), tmName)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	ctx := h.createContext(r)
	resp := toInventoryEntryResponse(ctx, *entry)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetInventoryByFetchName Get the metadata of the most recent TM version matching the name
// (GET /inventory/.latest/{fetchName})
func (h *TmcHandler) GetInventoryByFetchName(w http.ResponseWriter, r *http.Request, fetchName server.FetchName) {
	entry, err := h.Service.GetLatestTMMetadata(r.Context(), fetchName)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	ctx := h.createContext(r)
	resp := toInventoryEntryVersionResponse(ctx, *entry)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetThingModelByFetchName Get the content of a Thing Model by fetch name
// (GET /thing-models/.latest/{fetchName}
func (h *TmcHandler) GetThingModelByFetchName(w http.ResponseWriter, r *http.Request, fetchName server.FetchName, params server.GetThingModelByFetchNameParams) {
	restoreId := false
	if params.RestoreId != nil {
		restoreId = *params.RestoreId
	}

	data, err := h.Service.FetchLatestThingModel(r.Context(), fetchName, restoreId)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	HandleByteResponse(w, r, http.StatusOK, MimeJSON, data)
}

// GetInventoryById Get the metadata of a single TM by ID
// (GET /inventory/{tmID})
func (h *TmcHandler) GetInventoryByID(w http.ResponseWriter, r *http.Request, tmID server.TMID) {
	entry, err := h.Service.GetTMMetadata(r.Context(), tmID)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	ctx := h.createContext(r)
	resp := toInventoryEntryVersionResponse(ctx, *entry)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetThingModelById Get the content of a Thing Model by its ID
// (GET /thing-models/{id})
func (h *TmcHandler) GetThingModelById(w http.ResponseWriter, r *http.Request, id string, params server.GetThingModelByIdParams) {
	restoreId := false
	if params.RestoreId != nil {
		restoreId = *params.RestoreId
	}

	data, err := h.Service.FetchThingModel(r.Context(), id, restoreId)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	HandleByteResponse(w, r, http.StatusOK, MimeJSON, data)
}

// DeleteThingModelById Delete a Thing Model by ID
// (DELETE /thing-models/{id})
func (h *TmcHandler) DeleteThingModelById(w http.ResponseWriter, r *http.Request, tmID string, params server.DeleteThingModelByIdParams) {
	if params.Force != "true" {
		HandleErrorResponse(w, r, NewBadRequestError(nil, "invalid value of 'force' query parameter"))
		return
	}

	err := h.Service.DeleteThingModel(r.Context(), tmID)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	_, _ = w.Write(nil)
}

func (h *TmcHandler) PushThingModel(w http.ResponseWriter, r *http.Request, p server.PushThingModelParams) {
	contentType := r.Header.Get(HeaderContentType)

	if contentType != MimeJSON {
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

	opts := repos.PushOptions{}
	if p.Force != nil {
		parsedForce, err := strconv.ParseBool(*p.Force)
		opts.Force = parsedForce && err == nil
	}
	if p.OptPath != nil {
		opts.OptPath = *p.OptPath
	}

	res, err := h.Service.PushThingModel(r.Context(), b, opts)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toPushThingModelResponse(res)

	HandleJsonResponse(w, r, http.StatusCreated, resp)
}

func (h *TmcHandler) GetAuthors(w http.ResponseWriter, r *http.Request, params server.GetAuthorsParams) {

	searchParams := convertParams(params)

	authors, err := h.Service.ListAuthors(r.Context(), searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toAuthorsResponse(authors)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

func (h *TmcHandler) GetManufacturers(w http.ResponseWriter, r *http.Request, params server.GetManufacturersParams) {

	searchParams := convertParams(params)

	mans, err := h.Service.ListManufacturers(r.Context(), searchParams)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toManufacturersResponse(mans)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

func (h *TmcHandler) GetMpns(w http.ResponseWriter, r *http.Request, params server.GetMpnsParams) {

	searchParams := convertParams(params)

	mpns, err := h.Service.ListMpns(r.Context(), searchParams)

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

	err := h.Service.CheckHealth(r.Context())
	if err != nil {
		HandleErrorResponse(w, r, NewServiceUnavailableError(err, err.Error()))
		return
	}
	HandleHealthyResponse(w, r)
}

// GetHealthLive Returns the liveness of the service
// (GET /healthz/live)
func (h *TmcHandler) GetHealthLive(w http.ResponseWriter, r *http.Request) {

	err := h.Service.CheckHealthLive(r.Context())
	if err != nil {
		HandleErrorResponse(w, r, NewServiceUnavailableError(err, err.Error()))
		return
	}
	HandleHealthyResponse(w, r)
}

// GetHealthReady Returns the readiness of the service
// (GET /healthz/ready)
func (h *TmcHandler) GetHealthReady(w http.ResponseWriter, r *http.Request) {

	err := h.Service.CheckHealthReady(r.Context())
	if err != nil {
		HandleErrorResponse(w, r, NewServiceUnavailableError(err, err.Error()))
		return
	}
	HandleHealthyResponse(w, r)
}

// GetHealthStartup Returns whether the service is initialized
// (GET /healthz/startup)
func (h *TmcHandler) GetHealthStartup(w http.ResponseWriter, r *http.Request) {

	err := h.Service.CheckHealthStartup(r.Context())
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
	var args []string
	if params.Args != nil {
		args = *params.Args
	}
	vals, err := h.Service.GetCompletions(r.Context(), kind, args, toComplete)
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

	HandleByteResponse(w, r, http.StatusOK, MimeText, buf.Bytes())
}

func (h *TmcHandler) GetThingModelAttachmentByName(w http.ResponseWriter, r *http.Request, tmid, attachmentFileName string) {
	ref := model.NewTMIDAttachmentContainerRef(tmid)
	h.fetchAttachment(w, r, ref, attachmentFileName)
}
func (h *TmcHandler) GetTMNameAttachment(w http.ResponseWriter, r *http.Request, tmName server.TMName, attachmentFileName server.AttachmentFileName) {
	ref := model.NewTMNameAttachmentContainerRef(tmName)
	h.fetchAttachment(w, r, ref, attachmentFileName)
}

func (h *TmcHandler) fetchAttachment(w http.ResponseWriter, r *http.Request, ref model.AttachmentContainerRef, attachmentFileName string) {
	data, err := h.Service.FetchAttachment(r.Context(), ref, attachmentFileName)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}
	HandleByteResponse(w, r, http.StatusOK, MimeOctetStream, data)
}

func (h *TmcHandler) deleteAttachment(w http.ResponseWriter, r *http.Request, ref model.AttachmentContainerRef, attachmentFileName string) {
	err := h.Service.DeleteAttachment(r.Context(), ref, attachmentFileName)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	_, _ = w.Write(nil)
}

func (h *TmcHandler) DeleteThingModelAttachmentByName(w http.ResponseWriter, r *http.Request, tmID server.TMID, attachmentFileName string) {
	ref := model.NewTMIDAttachmentContainerRef(tmID)
	h.deleteAttachment(w, r, ref, attachmentFileName)
}

func (h *TmcHandler) DeleteTMNameAttachment(w http.ResponseWriter, r *http.Request, tmName server.TMName, attachmentFileName server.AttachmentFileName) {
	ref := model.NewTMNameAttachmentContainerRef(tmName)
	h.deleteAttachment(w, r, ref, attachmentFileName)
}

func (h *TmcHandler) PutThingModelAttachmentByName(w http.ResponseWriter, r *http.Request, tmID string, attachmentFileName string) {
	ref := model.NewTMIDAttachmentContainerRef(tmID)
	h.putAttachment(w, r, ref, attachmentFileName)
}

func (h *TmcHandler) PutTMNameAttachment(w http.ResponseWriter, r *http.Request, tmName server.TMName, attachmentFileName server.AttachmentFileName) {
	ref := model.NewTMNameAttachmentContainerRef(tmName)
	h.putAttachment(w, r, ref, attachmentFileName)
}

func (h *TmcHandler) putAttachment(w http.ResponseWriter, r *http.Request, ref model.AttachmentContainerRef, attachmentFileName string) {
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	err = r.Body.Close()

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	err = h.Service.PushAttachment(r.Context(), ref, attachmentFileName, b)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write(nil)
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
