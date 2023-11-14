package remotes

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
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
