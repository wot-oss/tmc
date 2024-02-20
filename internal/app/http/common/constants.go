package common

const (
	Error400Title  = "Bad Request"
	Error401Title  = "Unauthorized"
	Error404Title  = "Not Found"
	Error409Title  = "Conflict"
	Error503Title  = "Service Unavailable"
	Error500Title  = "Internal Server Error"
	Error500Detail = "An unhandled error has occurred. Try again later. If it is a bug we already recorded it. Retrying will most likely not help"

	HeaderAuthorization       = "Authorization"
	HeaderContentType         = "Content-Type"
	HeaderCacheControl        = "Cache-Control"
	HeaderXContentTypeOptions = "X-Content-Type-Options"
	MimeText                  = "text/plain"
	MimeJSON                  = "application/json"
	MimeProblemJSON           = "application/problem+json"
	NoSniff                   = "nosniff"
	NoCache                   = "no-cache, no-store, max-age=0, must-revalidate"
)
