package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/server"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

const (
	error400Title  = "Bad Request"
	error404Title  = "Not Found"
	error409Title  = "Conflict"
	error503Title  = "Service Unavailable"
	error500Title  = "Internal Server Error"
	error500Detail = "An unhandled error has occurred. Try again later. If it is a bug we already recorded it. Retrying will most likely not help"

	headerContentType         = "Content-Type"
	headerCacheControl        = "Cache-Control"
	headerXContentTypeOptions = "X-Content-Type-Options"
	mimeText                  = "text/plain"
	mimeJSON                  = "application/json"
	mimeProblemJSON           = "application/problem+json"
	noSniff                   = "nosniff"
	noCache                   = "no-cache, no-store, max-age=0, must-revalidate"

	basePathInventory   = "/inventory"
	basePathThingModels = "/thing-models"

	ctxUrlRoot      = "urlContextRoot"
	ctxRelPathDepth = "relPathDepth"
)

func HandleJsonResponse(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	body, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		HandleErrorResponse(w, r, err)
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func HandleByteResponse(w http.ResponseWriter, r *http.Request, status int, mime string, data []byte) {
	w.Header().Set(headerContentType, mime)
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func HandleHealthyResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headerCacheControl, noCache)
	w.WriteHeader(http.StatusNoContent)
	_, _ = w.Write(nil)
}

func HandleErrorResponse(w http.ResponseWriter, r *http.Request, err error) {

	if err != nil {
		//todo: log
		fmt.Println(err)
	}

	errTitle := error500Title
	errDetail := error500Detail
	errStatus := http.StatusInternalServerError

	var eErr *remotes.ErrTMExists
	var bErr *BaseHttpError
	if errors.Is(err, remotes.ErrTmNotFound) {
		errTitle = error404Title
		errDetail = err.Error()
		errStatus = http.StatusNotFound
	} else if errors.Is(err, model.ErrInvalidId) || errors.Is(err, commands.ErrInvalidFetchName) || errors.Is(err, remotes.ErrInvalidCompletionParams) {
		errTitle = error400Title
		errDetail = err.Error()
		errStatus = http.StatusBadRequest
	} else if errors.As(err, &bErr) {
		errTitle = bErr.Title
		errDetail = bErr.Detail
		errStatus = bErr.Status
	} else if errors.As(err, &eErr) {
		errTitle = error409Title
		errDetail = eErr.Error()
		errStatus = http.StatusConflict
	} else {
		switch err.(type) {
		case *jsonschema.ValidationError, *json.SyntaxError:
			errTitle = error400Title
			errDetail = err.Error()
			errStatus = http.StatusBadRequest
		case *server.InvalidParamFormatError, *server.RequiredParamError, *server.RequiredHeaderError,
			*server.UnmarshalingParamError, *server.TooManyValuesForParamError, *server.UnescapedCookieParamError:
			errTitle = error400Title
			errDetail = err.Error()
			errStatus = http.StatusBadRequest
		default:
		}
	}

	problem := server.ErrorResponse{
		Title:    errTitle,
		Detail:   &errDetail,
		Status:   errStatus,
		Instance: &r.RequestURI,
	}

	respBody, _ := json.MarshalIndent(problem, "", "  ")
	w.Header().Set(headerContentType, mimeProblemJSON)
	w.Header().Set(headerXContentTypeOptions, noSniff)
	w.WriteHeader(errStatus)
	_, _ = w.Write(respBody)
}

type BaseHttpError struct {
	Status int
	Title  string
	Detail string
	Err    error
}

func (e *BaseHttpError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%d: %s: %s", e.Status, e.Detail, e.Err.Error())
	} else {
		return fmt.Sprintf("%d: %s", e.Status, e.Detail)
	}
}

func (e *BaseHttpError) Unwrap() error {
	return e.Err
}

func NewNotFoundError(err error, detail string, args ...any) error {
	return newBaseHttpError(err, http.StatusNotFound, error404Title, detail, args...)
}

func NewBadRequestError(err error, detail string, args ...any) error {
	return newBaseHttpError(err, http.StatusBadRequest, error400Title, detail, args...)
}

func NewServiceUnavailableError(err error, detail string) error {
	return newBaseHttpError(err, http.StatusServiceUnavailable, error503Title, detail)
}

func newBaseHttpError(err error, status int, title string, detail string, args ...any) error {
	msg := fmt.Sprintf(detail, args...)

	if err != nil {
		msg = fmt.Sprintf(msg+": %s", err.Error())
	}

	be := &BaseHttpError{
		Status: status,
		Title:  title,
		Detail: msg,
		Err:    err,
	}
	return be
}

func convertParams(params any) *model.SearchParams {

	var filterAuthor *string
	var filterManufacturer *string
	var filterMpn *string
	var filterName *string
	var search *string

	if invParams, ok := params.(server.GetInventoryParams); ok {
		filterAuthor = invParams.FilterAuthor
		filterManufacturer = invParams.FilterManufacturer
		filterMpn = invParams.FilterMpn
		filterName = invParams.FilterName
		search = invParams.Search
	} else if authorsParams, ok := params.(server.GetAuthorsParams); ok {
		filterManufacturer = authorsParams.FilterManufacturer
		filterMpn = authorsParams.FilterMpn
		search = authorsParams.Search
	} else if manParams, ok := params.(server.GetManufacturersParams); ok {
		filterAuthor = manParams.FilterAuthor
		filterMpn = manParams.FilterMpn
		search = manParams.Search
	} else if mpnsParams, ok := params.(server.GetMpnsParams); ok {
		filterAuthor = mpnsParams.FilterAuthor
		filterManufacturer = mpnsParams.FilterManufacturer
		search = mpnsParams.Search
	}

	var searchParams model.SearchParams
	if filterAuthor != nil || filterManufacturer != nil || filterMpn != nil || filterName != nil || search != nil {
		searchParams = model.SearchParams{}
		if filterAuthor != nil {
			searchParams.Author = strings.Split(*filterAuthor, ",")
		}
		if filterManufacturer != nil {
			searchParams.Manufacturer = strings.Split(*filterManufacturer, ",")
		}
		if filterMpn != nil {
			searchParams.Mpn = strings.Split(*filterMpn, ",")
		}
		if filterName != nil {
			searchParams.Name = *filterName
			searchParams.Options.NameFilterType = model.PrefixMatch
		}
		if search != nil {
			searchParams.Query = *search
		}
	}
	return &searchParams
}

func toInventoryResponse(ctx context.Context, toc model.SearchResult) server.InventoryResponse {
	mapper := NewMapper(ctx)

	meta := mapper.GetInventoryMeta(toc)
	inv := mapper.GetInventoryData(toc.Entries)
	resp := server.InventoryResponse{
		Meta: &meta,
		Data: inv,
	}
	return resp
}

func toInventoryEntryResponse(ctx context.Context, tocEntry model.FoundEntry) server.InventoryEntryResponse {
	mapper := NewMapper(ctx)

	invEntry := mapper.GetInventoryEntry(tocEntry)
	resp := server.InventoryEntryResponse{
		Data: invEntry,
	}
	return resp
}

func toInventoryEntryVersionsResponse(ctx context.Context, tocVersions []model.FoundVersion) server.InventoryEntryVersionsResponse {
	mapper := NewMapper(ctx)

	invEntryVersions := mapper.GetInventoryEntryVersions(tocVersions)
	resp := server.InventoryEntryVersionsResponse{
		Data: invEntryVersions,
	}
	return resp
}

func toAuthorsResponse(authors []string) server.AuthorsResponse {
	resp := server.AuthorsResponse{
		Data: authors,
	}
	return resp
}

func toManufacturersResponse(manufacturers []string) server.ManufacturersResponse {
	resp := server.ManufacturersResponse{
		Data: manufacturers,
	}
	return resp
}

func toMpnsResponse(mpns []string) server.MpnsResponse {
	resp := server.MpnsResponse{
		Data: mpns,
	}
	return resp
}

func toPushThingModelResponse(tmID string) server.PushThingModelResponse {
	data := server.PushThingModelResult{
		TmID: tmID,
	}
	return server.PushThingModelResponse{
		Data: data,
	}
}
