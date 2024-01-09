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

	viper.Set(config.KeyLog, false)
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
	viper.Set(KeyRemotes, map[string]any{
		"r1": map[string]string{
			"type": "file",
			"loc":  "somewhere",
		},
	})

	_, err := rm.All()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid remote config")

	const ur = "http://example.com/{{ID}}"
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
	all, err := rm.All()
	assert.NoError(t, err)
	assert.Len(t, all, 2)
	assert.NotEqual(t, -1, slices.IndexFunc(all, func(remote Remote) bool { return reflect.TypeOf(remote) == reflect.TypeOf(&FileRemote{}) }))
	assert.NotEqual(t, -1, slices.IndexFunc(all, func(remote Remote) bool { return reflect.TypeOf(remote) == reflect.TypeOf(&HttpRemote{}) }))

	fr, err := rm.Get("r1")
	assert.NoError(t, err)
	assert.Equal(t, &FileRemote{
		root: "somewhere",
		name: "r1",
	}, fr)

	hr, err := rm.Get("r2")
	assert.NoError(t, err)
	u, _ := url.Parse(ur)
	assert.Equal(t, &HttpRemote{
		root:           ur,
		parsedRoot:     u,
		templatedPath:  true,
		templatedQuery: false,
		name:           "r2",
	}, hr)

}

func TestGetNamedOrAll(t *testing.T) {
	rm := NewMockRemoteManager(t)
	r1 := NewMockRemote(t)
	r2 := NewMockRemote(t)

	rm.On("All").Return([]Remote{r1, r2}, nil)
	all, err := GetNamedOrAll(rm, "")
	assert.NoError(t, err)
	assert.Equal(t, []Remote{r1, r2}, all)

	rm.On("Get", "r1").Return(r1, nil)
	all, err = GetNamedOrAll(rm, "r1")
	assert.NoError(t, err)
	assert.Equal(t, []Remote{r1}, all)
}
