package jwt

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/common"

	"github.com/golang-jwt/jwt/v5"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/server"
)

var jwksKeyFunc jwt.Keyfunc
var jwtServiceID string

// GetMiddleware starts a go routine that periodically fetches the JWKS
// key set and returns a middleware that uses that keyset to validate a
// token
func GetMiddleware(opts JWTValidationOpts) server.MiddlewareFunc {
	jwksKeyFunc = startJWKSFetch(opts).Keyfunc
	jwtServiceID = opts.JWTServiceID
	return jwtValidationMiddleware
}

func jwtValidationMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// existing scopes in ctx is the only hint for a protected endpoint
		scopes := extractAuthScopes(r)
		if scopes != nil {
			slog.Default().Debug("jwt: protected endpoint:", "path", r.URL)
			// protected endpoint, check for bearer tokenString in header
			tokenString, err := extractBearerToken(r)
			if err != nil {
				writeErrorResponse(w, err, http.StatusUnauthorized)
				return
			}
			// got token, validate it
			var token *jwt.Token
			if token, err = jwt.Parse(tokenString, jwksKeyFunc); err != nil {
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

var extractAuthScopes = func(r *http.Request) any {
	return r.Context().Value(server.BearerAuthScopes)
}

var TokenNotFoundError = errors.New("Authorization header does not contain a bearer token")

var extractBearerToken = func(r *http.Request) (string, error) {
	// get header and extract token string
	header := r.Header.Get(common.HeaderAuthorization)
	parts := strings.Split(header, " ")

	if !(len(parts) == 2 && parts[0] == "Bearer") {
		return "", TokenNotFoundError
	}

	token := parts[1]
	return token, nil
}

var InvalidAudClaimError = errors.New("Claim 'aud' did not contain valid service id")

func validateAudClaim(token *jwt.Token) error {
	audClaims, err := token.Claims.GetAudience()
	if err != nil {
		return err
	}
	for _, audClaim := range audClaims {
		if audClaim == jwtServiceID {
			return nil
		}
	}
	return InvalidAudClaimError
}

// TODO(pedram): refactor common.go into its own package to be reused here
func writeErrorResponse(w http.ResponseWriter, err error, status int) {
	slog.Default().Debug("jwt: " + err.Error())
	w.Header().Set(common.HeaderContentType, common.MimeJSON)
	w.WriteHeader(status)

	errString := fmt.Sprint(err)
	errorResponse := server.ErrorResponse{
		Status: status,
		Detail: &errString,
		Title:  common.Error401Title,
	}
	body, _ := json.MarshalIndent(errorResponse, "", "    ")
	_, _ = w.Write(body)
}
