package repos

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

func TestNewTmcRepo(t *testing.T) {
	t.Run("with url", func(t *testing.T) {
		root := "http://localhost:8000/"
		repo, err := NewTmcRepo(
			map[string]any{
				"type": "tmc",
				"loc":  root,
			}, model.NewRepoSpec("repoName"))
		assert.NoError(t, err)
		assert.Equal(t, root, repo.root)
		assert.Equal(t, model.NewRepoSpec("repoName"), repo.Spec())
	})
	t.Run("with env var", func(t *testing.T) {
		root := "http://localhost:8000/"
		os.Setenv("TMC_TEST_ENV_VAR_URL", root)
		defer os.Unsetenv("TMC_TEST_ENV_VAR_URL")
		repo, err := NewTmcRepo(
			map[string]any{
				"type": "tmc",
				"loc":  "$TMC_TEST_ENV_VAR_URL",
			}, model.NewRepoSpec("repoName"))
		assert.NoError(t, err)
		assert.Equal(t, root, repo.root)
		assert.Equal(t, model.NewRepoSpec("repoName"), repo.Spec())
	})
}

func TestCreateTmcRepoConfig(t *testing.T) {
	tests := []struct {
		fileConf string
		expRoot  string
		expErr   bool
	}{
		{``, "", true},
		{`[]`, "", true},
		{`{}`, "", true},
		{`{"loc":{}}`, "", true},
		{`{"loc":"http://localhost:8000/"}`, "http://localhost:8000/", false},
		{`{"loc":"http://localhost:8000/", "type":"tmc"}`, "http://localhost:8000/", false},
		{`{"loc":"$TMC_TEST_REPO_URL", "type":"tmc"}`, "$TMC_TEST_REPO_URL", false},
		{`{"loc":"http://localhost:8000/", "type":"file"}`, "", true},
	}

	for i, test := range tests {
		cf, err := createTmcRepoConfig([]byte(test.fileConf))
		if test.expErr {
			assert.Error(t, err, "error expected in test %d for %s", i, test.fileConf)
			continue
		} else {
			assert.NoError(t, err, "no error expected in test %d for %s", i, test.fileConf)
		}
		assert.Equalf(t, "tmc", cf[KeyRepoType], "in test %d for %s", i, test.fileConf)
		assert.Equalf(t, test.expRoot, fmt.Sprintf("%v", cf[KeyRepoLoc]), "in test %d for %s", i, test.fileConf)

	}
}

func TestTmcRepo_Fetch(t *testing.T) {
	const tmid = "manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json"
	const aid = "manufacturer/mpn/v1.0.0-20201205123243-c49617d2e4fc.tm.json"
	const tm = "{\"id\":\"manufacturer/mpn/v1.0.0-20201205123243-c49617d2e4fc.tm.json\"}"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/thing-models/"+tmid, r.URL.Path)
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(tm))
	}))
	defer srv.Close()

	config, err := createTmcRepoConfig([]byte(`{"loc":"` + srv.URL + `", "type":"tmc", "auth":{"bearer":"token123"}}`))
	assert.NoError(t, err)
	r, err := NewTmcRepo(config, model.NewRepoSpec("nameless"))
	assert.NoError(t, err)
	actId, b, err := r.Fetch(context.Background(), tmid)
	assert.NoError(t, err)
	assert.Equal(t, aid, actId)
	assert.Equal(t, []byte(tm), b)
}

func TestTmcRepo_UpdateIndex(t *testing.T) {
	config, _ := createTmcRepoConfig([]byte(`{"loc":"http://example.com", "type":"tmc"}`))
	r, err := NewTmcRepo(config, model.NewRepoSpec("nameless"))
	assert.NoError(t, err)
	err = r.Index(context.Background())
	assert.NoError(t, err)
}

func TestTmcRepo_List(t *testing.T) {
	_, inventory, _ := utils.ReadRequiredFile("../../test/data/repos/inventory_response.json")
	_, inventorySingle, _ := utils.ReadRequiredFile("../../test/data/repos/inventory_entry_response.json")

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

	config, err := createTmcRepoConfig([]byte(`{"loc":"` + srv.URL + `"}`))
	assert.NoError(t, err)
	config[keySubRepo] = "child"
	r, err := NewTmcRepo(config, model.NewRepoSpec("nameless"))
	assert.NoError(t, err)

	tests := []ht{
		{
			name:   "plain",
			body:   inventory,
			status: http.StatusOK,
			search: nil,
			expUrl: "/inventory?repo=child",
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
			expUrl: "/inventory?filter.name=autho&filter.author=author1%2Cauthor2&filter.manufacturer=manuf1%2Cman%26uf2&filter.mpn=mpn&search=some+string&repo=child",
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
				Name:         "author/corp/mpn",
				Query:        "some string",
				Options:      model.SearchOptions{NameFilterType: model.FullMatch},
			},
			expUrl: "/inventory/.tmName/author%2Fcorp%2Fmpn?repo=child",
			expErr: "",
			expRes: 1,
		},
		{
			name:   "retrieves single TM name entry",
			body:   inventorySingle,
			status: http.StatusOK,
			search: &model.SearchParams{
				Name: "author/omnicorp/senseall",
			},
			expUrl: "/inventory/.tmName/author%2Fomnicorp%2Fsenseall?repo=child",
			expErr: "",
			expRes: 1,
		},
		{
			name:   "bad request",
			body:   []byte(`{"detail":"invalid search parameter"}`),
			status: http.StatusBadRequest,
			search: nil,
			expUrl: "/inventory?repo=child",
			expErr: "invalid search parameter",
			expRes: 0,
		},
		{
			name:   "internal server error",
			body:   []byte(`{"detail":"something bad happened"}`),
			status: http.StatusInternalServerError,
			search: nil,
			expUrl: "/inventory?repo=child",
			expErr: "something bad happened",
			expRes: 0,
		},
		{
			name:   "unexpected status",
			body:   []byte(`{"detail":"no coffee for you"}`),
			status: http.StatusTeapot,
			search: nil,
			expUrl: "/inventory?repo=child",
			expErr: "received unexpected HTTP response",
			expRes: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			htc <- test
			sr, err := r.List(context.Background(), test.search)
			if test.expErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, test.expRes, len(sr.Entries))
				if reflect.DeepEqual(test.body, inventory) {
					assert.Equal(t, time.Date(2024, 12, 1, 10, 0, 0, 0, time.UTC), sr.LastUpdated)
				}
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

func TestTmcRepo_Versions(t *testing.T) {
	_, versionsResp, _ := utils.ReadRequiredFile("../../test/data/repos/inventory_entry_metadata.json")

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

	config, err := createTmcRepoConfig([]byte(`{"loc":"` + srv.URL + `"}`))
	assert.NoError(t, err)
	r, err := NewTmcRepo(config, model.NewRepoSpec("nameless"))
	assert.NoError(t, err)

	tests := []ht{
		{
			name:    "plain",
			body:    versionsResp,
			status:  http.StatusOK,
			reqName: "author/manufacturer/mpn/folder",
			expUrl:  "/inventory/.tmName/author%2Fmanufacturer%2Fmpn%2Ffolder",
			expErr:  "",
			expRes:  1,
		},
		{
			name:    "bad request",
			body:    []byte(`{"detail":"invalid name"}`),
			status:  http.StatusBadRequest,
			reqName: "zzzzzz",
			expUrl:  "/inventory/.tmName/zzzzzz",
			expErr:  "invalid name",
			expRes:  0,
		},
		{
			name:    "internal server error",
			body:    []byte(`{"detail":"something bad happened"}`),
			status:  http.StatusInternalServerError,
			reqName: "author/manufacturer/mpn",
			expUrl:  "/inventory/.tmName/author%2Fmanufacturer%2Fmpn",
			expErr:  "something bad happened",
			expRes:  0,
		},
		{
			name:    "unexpected status",
			body:    []byte(`{"detail":"no coffee for you"}`),
			status:  http.StatusTeapot,
			reqName: "author/manufacturer/mpn",
			expUrl:  "/inventory/.tmName/author%2Fmanufacturer%2Fmpn",
			expErr:  "received unexpected HTTP response",
			expRes:  0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			htc <- test
			vs, err := r.Versions(context.Background(), test.reqName)
			if test.expErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, test.expRes, len(vs))
				assert.Equal(t, "omnicorp/omnicorp/lightall/v1.0.1-20240606105140-5a3840060b05.tm.json", vs[0].Links["content"])
			} else {
				assert.ErrorContains(t, err, test.expErr)
			}
		})
	}
}

func TestTmcRepo_GetTMMetadata(t *testing.T) {
	_, tmMetaResp, _ := utils.ReadRequiredFile("../../test/data/repos/inventory_tmid_metadata.json")

	type ht struct {
		name   string
		body   []byte
		status int
		tmId   string
		expUrl string
		expErr string
		expRes []string
	}
	htc := make(chan ht, 1)
	defer close(htc)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := <-htc
		eu, _ := url.Parse(h.expUrl)
		assert.Equal(t, eu.Path, r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(h.status)
		_, _ = w.Write(h.body)
	}))
	defer srv.Close()

	config, err := createTmcRepoConfig([]byte(`{"loc":"` + srv.URL + `"}`))
	assert.NoError(t, err)
	r, err := NewTmcRepo(config, model.NewRepoSpec("nameless"))
	assert.NoError(t, err)

	tests := []ht{
		{
			name:   "tmid",
			body:   tmMetaResp,
			status: http.StatusOK,
			tmId:   "omniauthor/omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb.tm.json",
			expUrl: "/inventory/omniauthor/omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb.tm.json",
			expErr: "",
			expRes: []string{"README.md", "User Guide.pdf"},
		},
		{
			name:   "bad request",
			body:   []byte(`{"detail":"invalid id"}`),
			status: http.StatusBadRequest,
			tmId:   "omniauthor/omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb",
			expUrl: "/inventory/omniauthor/omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb",
			expErr: "id or name invalid",
			expRes: nil,
		},
		{
			name:   "not found",
			body:   []byte(`{"detail":"TM not found", "code": "TM"}`),
			status: http.StatusNotFound,
			tmId:   "omniauthor/omnicorp/senseall/v8.0.0-20231230153548-243d1b462bbb.tm.json",
			expUrl: "/inventory/omniauthor/omnicorp/senseall/v8.0.0-20231230153548-243d1b462bbb.tm.json",
			expErr: "TM not found",
			expRes: nil,
		},
		{
			name:   "internal server error",
			body:   []byte(`{"detail":"something bad happened"}`),
			status: http.StatusInternalServerError,
			tmId:   "omniauthor/omnicorp/senseall/v8.0.0-20231230153548-243d1b462bbb.tm.json",
			expUrl: "/inventory/omniauthor/omnicorp/senseall/v8.0.0-20231230153548-243d1b462bbb.tm.json",
			expErr: "something bad happened",
			expRes: nil,
		},
		{
			name:   "unexpected status",
			body:   []byte(`{"detail":"no coffee for you"}`),
			status: http.StatusTeapot,
			tmId:   "omniauthor/omnicorp/senseall/v8.0.0-20231230153548-243d1b462bbb.tm.json",
			expUrl: "/inventory/omniauthor/omnicorp/senseall/v8.0.0-20231230153548-243d1b462bbb.tm.json",
			expErr: "received unexpected HTTP response",
			expRes: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			htc <- test
			res, err := r.GetTMMetadata(context.Background(), test.tmId)
			if test.expErr == "" {
				assert.NoError(t, err)
				var attNames []string
				for _, v := range res {
					for _, a := range v.Attachments {
						attNames = append(attNames, a.Name)
					}
				}
				assert.Equal(t, test.expRes, attNames)
			} else {
				assert.ErrorContains(t, err, test.expErr)
			}
		})
	}
}
func TestTmcRepo_FetchAttachment(t *testing.T) {
	type ht struct {
		name       string
		body       []byte
		status     int
		tmNameOrId string
		expUrl     string
		expErr     string
		expRes     []byte
	}
	htc := make(chan ht, 1)
	defer close(htc)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := <-htc
		eu, _ := url.Parse(h.expUrl)
		assert.Equal(t, eu.Path, r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(h.status)
		_, _ = w.Write(h.body)
	}))
	defer srv.Close()

	config, err := createTmcRepoConfig([]byte(`{"loc":"` + srv.URL + `"}`))
	assert.NoError(t, err)
	r, err := NewTmcRepo(config, model.NewRepoSpec("nameless"))
	assert.NoError(t, err)

	tests := []ht{
		{
			name:       "tmname",
			body:       []byte("# README"),
			status:     http.StatusOK,
			tmNameOrId: "author/manufacturer/mpn",
			expUrl:     "/thing-models/author/manufacturer/mpn/.attachments/README.md",
			expErr:     "",
			expRes:     []byte("# README"),
		},
		{
			name:       "tmid",
			body:       []byte("# README"),
			status:     http.StatusOK,
			tmNameOrId: "omniauthor/omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb.tm.json",
			expUrl:     "/thing-models/omniauthor/omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb.tm.json/.attachments/README.md",
			expErr:     "",
			expRes:     []byte("# README"),
		},
		{
			name:       "bad request",
			body:       []byte(`{"detail":"invalid name"}`),
			status:     http.StatusBadRequest,
			tmNameOrId: "zzzzzz",
			expUrl:     "/thing-models/zzzzzz/.attachments/README.md",
			expErr:     "id or name invalid",
			expRes:     nil,
		},
		{
			name:       "not found",
			body:       []byte(`{"detail":"not found"}`),
			status:     http.StatusNotFound,
			tmNameOrId: "zzzzzz",
			expUrl:     "/thing-models/zzzzzz/.attachments/README.md",
			expErr:     "not found",
			expRes:     nil,
		},
		{
			name:       "internal server error",
			body:       []byte(`{"detail":"something bad happened"}`),
			status:     http.StatusInternalServerError,
			tmNameOrId: "author/manufacturer/mpn",
			expUrl:     "/thing-models/author/manufacturer/mpn/.attachments/README.md",
			expErr:     "something bad happened",
			expRes:     nil,
		},
		{
			name:       "unexpected status",
			body:       []byte(`{"detail":"no coffee for you"}`),
			status:     http.StatusTeapot,
			tmNameOrId: "author/manufacturer/mpn",
			expUrl:     "/thing-models/author/manufacturer/mpn/.attachments/README.md",
			expErr:     "received unexpected HTTP response",
			expRes:     nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			htc <- test
			content, err := r.FetchAttachment(context.Background(), model.NewTMIDAttachmentContainerRef(test.tmNameOrId), "README.md")
			if test.expErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, test.expRes, content)
			} else {
				assert.ErrorContains(t, err, test.expErr)
			}
		})
	}
}
func TestTmcRepo_DeleteAttachment(t *testing.T) {
	type ht struct {
		name       string
		body       []byte
		status     int
		tmNameOrId string
		expUrl     string
		expErr     string
	}
	htc := make(chan ht, 1)
	defer close(htc)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := <-htc
		eu, _ := url.Parse(h.expUrl)
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, eu.Path, r.URL.Path)
		w.WriteHeader(h.status)
		_, _ = w.Write(h.body)
	}))
	defer srv.Close()

	config, err := createTmcRepoConfig([]byte(`{"loc":"` + srv.URL + `"}`))
	assert.NoError(t, err)
	r, err := NewTmcRepo(config, model.NewRepoSpec("nameless"))
	assert.NoError(t, err)

	tests := []ht{
		{
			name:       "tmname",
			body:       nil,
			status:     http.StatusNoContent,
			tmNameOrId: "author/manufacturer/mpn",
			expUrl:     "/thing-models/author/manufacturer/mpn/.attachments/README.md",
			expErr:     "",
		},
		{
			name:       "tmid",
			body:       nil,
			status:     http.StatusNoContent,
			tmNameOrId: "omniauthor/omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb.tm.json",
			expUrl:     "/thing-models/omniauthor/omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb.tm.json/.attachments/README.md",
			expErr:     "",
		},
		{
			name:       "bad request",
			body:       []byte(`{"detail":"invalid name"}`),
			status:     http.StatusBadRequest,
			tmNameOrId: "zzzzzz",
			expUrl:     "/thing-models/zzzzzz/.attachments/README.md",
			expErr:     "id or name invalid",
		},
		{
			name:       "not found",
			body:       []byte(`{"detail":"TM not found", "code": "TM"}`),
			status:     http.StatusNotFound,
			tmNameOrId: "zzzzzz",
			expUrl:     "/thing-models/zzzzzz/.attachments/README.md",
			expErr:     "TM not found",
		},
		{
			name:       "internal server error",
			body:       []byte(`{"detail":"something bad happened"}`),
			status:     http.StatusInternalServerError,
			tmNameOrId: "author/manufacturer/mpn",
			expUrl:     "/thing-models/author/manufacturer/mpn/.attachments/README.md",
			expErr:     "something bad happened",
		},
		{
			name:       "unexpected status",
			body:       []byte(`{"detail":"no coffee for you"}`),
			status:     http.StatusTeapot,
			tmNameOrId: "author/manufacturer/mpn",
			expUrl:     "/thing-models/author/manufacturer/mpn/.attachments/README.md",
			expErr:     "received unexpected HTTP response",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			htc <- test
			err := r.DeleteAttachment(context.Background(), model.NewTMIDAttachmentContainerRef(test.tmNameOrId), "README.md")
			if test.expErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.expErr)
			}
		})
	}
}
func TestTmcRepo_ImportAttachment(t *testing.T) {
	type ht struct {
		name    string
		body    []byte
		status  int
		tmName  string
		tmID    string
		expUrl  string
		expErr  string
		reqBody []byte
	}
	htc := make(chan ht, 1)
	defer close(htc)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := <-htc
		eu, _ := url.Parse(h.expUrl)
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, eu.Path, r.URL.Path)
		rBody, _ := io.ReadAll(r.Body)
		assert.Equal(t, h.reqBody, rBody)
		w.WriteHeader(h.status)
		_, _ = w.Write(h.body)
	}))
	defer srv.Close()

	config, err := createTmcRepoConfig([]byte(`{"loc":"` + srv.URL + `"}`))
	assert.NoError(t, err)
	r, err := NewTmcRepo(config, model.NewRepoSpec("nameless"))
	assert.NoError(t, err)

	tests := []ht{
		{
			name:    "tmname",
			body:    nil,
			status:  http.StatusNoContent,
			tmName:  "author/manufacturer/mpn",
			expUrl:  "/thing-models/.tmName/author/manufacturer/mpn/.attachments/README.md",
			expErr:  "",
			reqBody: []byte("# README"),
		},
		{
			name:    "tmid",
			body:    nil,
			status:  http.StatusNoContent,
			tmID:    "omniauthor/omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb.tm.json",
			expUrl:  "/thing-models/omniauthor/omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb.tm.json/.attachments/README.md",
			expErr:  "",
			reqBody: []byte("# README"),
		},
		{
			name:    "bad request tmname",
			body:    []byte(`{"detail":"invalid name"}`),
			status:  http.StatusBadRequest,
			tmName:  "zzzzzz",
			expUrl:  "/thing-models/.tmName/zzzzzz/.attachments/README.md",
			expErr:  "id or name invalid",
			reqBody: []byte("# README"),
		},
		{
			name:    "bad request tmid",
			body:    []byte(`{"detail":"invalid name"}`),
			status:  http.StatusBadRequest,
			tmID:    "zzzzzz",
			expUrl:  "/thing-models/zzzzzz/.attachments/README.md",
			expErr:  "id or name invalid",
			reqBody: []byte("# README"),
		},
		{
			name:    "not found",
			body:    []byte(`{"detail":"not found", "code": "TM name"}`),
			status:  http.StatusNotFound,
			tmName:  "zzzzzz",
			expUrl:  "/thing-models/.tmName/zzzzzz/.attachments/README.md",
			expErr:  "TM name not found",
			reqBody: []byte("# README"),
		},
		{
			name:    "internal server error",
			body:    []byte(`{"detail":"something bad happened"}`),
			status:  http.StatusInternalServerError,
			tmName:  "author/manufacturer/mpn",
			expUrl:  "/thing-models/.tmName/author/manufacturer/mpn/.attachments/README.md",
			expErr:  "something bad happened",
			reqBody: []byte("# README"),
		},
		{
			name:    "attachment exists",
			body:    []byte(`{"detail":"attachment already exists"}`),
			status:  http.StatusConflict,
			tmName:  "author/manufacturer/mpn",
			expUrl:  "/thing-models/.tmName/author/manufacturer/mpn/.attachments/README.md",
			expErr:  "attachment already exists",
			reqBody: []byte("# README"),
		},
		{
			name:    "unexpected status",
			body:    []byte(`{"detail":"no coffee for you"}`),
			status:  http.StatusTeapot,
			tmName:  "author/manufacturer/mpn",
			expUrl:  "/thing-models/.tmName/author/manufacturer/mpn/.attachments/README.md",
			expErr:  "received unexpected HTTP response",
			reqBody: []byte("# README"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			htc <- test
			var ref model.AttachmentContainerRef
			if test.tmID != "" {
				ref = model.NewTMIDAttachmentContainerRef(test.tmID)
			} else {
				ref = model.NewTMNameAttachmentContainerRef(test.tmName)
			}
			err := r.ImportAttachment(context.Background(), ref, model.Attachment{Name: "README.md"}, test.reqBody, false)
			if test.expErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.expErr)
			}
		})
	}
}
func TestTmcRepo_Push(t *testing.T) {
	_, importBody, _ := utils.ReadRequiredFile("../../test/data/import/omnilamp.json")

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

	config, err := createTmcRepoConfig([]byte(`{"loc":"` + srv.URL + `"}`))
	assert.NoError(t, err)
	r, err := NewTmcRepo(config, model.NewRepoSpec("nameless"))
	assert.NoError(t, err)
	tmErr := &ErrTMIDConflict{Type: IdConflictSameContent, ExistingId: "omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb.tm.json"}

	tests := []ht{
		{
			name:     "plain",
			reqBody:  importBody,
			respBody: []byte(`{"data": {"tmID": "tmid"}}`),
			status:   http.StatusCreated,
			expErr:   nil,
		},
		{
			name:     "tm exists",
			reqBody:  importBody,
			respBody: []byte(`{"detail":"` + tmErr.Error() + `", "code": "` + tmErr.Code() + `"}`),
			status:   http.StatusConflict,
			expErr:   tmErr,
		},
		{
			name:     "bad request",
			reqBody:  importBody,
			respBody: []byte(`{"detail":"bad request"}`),
			status:   http.StatusBadRequest,
			expErr:   errors.New("received error response from remote tmc server: 400 Bad Request, bad request"),
		},
		{
			name:     "internal server error",
			reqBody:  importBody,
			respBody: []byte(`{"detail":"something bad happened"}`),
			status:   http.StatusInternalServerError,
			expErr:   errors.New("received error response from remote tmc server: 500 Internal Server Error, something bad happened"),
		},
		{
			name:     "unexpected status",
			reqBody:  importBody,
			respBody: []byte(`{"detail":"no coffee for you"}`),
			status:   http.StatusTeapot,
			expErr:   errors.New("received unexpected HTTP response from remote tmc server: 418 I'm a teapot"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			htc <- test
			_, err := r.Import(context.Background(), model.TMID{Name: "ignored in tmc repo"}, importBody, ImportOptions{})
			if test.expErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, test.expErr, err)
			}
		})
	}
}

func TestTmcRepo_ListCompletions(t *testing.T) {

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

	config, err := createTmcRepoConfig([]byte(`{"loc":"` + srv.URL + `"}`))
	assert.NoError(t, err)
	r, err := NewTmcRepo(config, model.NewRepoSpec("nameless"))
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

			cs, err := r.ListCompletions(context.Background(), test.kind, nil, test.toComplete)
			if test.expErr == nil {
				assert.NoError(t, err)
				assert.Equal(t, test.expComps, cs)
			} else {
				assert.Equal(t, test.expErr, err)
			}
		})
	}
}

func TestTmcRepo_Delete(t *testing.T) {
	type ht struct {
		name     string
		id       string
		status   int
		respBody []byte
		expErr   error
	}
	htc := make(chan ht, 1)
	defer close(htc)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := <-htc
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/thing-models/"+h.id, r.URL.Path)
		assert.Equal(t, url.Values{"force": []string{"true"}}, r.URL.Query())
		w.WriteHeader(h.status)
		_, _ = w.Write(h.respBody)
	}))
	defer srv.Close()

	config, err := createTmcRepoConfig([]byte(`{"loc":"` + srv.URL + `"}`))
	assert.NoError(t, err)
	r, err := NewTmcRepo(config, model.NewRepoSpec("nameless"))
	assert.NoError(t, err)

	tests := []ht{
		{
			name:     "invalid id",
			id:       "invalid-id",
			status:   http.StatusBadRequest,
			expErr:   model.ErrInvalidId,
			respBody: []byte(`{"detail":"id invalid"}`),
		},
		{
			name:     "non-existing id",
			id:       "omnicorp/lightall/v1.0.1-20240104165612-c81be4ed973d.tm.json",
			status:   http.StatusNotFound,
			respBody: []byte(`{"detail":"TM not found", "code": "TM"}`),
			expErr:   model.ErrTMNotFound,
		},
		{
			name:     "existing id",
			id:       "omnicorp/lightall/v1.0.1-20240104165612-c81be4ed973d.tm.json",
			status:   http.StatusNoContent,
			respBody: nil,
			expErr:   nil,
		},
		{
			name:     "internal error",
			id:       "omnicorp/lightall/v1.0.1-20240104165612-c81be4ed973d.tm.json",
			status:   http.StatusInternalServerError,
			respBody: []byte(`{"detail":"something bad happened"}`),
			expErr:   errors.New("something bad happened"),
		},
		{
			name:     "unexpected status",
			id:       "omnicorp/lightall/v1.0.1-20240104165612-c81be4ed973d.tm.json",
			status:   http.StatusTeapot,
			respBody: nil,
			expErr:   errors.New("received unexpected HTTP response from remote TM catalog: 418 I'm a teapot"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			htc <- test

			err := r.Delete(context.Background(), test.id)
			if test.expErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, test.expErr, err)
			}
		})
	}
}

func TestTmcRepo_CheckIntegrity(t *testing.T) {
	// given: a TMC Repo
	config, err := createTmcRepoConfig([]byte(`{"loc":"http://example.com"}`))
	assert.NoError(t, err)
	r, err := NewTmcRepo(config, model.NewRepoSpec("nameless"))
	assert.NoError(t, err)
	// when: AnalyzingIndex on the repo
	res, err := r.CheckIntegrity(context.Background(), nil)
	// then: it returns nil
	assert.Nil(t, res)
	assert.Nil(t, err)
}
