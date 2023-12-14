package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

const (
	error400Title  = "Bad request"
	error404Title  = "Not found"
	error409Title  = "Conflict"
	error500Title  = "Internal Server Error"
	error500Detail = "An unhandled error has occurred. Try again later. If it is a bug we already recorded it. Retrying will most likely not help"

	headerContentType         = "Content-Type"
	headerXContentTypeOptions = "X-Content-Type-Options"
	mimeJSON                  = "application/json"
	mimeProblemJSON           = "application/problem+json"
	noSniff                   = "nosniff"

	basePathInventory   = "/inventory"
	basePathThingModels = "/thing-models"
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
		Status: 404,
		Title:  error404Title,
		Detail: detail,
		Err:    err,
	}
}

func NewBadRequestError(err error, detail string, args ...any) error {
	detail = fmt.Sprintf(detail, args...)
	return &BaseHttpError{
		Status: 400,
		Title:  error400Title,
		Detail: detail,
		Err:    err,
	}
}

func convertParams(params any) *model.SearchParams {

	var filterAuthor *string
	var filterManufacturer *string
	var filterMpn *string
	var filterExternalID *string
	var searchContent *string

	if invParams, ok := params.(GetInventoryParams); ok {
		filterAuthor = invParams.FilterAuthor
		filterManufacturer = invParams.FilterManufacturer
		filterMpn = invParams.FilterMpn
		filterExternalID = invParams.FilterExternalID
		searchContent = invParams.SearchContent
	} else if authorsParams, ok := params.(GetAuthorsParams); ok {
		filterManufacturer = authorsParams.FilterManufacturer
		filterMpn = authorsParams.FilterMpn
		filterExternalID = authorsParams.FilterExternalID
		searchContent = authorsParams.SearchContent
	} else if manParams, ok := params.(GetManufacturersParams); ok {
		filterAuthor = manParams.FilterAuthor
		filterMpn = manParams.FilterMpn
		filterExternalID = manParams.FilterExternalID
		searchContent = manParams.SearchContent
	} else if mpnsParams, ok := params.(GetMpnsParams); ok {
		filterAuthor = mpnsParams.FilterAuthor
		filterManufacturer = mpnsParams.FilterManufacturer
		filterExternalID = mpnsParams.FilterExternalID
		searchContent = mpnsParams.SearchContent
	}

	var search model.SearchParams
	if filterAuthor != nil || filterManufacturer != nil || filterMpn != nil || filterExternalID != nil || searchContent != nil {
		search = model.SearchParams{}
		if filterAuthor != nil {
			search.Author = strings.Split(*filterAuthor, ",")
		}
		if filterManufacturer != nil {
			search.Manufacturer = strings.Split(*filterManufacturer, ",")
		}
		if filterMpn != nil {
			search.Mpn = strings.Split(*filterMpn, ",")
		}
		if filterExternalID != nil {
			search.ExternalID = strings.Split(*filterExternalID, ",")
		}
		if searchContent != nil {
			search.Query = *searchContent
		}
	}
	return &search
}

func toInventoryResponse(toc model.SearchResult) InventoryResponse {
	meta := mapInventoryMeta(toc)
	inv := mapInventoryData(toc.Entries)
	resp := InventoryResponse{
		Meta: &meta,
		Data: inv,
	}
	return resp
}

func toInventoryEntryResponse(tocEntry model.FoundEntry) InventoryEntryResponse {
	invEntry := mapInventoryEntry(tocEntry)
	resp := InventoryEntryResponse{
		Data: invEntry,
	}
	return resp
}

func toInventoryEntryVersionsResponse(tocVersions []model.FoundVersion) InventoryEntryVersionsResponse {
	invEntryVersions := mapInventoryEntryVersions(tocVersions)
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

func toPushThingModelResponse(tmID model.TMID) PushThingModelResponse {
	data := PushThingModelResult{
		TmID: tmID.String(),
	}
	return PushThingModelResponse{
		Data: data,
	}
}
