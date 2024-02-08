package remotes

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

func TestNewTmcRemote(t *testing.T) {
	root := "http://localhost:8000/"
	remote, err := NewTmcRemote(
		map[string]any{
			"type": "tmc",
			"loc":  root,
		}, NewRemoteSpec("remoteName"))
	assert.NoError(t, err)
	assert.Equal(t, root, remote.root)
	assert.Equal(t, NewRemoteSpec("remoteName"), remote.Spec())
}

func TestCreateTmcRemoteConfig(t *testing.T) {
	tests := []struct {
		strConf  string
		fileConf string
		expRoot  string
		expErr   bool
	}{
		{"http://localhost:8000/", "", "http://localhost:8000/", false},
		{"", ``, "", true},
		{"", `[]`, "", true},
		{"", `{}`, "", true},
		{"", `{"loc":{}}`, "", true},
		{"", `{"loc":"http://localhost:8000/"}`, "http://localhost:8000/", false},
		{"", `{"loc":"http://localhost:8000/", "type":"tmc"}`, "http://localhost:8000/", false},
		{"", `{"loc":"http://localhost:8000/", "type":"file"}`, "", true},
	}

	for i, test := range tests {
		cf, err := createTmcRemoteConfig(test.strConf, []byte(test.fileConf))
		if test.expErr {
			assert.Error(t, err, "error expected in test %d for %s %s", i, test.strConf, test.fileConf)
			continue
		} else {
			assert.NoError(t, err, "no error expected in test %d for %s %s", i, test.strConf, test.fileConf)
		}
		assert.Equalf(t, "tmc", cf[KeyRemoteType], "in test %d for %s %s", i, test.strConf, test.fileConf)
		assert.Equalf(t, test.expRoot, fmt.Sprintf("%v", cf[KeyRemoteLoc]), "in test %d for %s %s", i, test.strConf, test.fileConf)

	}
}

func TestTmcRemote_Fetch(t *testing.T) {
	const tmid = "manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json"
	const aid = "manufacturer/mpn/v1.0.0-20201205123243-c49617d2e4fc.tm.json"
	const tm = "{\"id\":\"manufacturer/mpn/v1.0.0-20201205123243-c49617d2e4fc.tm.json\"}"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/thing-models/"+tmid, r.URL.Path)
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(tm))
	}))
	defer srv.Close()

	config, err := createTmcRemoteConfig("", []byte(`{"loc":"`+srv.URL+`", "type":"tmc", "auth":{"bearer":"token123"}}`))
	assert.NoError(t, err)
	r, err := NewTmcRemote(config, NewRemoteSpec("nameless"))
	assert.NoError(t, err)
	actId, b, err := r.Fetch(tmid)
	assert.NoError(t, err)
	assert.Equal(t, aid, actId)
	assert.Equal(t, []byte(tm), b)
}

func TestTmcRemote_UpdateToc(t *testing.T) {
	config, _ := createTmcRemoteConfig("http://example.com", nil)
	r, err := NewTmcRemote(config, NewRemoteSpec("nameless"))
	assert.NoError(t, err)
	err = r.UpdateToc()
	assert.NoError(t, err)
}

func TestTmcRemote_List(t *testing.T) {
	_, inventory, _ := utils.ReadRequiredFile("../../test/data/remotes/inventory_response.json")
	_, inventorySingle, _ := utils.ReadRequiredFile("../../test/data/remotes/inventory_entry_response.json")

	type ht struct {
		name   string
		body   []byte
		status int
		search *model.SearchParams
		expUrl string
		expErr string
		expRes int
	}
	htc := make(chan ht, 1)
	defer close(htc)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := <-htc
		eu, _ := url.Parse(h.expUrl)
		assert.Equal(t, eu.RawPath, r.URL.RawPath)
		assert.Equal(t, eu.Path, r.URL.Path)
		assert.Equal(t, eu.Query(), r.URL.Query())
		w.WriteHeader(h.status)
		_, _ = w.Write(h.body)
	}))
	defer srv.Close()

	config, err := createTmcRemoteConfig(srv.URL, nil)
	assert.NoError(t, err)
	r, err := NewTmcRemote(config, NewRemoteSpec("nameless"))
	assert.NoError(t, err)

	tests := []ht{
		{
			name:   "plain",
			body:   inventory,
			status: http.StatusOK,
			search: nil,
			expUrl: "/inventory",
			expErr: "",
			expRes: 3,
		},
		{
			name:   "encodes search params",
			body:   inventory,
			status: http.StatusOK,
			search: &model.SearchParams{
				Author:       []string{"author1", "author2"},
				Manufacturer: []string{"manuf1", "man&uf2"},
				Mpn:          []string{"mpn"},
				Name:         "autho",
				Query:        "some string",
				Options:      model.SearchOptions{NameFilterType: model.PrefixMatch},
			},
			expUrl: "/inventory?filter.name=autho&filter.author=author1%2Cauthor2&filter.manufacturer=manuf1%2Cman%26uf2&filter.mpn=mpn&search=some+string",
			expErr: "",
			expRes: 3,
		},
		{
			name:   "ignores search params with name and full match",
			body:   inventorySingle,
			status: http.StatusOK,
			search: &model.SearchParams{
				Author:       []string{"author1", "author2"},
				Manufacturer: []string{"manuf1", "man&uf2"},
				Mpn:          []string{"mpn"},
				Name:         "corp/mpn",
				Query:        "some string",
				Options:      model.SearchOptions{NameFilterType: model.FullMatch},
			},
			expUrl: "/inventory/corp%2Fmpn",
			expErr: "",
			expRes: 1,
		},
		{
			name:   "retrieves single TM by name",
			body:   inventorySingle,
			status: http.StatusOK,
			search: &model.SearchParams{
				Name: "author/omnicorp/senseall",
			},
			expUrl: "/inventory/author%2Fomnicorp%2Fsenseall",
			expErr: "",
			expRes: 1,
		},
		{
			name:   "bad request",
			body:   []byte(`{"detail":"invalid search parameter"}`),
			status: http.StatusBadRequest,
			search: nil,
			expUrl: "/inventory",
			expErr: "invalid search parameter",
			expRes: 0,
		},
		{
			name:   "internal server error",
			body:   []byte(`{"detail":"something bad happened"}`),
			status: http.StatusInternalServerError,
			search: nil,
			expUrl: "/inventory",
			expErr: "something bad happened",
			expRes: 0,
		},
		{
			name:   "unexpected status",
			body:   []byte(`{"detail":"no coffee for you"}`),
			status: http.StatusTeapot,
			search: nil,
			expUrl: "/inventory",
			expErr: "received unexpected HTTP response",
			expRes: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			htc <- test
			sr, err := r.List(test.search)
			if test.expErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, test.expRes, len(sr.Entries))
				for _, e := range sr.Entries {
					for _, v := range e.Versions {
						assert.NotEmpty(t, v.TMID)
						assert.Equal(t, v.TMID, v.Links["content"])
					}
				}
			} else {
				assert.ErrorContains(t, err, test.expErr)
			}
		})
	}
}
func TestTmcRemote_Versions(t *testing.T) {
	_, versionsResp, _ := utils.ReadRequiredFile("../../test/data/remotes/inventory_entry_versions_response.json")

	type ht struct {
		name    string
		body    []byte
		status  int
		reqName string
		expUrl  string
		expErr  string
		expRes  int
	}
	htc := make(chan ht, 1)
	defer close(htc)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := <-htc
		eu, _ := url.Parse(h.expUrl)
		assert.Equal(t, eu.RawPath, r.URL.RawPath)
		w.WriteHeader(h.status)
		_, _ = w.Write(h.body)
	}))
	defer srv.Close()

	config, err := createTmcRemoteConfig(srv.URL, nil)
	assert.NoError(t, err)
	r, err := NewTmcRemote(config, NewRemoteSpec("nameless"))
	assert.NoError(t, err)

	tests := []ht{
		{
			name:    "plain",
			body:    versionsResp,
			status:  http.StatusOK,
			reqName: "author/manufacturer/mpn/folder",
			expUrl:  "/inventory/author%2Fmanufacturer%2Fmpn%2Ffolder/.versions",
			expErr:  "",
			expRes:  1,
		},
		{
			name:    "bad request",
			body:    []byte(`{"detail":"invalid name"}`),
			status:  http.StatusBadRequest,
			reqName: "zzzzzz",
			expUrl:  "/inventory/zzzzzz/.versions",
			expErr:  "invalid name",
			expRes:  0,
		},
		{
			name:    "internal server error",
			body:    []byte(`{"detail":"something bad happened"}`),
			status:  http.StatusInternalServerError,
			reqName: "author/manufacturer/mpn",
			expUrl:  "/inventory/author%2Fmanufacturer%2Fmpn/.versions",
			expErr:  "something bad happened",
			expRes:  0,
		},
		{
			name:    "unexpected status",
			body:    []byte(`{"detail":"no coffee for you"}`),
			status:  http.StatusTeapot,
			reqName: "author/manufacturer/mpn",
			expUrl:  "/inventory/author%2Fmanufacturer%2Fmpn/.versions",
			expErr:  "received unexpected HTTP response",
			expRes:  0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			htc <- test
			vs, err := r.Versions(test.reqName)
			if test.expErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, test.expRes, len(vs))
				assert.Equal(t, "omnicorp/lightall/v1.0.1-20240104165612-c81be4ed973d.tm.json", vs[0].Links["content"])
			} else {
				assert.ErrorContains(t, err, test.expErr)
			}
		})
	}
}
func TestTmcRemote_Push(t *testing.T) {
	_, pushBody, _ := utils.ReadRequiredFile("../../test/data/push/omnilamp.json")

	type ht struct {
		name     string
		respBody []byte
		status   int
		reqBody  []byte
		expErr   error
	}
	htc := make(chan ht, 1)
	defer close(htc)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := <-htc
		assert.Equal(t, "/thing-models", r.RequestURI)
		rBody, _ := io.ReadAll(r.Body)
		assert.Equal(t, h.reqBody, rBody)
		w.WriteHeader(h.status)
		_, _ = w.Write(h.respBody)
	}))
	defer srv.Close()

	config, err := createTmcRemoteConfig(srv.URL, nil)
	assert.NoError(t, err)
	r, err := NewTmcRemote(config, NewRemoteSpec("nameless"))
	assert.NoError(t, err)
	tmErr := &ErrTMExists{ExistingId: "omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb.tm.json"}

	tests := []ht{
		{
			name:     "plain",
			reqBody:  pushBody,
			respBody: nil,
			status:   http.StatusCreated,
			expErr:   nil,
		},
		{
			name:     "tm exists",
			reqBody:  pushBody,
			respBody: []byte(`{"detail":"` + tmErr.Error() + `"}`),
			status:   http.StatusConflict,
			expErr:   tmErr,
		},
		{
			name:     "bad request",
			reqBody:  pushBody,
			respBody: []byte(`{"detail":"bad request"}`),
			status:   http.StatusBadRequest,
			expErr:   errors.New("bad request"),
		},
		{
			name:     "internal server error",
			reqBody:  pushBody,
			respBody: []byte(`{"detail":"something bad happened"}`),
			status:   http.StatusInternalServerError,
			expErr:   errors.New("something bad happened"),
		},
		{
			name:     "unexpected status",
			reqBody:  pushBody,
			respBody: []byte(`{"detail":"no coffee for you"}`),
			status:   http.StatusTeapot,
			expErr:   errors.New("received unexpected HTTP response from remote TM catalog: 418 I'm a teapot"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			htc <- test
			err := r.Push(model.TMID{Name: "ignored in tmc remote"}, pushBody)
			if test.expErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, test.expErr, err)
			}
		})
	}
}

func TestTmcRemote_ListCompletions(t *testing.T) {

	type ht struct {
		name       string
		kind       string
		toComplete string
		status     int
		respBody   []byte
		expUrl     string
		expErr     error
		expComps   []string
	}
	htc := make(chan ht, 1)
	defer close(htc)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := <-htc
		eu, _ := url.Parse(h.expUrl)
		assert.Equal(t, eu.RequestURI(), r.RequestURI)
		assert.Equal(t, eu.Query(), r.URL.Query())
		w.WriteHeader(h.status)
		_, _ = w.Write(h.respBody)
	}))
	defer srv.Close()

	config, err := createTmcRemoteConfig(srv.URL, nil)
	assert.NoError(t, err)
	r, err := NewTmcRemote(config, NewRemoteSpec("nameless"))
	assert.NoError(t, err)

	tests := []ht{
		{
			name:       "invalid kind",
			kind:       "invalid",
			toComplete: "",
			status:     http.StatusBadRequest,
			respBody:   []byte(`{"detail":"` + "" + `"}`),
			expUrl:     "/.completions?kind=invalid&toComplete=",
			expErr:     ErrInvalidCompletionParams,
			expComps:   nil,
		},
		{
			name:       "names",
			kind:       "names",
			toComplete: "",
			status:     http.StatusOK,
			respBody:   []byte("abc\ndef\n"),
			expUrl:     "/.completions?kind=names&toComplete=",
			expErr:     nil,
			expComps:   []string{"abc", "def"},
		},
		{
			name:       "fetchNames",
			kind:       "fetchNames",
			toComplete: "abc:",
			status:     http.StatusOK,
			respBody:   []byte("abc:v1.0.2\nabc:v3.2.1\n"),
			expUrl:     "/.completions?kind=fetchNames&toComplete=abc%3A",
			expErr:     nil,
			expComps:   []string{"abc:v1.0.2", "abc:v3.2.1"},
		},
		{
			name:       "unexpected status",
			kind:       "names",
			toComplete: "",
			status:     http.StatusTeapot,
			respBody:   []byte(`{"detail":"something bad happened"}`),
			expUrl:     "/.completions?kind=names&toComplete=",
			expErr:     errors.New("received unexpected HTTP response from remote TM catalog: 418 I'm a teapot"),
			expComps:   nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			htc <- test

			cs, err := r.ListCompletions(test.kind, test.toComplete)
			if test.expErr == nil {
				assert.NoError(t, err)
				assert.Equal(t, test.expComps, cs)
			} else {
				assert.Equal(t, test.expErr, err)
			}
		})
	}
}
