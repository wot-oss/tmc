package http

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/server"
)

// Hint for generating the server code based on the openapi spec:
// 1. uncomment lines "// //go:generate" to "//go:generate" to be considered when calling "go generate"
// 2. after calling "go generate":
//       2.1. maybe reorder the properties in model.gen.go for a nicer JSON output, as oapi-codegen orders them alphabetically
//       2.2. for path parameters "name" and "tmID", add a regex for any character -> {name:.+}, {tmID:.+}
//       2.3. in server.gen.go, order the handler functions, in the way that the more specific routes are above the less specific
//          e.g. r.HandleFunc(options.BaseURL+"/inventory/{name:.+}/.versions" should be on top of r.HandleFunc(options.BaseURL+"/inventory/{name:.+}
// 3. when 2. is done, comment lines "// //go:generate" again, to prevent unwanted changes by calling "go generate"

// //go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.0.0 -package server -generate types -o server/models.gen.go ../../../api/tm-catalog.openapi.yaml
// //go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.0.0 -package server -generate gorilla-server -o server/server.gen.go ../../../api/tm-catalog.openapi.yaml

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
