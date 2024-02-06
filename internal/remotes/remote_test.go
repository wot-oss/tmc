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
	err = defaultManager.saveConfig(Config{
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
	rm := remoteManager{}
	t.Run("invalid remotes config", func(t *testing.T) {

		viper.Set(KeyRemotes, map[string]any{
			"r1": map[string]string{
				"type": "file",
				"loc":  "somewhere",
			},
		})

		_, err := rm.All()
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
			all, err := rm.All()
			assert.NoError(t, err)
			assert.Len(t, all, 2)
			assert.NotEqual(t, -1, slices.IndexFunc(all, func(remote Remote) bool { return reflect.TypeOf(remote) == reflect.TypeOf(&FileRemote{}) }))
			assert.NotEqual(t, -1, slices.IndexFunc(all, func(remote Remote) bool { return reflect.TypeOf(remote) == reflect.TypeOf(&HttpRemote{}) }))
		})
		t.Run("file remote", func(t *testing.T) {
			fr, err := rm.Get(NewRemoteSpec("r1"))
			assert.NoError(t, err)
			assert.Equal(t, &FileRemote{
				root: "somewhere",
				spec: NewRemoteSpec("r1"),
			}, fr)

		})
		t.Run("http remote", func(t *testing.T) {
			hr, err := rm.Get(NewRemoteSpec("r2"))
			assert.NoError(t, err)
			u, _ := url.Parse(ur)
			assert.Equal(t, &HttpRemote{
				templatedPath:  true,
				templatedQuery: false,
				baseHttpRemote: baseHttpRemote{
					root:       ur,
					parsedRoot: u,
					spec:       NewRemoteSpec("r2"),
				},
			}, hr)
		})
		t.Run("ad-hoc remote", func(t *testing.T) {
			ar, err := rm.Get(NewDirSpec("directory"))
			assert.NoError(t, err)
			assert.Equal(t, &FileRemote{
				root: "directory",
				spec: RepoSpec{"", "directory"},
			}, ar)
		})

		t.Run("invalid spec", func(t *testing.T) {
			_, err := rm.Get(RepoSpec{"name", "dir"})
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
			all, err := rm.All()
			assert.NoError(t, err)
			assert.Len(t, all, 1)
			assert.NotEqual(t, -1, slices.IndexFunc(all, func(remote Remote) bool { return reflect.TypeOf(remote) == reflect.TypeOf(&FileRemote{}) }))
		})
		t.Run("named file remote", func(t *testing.T) {
			fr, err := rm.Get(NewRemoteSpec("r1"))
			assert.NoError(t, err)
			assert.Equal(t, &FileRemote{
				root: "somewhere",
				spec: NewRemoteSpec("r1"),
			}, fr)

		})
		t.Run("empty spec", func(t *testing.T) {
			fr, err := rm.Get(EmptySpec)
			assert.NoError(t, err)
			assert.Equal(t, &FileRemote{
				root: "somewhere",
				spec: NewRemoteSpec("r1"),
			}, fr)

		})
		t.Run("http remote", func(t *testing.T) {
			_, err := rm.Get(NewRemoteSpec("r2"))
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
			all, err := rm.All()
			assert.NoError(t, err)
			assert.Len(t, all, 2)
			assert.NotEqual(t, -1, slices.IndexFunc(all, func(remote Remote) bool { return reflect.TypeOf(remote) == reflect.TypeOf(&FileRemote{}) }))
		})
		t.Run("named file remote", func(t *testing.T) {
			fr, err := rm.Get(NewRemoteSpec("r3"))
			assert.NoError(t, err)
			assert.Equal(t, &FileRemote{
				root: "somewhere/else",
				spec: NewRemoteSpec("r3"),
			}, fr)

		})
		t.Run("empty spec", func(t *testing.T) {
			_, err := rm.Get(EmptySpec)
			assert.ErrorIs(t, err, ErrAmbiguous)
		})
		t.Run("http remote", func(t *testing.T) {
			_, err := rm.Get(NewRemoteSpec("r2"))
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
			all, err := rm.All()
			assert.NoError(t, err)
			assert.Len(t, all, 0)
		})
		t.Run("named file remote", func(t *testing.T) {
			_, err := rm.Get(NewRemoteSpec("r1"))
			assert.ErrorIs(t, err, ErrRemoteNotFound)
		})
		t.Run("ad-hoc remote", func(t *testing.T) {
			ar, err := rm.Get(NewDirSpec("directory"))
			assert.NoError(t, err)
			assert.Equal(t, &FileRemote{
				root: "directory",
				spec: RepoSpec{"", "directory"},
			}, ar)
		})
		t.Run("empty spec", func(t *testing.T) {
			_, err := rm.Get(EmptySpec)
			assert.ErrorIs(t, err, ErrRemoteNotFound)
		})
	})
}

func TestGetSpecdOrAll(t *testing.T) {
	rm := NewMockRemoteManager(t)
	r1 := NewMockRemote(t)
	r2 := NewMockRemote(t)
	r3 := NewMockRemote(t)

	rm.On("All").Return([]Remote{r1, r2}, nil)
	all, err := GetSpecdOrAll(rm, EmptySpec)
	assert.NoError(t, err)
	assert.Equal(t, &UnionRemote{rs: []Remote{r1, r2}}, all)

	rm.On("Get", NewRemoteSpec("r1")).Return(r1, nil)
	all, err = GetSpecdOrAll(rm, NewRemoteSpec("r1"))
	assert.NoError(t, err)
	assert.Equal(t, r1, all)

	rm.On("Get", NewDirSpec("dir1")).Return(r3, nil)
	all, err = GetSpecdOrAll(rm, NewDirSpec("dir1"))
	assert.NoError(t, err)
	assert.Equal(t, r3, all)

}
