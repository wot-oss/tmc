package repos

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

func TestNewHttpRepo(t *testing.T) {
	root := "http://localhost:8000/"
	repo, err := NewHttpRepo(
		map[string]any{
			"type": "http",
			"loc":  root,
		}, model.NewRepoSpec("repoName"))
	assert.NoError(t, err)
	assert.Equal(t, root, repo.root)
	assert.Equal(t, model.NewRepoSpec("repoName"), repo.Spec())
}

func TestCreateHttpRepoConfig(t *testing.T) {
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
		{"", `{"loc":"http://localhost:8000/", "type":"http"}`, "http://localhost:8000/", false},
		{"", `{"loc":"http://localhost:8000/", "type":"file"}`, "", true},
	}

	for i, test := range tests {
		cf, err := createHttpRepoConfig(test.strConf, []byte(test.fileConf))
		if test.expErr {
			assert.Error(t, err, "error expected in test %d for %s %s", i, test.strConf, test.fileConf)
			continue
		} else {
			assert.NoError(t, err, "no error expected in test %d for %s %s", i, test.strConf, test.fileConf)
		}
		assert.Equalf(t, "http", cf[KeyRepoType], "in test %d for %s %s", i, test.strConf, test.fileConf)
		assert.Equalf(t, test.expRoot, fmt.Sprintf("%v", cf[KeyRepoLoc]), "in test %d for %s %s", i, test.strConf, test.fileConf)

	}
}

func TestHttpRepo_Fetch(t *testing.T) {
	const tmid = "manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json"
	const aid = "manufacturer/mpn/v1.0.0-20201205123243-c49617d2e4fc.tm.json"
	const tm = "{\"id\":\"manufacturer/mpn/v1.0.0-20201205123243-c49617d2e4fc.tm.json\"}"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/"+tmid, r.URL.Path)
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(tm))
	}))
	defer srv.Close()

	config, err := createHttpRepoConfig("", []byte(`{"loc":"`+srv.URL+`", "type":"http", "auth":{"bearer":"token123"}}`))
	assert.NoError(t, err)
	r, err := NewHttpRepo(config, model.NewRepoSpec("nameless"))
	assert.NoError(t, err)
	actId, b, err := r.Fetch(context.Background(), tmid)
	assert.NoError(t, err)
	assert.Equal(t, aid, actId)
	assert.Equal(t, []byte(tm), b)
}

func TestHttpRepo_ListCompletions(t *testing.T) {
	_, idx, err := utils.ReadRequiredFile("../../test/data/list/tm-catalog.toc.json")
	assert.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/.tmc/"+IndexFilename, r.URL.Path)
		_, _ = w.Write(idx)
	}))
	defer srv.Close()
	config, err := createHttpRepoConfig("", []byte(`{"loc":"`+srv.URL+`", "type":"http", "auth":{"bearer":"token123"}}`))
	assert.NoError(t, err)
	r, err := NewHttpRepo(config, model.NewRepoSpec("nameless"))
	assert.NoError(t, err)

	t.Run("names", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, "")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/"}, completions)
		})
		t.Run("some letters", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, "om")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/"}, completions)
		})
		t.Run("some letters non existing", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, "aaa")
			assert.NoError(t, err)
			var expRes []string
			assert.Equal(t, expRes, completions)
		})
		t.Run("full first name part", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, "omnicorp-r-d-research/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/"}, completions)
		})
		t.Run("some letters second part", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, "omnicorp-r-d-research/omnicorp")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/"}, completions)
		})
		t.Run("full second part", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall", "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/"}, completions)
		})
		t.Run("full third part", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/a/", "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/subpath"}, completions)
		})
		t.Run("full fourth part", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/a/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/a/b"}, completions)
		})
		t.Run("full name", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/subpath")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/subpath"}, completions)
		})
	})
	t.Run("fetch names", func(t *testing.T) {
		completions, err := r.ListCompletions(context.Background(), CompletionKindFetchNames, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall:")
		assert.NoError(t, err)

		slices.Sort(completions)
		assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall:1.0.1"}, completions)

		completions, err = r.ListCompletions(context.Background(), CompletionKindFetchNames, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall:v1")
		assert.NoError(t, err)

		slices.Sort(completions)
		assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall:1.0.1"}, completions)
	})
}

func TestHttpRepo_AnalyzeIndex(t *testing.T) {
	// given: a Http Repo
	config, err := createHttpRepoConfig("http://example.com", nil)
	assert.NoError(t, err)
	r, err := NewHttpRepo(config, model.NewRepoSpec("nameless"))
	assert.NoError(t, err)
	// when: AnalyzingIndex on the repo
	err = r.AnalyzeIndex(context.Background())
	// then: it returns NotSupported error
	assert.True(t, errors.Is(err, ErrNotSupported))
}

func TestHttpRepo_RangeResources(t *testing.T) {
	// given: a Http Repo
	config, err := createHttpRepoConfig("http://example.com", nil)
	assert.NoError(t, err)
	r, err := NewHttpRepo(config, model.NewRepoSpec("nameless"))
	assert.NoError(t, err)
	// when: RangeResources on the repo
	err = r.RangeResources(context.Background(), model.ResourceFilter{}, func(resource model.Resource, err error) bool {
		return true
	})
	// then: it returns NotSupported error
	assert.True(t, errors.Is(err, ErrNotSupported))
}
