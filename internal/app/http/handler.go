package http

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

// Hint for generating the server code based on the openapi spec:
// 1. uncomment lines "// //go:generate" to "//go:generate" to be considered when calling "go generate"
// 2. after calling "go generate":
//       2.1. maybe reorder the properties in model.gen.go for a nicer JSON output, as oapi-codegen orders them alphabetically
//       2.2. for path parameters "name" and "tmID", add a regex for any character -> {name:.+}, {tmID:.+}
//       2.3. in server.gen.go, order the handler functions, in the way that the more specific are on top on less specific
//          e.g. r.HandleFunc(options.BaseURL+"/inventory/{name:.+}/versions" should be on top of r.HandleFunc(options.BaseURL+"/inventory/{name:.+}
// 3. when 2. is done, comment lines "// //go:generate" again, to prevent unwanted changes by calling "go generate"

// //go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.0.0 -package http -generate types -o models.gen.go ../../../api/tm-catalog.openapi.yaml
// //go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.0.0 -package http -generate gorilla-server -o server.gen.go ../../../api/tm-catalog.openapi.yaml

type TmcHandler struct {
	Options TmcHandlerOptions
}

type TmcHandlerOptions struct {
	UrlContextRoot string
	PushTarget     remotes.RepoSpec
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

	searchParams := convertParams(params)

	toc, err := listToc(searchParams)

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

// GetInventoryVersionsByName Get the versions of an inventory entry
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
	tmID, err := pushThingModel(b, h.Options.PushTarget)
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

// GetHealth Get the overall health of the service
// (GET /healthz)
func (h *TmcHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	err := checkHealth()
	if err != nil {
		HandleErrorResponse(w, r, NewServiceUnavailableError(err, err.Error()))
		return
	}
	HandleHealthyResponse(w, r)
}

// GetHealthLive Returns the liveness of the service
// (GET /healthz/live)
func (h *TmcHandler) GetHealthLive(w http.ResponseWriter, r *http.Request) {
	err := checkHealthLive()
	if err != nil {
		HandleErrorResponse(w, r, NewServiceUnavailableError(err, err.Error()))
		return
	}
	HandleHealthyResponse(w, r)
}

// GetHealthReady Returns the readiness of the service
// (GET /healthz/ready)
func (h *TmcHandler) GetHealthReady(w http.ResponseWriter, r *http.Request) {
	err := checkHealthReady()
	if err != nil {
		HandleErrorResponse(w, r, NewServiceUnavailableError(err, err.Error()))
		return
	}
	HandleHealthyResponse(w, r)
}

// GetHealthStartup Returns whether the service is initialized
// (GET /healthz/startup)
func (h *TmcHandler) GetHealthStartup(w http.ResponseWriter, r *http.Request) {
	err := checkHealthStartup()
	if err != nil {
		HandleErrorResponse(w, r, NewServiceUnavailableError(err, err.Error()))
		return
	}
	HandleHealthyResponse(w, r)
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
