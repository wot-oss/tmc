package jwt

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	httptmc "github.com/wot-oss/tmc/internal/app/http"
	"github.com/wot-oss/tmc/internal/app/http/server"
	"github.com/wot-oss/tmc/internal/utils"
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
			log := utils.GetLogger(r.Context(), "jwt.validation.middleware").With("authentication", true)
			log.Debug("jwt: protected endpoint:", "path", r.URL)
			// protected endpoint, check for bearer tokenString in header
			tokenString, err := extractBearerToken(r)
			if err != nil {
				log.Warn("failed to extract token", "error", err)
				httptmc.HandleErrorResponse(w, r, httptmc.NewUnauthorizedError(nil, err.Error()))
				return
			}
			// got token, validate it
			var token *jwt.Token
			if token, err = jwt.Parse(tokenString, jwksKeyFunc); err != nil {
				log.Warn("token validation failed", "error", err)
				httptmc.HandleErrorResponse(w, r, httptmc.NewUnauthorizedError(nil, err.Error()))
				return
			}
			// valid token, identify our service in the "aud" claim
			// https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.3
			if err := validateAudClaim(token); err != nil {
				log.Warn("failed to validate 'aud' claim", "error", err)
				httptmc.HandleErrorResponse(w, r, httptmc.NewUnauthorizedError(nil, err.Error()))
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

var extractAuthScopes = func(r *http.Request) any {
	return r.Context().Value(server.BearerAuthScopes)
}

var TokenNotFoundError = errors.New("'Authorization' header does not contain a bearer token")

var extractBearerToken = func(r *http.Request) (string, error) {
	// get header and extract token string
	header := r.Header.Get(httptmc.HeaderAuthorization)
	parts := strings.Split(header, " ")

	if !(len(parts) == 2 && parts[0] == "Bearer") {
		return "", TokenNotFoundError
	}

	token := parts[1]
	return token, nil
}

var InvalidAudClaimError = errors.New("claim 'aud' did not contain valid service id")

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
