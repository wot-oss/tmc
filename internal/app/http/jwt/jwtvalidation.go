package jwt

import (
	"net/http"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/server"
)

func JWTValidationMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scopes := r.Context().Value(server.ApiKeyAuthScopes)
		if scopes != nil {
			// protected endpoint, check token

		}
		h.ServeHTTP(w, r)
	})
}
