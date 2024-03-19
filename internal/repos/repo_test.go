package repos

import (
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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
	orgDir := config.DefaultConfigDir
	config.DefaultConfigDir = temp
	defer func() { config.DefaultConfigDir = orgDir }()

	configFile := filepath.Join(config.DefaultConfigDir, "config.json")
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
	defer viper.Reset()

	viper.Set(config.KeyLogLevel, "")
	err = saveConfig(Config{
		"remote": map[string]any{
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
  "remotes": {
    "remote": {
      "type": "http",
      "loc": "http://example.com/"
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
