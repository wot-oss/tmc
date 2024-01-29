package http

import (
	"net/http"
	"slices"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/server"
)

type ServerOptions struct {
	CORS CORSOptions
}

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

func NewHttpHandler(si server.ServerInterface) http.Handler {
	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(handleNoRoute)
	options := server.GorillaServerOptions{
		BaseRouter:       r,
		ErrorHandlerFunc: HandleErrorResponse,
	}
	return server.HandlerWithOptions(si, options)
}

func WithCORS(h http.Handler, opts ServerOptions) http.Handler {
	// add supported default values to the CORS options
	opts.CORS.AddAllowedHeaders(headerContentType)

	// add CORS middleware to the http handler
	var corsOpts []handlers.CORSOption
	corsOpts = append(corsOpts, handlers.AllowedHeaders(opts.CORS.allowedHeaders))
	corsOpts = append(corsOpts, handlers.AllowedOrigins(opts.CORS.allowedOrigins))
	corsOpts = append(corsOpts, handlers.AllowedMethods([]string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPatch,
		http.MethodOptions,
		http.MethodHead,
		http.MethodDelete}))

	if opts.CORS.allowCredentials {
		corsOpts = append(corsOpts, handlers.AllowCredentials())
	}
	if opts.CORS.maxAge > 0 {
		corsOpts = append(corsOpts, handlers.MaxAge(opts.CORS.maxAge))
	}

	return handlers.CORS(corsOpts...)(h)
}

func handleNoRoute(w http.ResponseWriter, r *http.Request) {
	HandleErrorResponse(w, r, NewNotFoundError(nil, "Path not handled by Thing Model Catalog"))
}
