package repos

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/kinbiko/jsonassert"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/config"
	"github.com/wot-oss/tmc/internal/model"
)

func TestSaveConfigOverwritesOnlyRepos(t *testing.T) {
	temp, err := os.MkdirTemp("", "config")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(temp)
	orgDir := config.ConfigDir
	config.ConfigDir = temp
	defer func() { config.ConfigDir = orgDir }()

	configFile := filepath.Join(config.ConfigDir, "config.json")
	err = os.WriteFile(configFile, []byte(`{
  "log": true,
  "repos": {
    "local": {
      "type": "file",
      "loc": "/tmp/tmc"
    }
  }
}`), 0660)
	assert.NoError(t, err)

	viper.SetConfigFile(configFile)
	assert.NoError(t, viper.ReadInConfig())
	defer viper.Reset()

	viper.Set(config.KeyLogLevel, "")
	err = saveConfig(Config{
		"httprepo": map[string]any{
			"type": "http",
			"loc":  "http://example.com/",
		},
	})
	assert.NoError(t, err)
	file, err := os.ReadFile(configFile)
	assert.NoError(t, err)

	jsa := jsonassert.New(t)
	jsa.Assertf(string(file), `{
  "log": true,
  "repos": {
    "httprepo": {
      "type": "http",
      "loc": "http://example.com/"
    }
  }
}`)

}
func TestReadConfigOverwritesRemotesWithRepos(t *testing.T) {
	temp, err := os.MkdirTemp("", "config")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(temp)
	orgDir := config.ConfigDir
	config.ConfigDir = temp
	defer func() { config.ConfigDir = orgDir }()

	configFile := filepath.Join(config.ConfigDir, "config.json")
	err = os.WriteFile(configFile, []byte(`{
  "log": true,
  "remotes": {
    "local": {
      "type": "file",
      "loc": "/tmp/tmc"
    }
  }
}`), 0660)
	assert.NoError(t, err)

	viper.SetConfigFile(configFile)
	assert.NoError(t, viper.ReadInConfig())
	defer viper.Reset()

	conf, err := ReadConfig()
	assert.NoError(t, err)
	assert.Contains(t, conf, "local")

	file, err := os.ReadFile(configFile)
	assert.NoError(t, err)

	jsa := jsonassert.New(t)
	jsa.Assertf(string(file), `{
  "log": true,
  "repos": {
    "local": {
      "type": "file",
      "loc": "/tmp/tmc"
    }
  }
}`)

}

func TestRepoManager_All_And_Get(t *testing.T) {
	t.Run("invalid repo config", func(t *testing.T) {

		viper.Set(KeyRepos, map[string]any{
			"r1": map[string]string{
				"type": "file",
				"loc":  "somewhere",
			},
		})

		_, err := All()
		assert.Error(t, err)
		assert.ErrorContains(t, err, "invalid repo config")

	})
	const ur = "http://example.com/{{ID}}"

	t.Run("two repos", func(t *testing.T) {

		viper.Set(KeyRepos, map[string]any{
			"r1": map[string]any{
				"type": "file",
				"loc":  "somewhere",
			},
			"r2": map[string]any{
				"type": "http",
				"loc":  ur,
			},
		})

		t.Run("all", func(t *testing.T) {
			all, err := All()
			assert.NoError(t, err)
			assert.Len(t, all, 2)
			assert.NotEqual(t, -1, slices.IndexFunc(all, func(repo Repo) bool { return reflect.TypeOf(repo) == reflect.TypeOf(&FileRepo{}) }))
			assert.NotEqual(t, -1, slices.IndexFunc(all, func(repo Repo) bool { return reflect.TypeOf(repo) == reflect.TypeOf(&HttpRepo{}) }))
		})
		t.Run("file repo", func(t *testing.T) {
			fr, err := Get(model.NewRepoSpec("r1"))
			assert.NoError(t, err)
			assert.Equal(t, &FileRepo{
				root: "somewhere",
				spec: model.NewRepoSpec("r1"),
			}, fr)

		})
		t.Run("http repo", func(t *testing.T) {
			hr, err := Get(model.NewRepoSpec("r2"))
			assert.NoError(t, err)
			u, _ := url.Parse(ur)
			assert.Equal(t, &HttpRepo{
				templatedPath:  true,
				templatedQuery: false,
				baseHttpRepo: baseHttpRepo{
					root:       ur,
					parsedRoot: u,
					spec:       model.NewRepoSpec("r2"),
				},
			}, hr)
		})
		t.Run("ad-hoc repo", func(t *testing.T) {
			ar, err := Get(model.NewDirSpec("directory"))
			assert.NoError(t, err)
			assert.Equal(t, &FileRepo{
				root: "directory",
				spec: model.NewDirSpec("directory"),
			}, ar)
		})

		t.Run("invalid spec", func(t *testing.T) {
			_, err := model.NewSpec("name", "dir")
			assert.Error(t, err)
		})

	})

	t.Run("one enabled repo", func(t *testing.T) {
		viper.Set(KeyRepos, map[string]any{
			"r1": map[string]any{
				"type": "file",
				"loc":  "somewhere",
			},
			"r2": map[string]any{
				"type":    "http",
				"loc":     ur,
				"enabled": false,
			},
		})
		t.Run("all", func(t *testing.T) {
			all, err := All()
			assert.NoError(t, err)
			assert.Len(t, all, 1)
			assert.NotEqual(t, -1, slices.IndexFunc(all, func(repo Repo) bool { return reflect.TypeOf(repo) == reflect.TypeOf(&FileRepo{}) }))
		})
		t.Run("named file repo", func(t *testing.T) {
			fr, err := Get(model.NewRepoSpec("r1"))
			assert.NoError(t, err)
			assert.Equal(t, &FileRepo{
				root: "somewhere",
				spec: model.NewRepoSpec("r1"),
			}, fr)

		})
		t.Run("empty spec", func(t *testing.T) {
			fr, err := Get(model.EmptySpec)
			assert.NoError(t, err)
			assert.Equal(t, &FileRepo{
				root: "somewhere",
				spec: model.NewRepoSpec("r1"),
			}, fr)

		})
		t.Run("http repo", func(t *testing.T) {
			_, err := Get(model.NewRepoSpec("r2"))
			assert.ErrorIs(t, err, ErrRepoNotFound)
		})

	})
	t.Run("two enabled repos", func(t *testing.T) {
		viper.Set(KeyRepos, map[string]any{
			"r1": map[string]any{
				"type": "file",
				"loc":  "somewhere",
			},
			"r2": map[string]any{
				"type":    "http",
				"loc":     ur,
				"enabled": false,
			},
			"r3": map[string]any{
				"type": "file",
				"loc":  "somewhere/else",
			},
		})
		t.Run("all", func(t *testing.T) {
			all, err := All()
			assert.NoError(t, err)
			assert.Len(t, all, 2)
			assert.NotEqual(t, -1, slices.IndexFunc(all, func(repo Repo) bool { return reflect.TypeOf(repo) == reflect.TypeOf(&FileRepo{}) }))
		})
		t.Run("named file repo", func(t *testing.T) {
			fr, err := Get(model.NewRepoSpec("r3"))
			assert.NoError(t, err)
			assert.Equal(t, &FileRepo{
				root: "somewhere/else",
				spec: model.NewRepoSpec("r3"),
			}, fr)

		})
		t.Run("empty spec", func(t *testing.T) {
			_, err := Get(model.EmptySpec)
			assert.ErrorIs(t, err, ErrAmbiguous)
		})
		t.Run("http repo", func(t *testing.T) {
			_, err := Get(model.NewRepoSpec("r2"))
			assert.ErrorIs(t, err, ErrRepoNotFound)
		})

	})
	t.Run("no enabled repos", func(t *testing.T) {
		viper.Set(KeyRepos, map[string]any{
			"r1": map[string]any{
				"type":    "file",
				"loc":     "somewhere",
				"enabled": false,
			},
			"r2": map[string]any{
				"type":    "http",
				"loc":     ur,
				"enabled": false,
			},
		})
		t.Run("all", func(t *testing.T) {
			all, err := All()
			assert.NoError(t, err)
			assert.Len(t, all, 0)
		})
		t.Run("named file repo", func(t *testing.T) {
			_, err := Get(model.NewRepoSpec("r1"))
			assert.ErrorIs(t, err, ErrRepoNotFound)
		})
		t.Run("local repo", func(t *testing.T) {
			ar, err := Get(model.NewDirSpec("directory"))
			assert.NoError(t, err)
			assert.Equal(t, &FileRepo{
				root: "directory",
				spec: model.NewDirSpec("directory"),
			}, ar)
		})
		t.Run("empty spec", func(t *testing.T) {
			_, err := Get(model.EmptySpec)
			assert.ErrorIs(t, err, ErrRepoNotFound)
		})
	})
}

func TestGetSpecdOrAll(t *testing.T) {
	viper.Set(KeyRepos, map[string]any{
		"r1": map[string]any{
			"type": "file",
			"loc":  "somewhere",
		},
		"r2": map[string]any{
			"type": "file",
			"loc":  "somewhere-else",
		},
	})

	// check if all repos are returned, when passing EmptySpec
	all, err := GetSpecdOrAll(model.EmptySpec)
	assert.NoError(t, err)
	expLocs := []string{"somewhere", "somewhere-else"}
	for _, r := range all.rs {
		if fr, ok := r.(*FileRepo); assert.True(t, ok) {
			idx := slices.IndexFunc(expLocs, func(s string) bool { return s == fr.root })
			if assert.Greater(t, idx, -1) {
				expLocs = slices.Delete(expLocs, idx, idx+1)
			}
		}
	}
	assert.Len(t, expLocs, 0) // no locations remained that were not found

	all, err = GetSpecdOrAll(model.NewRepoSpec("r1"))
	assert.NoError(t, err)
	if assert.Len(t, all.rs, 1) {
		if r1, ok := all.rs[0].(*FileRepo); assert.True(t, ok) {
			assert.Equal(t, "somewhere", r1.root)
		}
	}

	// get local repo
	all, err = GetSpecdOrAll(model.NewDirSpec("dir1"))
	assert.NoError(t, err)
	if assert.Len(t, all.rs, 1) {
		if r1, ok := all.rs[0].(*FileRepo); assert.True(t, ok) {
			assert.Equal(t, "dir1", r1.root)
		}
	}

}

func TestGet_SplitsSubRepoName(t *testing.T) {
	viper.Set(KeyRepos, map[string]any{
		"r1": map[string]any{
			"type": "tmc",
			"loc":  "http://example.com/tmc",
		},
	})

	t.Run("with empty repo name", func(t *testing.T) {
		repo, err := Get(model.NewRepoSpec(""))
		assert.NoError(t, err)
		tmcRepo, ok := repo.(*TmcRepo)
		assert.True(t, ok)
		assert.Equal(t, "r1", tmcRepo.Spec().RepoName())
		assert.Equal(t, "", tmcRepo.subRepo)
	})

	t.Run("with simple repo name", func(t *testing.T) {
		repo, err := Get(model.NewRepoSpec("r1"))
		assert.NoError(t, err)
		tmcRepo, ok := repo.(*TmcRepo)
		assert.True(t, ok)
		assert.Equal(t, "r1", tmcRepo.Spec().RepoName())
		assert.Equal(t, "", tmcRepo.subRepo)
	})

	t.Run("with subrepo", func(t *testing.T) {
		repo, err := Get(model.NewRepoSpec("r1/child"))
		assert.NoError(t, err)
		tmcRepo, ok := repo.(*TmcRepo)
		assert.True(t, ok)
		assert.Equal(t, "r1", tmcRepo.Spec().RepoName())
		assert.Equal(t, "child", tmcRepo.subRepo)
	})

	t.Run("with chained subrepo", func(t *testing.T) {
		repo, err := Get(model.NewRepoSpec("r1/child/grandchild"))
		assert.NoError(t, err)
		tmcRepo, ok := repo.(*TmcRepo)
		assert.True(t, ok)
		assert.Equal(t, "r1", tmcRepo.Spec().RepoName())
		assert.Equal(t, "child/grandchild", tmcRepo.subRepo)
	})

}

func TestGetDescriptions(t *testing.T) {
	rdComparer := func(a, b model.RepoDescription) int {
		return strings.Compare(a.Name, b.Name)
	}

	t.Run("without tmc repos", func(t *testing.T) {
		viper.Set(KeyRepos, map[string]any{
			"r1": map[string]any{
				"type":        "file",
				"loc":         "somewhere",
				"description": "r1 description",
			},
			"r2": map[string]any{
				"type":        "file",
				"loc":         "somewhere-else",
				"description": "r2 description",
			},
		})

		t.Run("with dir spec", func(t *testing.T) {
			ds, err := GetDescriptions(context.Background(), model.NewDirSpec("somewhere"))
			assert.NoError(t, err)
			assert.Len(t, ds, 0)
		})
		t.Run("with single file repo", func(t *testing.T) {
			ds, err := GetDescriptions(context.Background(), model.NewRepoSpec("r1"))
			assert.NoError(t, err)
			expDs := []model.RepoDescription{{Name: "r1", Type: "file", Description: "r1 description"}}
			slices.SortFunc(ds, rdComparer)
			assert.Equal(t, expDs, ds)

		})
		t.Run("with empty spec", func(t *testing.T) {
			ds, err := GetDescriptions(context.Background(), model.EmptySpec)
			assert.NoError(t, err)
			expDs := []model.RepoDescription{{Name: "r1", Type: "file", Description: "r1 description"}, {Name: "r2", Type: "file", Description: "r2 description"}}
			slices.SortFunc(ds, rdComparer)
			assert.Equal(t, expDs, ds)
		})

	})
	t.Run("with a tmc repo", func(t *testing.T) {
		type ht struct {
			name      string
			spec      model.RepoSpec
			status    int
			respBody  []byte
			expErr    error
			expDescrs []model.RepoDescription
		}
		htc := make(chan ht, 1)
		defer close(htc)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := <-htc
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/repos", r.URL.Path)
			w.WriteHeader(h.status)
			_, _ = w.Write(h.respBody)
		}))
		defer srv.Close()
		viper.Set(KeyRepos, map[string]any{
			"r1": map[string]any{
				"type":        "file",
				"loc":         "somewhere",
				"description": "r1 description",
			},
			"r2": map[string]any{
				"type":        "tmc",
				"loc":         srv.URL,
				"description": "r2 description",
			},
		})

		tests := []ht{
			{
				name:      "with empty spec and error expanding",
				spec:      model.EmptySpec,
				status:    http.StatusBadGateway,
				expErr:    &RepoAccessError{model.NewRepoSpec("r2"), errors.New("oops")},
				expDescrs: nil,
				respBody:  []byte(`{"detail":"oops"}`),
			},
			{
				name:      "with empty spec and two sources",
				spec:      model.EmptySpec,
				status:    http.StatusOK,
				expErr:    nil,
				expDescrs: []model.RepoDescription{{Name: "r1", Type: "file", Description: "r1 description"}, {Name: "r2/r2-1", Description: "r2-1 description"}, {Name: "r2/r2-2", Description: "r2-2 description"}},
				respBody:  []byte(`{"data": [{"name": "r2-1", "description": "r2-1 description"}, {"name": "r2-2", "description": "r2-2 description"}]}`),
			},
			{
				name:      "with empty spec and one source",
				spec:      model.EmptySpec,
				status:    http.StatusOK,
				expErr:    nil,
				expDescrs: []model.RepoDescription{{Name: "r1", Type: "file", Description: "r1 description"}, {Name: "r2", Type: "tmc", Description: "r2 description"}},
				respBody:  []byte(`{"data": [{"name": "r2-1", "description": "r2-1 description"}]}`), // actually, the tmc api should return empty list in this case, but the client side should be able to handle this anyway
			},
			{
				name:      "with empty spec and no named sources",
				spec:      model.EmptySpec,
				status:    http.StatusOK,
				expErr:    nil,
				expDescrs: []model.RepoDescription{{Name: "r1", Type: "file", Description: "r1 description"}, {Name: "r2", Type: "tmc", Description: "r2 description"}},
				respBody:  []byte(`{"data": []}`),
			},
			{
				name:      "with repo spec and two sources",
				spec:      model.NewRepoSpec("r2"),
				status:    http.StatusOK,
				expErr:    nil,
				expDescrs: []model.RepoDescription{{Name: "r2/r2-1", Description: "r2-1 description"}, {Name: "r2/r2-2", Description: "r2-2 description"}},
				respBody:  []byte(`{"data": [{"name": "r2-1", "description": "r2-1 description"}, {"name": "r2-2", "description": "r2-2 description"}]}`),
			},
			{
				name:      "with repo spec and one source",
				spec:      model.NewRepoSpec("r2"),
				status:    http.StatusOK,
				expErr:    nil,
				expDescrs: []model.RepoDescription{{Name: "r2", Type: "tmc", Description: "r2 description"}},
				respBody:  []byte(`{"data": [{"name": "r2-1", "description": "r2-1 description"}]}`), // actually, the tmc api should return empty list in this case, but the client side should be able to handle this anyway
			},
			{
				name:      "with repo spec and no named sources",
				spec:      model.NewRepoSpec("r2"),
				status:    http.StatusOK,
				expErr:    nil,
				expDescrs: []model.RepoDescription{{Name: "r2", Type: "tmc", Description: "r2 description"}},
				respBody:  []byte(`{"data": []}`),
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				htc <- test

				ds, err := GetDescriptions(context.Background(), test.spec)
				if test.expErr == nil {
					assert.NoError(t, err)
					slices.SortFunc(ds, rdComparer)
					slices.SortFunc(test.expDescrs, rdComparer)
					assert.Equal(t, test.expDescrs, ds)
				} else {
					assert.Equal(t, test.expErr, err)
				}
			})
		}
	})
}
