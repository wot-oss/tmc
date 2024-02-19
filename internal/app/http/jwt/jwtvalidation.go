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
var JWTServiceID string

// GetMiddleware starts a go routine that periodically fetches the JWKS
// key set and returns a middleware that uses that keyset to validate a
// token
func GetMiddleware(opts JWTValidationOpts) server.MiddlewareFunc {
	JWKSKeyFunc = startJWKSFetch(opts)
	JWTServiceID = opts.JWTServiceID
	return jwtValidationMiddleware
}

func jwtValidationMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// existing scopes in ctx is the only hint for a protected endpoint
		scopes := r.Context().Value(server.BearerAuthScopes)

		if scopes != nil {
			// protected endpoint, check for bearer tokenString in header
			tokenString, err := extractBearerToken(r)
			if err != nil {
				writeErrorResponse(w, err, http.StatusUnauthorized)
				return
			}
			// got token, validate it
			var token *jwt.Token
			if token, err = jwt.Parse(tokenString, JWKSKeyFunc.Keyfunc); err != nil {
				writeErrorResponse(w, err, http.StatusUnauthorized)
				return
			}
			// valid token, identify our service in the "aud" claim
			// https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.3
			if err := validateAudClaim(token); err != nil {
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

func validateAudClaim(token *jwt.Token) error {
	audClaims, err := token.Claims.GetAudience()
	if err != nil {
		return err
	}
	for _, audClaim := range audClaims {
		if audClaim == JWTServiceID {
			return nil
		}
	}
	return errors.New("Claim 'aud' did not contain valid service id")

}
