package http

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/wot-oss/tmc/internal/app/http/server"
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

func handleNoRoute(w http.ResponseWriter, r *http.Request) {
	HandleErrorResponse(w, r, NewNotFoundError(nil, "Path not handled by Thing Model Catalog"))
}
