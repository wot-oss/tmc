package http

import (
	"net/http"
	"slices"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/jwt"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/server"
)

// READ ME !!!
// Hint for generating the server code based on the openapi spec:
// oapi-codegen does not support all features of an OpenAPI spec like "pattern" in path variables, they will be lost in code after fresh generation.
// To prevent manual work after code generation, we execute the server/patch/patch.go as last step to fix these details.
//
// IF YOU ADD A NEW ROUTE to tm-catalog.openapi.yaml, AND YOUR ROUTE HAS A PATH VARIABLE WITH A REGEX PATTERN,
// ADD THE PATH VARIABLE REPLACEMENT TO server/patch/patch.go -> routePathVarPatch FOR PERMANENT FIXING.
//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.1.0 -package server -generate types -o server/models.gen.go ../../../api/tm-catalog.openapi.yaml
//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.1.0 -package server -generate gorilla-server -o server/server.gen.go ../../../api/tm-catalog.openapi.yaml
//go:generate go run server/patch/patch.go

type ServerOptions struct {
	JWTValidation bool
	jwt.JWTValidationOpts
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

func NewHttpHandler(si server.ServerInterface, mws []server.MiddlewareFunc) http.Handler {
	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(handleNoRoute)
	options := server.GorillaServerOptions{
		BaseRouter:       r,
		ErrorHandlerFunc: HandleErrorResponse,
		Middlewares:      mws,
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
