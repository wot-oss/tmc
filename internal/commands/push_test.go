package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

func TestMoveIdToOriginalLink(t *testing.T) {
	tests := []struct {
		json string
		id   string
		exp  string
	}{
		{
			json: "{}",
			id:   "myId",
			exp:  "myId",
		},
		{
			json: `{
  "@context": [
    "https://www.w3.org/2022/wot/td/v1.1",
    "https://schema.org/docs/jsonldcontext.json"
  ],
  "@type": "tm:ThingModel",
  "title": "Lamp Thing Model",
  "properties": {
    "status": {
      "description": "current status of the lamp (on|off)",
      "type": "string",
      "readOnly": true
    }
  },
  "version": {
    "model": "v1.0.1"
  }
}`,
			id:  "myId",
			exp: "myId",
		},
		{
			json: `{
  "links": [{
    "rel": "manifest",
    "href": "https://example.org/docs/jsonldcontext.json"
  }]
}`,
			id:  "myId",
			exp: "myId",
		},
		{
			json: `{
  "links": [{
    "rel": "original",
    "href": "https://example.org/docs/jsonldcontext.json"
  }]
}`,
			id:  "myId",
			exp: "https://example.org/docs/jsonldcontext.json",
		},
	}

	for i, test := range tests {
		res := moveIdToOriginalLink([]byte(test.json), test.id)
		var js map[string]any
		err := json.Unmarshal(res, &js)
		assert.NoError(t, err)
		links, ok := js["links"]
		assert.True(t, ok)
		linksSlice, ok := links.([]any)
		assert.True(t, ok)
		found := false
		for _, link := range linksSlice {
			linkObj, ok := link.(map[string]any)
			assert.True(t, ok)
			if rel, ok := linkObj["rel"]; ok {
				if rel == "original" {
					found = true
					assert.Equal(t, test.exp, linkObj["href"], "unexpected original href at %d", i)
					break
				}
			}
		}
		assert.True(t, found)
	}
}

func TestGenerateNewID(t *testing.T) {
	now = func() time.Time {
		return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC)
	}
	defer func() {
		now = time.Now
	}()

	id := generateNewId(&model.ThingModel{
		Manufacturer: model.SchemaManufacturer{"omnicorp"},
		Mpn:          "senseall",
		Author:       model.SchemaAuthor{"author"},
		Version:      model.Version{"v3.2.1"},
	}, []byte("{}"), "opt/dir")

	assert.Equal(t, "author/omnicorp/senseall/opt/dir/v3.2.1-20231110123243-bf21a9e8fbc5.tm.json", id.String())
}

func TestPushToRemoteUnversioned(t *testing.T) {
	root, err := os.MkdirTemp(os.TempDir(), "tm-catalog")
	assert.NoError(t, err)
	t.Logf("test root: %s", root)
	defer func() { _ = os.RemoveAll(root) }()

	remote, err := remotes.NewFileRemote(
		map[string]any{
			"type": "file",
			"url":  "file:" + root,
		})
	assert.NoError(t, err)

	// write first TM
	_, raw, err := internal.ReadRequiredFile("../../test/data/push/omnilamp.json")
	assert.NoError(t, err)
	id, err := PushFile(raw, remote, "")
	assert.NoError(t, err)
	testTMDir := filepath.Join(root, filepath.Dir(id.String()))
	t.Logf("test TM dir: %s", testTMDir)
	_, err = os.Stat(filepath.Join(root, id.String()))
	assert.NoError(t, err)
	entries, _ := os.ReadDir(testTMDir)
	assert.Len(t, entries, 1)
	firstSaved := entries[0].Name()

	// attempt overwriting with the same content - no change
	time.Sleep(1050 * time.Millisecond)
	id, err = PushFile(raw, remote, "")
	var errExists *remotes.ErrTMExists
	assert.ErrorAs(t, err, &errExists)
	entries, _ = os.ReadDir(filepath.Join(root, filepath.Dir(id.String())))
	assert.Len(t, entries, 1)
	assert.Equal(t, firstSaved, entries[0].Name())

	// write a changed file - saves new version
	time.Sleep(1050 * time.Millisecond)
	raw = bytes.Replace(raw, []byte("Lamp Thing Model"), []byte("Lamp Thing"), 1)
	id, err = PushFile(raw, remote, "")
	assert.NoError(t, err)
	entries, _ = os.ReadDir(filepath.Join(root, filepath.Dir(id.String())))
	assert.Len(t, entries, 2)
	assert.Equal(t, firstSaved, entries[0].Name())

	// change the file back and write - saves new version
	time.Sleep(1050 * time.Millisecond)
	raw = bytes.Replace(raw, []byte("Lamp Thing"), []byte("Lamp Thing Model"), 1)
	id, err = PushFile(raw, remote, "")
	assert.NoError(t, err)
	entries, _ = os.ReadDir(filepath.Join(root, filepath.Dir(id.String())))
	assert.Len(t, entries, 3)
	assert.Equal(t, firstSaved, entries[0].Name())

}
func TestPushToRemoteVersioned(t *testing.T) {
	root, err := os.MkdirTemp(os.TempDir(), "tm-catalog")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(root) }()

	remote, err := remotes.NewFileRemote(
		map[string]any{
			"type": "file",
			"url":  "file:" + root,
		})
	assert.NoError(t, err)

	// write first TM
	_, raw, err := internal.ReadRequiredFile("../../test/data/push/omnilamp-versioned.json")
	assert.NoError(t, err)

	id, err := PushFile(raw, remote, "")
	assert.NoError(t, err)
	entries, _ := os.ReadDir(filepath.Join(root, filepath.Dir(id.String())))
	assert.Len(t, entries, 1)
	assert.True(t, strings.HasPrefix(entries[0].Name(), "v3.2.1"))

	// write a new version of ThingModel - saves new version
	time.Sleep(1050 * time.Millisecond)
	raw = bytes.Replace(raw, []byte("\"v3.2.1\""), []byte("\"v4.0.0\""), 1)
	id, err = PushFile(raw, remote, "")
	assert.NoError(t, err)
	entries, _ = os.ReadDir(filepath.Join(root, filepath.Dir(id.String())))
	assert.Len(t, entries, 2)
	assert.True(t, strings.HasPrefix(entries[1].Name(), "v4.0.0"))

	// change an older version and push - saves new version
	_, raw, err = internal.ReadRequiredFile("../../test/data/push/omnilamp-versioned.json")
	time.Sleep(1050 * time.Millisecond)
	raw = bytes.Replace(raw, []byte("Lamp Thing Model"), []byte("Lamp Thing"), 1)
	id, err = PushFile(raw, remote, "")
	assert.NoError(t, err)
	entries, _ = os.ReadDir(filepath.Join(root, filepath.Dir(id.String())))
	assert.Len(t, entries, 3)
	assert.True(t, strings.HasPrefix(entries[0].Name(), "v3.2.1"))
	assert.True(t, strings.HasPrefix(entries[1].Name(), "v3.2.1"))
	assert.True(t, strings.HasPrefix(entries[2].Name(), "v4.0.0"))

}
