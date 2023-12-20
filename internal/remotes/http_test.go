package remotes

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHttpRemote(t *testing.T) {
	root := "http://localhost:8000/"
	remote, err := NewHttpRemote(
		map[string]any{
			"type": "http",
			"loc":  root,
		}, "name")
	assert.NoError(t, err)
	assert.Equal(t, root, remote.root)
	assert.Equal(t, "name", remote.Name())
}

func TestCreateHttpRemoteConfig(t *testing.T) {
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
		cf, err := createHttpRemoteConfig(test.strConf, []byte(test.fileConf))
		if test.expErr {
			assert.Error(t, err, "error expected in test %d for %s %s", i, test.strConf, test.fileConf)
			continue
		} else {
			assert.NoError(t, err, "no error expected in test %d for %s %s", i, test.strConf, test.fileConf)
		}
		assert.Equalf(t, "http", cf[KeyRemoteType], "in test %d for %s %s", i, test.strConf, test.fileConf)
		assert.Equalf(t, test.expRoot, fmt.Sprintf("%v", cf[KeyRemoteLoc]), "in test %d for %s %s", i, test.strConf, test.fileConf)

	}
}
