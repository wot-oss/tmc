package remotes

import (
	"os"
	"path/filepath"
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
