package remotes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFileRemote(t *testing.T) {
	root := "/tmp/tm-catalog1157316148"
	remote, err := NewFileRemote(
		map[string]any{
			"type": "file",
			"url":  "file:" + root,
		})
	assert.NoError(t, err)
	assert.Equal(t, root, remote.root)

	root = "/tmp/tm-catalog1157316148"
	remote, err = NewFileRemote(
		map[string]any{
			"type": "file",
			"url":  "file://" + root,
		})
	assert.NoError(t, err)
	assert.Equal(t, root, remote.root)

	root = "~/tm-catalog"
	remote, err = NewFileRemote(
		map[string]any{
			"type": "file",
			"url":  "file:/" + root,
		})
	assert.NoError(t, err)
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, "tm-catalog"), remote.root)

	root = "~/tm-catalog"
	remote, err = NewFileRemote(
		map[string]any{
			"type": "file",
			"url":  "file:" + root,
		})
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "tm-catalog"), remote.root)

	root = "~/tm-catalog"
	remote, err = NewFileRemote(
		map[string]any{
			"type": "file",
			"url":  "file:///" + root,
		})
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "tm-catalog"), remote.root)

	root = "c:\\Users\\user\\Desktop\\tm-catalog"
	remote, err = NewFileRemote(
		map[string]any{
			"type": "file",
			"url":  "file:/" + root,
		})
	assert.NoError(t, err)
	assert.Equal(t, filepath.ToSlash("c:\\Users\\user\\Desktop\\tm-catalog"), filepath.ToSlash(remote.root))

	root = "C:\\Users\\user\\Desktop\\tm-catalog"
	remote, err = NewFileRemote(
		map[string]any{
			"type": "file",
			"url":  "file:///" + root,
		})
	assert.NoError(t, err)
	assert.Equal(t, filepath.ToSlash("C:\\Users\\user\\Desktop\\tm-catalog"), filepath.ToSlash(remote.root))

}

func TestCreateFileRemoteConfig(t *testing.T) {
	wd, _ := os.Getwd()

	tests := []struct {
		strConf  string
		fileConf string
		expRoot  string
		expErr   bool
	}{
		{"../dir/name", "", "file:/" + filepath.Join(filepath.Dir(wd), "/dir/name"), false},
		{"./dir/name", "", "file:/" + filepath.Join(wd, "dir/name"), false},
		{"dir/name", "", "file:/" + filepath.Join(wd, "dir/name"), false},
		{".", "", "file:/" + filepath.Join(wd), false},
		{filepath.Join(wd, "dir/name"), "", "file:" + filepath.Join(wd, "dir/name"), false},
		{"~/dir/name", "", "file:/~/dir/name", false},
		{"", ``, "", true},
		{"", `[]`, "", true},
		{"", `{}`, "", true},
		//{"", `{"url":{}}`, "", true},
		//{"", `{"url":"dir/name"}`, "file:/dir/name", false},
		//{"", `{"url":"/dir/name"}`, "file:/dir/name", false},
		//{"", `{"url":"dir/name", "type":"http"}`, "file:dir/name", false},
	}

	for i, test := range tests {
		cf, err := createFileRemoteConfig(test.strConf, []byte(test.fileConf))
		if test.expErr {
			assert.Error(t, err, "error expected in test %d for %s %s", i, test.strConf, test.fileConf)
			continue
		} else {
			assert.NoError(t, err, "no error expected in test %d for %s %s", i, test.strConf, test.fileConf)
		}
		assert.Equalf(t, "file", cf[KeyRemoteType], "in test %d for %s %s", i, test.strConf, test.fileConf)
		assert.Equalf(t, test.expRoot, cf[KeyRemoteUrl], "in test %d for %s %s", i, test.strConf, test.fileConf)

	}
}
