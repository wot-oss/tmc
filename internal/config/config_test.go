package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kinbiko/jsonassert"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func setupDefaultConfigDir() func() {
	temp, err := os.MkdirTemp("", "config")
	if err != nil {
		panic(err)
	}

	orgDir := DefaultConfigDir
	DefaultConfigDir = temp
	return func() {
		DefaultConfigDir = orgDir
		os.RemoveAll(temp)
	}
}

func TestSaveConfigOverwritesOnlyKeyValue(t *testing.T) {
	defer setupDefaultConfigDir()()

	// given: a config file
	configFile := filepath.Join(DefaultConfigDir, "cfg.json")
	err := os.WriteFile(configFile, []byte(`{
  "loglevel": "debug",
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

	// and given: a key-value pair that shall be overwritten in config file
	key := "repos"
	val := map[string]any{
		"http": map[string]any{
			"type": "http",
			"loc":  "http://example.com/"},
	}
	// and given: an in memory overwritten key-value pair that exists in config file
	viper.Set(KeyLogLevel, "error")
	// and given: an in memory key-value pair that does not exist in config file
	viper.Set("someKey", "someValue")

	// when: saving the key-value pair in the config file
	err = Save(key, val)

	// then: only the intended key-value pair is overwritten,
	//       everything else has not been changed or added
	assert.NoError(t, err)
	file, err := os.ReadFile(configFile)
	assert.NoError(t, err)
	jsa := jsonassert.New(t)
	jsa.Assertf(string(file), `{
  "loglevel": "debug",
  "repos": {
    "http": {
      "type": "http",
      "loc": "http://example.com/"
    }
  }
}`)
	// and then: the key-value pair is also overwritten in memory
	repos := viper.Get("repos")
	assert.Equal(t, val, repos)
}

func TestDeleteConfigRemovesOnlyKeyValue(t *testing.T) {
	defer setupDefaultConfigDir()()

	// given: a config file
	configFile := filepath.Join(DefaultConfigDir, "config.json")
	err := os.WriteFile(configFile, []byte(`{
  "loglevel": "debug",
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

	// and given: a key-value pair that shall be deleted in config file
	key := "repos"
	// and given: an in memory overwritten key-value pair that exists in config file
	viper.Set(KeyLogLevel, "error")
	// and given: an in memory key-value pair that does not exist in config file
	viper.Set("someKey", "someValue")

	// when: deleting the key-value pair in the config file
	err = Delete(key)

	// then: only the intended key-value pair is deleted,
	//       everything else has not been changed or added
	assert.NoError(t, err)
	file, err := os.ReadFile(configFile)
	assert.NoError(t, err)
	jsa := jsonassert.New(t)
	jsa.Assertf(string(file), `{ "loglevel": "debug" }`)
}
