package cors

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/oapi-codegen/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCORSOptions(t *testing.T) {

	underTest := CORSOptions{}

	// when: adding origins
	underTest.AddAllowedOrigins("http://example.org", "", "*", "https://example.org:8080", "http://example.org")
	// then: duplicates and empty ones are removed
	assert.Equal(t, []string{"http://example.org", "*", "https://example.org:8080"}, underTest.allowedOrigins)

	// when: adding headers
	underTest.AddAllowedHeaders("X-Api-Key", "", "Content-Type", "Content-Type")
	// then: duplicates and empty ones are removed
	assert.Equal(t, []string{"X-Api-Key", "Content-Type"}, underTest.allowedHeaders)

	assert.False(t, underTest.allowCredentials)
	underTest.AllowCredentials(true)
	assert.True(t, underTest.allowCredentials)

	assert.Equal(t, 0, underTest.maxAge)
	underTest.MaxAge(120)
	assert.Equal(t, 120, underTest.maxAge)
}

func TestWithCORS(t *testing.T) {
	// given: some CORS options to be set on the CORS middleware handler
	cOpts := CORSOptions{}
	cOpts.AddAllowedHeaders("X-Api-Key", "X-Bar")
	cOpts.AddAllowedOrigins("http://example.org", "https://sample.com")
	cOpts.AllowCredentials(true)
	cOpts.MaxAge(120)

	// and given: a http handler not CORS aware
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// when: setting CORS on the http handler
	corsHdl := Protect(hf, cOpts)

	immutable := reflect.ValueOf(corsHdl)
	corsOrigins := fmt.Sprintf("%v", reflect.Indirect(immutable).FieldByName("allowedOrigins"))
	corsHeaders := fmt.Sprintf("%v", reflect.Indirect(immutable).FieldByName("allowedHeaders"))
	corsCredentials := fmt.Sprintf("%v", reflect.Indirect(immutable).FieldByName("allowCredentials"))
	corsMaxAge := fmt.Sprintf("%v", reflect.Indirect(immutable).FieldByName("maxAge"))

	// then: origins are set correct on CORS middleware handler
	assert.Equal(t, "[http://example.org https://sample.com]", corsOrigins)
	// then: headers contain the default CORS allowed header, the manual allowed header and the default header set by WithCORS()
	assert.Equal(t, "[Accept Accept-Language Content-Language Origin X-Api-Key X-Bar Content-Type]", corsHeaders)
	// then: allow credentials is set correct on CORS middleware handler
	assert.Equal(t, "true", corsCredentials)
	// then: max age is set correct on CORS middleware handler
	assert.Equal(t, "120", corsMaxAge)
}

func TestWithCORSOnRequest(t *testing.T) {
	route := "/inventory"
	allowedOrigin := "http://example.org"
	notAllowedOrigin := "http://not-allowed.org"

	// given: CORS options with an allowed origin
	opts := CORSOptions{}
	opts.AddAllowedOrigins(allowedOrigin)
	opts.MaxAge(120)

	// and given: a http handler without CORS
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// when setting CORS on the handler
	hdl := Protect(hf, opts)

	t.Run("request with an allowed origin", func(t *testing.T) {
		// and when: making an options request
		rec := testutil.NewRequest().WithMethod(http.MethodOptions, route).
			WithHeader("origin", allowedOrigin).
			WithHeader("Access-Control-Request-Method", http.MethodPost).
			GoWithHTTPHandler(t, hdl).
			Recorder

		// then: the response header "Access-Control-Allow-Origin" contains the allowed origin
		assert.Equal(t, allowedOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
		// and then: the response header "Access-Control-Max-Age" is set
		assert.Equal(t, "120", rec.Header().Get("Access-Control-Max-Age"))
	})

	t.Run("request with a not allowed origin", func(t *testing.T) {
		// and when: making an options request
		rec := testutil.NewRequest().WithMethod(http.MethodOptions, route).
			WithHeader("origin", notAllowedOrigin).
			WithHeader("Access-Control-Request-Method", http.MethodPost).
			GoWithHTTPHandler(t, hdl).
			Recorder

		// then: the response header "Access-Control-Allow-Origin" is not present
		assert.Equal(t, "", rec.Header().Get("Access-Control-Allow-Origin"))
	})
}
