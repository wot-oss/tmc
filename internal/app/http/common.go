package http

import (
	"encoding/json"
	"fmt"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"net/http"
	"strings"
)

const (
	error400Title  = "Malformed request"
	error404Title  = "Not found"
	error500Title  = "Internal Server Error"
	error500Detail = "An unhandled error has occurred. Try again later. If it is a bug we already recorded it. Retrying will most likely not help"

	headerContentType         = "Content-Type"
	headerXContentTypeOptions = "X-Content-Type-Options"
	mimeJSON                  = "application/json"
	mimeProblemJSON           = "application/problem+json"
	noSniff                   = "nosniff"
)

func HandleJsonResponse(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	body, err := json.Marshal(data)
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
	} else {
		switch err.(type) {
		case *InvalidParamFormatError, *RequiredParamError, *RequiredHeaderError,
			*UnmarshalingParamError, *TooManyValuesForParamError, *UnescapedCookieParamError:
			errTitle = error400Title
			errDetail = err.Error()
			errStatus = 400
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

func hasFilterQuerySet(params GetInventoryParams) bool {
	return params.FilterAuthor != nil || params.FilterManufacturer != nil ||
		params.FilterMpn != nil || params.FilterOriginal != nil
}

func hasSearchQuerySet(params GetInventoryParams) bool {
	return params.SearchContent != nil
}

func toFilterParams(params GetInventoryParams) *FilterParams {
	if !hasFilterQuerySet(params) {
		return nil
	}

	filter := &FilterParams{}
	if params.FilterAuthor != nil {
		filter.Author = strings.Split(*params.FilterAuthor, ",")
	}
	if params.FilterManufacturer != nil {
		filter.Manufacturer = strings.Split(*params.FilterManufacturer, ",")
	}
	if params.FilterMpn != nil {
		filter.Mpn = strings.Split(*params.FilterMpn, ",")
	}
	if params.FilterOriginal != nil {
		filter.Original = strings.Split(*params.FilterOriginal, ",")
	}
	return filter
}

func toSearchParams(params GetInventoryParams) *SearchParams {
	if !hasSearchQuerySet(params) {
		return nil
	}

	search := &SearchParams{}
	if params.SearchContent != nil {
		search.query = *params.SearchContent
	}
	return search
}

func toInventoryResponse(toc model.Toc) InventoryResponse {
	inv := mapInventory(toc)
	resp := InventoryResponse{
		Data: inv,
	}
	return resp
}

func toInventoryEntryResponse(tocEntryId string, tocThing model.TocThing) InventoryEntryResponse {
	invEntry := mapInventoryEntry(tocEntryId, tocThing)
	resp := InventoryEntryResponse{
		Data: invEntry,
	}
	return resp
}

func toInventoryEntryVersionsResponse(tocVersions []model.TocVersion) InventoryEntryVersionsResponse {
	invEntryVersions := mapInvtoryEntryVersions(tocVersions)
	resp := InventoryEntryVersionsResponse{
		Data: invEntryVersions,
	}
	return resp
}
