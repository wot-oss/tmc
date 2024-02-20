package jwt

// TODO(pedram): refactor common.go into its own package to be reused here
const (
	error400Title  = "Bad Request"
	error401Title  = "Unauthorized"
	error404Title  = "Not Found"
	error409Title  = "Conflict"
	error503Title  = "Service Unavailable"
	error500Title  = "Internal Server Error"
	error500Detail = "An unhandled error has occurred. Try again later. If it is a bug we already recorded it. Retrying will most likely not help"

	headerAuthorization       = "Authorization"
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
