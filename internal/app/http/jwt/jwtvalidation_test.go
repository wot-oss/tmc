package jwt

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	httpt "net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	auth "github.com/wot-oss/tmc/internal/app/http/auth"
	"github.com/wot-oss/tmc/internal/app/http/server"
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
	scopeAdmin := []string{"tmc.admin"}
	auth.InitializeAccessControl()

	tests := []struct {
		privateKey     *rsa.PrivateKey
		serviceID      string
		tokenString    string
		expectedStatus int
		authorized     bool
	}{
		// Good case
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud":   jwtServiceID,
				"nbf":   pastDate,
				"exp":   futureDate,
				"scope": scopeAdmin,
			}, keyA),
			http.StatusOK,
			true,
		},
		// sign with A, validate with B
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud":   jwtServiceID,
				"nbf":   pastDate,
				"exp":   futureDate,
				"scope": scopeAdmin,
			}, keyB),
			http.StatusUnauthorized,
			false,
		},
		// wrong audience
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud":   "wrong-service-id",
				"nbf":   pastDate,
				"exp":   futureDate,
				"scope": scopeAdmin,
			}, keyA),
			http.StatusUnauthorized,
			false,
		},
		// expired token
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud":   jwtServiceID,
				"nbf":   pastDate,
				"exp":   pastDate,
				"scope": scopeAdmin,
			}, keyA),
			http.StatusUnauthorized,
			false,
		},
		// not yet valid token
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud":   jwtServiceID,
				"nbf":   futureDate,
				"exp":   futureDate,
				"scope": scopeAdmin,
			}, keyA),
			http.StatusUnauthorized,
			false,
		},
	}

	for _, test := range tests {
		// Capture test values by value before creating closures
		tokenString := test.tokenString
		keyForValidation := test.privateKey
		expectedStatus := test.expectedStatus
		expectedAuthorized := test.authorized

		// Reset authorized flag for each test case
		authorized := false

		// Create wrapped handler that sets the authorized flag
		wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorized = true
		})
		protectedTest := jwtValidationMiddleware(wrappedHandler)

		extractBearerToken = func(r *http.Request) (string, error) {
			return tokenString, nil
		}

		jwksKeyFunc = func(*jwt.Token) (any, error) {
			return &keyForValidation.PublicKey, nil
		}

		out := httpt.NewRecorder()
		req := httpt.NewRequest("", "/inventory", nil)
		req = req.WithContext(context.WithValue(req.Context(), server.BearerAuthScopes, []string{}))
		protectedTest.ServeHTTP(out, req)

		if out.Result().StatusCode != expectedStatus || authorized != expectedAuthorized {
			t.Fatalf("Test case failed: expected status=%d authorized=%v, got status=%d authorized=%v",
				expectedStatus, expectedAuthorized, out.Result().StatusCode, authorized)
		}
	}
}

func Test_Authorization_GetTMsWithToken(t *testing.T) {
	keyA, _ := rsa.GenerateKey(rand.Reader, 1024)
	futureDate := time.Now().Add(24 * time.Hour).Unix()
	pastDate := time.Now().Add(-24 * time.Hour).Unix()
	jwtServiceID := "some-service-id"
	scopeAdmin := []string{"tmc.admin"}
	scopeOnlyBCorpNSRead := []string{"tmc.ns.b-corp.read"}
	scopeOnlyBCorpNSWrite := []string{"tmc.ns.b-corp.write"}
	auth.InitializeAccessControl()
	filePath := "../../../../test/data/validate/omnilamp.json"
	jsonData, _ := os.ReadFile(filePath)

	tests := []struct {
		privateKey  *rsa.PrivateKey
		serviceID   string
		tokenString string
		requests    []struct {
			method         string
			endpoint       string
			body           []byte
			expectedStatus int
			authorized     bool
		}
	}{
		// admin user
		{
			keyA,
			jwtServiceID,
			newToken(jwt.MapClaims{
				"aud":   jwtServiceID,
				"nbf":   pastDate,
				"exp":   futureDate,
				"scope": scopeAdmin,
			}, keyA),
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
				"aud":   jwtServiceID,
				"nbf":   pastDate,
				"exp":   futureDate,
				"scope": scopeOnlyBCorpNSRead,
			}, keyA),
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
				"aud":   jwtServiceID,
				"nbf":   pastDate,
				"exp":   futureDate,
				"scope": scopeOnlyBCorpNSWrite,
			}, keyA),
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

	for _, test := range tests {
		tokenString := test.tokenString
		keyForValidation := test.privateKey

		extractBearerToken = func(r *http.Request) (string, error) {
			return tokenString, nil
		}

		jwksKeyFunc = func(*jwt.Token) (any, error) {
			return &keyForValidation.PublicKey, nil
		}

		for _, req := range test.requests {
			authorized := false

			wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authorized = true
			})
			protected := jwtValidationMiddleware(wrappedHandler)

			out := httpt.NewRecorder()
			var httpReq *http.Request
			if req.body != nil {
				body := bytes.NewReader(req.body)
				httpReq = httpt.NewRequest(req.method, req.endpoint, body)
			} else {
				httpReq = httpt.NewRequest(req.method, req.endpoint, nil)
			}
			httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), server.BearerAuthScopes, []string{}))
			protected.ServeHTTP(out, httpReq)

			if out.Result().StatusCode != req.expectedStatus || authorized != req.authorized {
				t.Fatalf("Unexpected result for request: %s %s\nExpected status: %d, Authorized: %v\nGot status: %d, Authorized: %v",
					req.method, req.endpoint, req.expectedStatus, req.authorized, out.Result().StatusCode, authorized)
			}
		}
	}
}

func Test_Authorization_Scopes(t *testing.T) {
	keyA, _ := rsa.GenerateKey(rand.Reader, 1024)
	futureDate := time.Now().Add(24 * time.Hour).Unix()
	pastDate := time.Now().Add(-24 * time.Hour).Unix()
	jwtServiceID := "some-service-id"
	auth.InitializeAccessControl()

	// Define scope strings
	scopeReadACorp := []string{"tmc.ns.a-corp.read"}
	scopeWriteACorp := []string{"tmc.ns.a-corp.write"}
	scopeAttachmentsDeleteACorp := []string{"tmc.ns.a-corp.attachments.delete"}
	scopeTMDeleteACorp := []string{"tmc.ns.a-corp.thingmodels.delete"}
	scopeReposRead := []string{"tmc.repos.read"}
	scopeInternalRead := []string{"tmc.internal.read"}
	scopeHealthRead := []string{"tmc.health.read"}

	tests := []struct {
		name     string
		scope    []string
		requests []struct {
			method         string
			endpoint       string
			body           []byte
			expectedStatus int
			authorized     bool
		}
	}{
		{
			name:  "ns a-corp read",
			scope: scopeReadACorp,
			requests: []struct {
				method, endpoint string
				body             []byte
				expectedStatus   int
				authorized       bool
			}{
				{"GET", "/thing-models/a-corp/eagle/bt2000/v1.0.0.tm.json", nil, http.StatusOK, true},
				{"GET", "/thing-models/b-corp/frog/bt3000/v1.0.0.tm.json", nil, http.StatusUnauthorized, false},
				{"GET", "/thing-models/.latest/a-corp/eagle/bt2000", nil, http.StatusOK, true},
				{"POST", "/thing-models", nil, http.StatusUnauthorized, false},
				{"GET", "/inventory", nil, http.StatusUnauthorized, false},
			},
		},
		{
			name:  "ns a-corp write",
			scope: scopeWriteACorp,
			requests: []struct {
				method, endpoint string
				body             []byte
				expectedStatus   int
				authorized       bool
			}{
				{"POST", "/thing-models", nil, http.StatusOK, true},
				{"GET", "/thing-models/a-corp/eagle/bt2000/v1.0.0.tm.json", nil, http.StatusUnauthorized, false},
			},
		},
		{
			name:  "ns a-corp attachments.delete",
			scope: scopeAttachmentsDeleteACorp,
			requests: []struct {
				method, endpoint string
				body             []byte
				expectedStatus   int
				authorized       bool
			}{
				{"DELETE", "/thing-models/a-corp/.attachments/att123", nil, http.StatusOK, true},
				{"DELETE", "/thing-models/b-corp/.attachments/att123", nil, http.StatusUnauthorized, false},
				{"DELETE", "/thing-models/a-corp/something/.attachments/att123", nil, http.StatusOK, true},
			},
		},
		{
			name:  "ns a-corp thingmodels.delete",
			scope: scopeTMDeleteACorp,
			requests: []struct {
				method, endpoint string
				body             []byte
				expectedStatus   int
				authorized       bool
			}{
				{"DELETE", "/thing-models/a-corp", nil, http.StatusOK, true},
				{"DELETE", "/thing-models/b-corp", nil, http.StatusUnauthorized, false},
				{"DELETE", "/thing-models/a-corp/eagle", nil, http.StatusUnauthorized, false},
			},
		},
		{
			name:  "repos read",
			scope: scopeReposRead,
			requests: []struct {
				method, endpoint string
				body             []byte
				expectedStatus   int
				authorized       bool
			}{
				{"GET", "/repos", nil, http.StatusOK, true},
				{"GET", "/inventory", nil, http.StatusUnauthorized, false},
			},
		},
		{
			name:  "internal read",
			scope: scopeInternalRead,
			requests: []struct {
				method, endpoint string
				body             []byte
				expectedStatus   int
				authorized       bool
			}{
				{"GET", "/info/some", nil, http.StatusOK, true},
				{"GET", "/info/other/more", nil, http.StatusOK, true},
				{"GET", "/inventory", nil, http.StatusUnauthorized, false},
			},
		},
		{
			name:  "health read",
			scope: scopeHealthRead,
			requests: []struct {
				method, endpoint string
				body             []byte
				expectedStatus   int
				authorized       bool
			}{
				{"GET", "/healthz", nil, http.StatusOK, true},
				{"GET", "/inventory", nil, http.StatusUnauthorized, false},
			},
		},
	}

	for _, tt := range tests {
		tokenString := newToken(jwt.MapClaims{"aud": jwtServiceID, "nbf": pastDate, "exp": futureDate, "scope": tt.scope}, keyA)
		keyForValidation := keyA

		extractBearerToken = func(r *http.Request) (string, error) {
			return tokenString, nil
		}

		jwksKeyFunc = func(*jwt.Token) (any, error) {
			return &keyForValidation.PublicKey, nil
		}

		for _, req := range tt.requests {
			authorized := false
			wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authorized = true
			})
			protected := jwtValidationMiddleware(wrappedHandler)

			out := httpt.NewRecorder()
			var httpReq *http.Request
			if req.body != nil {
				body := bytes.NewReader(req.body)
				httpReq = httpt.NewRequest(req.method, req.endpoint, body)
			} else {
				httpReq = httpt.NewRequest(req.method, req.endpoint, nil)
			}
			httpReq = httpReq.WithContext(context.WithValue(httpReq.Context(), server.BearerAuthScopes, []string{}))
			protected.ServeHTTP(out, httpReq)

			if out.Result().StatusCode != req.expectedStatus || authorized != req.authorized {
				t.Fatalf("[%s] Unexpected result for request: %s %s\nExpected status: %d, Authorized: %v\nGot status: %d, Authorized: %v",
					tt.name, req.method, req.endpoint, req.expectedStatus, req.authorized, out.Result().StatusCode, authorized)
			}
		}
	}
}
