package jwt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	httptmc "github.com/wot-oss/tmc/internal/app/http"
	auth "github.com/wot-oss/tmc/internal/app/http/auth"
	"github.com/wot-oss/tmc/internal/app/http/server"
	"github.com/wot-oss/tmc/internal/utils"

	"github.com/golang-jwt/jwt/v5"
)

var jwksKeyFunc jwt.Keyfunc
var jwtServiceID string

// GetMiddleware starts a go routine that periodically fetches the JWKS
// key set and returns a middleware that uses that keyset to validate a
// token
func GetMiddleware(opts JWTValidationOpts) server.MiddlewareFunc {
	jwksKeyFunc = startJWKSFetch(opts).Keyfunc
	jwtServiceID = opts.JWTServiceID

	if err := auth.InitializeAccessControl(); err != nil {
		utils.GetLogger(context.Background(), "jwt.validation.init").Warn("failed to initialize access control", "error", err)
	}

	return jwtValidationMiddleware
}

func jwtValidationMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// existing scopes in ctx is the only hint for a protected endpoint
		if auth.GlobalAccessControl == nil {
			http.Error(w, "Access control not initialized", http.StatusInternalServerError)
			return
		}
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
			if token, err = jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, jwksKeyFunc); err != nil {
				log.Warn("token validation failed", "error", err)
				httptmc.HandleErrorResponse(w, r, httptmc.NewUnauthorizedError(nil, err.Error()))
				return
			}

			// Validate audience claim
			if err := validateAudClaim(token); err != nil {
				log.Warn("audience validation failed", "error", err)
				httptmc.HandleErrorResponse(w, r, httptmc.NewUnauthorizedError(nil, err.Error()))
				return
			}

			scopes, err := getScopesFromToken(token, nil)
			if err != nil {
				log.Warn("failed to get scopes from token", "error", err)
				httptmc.HandleErrorResponse(w, r, httptmc.NewUnauthorizedError(nil, err.Error()))
				return
			}
			_, err = getAuthStatus(r, scopes)
			if err != nil {
				log.Warn("the user doesn't have a whitelist entry for the requested endpoint", "error", err)
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
	header := r.Header.Get("Authorization")
	parts := strings.Split(header, " ")

	if !(len(parts) == 2 && parts[0] == "Bearer") {
		return "", TokenNotFoundError
	}

	token := parts[1]
	return token, nil
}

var ErrInvalidAudClaim = errors.New("claim 'aud' did not contain valid service id")
var ErrToken = errors.New("token fields do not match the expected values")

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
	return ErrInvalidAudClaim
}

func getScopesFromToken(token *jwt.Token, signingKey []byte) ([]string, error) {
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		scopeInterface, exists := claims["scope"]
		if !exists {
			return nil, fmt.Errorf("scope claim not found in token")
		}
		var scopes []string
		switch v := scopeInterface.(type) {
		case []string:
			scopes = v
		case []interface{}:
			for _, item := range v {
				if str, ok := item.(string); ok {
					scopes = append(scopes, str)
				} else {
					return nil, fmt.Errorf("scope item is not a string, got type %T", item)
				}
			}
		default:
			return nil, fmt.Errorf("scope claim has unexpected type %T", scopeInterface)
		}

		return scopes, nil
	} else {
		return nil, fmt.Errorf("claims type assertion failed")
	}
}

func getAuthStatus(r *http.Request, scopes []string) (bool, error) {
	pathParts := strings.Split(r.URL.Path[1:], "/")
	aliasToBeChecked := ""

	if len(pathParts) > 1 {
		if pathParts[1] == ".latest" || pathParts[1] == ".tmName" {
			if len(pathParts) > 2 {
				aliasToBeChecked = pathParts[2]
			}
		} else {
			aliasToBeChecked = pathParts[1]
		}
	}

	for _, scope := range scopes {
		if scope == "tmc.admin" {
			return true, nil
		}
		if scope == "tmc.repos.read" && pathParts[0] == "repos" && r.Method == "GET" {
			return true, nil
		}
		if scope == "tmc.internal.read" && pathParts[0] == "info" && r.Method == "GET" {
			return true, nil
		} // Support both legacy `tmc.repos.health` and the preferred `tmc.health.read`
		if (scope == "tmc.health.read") && pathParts[0] == "healthz" && r.Method == "GET" {
			return true, nil
		}
		if strings.HasPrefix(scope, "tmc.ns.") {
			parts := strings.Split(scope, ".")
			if len(parts) >= 4 {
				namespaceNameToExtract := parts[2]
				if aliasToBeChecked == "" && r.Method == "POST" && pathParts[0] == "thing-models" {
					if strings.HasSuffix(scope, ".write") {
						return true, nil
					}
				}
				if namespaceNameToExtract == aliasToBeChecked {
					if strings.HasSuffix(scope, ".read") && r.Method == "GET" {
						return true, nil
					} else if strings.HasSuffix(scope, ".write") && (r.Method == "POST" || r.Method == "PUT") {
						return true, nil
					} else if strings.HasSuffix(scope, "attachments.delete") && r.Method == "DELETE" && (pathParts[2] == ".attachments" || pathParts[3] == ".attachments") {
						return true, nil
					} else if strings.HasSuffix(scope, "thingmodels.delete") && r.Method == "DELETE" && pathParts[0] == "thing-models" && len(pathParts) == 2 {
						return true, nil
					}
				}
			} else {
				return false, fmt.Errorf("scope '%s' malformed: expected format 'tmc.ns.<namespace>.<operation>'", scope)
			}
		}
	}
	return false, fmt.Errorf("user does not have access to this resource")
}
