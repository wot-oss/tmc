package remotes

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
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/config"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

func TestSaveConfigOverwritesOnlyRemotes(t *testing.T) {
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

func TestRemoteManager_All_And_Get(t *testing.T) {
	t.Run("invalid remotes config", func(t *testing.T) {

		viper.Set(KeyRemotes, map[string]any{
			"r1": map[string]string{
				"type": "file",
				"loc":  "somewhere",
			},
		})

		_, err := All()
		assert.Error(t, err)
		assert.ErrorContains(t, err, "invalid remote config")

	})
	const ur = "http://example.com/{{ID}}"

	t.Run("two remotes", func(t *testing.T) {

		viper.Set(KeyRemotes, map[string]any{
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
			assert.NotEqual(t, -1, slices.IndexFunc(all, func(remote Remote) bool { return reflect.TypeOf(remote) == reflect.TypeOf(&FileRemote{}) }))
			assert.NotEqual(t, -1, slices.IndexFunc(all, func(remote Remote) bool { return reflect.TypeOf(remote) == reflect.TypeOf(&HttpRemote{}) }))
		})
		t.Run("file remote", func(t *testing.T) {
			fr, err := Get(model.NewRemoteSpec("r1"))
			assert.NoError(t, err)
			assert.Equal(t, &FileRemote{
				root: "somewhere",
				spec: model.NewRemoteSpec("r1"),
			}, fr)

		})
		t.Run("http remote", func(t *testing.T) {
			hr, err := Get(model.NewRemoteSpec("r2"))
			assert.NoError(t, err)
			u, _ := url.Parse(ur)
			assert.Equal(t, &HttpRemote{
				templatedPath:  true,
				templatedQuery: false,
				baseHttpRemote: baseHttpRemote{
					root:       ur,
					parsedRoot: u,
					spec:       model.NewRemoteSpec("r2"),
				},
			}, hr)
		})
		t.Run("ad-hoc remote", func(t *testing.T) {
			ar, err := Get(model.NewDirSpec("directory"))
			assert.NoError(t, err)
			assert.Equal(t, &FileRemote{
				root: "directory",
				spec: model.NewDirSpec("directory"),
			}, ar)
		})

		t.Run("invalid spec", func(t *testing.T) {
			_, err := model.NewSpec("name", "dir")
			assert.Error(t, err)
		})

	})

	t.Run("one enabled remote", func(t *testing.T) {
		viper.Set(KeyRemotes, map[string]any{
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
			assert.NotEqual(t, -1, slices.IndexFunc(all, func(remote Remote) bool { return reflect.TypeOf(remote) == reflect.TypeOf(&FileRemote{}) }))
		})
		t.Run("named file remote", func(t *testing.T) {
			fr, err := Get(model.NewRemoteSpec("r1"))
			assert.NoError(t, err)
			assert.Equal(t, &FileRemote{
				root: "somewhere",
				spec: model.NewRemoteSpec("r1"),
			}, fr)

		})
		t.Run("empty spec", func(t *testing.T) {
			fr, err := Get(model.EmptySpec)
			assert.NoError(t, err)
			assert.Equal(t, &FileRemote{
				root: "somewhere",
				spec: model.NewRemoteSpec("r1"),
			}, fr)

		})
		t.Run("http remote", func(t *testing.T) {
			_, err := Get(model.NewRemoteSpec("r2"))
			assert.ErrorIs(t, err, ErrRemoteNotFound)
		})

	})
	t.Run("two enabled remotes", func(t *testing.T) {
		viper.Set(KeyRemotes, map[string]any{
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
			assert.NotEqual(t, -1, slices.IndexFunc(all, func(remote Remote) bool { return reflect.TypeOf(remote) == reflect.TypeOf(&FileRemote{}) }))
		})
		t.Run("named file remote", func(t *testing.T) {
			fr, err := Get(model.NewRemoteSpec("r3"))
			assert.NoError(t, err)
			assert.Equal(t, &FileRemote{
				root: "somewhere/else",
				spec: model.NewRemoteSpec("r3"),
			}, fr)

		})
		t.Run("empty spec", func(t *testing.T) {
			_, err := Get(model.EmptySpec)
			assert.ErrorIs(t, err, ErrAmbiguous)
		})
		t.Run("http remote", func(t *testing.T) {
			_, err := Get(model.NewRemoteSpec("r2"))
			assert.ErrorIs(t, err, ErrRemoteNotFound)
		})

	})
	t.Run("no enabled remotes", func(t *testing.T) {
		viper.Set(KeyRemotes, map[string]any{
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
		t.Run("named file remote", func(t *testing.T) {
			_, err := Get(model.NewRemoteSpec("r1"))
			assert.ErrorIs(t, err, ErrRemoteNotFound)
		})
		t.Run("ad-hoc remote", func(t *testing.T) {
			ar, err := Get(model.NewDirSpec("directory"))
			assert.NoError(t, err)
			assert.Equal(t, &FileRemote{
				root: "directory",
				spec: model.NewDirSpec("directory"),
			}, ar)
		})
		t.Run("empty spec", func(t *testing.T) {
			_, err := Get(model.EmptySpec)
			assert.ErrorIs(t, err, ErrRemoteNotFound)
		})
	})
}

func TestGetSpecdOrAll(t *testing.T) {
	viper.Set(KeyRemotes, map[string]any{
		"r1": map[string]any{
			"type": "file",
			"loc":  "somewhere",
		},
		"r2": map[string]any{
			"type": "file",
			"loc":  "somewhere-else",
		},
	})

	// check if all remotes are returned, when passing EmptySpec
	all, err := GetSpecdOrAll(model.EmptySpec)
	assert.NoError(t, err)
	if assert.Len(t, all.rs, 2) {
		var fileRemote *FileRemote
		var ok bool
		for _, remote := range all.rs {
			if fileRemote, ok = remote.(*FileRemote); !ok {
				t.Fatalf("expected file remote, got %T", remote)
			}
			switch fileRemote.Spec().RemoteName() {
			case "r1":
				assert.Equal(t, "somewhere", fileRemote.root)
			case "r2":
				assert.Equal(t, "somewhere-else", fileRemote.root)
			default:
				t.Fatalf("unexpected remote found: %v", *fileRemote)
			}
		}
	}

	// get r1 remote
	all, err = GetSpecdOrAll(model.NewRemoteSpec("r1"))
	assert.NoError(t, err)
	if assert.Len(t, all.rs, 1) {
		if r1, ok := all.rs[0].(*FileRemote); assert.True(t, ok) {
			assert.Equal(t, "somewhere", r1.root)
		}
	}

	// get local repo
	all, err = GetSpecdOrAll(model.NewDirSpec("dir1"))
	assert.NoError(t, err)
	if assert.Len(t, all.rs, 1) {
		if r1, ok := all.rs[0].(*FileRemote); assert.True(t, ok) {
			assert.Equal(t, "dir1", r1.root)
		}
	}

}
