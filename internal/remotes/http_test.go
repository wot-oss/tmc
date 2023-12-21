package remotes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestHttpRemote_Fetch(t *testing.T) {
	const tmid = "manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json"
	const aid = "manufacturer/mpn/v1.0.0-20201205123243-c49617d2e4fc.tm.json"
	const tm = "{\"id\":\"manufacturer/mpn/v1.0.0-20201205123243-c49617d2e4fc.tm.json\"}"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/"+tmid, r.URL.Path)
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(tm))
	}))
	defer srv.Close()

	config, err := createHttpRemoteConfig("", []byte(`{"loc":"`+srv.URL+`", "type":"http", "auth":{"bearer":"token123"}}`))
	assert.NoError(t, err)
	r, err := NewHttpRemote(config, "nameless")
	assert.NoError(t, err)
	actId, b, err := r.Fetch(tmid)
	assert.NoError(t, err)
	assert.Equal(t, aid, actId)
	assert.Equal(t, []byte(tm), b)
}
