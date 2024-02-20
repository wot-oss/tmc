package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	httpt "net/http/httptest"
	"testing"
	"time"

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

	tests := []struct {
		privateKey     *rsa.PrivateKey
		serviceID      string
		tokenString    string
		authScopes     []string
		expectedStatus int
		authorized     bool
	}{
		// Good case
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud": jwtServiceID,
				"nbf": pastDate,
				"exp": futureDate,
			}, keyA),
			[]string{},
			http.StatusOK,
			true,
		},
		// sign with A, validate with B
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud": jwtServiceID,
				"nbf": pastDate,
				"exp": futureDate,
			}, keyB),
			[]string{},
			http.StatusUnauthorized,
			false,
		},
		// wrong service ID
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud": "wrong-service-id",
				"nbf": pastDate,
				"exp": futureDate,
			}, keyA),
			[]string{},
			http.StatusUnauthorized,
			false,
		},
		// expired token
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud": jwtServiceID,
				"nbf": pastDate,
				"exp": pastDate,
			}, keyA),
			[]string{},
			http.StatusUnauthorized,
			false,
		},
		// not yet valid token
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud": jwtServiceID,
				"nbf": futureDate,
				"exp": futureDate,
			}, keyA),
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

func Test_invalidToken(t *testing.T) {

}

func logRSAKeyPair(keyPair *rsa.PrivateKey) {

}
