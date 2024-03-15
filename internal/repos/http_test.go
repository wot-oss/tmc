package repos

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
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
	actId, b, err := r.Fetch(tmid)
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
			completions, err := r.ListCompletions(CompletionKindNames, "")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-R-D-research/"}, completions)
		})
		t.Run("some letters", func(t *testing.T) {
			completions, err := r.ListCompletions(CompletionKindNames, "om")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-R-D-research/"}, completions)
		})
		t.Run("some letters non existing", func(t *testing.T) {
			completions, err := r.ListCompletions(CompletionKindNames, "aaa")
			assert.NoError(t, err)
			var expRes []string
			assert.Equal(t, expRes, completions)
		})
		t.Run("full first name part", func(t *testing.T) {
			completions, err := r.ListCompletions(CompletionKindNames, "omnicorp-R-D-research/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/"}, completions)
		})
		t.Run("some letters second part", func(t *testing.T) {
			completions, err := r.ListCompletions(CompletionKindNames, "omnicorp-R-D-research/omnicorp")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/"}, completions)
		})
		t.Run("full second part", func(t *testing.T) {
			completions, err := r.ListCompletions(CompletionKindNames, "omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall", "omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall/"}, completions)
		})
		t.Run("full third part", func(t *testing.T) {
			completions, err := r.ListCompletions(CompletionKindNames, "omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall/a/", "omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall/subpath"}, completions)
		})
		t.Run("full fourth part", func(t *testing.T) {
			completions, err := r.ListCompletions(CompletionKindNames, "omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall/a/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall/a/b"}, completions)
		})
		t.Run("full name", func(t *testing.T) {
			completions, err := r.ListCompletions(CompletionKindNames, "omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall/subpath")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall/subpath"}, completions)
		})
	})
	t.Run("fetch names", func(t *testing.T) {
		completions, err := r.ListCompletions(CompletionKindFetchNames, "omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall:")
		assert.NoError(t, err)

		slices.Sort(completions)
		assert.Equal(t, []string{"omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall:1.0.1"}, completions)

		completions, err = r.ListCompletions(CompletionKindFetchNames, "omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall:v1")
		assert.NoError(t, err)

		slices.Sort(completions)
		assert.Equal(t, []string{"omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall:1.0.1"}, completions)
	})
}
