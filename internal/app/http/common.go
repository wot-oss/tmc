package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
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
	mimeJSON                  = "application/json"
	mimeProblemJSON           = "application/problem+json"
	noSniff                   = "nosniff"
	noCache                   = "no-cache, no-store, max-age=0, must-revalidate"

	basePathInventory   = "/inventory"
	basePathThingModels = "/thing-models"

	ctxUrlRoot       = "urlContextRoot"
	ctxRelPathDepth  = "relPathDepth"
	ctxRemoteManager = "remoteManager"
	ctxPushRemote    = "pushRemote"
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

	if sErr, ok := err.(*BaseHttpError); ok {
		errTitle = sErr.Title
		errDetail = sErr.Detail
		errStatus = sErr.Status
	} else if sErr, ok := err.(*remotes.ErrTMExists); ok {
		errTitle = error409Title
		errDetail = sErr.Error()
		errStatus = http.StatusConflict
	} else {
		switch err.(type) {
		case *jsonschema.ValidationError, *json.SyntaxError:
			errTitle = error400Title
			errDetail = err.Error()
			errStatus = http.StatusBadRequest
		case *InvalidParamFormatError, *RequiredParamError, *RequiredHeaderError,
			*UnmarshalingParamError, *TooManyValuesForParamError, *UnescapedCookieParamError:
			errTitle = error400Title
			errDetail = err.Error()
			errStatus = http.StatusBadRequest
		default:
		}
	}

	problem := ErrorResponse{
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
	detail = fmt.Sprintf(detail, args...)
	return &BaseHttpError{
		Status: http.StatusNotFound,
		Title:  error404Title,
		Detail: detail,
		Err:    err,
	}
}

func NewBadRequestError(err error, detail string, args ...any) error {
	detail = fmt.Sprintf(detail, args...)
	return &BaseHttpError{
		Status: http.StatusBadRequest,
		Title:  error400Title,
		Detail: detail,
		Err:    err,
	}
}

func NewServiceUnavailableError(err error, detail string) error {
	return &BaseHttpError{
		Status: http.StatusServiceUnavailable,
		Title:  error503Title,
		Detail: detail,
		Err:    err,
	}
}

func convertParams(params any) *model.SearchParams {

	var filterAuthor *string
	var filterManufacturer *string
	var filterMpn *string
	var filterExternalID *string
	var search *string

	if invParams, ok := params.(GetInventoryParams); ok {
		filterAuthor = invParams.FilterAuthor
		filterManufacturer = invParams.FilterManufacturer
		filterMpn = invParams.FilterMpn
		filterExternalID = invParams.FilterExternalID
		search = invParams.Search
	} else if authorsParams, ok := params.(GetAuthorsParams); ok {
		filterManufacturer = authorsParams.FilterManufacturer
		filterMpn = authorsParams.FilterMpn
		filterExternalID = authorsParams.FilterExternalID
		search = authorsParams.Search
	} else if manParams, ok := params.(GetManufacturersParams); ok {
		filterAuthor = manParams.FilterAuthor
		filterMpn = manParams.FilterMpn
		filterExternalID = manParams.FilterExternalID
		search = manParams.Search
	} else if mpnsParams, ok := params.(GetMpnsParams); ok {
		filterAuthor = mpnsParams.FilterAuthor
		filterManufacturer = mpnsParams.FilterManufacturer
		filterExternalID = mpnsParams.FilterExternalID
		search = mpnsParams.Search
	}

	var searchParams model.SearchParams
	if filterAuthor != nil || filterManufacturer != nil || filterMpn != nil || filterExternalID != nil || search != nil {
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
		if filterExternalID != nil {
			searchParams.ExternalID = strings.Split(*filterExternalID, ",")
		}
		if search != nil {
			searchParams.Query = *search
		}
	}
	return &searchParams
}

func toInventoryResponse(ctx context.Context, toc model.SearchResult) InventoryResponse {
	mapper := NewMapper(ctx)

	meta := mapper.GetInventoryMeta(toc)
	inv := mapper.GetInventoryData(toc.Entries)
	resp := InventoryResponse{
		Meta: &meta,
		Data: inv,
	}
	return resp
}

func toInventoryEntryResponse(ctx context.Context, tocEntry model.FoundEntry) InventoryEntryResponse {
	mapper := NewMapper(ctx)

	invEntry := mapper.GetInventoryEntry(tocEntry)
	resp := InventoryEntryResponse{
		Data: invEntry,
	}
	return resp
}

func toInventoryEntryVersionsResponse(ctx context.Context, tocVersions []model.FoundVersion) InventoryEntryVersionsResponse {
	mapper := NewMapper(ctx)

	invEntryVersions := mapper.GetInventoryEntryVersions(tocVersions)
	resp := InventoryEntryVersionsResponse{
		Data: invEntryVersions,
	}
	return resp
}

func toAuthorsResponse(authors []string) AuthorsResponse {
	resp := AuthorsResponse{
		Data: authors,
	}
	return resp
}

func toManufacturersResponse(manufacturers []string) ManufacturersResponse {
	resp := ManufacturersResponse{
		Data: manufacturers,
	}
	return resp
}

func toMpnsResponse(mpns []string) MpnsResponse {
	resp := MpnsResponse{
		Data: mpns,
	}
	return resp
}

func toPushThingModelResponse(tmID string) PushThingModelResponse {
	data := PushThingModelResult{
		TmID: tmID,
	}
	return PushThingModelResponse{
		Data: data,
	}
}
