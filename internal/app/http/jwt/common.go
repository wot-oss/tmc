package jwt

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/server"
)

// TODO(pedram): refactor common.go into its own package to be reused here
const HTTPHeaderAuthorization = "Authorization"

// TODO(pedram): refactor common.go into its own package to be reused here
const (
	error400Title  = "Bad Request"
	error401Title  = "Unauthorized"
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

// TODO(pedram): refactor common.go into its own package to be reused here
func writeErrorResponse(w http.ResponseWriter, err error, status int) {
	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(status)

	errString := fmt.Sprint(err)
	errorResponse := server.ErrorResponse{
		Status: status,
		Detail: &errString,
		Title:  error401Title,
	}
	body, _ := json.MarshalIndent(errorResponse, "", "    ")
	_, _ = w.Write(body)
}
