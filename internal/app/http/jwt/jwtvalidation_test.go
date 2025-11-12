package jwt

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	httpt "net/http/httptest"
	"os"
	"testing"
	"time"

	auth "github.com/wot-oss/tmc/internal/app/http/auth"

	"github.com/golang-jwt/jwt/v5"
)

func newToken(claims jwt.MapClaims, key *rsa.PrivateKey) string {
	tokenString, _ := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	return tokenString
}

func Test_Authorization_Inventory(t *testing.T) {
	keyA, _ := rsa.GenerateKey(rand.Reader, 1024)
	keyB, _ := rsa.GenerateKey(rand.Reader, 1024)
	pastDate := time.Now().Add(-24 * time.Hour).Unix()
	futureDate := time.Now().Add(24 * time.Hour).Unix()
	jwtServiceID = "some-service-id"
	username := "tmc testuser god mode"
	userWithoutInventoryAccess := "tmc testuser no inventory"
	wrongUsername := "wrong tmc testuser"
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
		//tmc testuser no inventory
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud":      jwtServiceID,
				"nbf":      pastDate,
				"exp":      futureDate,
				"username": userWithoutInventoryAccess,
			}, keyA),
			whitelistFile,
			[]string{},
			http.StatusUnauthorized,
			false,
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
				"username": wrongUsername,
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
				"username": wrongUsername,
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

func Test_Authorization_GetTMsWithToken(t *testing.T) {
	keyA, _ := rsa.GenerateKey(rand.Reader, 1024)
	futureDate := time.Now().Add(24 * time.Hour).Unix()
	pastDate := time.Now().Add(-24 * time.Hour).Unix()
	jwtServiceID := "some-service-id"
	username := "tmc testuser god mode"
	userOnlyBCorpAllowed := "tmc testuser only b-corp"
	userOnlyPOSTAllowed := "tmc testuser only post"
	whitelistFile := "../../../../test/data/jwt/whitelist.json"
	auth.InitializeAccessControl(whitelistFile)
	filePath := "../../../../test/data/validate/omnilamp.json"
	jsonData, _ := os.ReadFile(filePath)

	tests := []struct {
		privateKey    *rsa.PrivateKey
		serviceID     string
		tokenString   string
		whitelistFile string
		authScopes    []string
		requests      []struct {
			method         string
			endpoint       string
			body           []byte
			expectedStatus int
			authorized     bool
		}
	}{
		// God mode user
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
			[]struct {
				method         string
				endpoint       string
				body           []byte
				expectedStatus int
				authorized     bool
			}{
				{
					"GET",
					"/thing-models/a-corp/eagle/bt2000/v1.0.0-20240108140117-243d1b462ccc.tm.json",
					nil,
					http.StatusOK,
					true,
				},
				{
					"GET",
					"/thing-models/b-corp/frog/bt3000/v1.0.0-20240108140117-743d1b462uuu.tm.json",
					nil,
					http.StatusOK,
					true,
				},
				{
					"GET",
					"/thing-models/.latest/a-corp/eagle/bt2000",
					nil,
					http.StatusOK,
					true,
				},
			},
		},
		// only bcorp namespace allowed
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud":      jwtServiceID,
				"nbf":      pastDate,
				"exp":      futureDate,
				"username": userOnlyBCorpAllowed,
			}, keyA),
			whitelistFile,
			[]string{},
			[]struct {
				method         string
				endpoint       string
				body           []byte
				expectedStatus int
				authorized     bool
			}{
				{
					"GET",
					"/thing-models/a-corp/eagle/bt2000/v1.0.0-20240108140117-243d1b462ccc.tm.json",
					nil,
					http.StatusUnauthorized,
					false,
				},
				{
					"GET",
					"/thing-models/b-corp/frog/bt3000/v1.0.0-20240108140117-743d1b462uuu.tm.json",
					nil,
					http.StatusOK,
					true,
				},
				{
					"GET",
					"/thing-models/.latest/a-corp/eagle/bt2000",
					nil,
					http.StatusUnauthorized,
					false,
				},
			},
		},
		// User with only POST operation allowed
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud":      jwtServiceID,
				"nbf":      pastDate,
				"exp":      futureDate,
				"username": userOnlyPOSTAllowed,
			}, keyA),
			whitelistFile,
			[]string{},
			[]struct {
				method         string
				endpoint       string
				body           []byte
				expectedStatus int
				authorized     bool
			}{
				{
					"GET",
					"/thing-models/a-corp/eagle/bt2000/v1.0.0-20240108140117-243d1b462ccc.tm.json",
					nil,
					http.StatusUnauthorized,
					false,
				},
				{
					"GET",
					"/thing-models/b-corp/frog/bt3000/v1.0.0-20240108140117-743d1b462uuu.tm.json",
					nil,
					http.StatusUnauthorized,
					false,
				},
				{
					"GET",
					"/thing-models/.latest/a-corp/eagle/bt2000",
					nil,
					http.StatusUnauthorized,
					false,
				},
				{
					"POST",
					"/thing-models",
					jsonData,
					http.StatusOK,
					true,
				},
			},
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

		// inject auth scopes, so the endpoint is protected
		extractAuthScopes = func(r *http.Request) any {
			return test.authScopes
		}
		for _, req := range test.requests {
			out := httpt.NewRecorder()
			var body *bytes.Reader
			if req.body != nil {
				body = bytes.NewReader(req.body)
				protected.ServeHTTP(out, httpt.NewRequest(req.method, req.endpoint, body))
			} else {
				protected.ServeHTTP(out, httpt.NewRequest(req.method, req.endpoint, nil))
			}

			// Ensure the correct status and authorization flags
			if out.Result().StatusCode != req.expectedStatus || authorized != req.authorized {
				t.Fatalf("Unexpected result for request: %s %s\nExpected status: %d, Authorized: %v\nGot status: %d, Authorized: %v",
					req.method, req.endpoint, req.expectedStatus, req.authorized, out.Result().StatusCode, authorized)
			}
			authorized = false // Reset authorization flag for next request
		}
	}
}
