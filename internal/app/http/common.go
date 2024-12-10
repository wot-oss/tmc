package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/wot-oss/tmc/internal/app/http/server"
	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/utils"
)

const (
	Error400Title            = "Bad Request"
	Error401Title            = "Unauthorized"
	Error404Title            = "Not Found"
	Error409Title            = "Conflict"
	Error503Title            = "Service Unavailable"
	Error500Title            = "Internal Server Error"
	Error500Detail           = "An unhandled error has occurred. Try again later. If it is a bug we already recorded it. Retrying will most likely not help"
	Error502Title            = "Bad Gateway"
	Error502Detail           = "An upstream Thing Model repository returned an error"
	ErrorRepoAmbiguousDetail = "Repository ambiguous. Repeat the request with the 'repo' query parameter"

	HeaderAuthorization       = "Authorization"
	HeaderContentType         = "Content-Type"
	HeaderCacheControl        = "Cache-Control"
	HeaderXContentTypeOptions = "X-Content-Type-Options"
	MimeText                  = "text/plain"
	MimeJSON                  = "application/json"
	MimeOctetStream           = "application/octet-stream"
	MimeProblemJSON           = "application/problem+json"
	NoSniff                   = "nosniff"
	NoCache                   = "no-cache, no-store, max-age=0, must-revalidate"

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

	w.Header().Set(HeaderContentType, MimeJSON)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func HandleByteResponse(w http.ResponseWriter, r *http.Request, status int, mime string, data []byte) {
	w.Header().Set(HeaderContentType, mime)
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func HandleHealthyResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(HeaderCacheControl, NoCache)
	w.WriteHeader(http.StatusNoContent)
	_, _ = w.Write(nil)
}

func HandleErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	errTitle := Error500Title
	errDetail := Error500Detail
	errStatus := http.StatusInternalServerError
	errCode := ""

	var nfErr *model.ErrNotFound
	var eErr *repos.ErrTMIDConflict
	var aErr *repos.RepoAccessError
	var bErr *BaseHttpError

	switch true {
	// handle sentinel errors with errors.Is()
	case errors.Is(err, model.ErrInvalidId),
		errors.Is(err, model.ErrInvalidIdOrName),
		errors.Is(err, model.ErrInvalidFetchName),
		errors.Is(err, commands.ErrTMNameTooLong),
		errors.Is(err, repos.ErrRepoNotFound),
		errors.Is(err, repos.ErrInvalidCompletionParams):
		errTitle = Error400Title
		errDetail = err.Error()
		errStatus = http.StatusBadRequest
	case errors.Is(err, repos.ErrAmbiguous):
		errTitle = Error400Title
		errDetail = ErrorRepoAmbiguousDetail
		errStatus = http.StatusBadRequest
	case errors.Is(err, repos.ErrAttachmentExists):
		errTitle = Error409Title
		errDetail = err.Error()
		errStatus = http.StatusConflict
	// handle error values we want to access with errors.As()
	case errors.As(err, &nfErr):
		errTitle = Error404Title
		errDetail = err.Error()
		errStatus = http.StatusNotFound
		errCode = nfErr.Code()
	case errors.As(err, &bErr):
		errTitle = bErr.Title
		errDetail = bErr.Detail
		errStatus = bErr.Status
	case errors.As(err, &aErr):
		errTitle = Error502Title
		errDetail = Error502Detail
		errStatus = http.StatusBadGateway
	case errors.As(err, &eErr):
		errTitle = Error409Title
		errDetail = eErr.Error()
		errCode = eErr.Code()
		errStatus = http.StatusConflict
	// handle error values we don't need to access with errors.As,
	// but don't create a separate var above
	case errors.As(err, new(*jsonschema.ValidationError)),
		errors.As(err, new(*json.SyntaxError)):
		errTitle = Error400Title
		errDetail = err.Error()
		errStatus = http.StatusBadRequest
	case errors.As(err, new(*server.InvalidParamFormatError)),
		errors.As(err, new(*server.RequiredParamError)),
		errors.As(err, new(*server.RequiredHeaderError)),
		errors.As(err, new(*server.UnmarshalingParamError)),
		errors.As(err, new(*server.TooManyValuesForParamError)),
		errors.As(err, new(*server.UnescapedCookieParamError)):
		errTitle = Error400Title
		errDetail = err.Error()
		errStatus = http.StatusBadRequest
	default:
	}

	if err != nil {
		log := utils.GetLogger(r.Context(), "http.HandleErrorResponse")
		log = log.With("title", errTitle, "status", errStatus)
		if errStatus < http.StatusInternalServerError {
			log.Info(errDetail)
		} else {
			log.Error(errDetail)
		}
	}

	problem := server.ErrorResponse{
		Title:    errTitle,
		Detail:   &errDetail,
		Status:   errStatus,
		Instance: &r.RequestURI,
		Code:     &errCode,
	}

	respBody, _ := json.MarshalIndent(problem, "", "  ")
	w.Header().Set(HeaderContentType, MimeProblemJSON)
	w.Header().Set(HeaderXContentTypeOptions, NoSniff)
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

func NewUnauthorizedError(err error, detail string, args ...any) error {
	return newBaseHttpError(err, http.StatusUnauthorized, Error401Title, detail, args...)
}

func NewNotFoundError(err error, detail string, args ...any) error {
	return newBaseHttpError(err, http.StatusNotFound, Error404Title, detail, args...)
}

func NewBadRequestError(err error, detail string, args ...any) error {
	return newBaseHttpError(err, http.StatusBadRequest, Error400Title, detail, args...)
}

func NewServiceUnavailableError(err error, detail string) error {
	return newBaseHttpError(err, http.StatusServiceUnavailable, Error503Title, detail)
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

func convertRepoName(repo *string) string {
	if repo != nil {
		return *repo
	}
	return ""
}

func convertForceParam(p *server.ForceImport) bool {
	if p != nil {
		return *p
	}
	return false
}

func convertParams(params any) *model.SearchParams {

	var filterAuthor *string
	var filterManufacturer *string
	var filterMpn *string
	var filterProtocol *string
	var filterName *string
	var search *string

	if invParams, ok := params.(server.GetInventoryParams); ok {
		filterAuthor = invParams.FilterAuthor
		filterManufacturer = invParams.FilterManufacturer
		filterMpn = invParams.FilterMpn
		filterProtocol = invParams.FilterProtocol
		filterName = invParams.FilterName
		search = invParams.Search
	} else if authorsParams, ok := params.(server.GetAuthorsParams); ok {
		filterManufacturer = authorsParams.FilterManufacturer
		filterMpn = authorsParams.FilterMpn
		filterProtocol = authorsParams.FilterProtocol
		search = authorsParams.Search
	} else if manParams, ok := params.(server.GetManufacturersParams); ok {
		filterAuthor = manParams.FilterAuthor
		filterMpn = manParams.FilterMpn
		filterProtocol = manParams.FilterProtocol
		search = manParams.Search
	} else if mpnsParams, ok := params.(server.GetMpnsParams); ok {
		filterAuthor = mpnsParams.FilterAuthor
		filterManufacturer = mpnsParams.FilterManufacturer
		filterProtocol = mpnsParams.FilterProtocol
		search = mpnsParams.Search
	}

	return model.ToSearchParams(filterAuthor, filterManufacturer, filterMpn, filterProtocol, filterName, search,
		&model.SearchOptions{NameFilterType: model.PrefixMatch})
}

func toInventoryResponse(ctx context.Context, res model.SearchResult) server.InventoryResponse {
	mapper := NewMapper(ctx)

	meta := mapper.GetInventoryMeta(res)
	inv := mapper.GetInventoryData(res.Entries)
	resp := server.InventoryResponse{
		Meta: &meta,
		Data: inv,
	}
	return resp
}

func toInventoryEntryResponse(ctx context.Context, es []model.FoundEntry) server.InventoryEntryResponse {
	mapper := NewMapper(ctx)

	var ses []server.InventoryEntry
	for _, e := range es {
		ses = append(ses, mapper.GetInventoryEntry(e))
	}
	resp := server.InventoryEntryResponse{
		Data: ses,
	}
	return resp
}

func toInventoryEntryVersionResponse(ctx context.Context, v model.FoundVersion) server.InventoryEntryVersionResponse {
	mapper := NewMapper(ctx)

	ev := mapper.GetInventoryEntryVersion(v)
	resp := server.InventoryEntryVersionResponse{
		Data: ev,
	}
	return resp
}

func toInventoryEntryVersionsResponse(ctx context.Context, v []model.FoundVersion) server.InventoryEntryVersionsResponse {
	mapper := NewMapper(ctx)

	ev := mapper.GetInventoryEntryVersions(v)
	resp := server.InventoryEntryVersionsResponse{
		Data: ev,
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

func toReposResponse(repos []model.RepoDescription) server.ReposResponse {
	rds := []server.RepoDescription{}
	for _, r := range repos {
		r := r
		rds = append(rds, server.RepoDescription{
			Description: func(s string) *string {
				if s == "" {
					return nil
				}
				return &s
			}(r.Description),
			Name: r.Name,
		})
	}
	resp := server.ReposResponse{
		Data: rds,
	}
	return resp
}

func toImportThingModelResponse(res repos.ImportResult) server.ImportThingModelResponse {
	data := server.ImportThingModelResult{
		TmID: res.TmID,
	}
	data.Message = &res.Message
	var ce repos.CodedError
	if errors.As(res.Err, &ce) {
		code := ce.Code()
		data.Code = &code
	}
	return server.ImportThingModelResponse{
		Data: data,
	}
}

func infoResponse() server.InfoResponse {
	return server.InfoResponse{
		Name: "tmc",
		Version: server.InfoVersion{
			Implementation: utils.GetTmcVersion(),
		},
		Details: &[]string{},
	}
}

func WithRequestLogger(handler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log := loggerForRequest(r)
			ctx := context.WithValue(r.Context(), utils.CtxKeyLogger, log)
			utils.GetLogger(ctx, "http.serve").Info("received request")
			r = r.WithContext(ctx)
			handler.ServeHTTP(w, r)
		})
}

func WithLogAfterRequestProcessing(handler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				utils.GetLogger(r.Context(), "http.serve").Info("processed request")
			}()
			handler.ServeHTTP(w, r)
		})
}

func loggerForRequest(r *http.Request) *slog.Logger {
	requestID := r.Header.Get("X-Request-Id")
	if requestID == "" {
		requestID = newRequestID()
		r.Header.Set("X-Request-Id", requestID)
	}
	log := slog.Default().With("request-id", requestID, "method", r.Method, "path", r.URL.Path)
	return log
}

func newRequestID() string {
	rId, _ := uuid.NewRandom()
	return rId.String()
}
