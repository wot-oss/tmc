package jwt

import (
	"context"
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
var whitelistFile string

var globalAccessControl *AccessControl

func InitializeAccessControl(filePath string) error {
	if globalAccessControl == nil {
		globalAccessControl = &AccessControl{}
	}
	globalAccessControl.filePath = filePath
	return globalAccessControl.reload()
}

// GetMiddleware starts a go routine that periodically fetches the JWKS
// key set and returns a middleware that uses that keyset to validate a
// token
func GetMiddleware(opts JWTValidationOpts) server.MiddlewareFunc {
	// start fetching JWKS and set keyfunc
	jwksKeyFunc = startJWKSFetch(opts).Keyfunc
	jwtServiceID = opts.JWTServiceID

	// ensure access control is initialized (loads whitelist file)
	if err := InitializeAccessControl(opts.WhitelistFile); err != nil {
		// log the error and continue; middleware will fail requests if access control is not available
		utils.GetLogger(context.Background(), "jwt.validation.init").Warn("failed to initialize access control", "error", err)
	}

	return jwtValidationMiddleware
}

func jwtValidationMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// existing scopes in ctx is the only hint for a protected endpoint
		if globalAccessControl == nil {
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
			if token, err = jwt.Parse(tokenString, jwksKeyFunc); err != nil {
				log.Warn("token validation failed", "error", err)
				httptmc.HandleErrorResponse(w, r, httptmc.NewUnauthorizedError(nil, err.Error()))
				return
			}
			// valid token, identify our service in the "aud" claim
			_, err = getAuthStatus(w, r, token)
			if err != nil {
				log.Warn("the user doesn't have a whitelist entry for the requested endpoint", "error", err)
				httptmc.HandleErrorResponse(w, r, httptmc.NewUnauthorizedError(nil, err.Error()))
				return
			}
			// https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.3
			// if err := validateAudClaim(token); err != nil {
			// 	//log.Warn("failed to validate 'aud' claim", "error", err)
			// 	//TODO: what to do?? httptmc.HandleErrorResponse(w, r, httptmc.NewUnauthorizedError(nil, err.Error()))
			// 	return
			// }
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

func getAuthStatus(w http.ResponseWriter, r *http.Request, token *jwt.Token) (bool, error) {

	// ac, err := NewAccessControl(config.WhitelistPath)
	// if err != nil {
	// 	log.Fatalf("Failed to initialize access control: %v", err)
	// }
	valid, userInfo := ValidateJWT(token.Raw)
	if !valid {
		err := errors.New("validation failed, check if it is needed at all")
		httptmc.HandleErrorResponse(w, r, httptmc.NewUnauthorizedError(err, ""))
		return false, ErrToken
	}
	aliasToBeChecked := ""
	switch strings.Split(r.URL.Path[1:], "/")[0] {
	case "inventory":
		aliasToBeChecked = "inventory"
	case "thing-models":
		if strings.Split(r.URL.Path[1:], "/")[1] == ".latest" || strings.Split(r.URL.Path[1:], "/")[1] == ".tmName" {
			aliasToBeChecked = strings.Split(r.URL.Path[1:], "/")[2]
		} else {
			aliasToBeChecked = strings.Split(r.URL.Path[1:], "/")[1]
		}
	default:
		return true, nil
	}
	if !HasAccess(userInfo, aliasToBeChecked, r) {
		err := errors.New("the user doesn't have access")
		httptmc.HandleErrorResponse(w, r, httptmc.NewUnauthorizedError(err, ""))
		return false, ErrToken
	}
	return true, nil
}
