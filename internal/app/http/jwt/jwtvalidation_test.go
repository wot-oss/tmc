package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	httpt "net/http/httptest"
	"testing"
	"time"

	auth "github.com/wot-oss/tmc/internal/app/http/auth"

	"github.com/golang-jwt/jwt/v5"
)

func newToken(claims jwt.MapClaims, key *rsa.PrivateKey) string {
	tokenString, _ := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	return tokenString
}

func Test_Authorization(t *testing.T) {
	keyA, _ := rsa.GenerateKey(rand.Reader, 1024)
	keyB, _ := rsa.GenerateKey(rand.Reader, 1024)
	pastDate := time.Now().Add(-24 * time.Hour).Unix()
	futureDate := time.Now().Add(24 * time.Hour).Unix()
	jwtServiceID = "some-service-id"
	username := "tmc testuser"
	wrongusername := "wrong tmc testuser"
	whitelistFile := "../../../../test/data/jwt/whitelist.json"
	auth.InitializeAccessControl(whitelistFile)

	tests := []struct {
		privateKey     *rsa.PrivateKey
		serviceID      string
		tokenString    string
		whitelistFile  string
		authScopes     []string
		expectedStatus int
		authorized     bool
	}{
		// Good case
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud":      jwtServiceID,
				"nbf":      pastDate,
				"exp":      futureDate,
				"username": username,
			}, keyA),
			whitelistFile,
			[]string{},
			http.StatusOK,
			true,
		},
		// sign with A, validate with B
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud":      jwtServiceID,
				"nbf":      pastDate,
				"exp":      futureDate,
				"username": username,
			}, keyB),
			whitelistFile,
			[]string{},
			http.StatusUnauthorized,
			false,
		},
		// wrong username
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud":      "wrong-service-id",
				"nbf":      pastDate,
				"exp":      futureDate,
				"username": wrongusername,
			}, keyA),
			whitelistFile,
			[]string{},
			http.StatusUnauthorized,
			false,
		},
		// expired token
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud":      jwtServiceID,
				"nbf":      pastDate,
				"exp":      pastDate,
				"username": wrongusername,
			}, keyA),
			whitelistFile,
			[]string{},
			http.StatusUnauthorized,
			false,
		},
		// not yet valid token
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud":      jwtServiceID,
				"nbf":      futureDate,
				"exp":      futureDate,
				"username": username,
			}, keyA),
			whitelistFile,
			[]string{},
			http.StatusUnauthorized,
			false,
		},
	}

	// inject authorization check
	authorized := false
	authorizedFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorized = true
	})
	protected := jwtValidationMiddleware(authorizedFunc)

	for _, test := range tests {
		// inject bearer token to test
		extractBearerToken = func(r *http.Request) (string, error) {
			return test.tokenString, nil
		}

		// inject key for validation
		jwksKeyFunc = func(*jwt.Token) (any, error) {
			return &test.privateKey.PublicKey, nil
		}

		// inject auth scopes, so the endpoint it protected
		extractAuthScopes = func(r *http.Request) any {
			return test.authScopes
		}

		out := httpt.NewRecorder()
		protected.ServeHTTP(out, httpt.NewRequest("", "/inventory", nil))

		if out.Result().StatusCode != test.expectedStatus || authorized != test.authorized {
			t.Fatal(out)
		}
		authorized = false
	}
}
