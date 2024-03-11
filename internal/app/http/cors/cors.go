package cors

import (
	"net/http"
	"slices"

	"github.com/gorilla/handlers"
	httptmc "github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http"
)

type CORSOptions struct {
	allowedOrigins   []string
	allowedHeaders   []string
	allowCredentials bool
	maxAge           int
}

func (co *CORSOptions) AddAllowedOrigins(origins ...string) {
	for _, origin := range origins {
		if origin != "" && !slices.Contains(co.allowedOrigins, origin) {
			co.allowedOrigins = append(co.allowedOrigins, origin)
		}
	}
}

func (co *CORSOptions) AddAllowedHeaders(headers ...string) {
	for _, header := range headers {
		if header != "" && !slices.Contains(co.allowedHeaders, header) {
			co.allowedHeaders = append(co.allowedHeaders, header)
		}
	}
}

func (co *CORSOptions) AllowCredentials(allow bool) {
	co.allowCredentials = allow
}

func (co *CORSOptions) MaxAge(max int) {
	co.maxAge = max
}

func Protect(h http.Handler, opts CORSOptions) http.Handler {
	// add supported default values to the CORS options
	opts.AddAllowedHeaders(httptmc.HeaderContentType)

	// add CORS middleware to the http handler
	var corsOpts []handlers.CORSOption
	corsOpts = append(corsOpts, handlers.AllowedHeaders(opts.allowedHeaders))
	corsOpts = append(corsOpts, handlers.AllowedOrigins(opts.allowedOrigins))
	corsOpts = append(corsOpts, handlers.AllowedMethods([]string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPatch,
		http.MethodOptions,
		http.MethodHead,
		http.MethodDelete}))

	if opts.allowCredentials {
		corsOpts = append(corsOpts, handlers.AllowCredentials())
	}
	if opts.maxAge > 0 {
		corsOpts = append(corsOpts, handlers.MaxAge(opts.maxAge))
	}

	return handlers.CORS(corsOpts...)(h)
}
