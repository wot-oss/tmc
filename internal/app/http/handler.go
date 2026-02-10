package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/wot-oss/tmc/internal/app/http/server"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/utils"
)

const ContextKeyBearerAuthNamespaces = "BearerAuth.Namespaces"

type TmcHandler struct {
	Service     HandlerService
	Options     TmcHandlerOptions
	JobManager  *JobManager
	zipData     []byte
	zipDataName string
}

type TmcHandlerOptions struct {
	UrlContextRoot string
	JWTValidation  bool
}

type ExportJobStatus struct {
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Error     string    `json:"error,omitempty"`
}

type JobManager struct {
	job               ExportJobStatus
	activeJobLock     sync.Mutex
	isExportingActive bool
}

func NewTmcHandler(handlerService HandlerService, options TmcHandlerOptions) *TmcHandler {
	return &TmcHandler{
		Service:     handlerService,
		Options:     options,
		JobManager:  NewJobManager(),
		zipData:     make([]byte, 0),
		zipDataName: "",
	}
}

func NewJobManager() *JobManager {
	return &JobManager{
		job: ExportJobStatus{
			Status:    "idle",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		isExportingActive: false,
	}
}

// GetInventory returns the inventory of the catalog
// (GET /inventory)
func (h *TmcHandler) GetInventory(w http.ResponseWriter, r *http.Request, params server.GetInventoryParams) {
	var page, pageSize, offset, limit int
	filters := convertParams(params)
	if params.Page != nil || params.PageSize != nil {
		page = 1
		if params.Page != nil && *params.Page > 0 {
			page = *params.Page
		}
		pageSize = 100
		if params.PageSize != nil && *params.PageSize > 0 {
			pageSize = *params.PageSize
		}
		offset = (page - 1) * pageSize
		limit = pageSize
	} else {
		offset = -1
		limit = -1
	}
	if h.Options.JWTValidation {
		namespaces := extractNamespacesFromContext(r.Context())
		if namespaces != nil {
			if filters == nil {
				filters = &model.Filters{}
				if !slices.Contains(namespaces, "*") {
					filters.Author = namespaces
				} else {
					filters = nil
				}
			} else if filters.Author != nil {
				authorSet := make(map[string]struct{})
				for _, a := range filters.Author {
					authorSet[a] = struct{}{}
				}
				var intersection []string
				if slices.Contains(namespaces, "*") {
					for _, a := range filters.Author {
						intersection = append(intersection, a)
					}
				} else {
					for _, ns := range namespaces {
						if _, exists := authorSet[ns]; exists {
							intersection = append(intersection, ns)
						}
					}
				}
				if len(intersection) == 0 {
					resp := toInventoryResponse(h.createContext(r), model.SearchResult{
						LastUpdated: time.Now(),
						Entries:     []model.FoundEntry{},
					}, page, pageSize)
					HandleJsonResponse(w, r, http.StatusOK, resp)
					return
				}
				filters.Author = intersection
			}
		}
	}
	repo := convertRepoName(params.Repo)
	var search string
	if params.Search != nil {
		search = *params.Search
	}

	if filters != nil && params.Search != nil {
		HandleErrorResponse(w, r, fmt.Errorf("%w: filters and search are mutually exclusive", ErrIncompatibleParameters))
		return
	}

	var inv *model.SearchResult
	var err error
	if search != "" {
		inv, err = h.Service.SearchInventory(r.Context(), repo, search, offset, limit)
	} else {
		utils.GetLogger(r.Context(), "handler").Info(fmt.Sprintf("filters %v", filters))
		inv, err = h.Service.ListInventory(r.Context(), repo, filters, offset, limit)
	}
	if params.FilterLatest != nil && *params.FilterLatest {
		inv = filterLatestVersions(inv)
	}
	if params.FilterLatest != nil && *params.FilterLatest {
		inv = filterLatestVersions(inv)
	}

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	ctx := h.createContext(r)
	resp := toInventoryResponse(ctx, *inv, page, pageSize)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

func filterLatestVersions(inv *model.SearchResult) *model.SearchResult {
	for i, entry := range inv.Entries {
		filteredEntry := model.FoundEntry{}
		if len(entry.Versions) > 1 {
			if len(entry.Versions) > 1 {
				var latestVersion *model.FoundVersion
				if len(entry.Versions) > 0 {
					latestVersion = &entry.Versions[0]
				}
				for i := 1; i < len(entry.Versions); i++ {
					currentVersion := &entry.Versions[i]
					latestTime, err := time.Parse(time.RFC3339, latestVersion.TimeStamp)
					currentVersionTime, err2 := time.Parse(time.RFC3339, currentVersion.TimeStamp)
					if currentVersionTime.After(latestTime) && err == nil && err2 == nil {
						latestVersion = currentVersion
					}
				}
				filteredEntry.Versions = []model.FoundVersion{*latestVersion}
			}
			inv.Entries[i].Versions = filteredEntry.Versions
		}
	}
	return inv
}

func extractNamespacesFromContext(ctx context.Context) []string {
	val := ctx.Value(ContextKeyBearerAuthNamespaces)
	if val == nil {
		return nil
	}
	namespaces, ok := val.([]string)
	if !ok {
		return nil
	}
	return namespaces
}

// GetInventoryByName Get an inventory entry by inventory name
// (GET /inventory/.tmName/{tmName})
func (h *TmcHandler) GetInventoryByName(w http.ResponseWriter, r *http.Request, tmName string, params server.GetInventoryByNameParams) {
	repo := convertRepoName(params.Repo)
	entries, err := h.Service.FindInventoryEntries(r.Context(), repo, tmName)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	ctx := h.createContext(r)
	resp := toInventoryEntryResponse(ctx, entries)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetInventoryByFetchName Get the metadata of the most recent TM version matching the name
// (GET /inventory/.latest/{fetchName})
func (h *TmcHandler) GetInventoryByFetchName(w http.ResponseWriter, r *http.Request, fetchName server.FetchName, params server.GetInventoryByFetchNameParams) {
	entry, err := h.Service.GetLatestTMMetadata(r.Context(), convertRepoName(params.Repo), fetchName)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	ctx := h.createContext(r)
	resp := toInventoryEntryVersionResponse(ctx, entry)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetThingModelByFetchName Get the content of a Thing Model by fetch name
// (GET /thing-models/.latest/{fetchName}
func (h *TmcHandler) GetThingModelByFetchName(w http.ResponseWriter, r *http.Request, fetchName server.FetchName, params server.GetThingModelByFetchNameParams) {
	restoreId := false
	if params.RestoreId != nil {
		restoreId = *params.RestoreId
	}

	data, err := h.Service.FetchLatestThingModel(r.Context(), convertRepoName(params.Repo), fetchName, restoreId)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	HandleByteResponse(w, r, http.StatusOK, MimeTMJSON, data)
}

// GetInventoryByID returns the metadata of a single TM by ID
// (GET /inventory/{tmID})
func (h *TmcHandler) GetInventoryByID(w http.ResponseWriter, r *http.Request, tmID server.TMID, params server.GetInventoryByIDParams) {
	repo := convertRepoName(params.Repo)

	versions, err := h.Service.GetTMMetadata(r.Context(), repo, tmID)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	ctx := h.createContext(r)
	resp := toInventoryEntryVersionsResponse(ctx, versions)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

// GetThingModelById Get the content of a Thing Model by its ID
// (GET /thing-models/{id})
func (h *TmcHandler) GetThingModelById(w http.ResponseWriter, r *http.Request, id string, params server.GetThingModelByIdParams) {
	restoreId := false
	if params.RestoreId != nil {
		restoreId = *params.RestoreId
	}

	data, err := h.Service.FetchThingModel(r.Context(), convertRepoName(params.Repo), id, restoreId)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}
	HandleByteResponse(w, r, http.StatusOK, MimeTMJSON, data)
}

// ExportCatalog Export the entire catalog as a zip file
// (GET /repos/export)
func (h *TmcHandler) GetExportedCatalog(w http.ResponseWriter, r *http.Request) {
	jobStatus := h.JobManager.GetJob()

	if jobStatus.Status != "completed" {
		err := NewConflictError(nil, "Exporting not yet complete or failed. Current status: %s", jobStatus.Status)
		HandleErrorResponse(w, r, err)
		return
	}

	data := h.zipData
	if len(data) == 0 {
		fmt.Printf("Error: Zip data not found in store for completed job when trying to download.\n")
		http.Error(w, "Internal server error: Zip data missing", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename="+h.zipDataName)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))

	_, err := w.Write(data)
	if err != nil {
		fmt.Printf("Error writing zip data to response: %v\n", err)
	}
}

// ExportCatalog Export the entire catalog as a zip file
// (POST /repos/export)
func (h *TmcHandler) ExportCatalog(w http.ResponseWriter, r *http.Request, params server.ExportCatalogParams) {
	if !h.JobManager.TryAcquireExportingLock() {
		currentJobStatus := h.JobManager.GetJob()
		err := NewConflictError(nil, "An export job is already in progress. Current status: %s", currentJobStatus.Status)
		HandleErrorResponse(w, r, err)
		return
	}
	if params.Repo == nil {
		h.JobManager.ReleaseExportingLock()
		err := NewBadRequestError(nil, "missing required query parameter: repo")
		HandleErrorResponse(w, r, err)
		return
	}
	_, err := h.Service.ListInventory(context.Background(), *params.Repo, nil)
	if err != nil {
		h.JobManager.ReleaseExportingLock()
		HandleErrorResponse(w, r, err)
		return
	}
	h.zipData = nil
	h.zipDataName = fmt.Sprintf("%s.zip", *params.Repo)
	go h.performExportCatalogAsync(convertRepoName(params.Repo))
	HandleJsonResponse(w, r, http.StatusAccepted, map[string]string{
		"status":  "packing",
		"message": "Exporting initiated.",
	})
}

func (h *TmcHandler) performExportCatalogAsync(repoName string) {
	defer h.JobManager.ReleaseExportingLock()

	h.JobManager.UpdateJobStatus("packing", "Currently packing the catalog...")
	ctx := context.Background()

	data, err := h.Service.ExportCatalog(ctx, repoName)
	if err != nil {
		h.JobManager.MarkJobFailed(err.Error())
		fmt.Printf("Error during async export: %v\n", err)
		return
	}
	h.zipData = data
	h.JobManager.UpdateJobStatus("completed", "Export complete. Ready for download.")
}

// DeleteThingModelById Delete a Thing Model by ID
// (DELETE /thing-models/{id})
func (h *TmcHandler) DeleteThingModelById(w http.ResponseWriter, r *http.Request, tmID string, params server.DeleteThingModelByIdParams) {
	if params.Force != "true" {
		HandleErrorResponse(w, r, NewBadRequestError(nil, "invalid value of 'force' query parameter"))
		return
	}

	err := h.Service.DeleteThingModel(r.Context(), convertRepoName(params.Repo), tmID)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	_, _ = w.Write(nil)
}

func (h *TmcHandler) ImportThingModel(w http.ResponseWriter, r *http.Request, p server.ImportThingModelParams) {
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
	if len(b) == 0 {
		HandleErrorResponse(w, r, NewBadRequestError(nil, "Empty request body"))
		return
	}

	opts := repos.ImportOptions{}
	opts.Force = convertForceParam(p.Force)
	if p.OptPath != nil {
		opts.OptPath = *p.OptPath
	}

	res, err := h.Service.ImportThingModel(r.Context(), convertRepoName(p.Repo), b, opts)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toImportThingModelResponse(res)

	HandleJsonResponse(w, r, http.StatusCreated, resp)

}

func (h *TmcHandler) GetAuthors(w http.ResponseWriter, r *http.Request, params server.GetAuthorsParams) {

	filters := convertParams(params)

	authors, err := h.Service.ListAuthors(r.Context(), filters)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toAuthorsResponse(authors)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

func (h *TmcHandler) GetManufacturers(w http.ResponseWriter, r *http.Request, params server.GetManufacturersParams) {

	filters := convertParams(params)

	mans, err := h.Service.ListManufacturers(r.Context(), filters)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toManufacturersResponse(mans)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

func (h *TmcHandler) GetMpns(w http.ResponseWriter, r *http.Request, params server.GetMpnsParams) {

	filters := convertParams(params)

	mpns, err := h.Service.ListMpns(r.Context(), filters)

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toMpnsResponse(mpns)
	HandleJsonResponse(w, r, http.StatusOK, resp)
}

func (h *TmcHandler) GetRepos(w http.ResponseWriter, r *http.Request) {
	rs, err := h.Service.ListRepos(r.Context())

	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	resp := toReposResponse(rs)
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

// GetInfo Returns some static information about the Thing Model Catalog API
// (GET /info)
func (h *TmcHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	resp := infoResponse()
	HandleJsonResponse(w, r, http.StatusOK, resp)
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

func (h *TmcHandler) GetThingModelAttachmentByName(w http.ResponseWriter, r *http.Request, tmid, attachmentFileName string, params server.GetThingModelAttachmentByNameParams) {
	ref := model.NewTMIDAttachmentContainerRef(tmid)
	h.fetchAttachment(w, r, convertRepoName(params.Repo), ref, attachmentFileName, false)
}
func (h *TmcHandler) GetTMNameAttachment(w http.ResponseWriter, r *http.Request, tmName server.TMName, attachmentFileName server.AttachmentFileName, params server.GetTMNameAttachmentParams) {
	ref := model.NewTMNameAttachmentContainerRef(tmName)
	concat := false
	if params.Concat != nil {
		concat = *params.Concat
	}
	h.fetchAttachment(w, r, convertRepoName(params.Repo), ref, attachmentFileName, concat)
}

func (h *TmcHandler) fetchAttachment(w http.ResponseWriter, r *http.Request, repo string, ref model.AttachmentContainerRef, attachmentFileName string, concat bool) {
	data, err := h.Service.FetchAttachment(r.Context(), repo, ref, attachmentFileName, concat)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}
	HandleByteResponse(w, r, http.StatusOK, MimeOctetStream, data)
}

func (h *TmcHandler) deleteAttachment(w http.ResponseWriter, r *http.Request, repo string, ref model.AttachmentContainerRef, attachmentFileName string) {
	err := h.Service.DeleteAttachment(r.Context(), repo, ref, attachmentFileName)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	_, _ = w.Write(nil)
}

func (h *TmcHandler) DeleteThingModelAttachmentByName(w http.ResponseWriter, r *http.Request, tmID server.TMID, attachmentFileName string, params server.DeleteThingModelAttachmentByNameParams) {
	ref := model.NewTMIDAttachmentContainerRef(tmID)
	h.deleteAttachment(w, r, convertRepoName(params.Repo), ref, attachmentFileName)
}

func (h *TmcHandler) DeleteTMNameAttachment(w http.ResponseWriter, r *http.Request, tmName server.TMName, attachmentFileName server.AttachmentFileName, params server.DeleteTMNameAttachmentParams) {
	ref := model.NewTMNameAttachmentContainerRef(tmName)
	h.deleteAttachment(w, r, convertRepoName(params.Repo), ref, attachmentFileName)
}

func (h *TmcHandler) PutTMIDAttachment(w http.ResponseWriter, r *http.Request, tmID string, attachmentFileName string, params server.PutTMIDAttachmentParams) {
	ref := model.NewTMIDAttachmentContainerRef(tmID)
	h.putAttachment(w, r, convertRepoName(params.Repo), ref, attachmentFileName, r.Header.Get(HeaderContentType), convertForceParam(params.Force))
}

func (h *TmcHandler) PutTMNameAttachment(w http.ResponseWriter, r *http.Request, tmName server.TMName, attachmentFileName server.AttachmentFileName, params server.PutTMNameAttachmentParams) {
	ref := model.NewTMNameAttachmentContainerRef(tmName)
	h.putAttachment(w, r, convertRepoName(params.Repo), ref, attachmentFileName, r.Header.Get(HeaderContentType), convertForceParam(params.Force))
}

func (h *TmcHandler) putAttachment(w http.ResponseWriter, r *http.Request, repo string, ref model.AttachmentContainerRef, attachmentFileName string, contentType string, force bool) {
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}
	if len(b) == 0 {
		HandleErrorResponse(w, r, NewBadRequestError(nil, "Empty request body"))
		return
	}

	err = h.Service.ImportAttachment(r.Context(), repo, ref, attachmentFileName, b, contentType, force)
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func (jm *JobManager) TryAcquireExportingLock() bool {
	jm.activeJobLock.Lock()
	defer jm.activeJobLock.Unlock()

	if jm.isExportingActive {
		return false
	}

	jm.isExportingActive = true
	jm.job = ExportJobStatus{
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return true
}

func (jm *JobManager) ReleaseExportingLock() {
	jm.activeJobLock.Lock()
	defer jm.activeJobLock.Unlock()
	jm.isExportingActive = false
}

func (jm *JobManager) CreateJob() ExportJobStatus {
	job := ExportJobStatus{
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	jm.job = job
	return job
}

func (jm *JobManager) GetJob() ExportJobStatus {
	jm.activeJobLock.Lock()
	defer jm.activeJobLock.Unlock()
	return jm.job
}

func (jm *JobManager) UpdateJobStatus(status, message string) {
	jm.activeJobLock.Lock()
	defer jm.activeJobLock.Unlock()
	jm.job.Status = status
	jm.job.UpdatedAt = time.Now()
	jm.job.Error = ""
}

func (jm *JobManager) MarkJobFailed(errorMessage string) {
	jm.activeJobLock.Lock()
	defer jm.activeJobLock.Unlock()
	jm.job.Status = "failed"
	jm.job.Error = errorMessage
	jm.job.UpdatedAt = time.Now()
}
