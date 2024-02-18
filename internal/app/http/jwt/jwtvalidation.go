package jwt

import (
	"errors"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/server"
)

var JWKSKeyFunc keyfunc.Keyfunc

// GetMiddleware starts a go routine that periodically fetches the JWKS
// key set and returns a middleware that uses that keyset to validate a
// token
func GetMiddleware(opts JWKSOpts) server.MiddlewareFunc {
	JWKSKeyFunc = startJWKSFetch(opts)
	return jwtValidationMiddleware
}

func jwtValidationMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// existing scopes in ctx is the only hint for a protected endpoint
		scopes := r.Context().Value(server.BearerAuthScopes)

		if scopes != nil {
			// protected endpoint, check for bearer token in header
			token, err := extractBearerToken(r)
			if err != nil {
				writeErrorResponse(w, err, http.StatusUnauthorized)
				return
			}
			// got token, validate it
			if _, err := jwt.Parse(token, JWKSKeyFunc.Keyfunc); err != nil {
				writeErrorResponse(w, err, http.StatusUnauthorized)
				return
			}
		}

		h.ServeHTTP(w, r)
	})
}

func extractBearerToken(r *http.Request) (string, error) {
	// get header and extract token string
	header := r.Header.Get(HTTPHeaderAuthorization)
	parts := strings.Split(header, " ")

	if !(len(parts) == 2 && parts[0] == "Bearer") {
		return "", errors.New("Authorization header does not contain a bearer token")
	}

	token := parts[1]
	return token, nil
}
