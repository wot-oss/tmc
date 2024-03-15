package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/repos"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/testutils"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
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
	now := func() time.Time {
		return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC)
	}

	id, _ := generateNewId(now, &model.ThingModel{
		Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
		Mpn:          "senseall",
		Author:       model.SchemaAuthor{Name: "author"},
		Version:      model.Version{Model: "v3.2.1"},
	}, []byte("{\n\"title\":\"test\"\n}"), "opt/dir")

	assert.Equal(t, "author/omnicorp/senseall/opt/dir/v3.2.1-20231110123243-7ae21a619c71.tm.json", id.String())
}

func TestPrepareToImport(t *testing.T) {
	now := func() time.Time {
		return time.Date(2023, time.November, 10, 12, 32, 43, 0, time.UTC)
	}
	t.Run("no id in original", func(t *testing.T) {
		b, _, err := prepareToImport(now, &model.ThingModel{
			Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
			Mpn:          "senseall",
			Author:       model.SchemaAuthor{Name: "author"},
			Version:      model.Version{Model: "v3.2.1"},
		}, []byte("{\r\n\"title\":\"test\"\r\n}"), "opt/dir")
		assert.NoError(t, err)
		assert.False(t, bytes.Contains(b, []byte{'\r'})) // make sure line endings were normalized
		assert.True(t, bytes.Contains(b, []byte("author/omnicorp/senseall/opt/dir/v3.2.1-20231110123243-7ae21a619c71.tm.json")))
	})
	t.Run("foreign string id in original", func(t *testing.T) {
		b, _, err := prepareToImport(now, &model.ThingModel{
			Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
			Mpn:          "senseall",
			Author:       model.SchemaAuthor{Name: "author"},
			Version:      model.Version{Model: "v3.2.1"},
		}, []byte("{\r\n\"title\":\"test\"\r\n,\"id\":\"foreign-id\"}"), "opt/dir")
		assert.NoError(t, err)
		assert.True(t, bytes.Contains(b, []byte("\"href\":\"foreign-id\"")))
		assert.True(t, bytes.Contains(b, []byte("author/omnicorp/senseall/opt/dir/v3.2.1-20231110123243-e7dac5728be6.tm.json")))
	})
	t.Run("our string id in original/correct hash", func(t *testing.T) {
		b, _, err := prepareToImport(now, &model.ThingModel{
			Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
			Mpn:          "senseall",
			Author:       model.SchemaAuthor{Name: "author"},
			Version:      model.Version{Model: "v3.2.1"},
		}, []byte("{\r\n\"title\":\"test\"\r\n,\"id\":\"author/omnicorp/senseall/opt/dir/v3.2.1-20221010123243-7ae21a619c71.tm.json\"}"), "opt/dir")
		assert.NoError(t, err)
		// no change in id
		assert.True(t, bytes.Contains(b, []byte("author/omnicorp/senseall/opt/dir/v3.2.1-20221010123243-7ae21a619c71.tm.json")))
	})
	t.Run("our string id in original/incorrect author", func(t *testing.T) {
		b, _, err := prepareToImport(now, &model.ThingModel{
			Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
			Mpn:          "senseall",
			Author:       model.SchemaAuthor{Name: "author"},
			Version:      model.Version{Model: "v3.2.1"},
		}, []byte("{\r\n\"title\":\"test\"\r\n,\"id\":\"publisher/omnicorp/senseall/opt/dir/v3.2.1-20221010123243-e7dac5728be6.tm.json\"}"), "opt/dir")
		assert.NoError(t, err)
		// new generated id
		assert.True(t, bytes.Contains(b, []byte("author/omnicorp/senseall/opt/dir/v3.2.1-20231110123243-7ae21a619c71.tm.json")))
	})
	t.Run("our string id in original/incorrect hash", func(t *testing.T) {
		b, _, err := prepareToImport(now, &model.ThingModel{
			Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
			Mpn:          "senseall",
			Author:       model.SchemaAuthor{Name: "author"},
			Version:      model.Version{Model: "v3.2.1"},
		}, []byte("{\r\n\"title\":\"test\"\r\n,\"id\":\"author/omnicorp/senseall/opt/dir/v3.2.1-20221010123243-863e9f0f950a.tm.json\"}"), "opt/dir")
		assert.NoError(t, err)
		// new generated id
		assert.True(t, bytes.Contains(b, []byte("author/omnicorp/senseall/opt/dir/v3.2.1-20231110123243-7ae21a619c71.tm.json")))
	})

}

func TestPushToRepoUnversioned(t *testing.T) {
	root, err := os.MkdirTemp(os.TempDir(), "tm-catalog")
	assert.NoError(t, err)
	t.Logf("test root: %s", root)
	defer func() { _ = os.RemoveAll(root) }()

	repo, err := repos.NewFileRepo(map[string]any{
		"type": "file",
		"loc":  root,
	}, model.EmptySpec)
	assert.NoError(t, err)

	clk := testutils.NewTestClock(time.Now(), 1050*time.Millisecond)
	c := NewPushCommand(clk.Now)

	var firstSaved string
	_, raw, err := utils.ReadRequiredFile("../../test/data/push/omnilamp.json")
	t.Run("write first TM", func(t *testing.T) {

		assert.NoError(t, err)
		id, err := c.PushFile(raw, repo, "")
		assert.NoError(t, err)
		testTMDir := filepath.Join(root, filepath.Dir(id))
		t.Logf("test TM dir: %s", testTMDir)
		_, err = os.Stat(filepath.Join(root, id))
		assert.NoError(t, err)
		entries, _ := os.ReadDir(testTMDir)
		assert.Len(t, entries, 1)
		firstSaved = entries[0].Name()
	})

	t.Run("attempt overwriting with the same content", func(t *testing.T) {
		// attempt overwriting with the same content - no change
		id, err := c.PushFile(raw, repo, "")
		var errExists *repos.ErrTMIDConflict
		assert.ErrorAs(t, err, &errExists)
		entries, _ := os.ReadDir(filepath.Join(root, filepath.Dir(id)))
		assert.Len(t, entries, 1)
		assert.Equal(t, firstSaved, entries[0].Name())
	})

	t.Run("write a changed file", func(t *testing.T) {
		// write a changed file - saves new version
		raw = bytes.Replace(raw, []byte("Lamp Thing Model"), []byte("Lamp Thing"), 1)
		id, err := c.PushFile(raw, repo, "")
		assert.NoError(t, err)
		entries, _ := os.ReadDir(filepath.Join(root, filepath.Dir(id)))
		assert.Len(t, entries, 2)
		assert.Equal(t, firstSaved, entries[0].Name())
	})

	t.Run("change the file back and write", func(t *testing.T) {

		// change the file back and write - saves new version
		raw = bytes.Replace(raw, []byte("Lamp Thing"), []byte("Lamp Thing Model"), 1)
		id, err := c.PushFile(raw, repo, "")
		assert.NoError(t, err)
		entries, _ := os.ReadDir(filepath.Join(root, filepath.Dir(id)))
		assert.Len(t, entries, 3)
		assert.Equal(t, firstSaved, entries[0].Name())
	})

	t.Run("write multiple content versions in the same second", func(t *testing.T) {
		c = NewPushCommand(time.Now) // use real clock to be able to produce timestamp clash
		// change content and write multiple times in the same second - produces no files with the same timestamp
		var id string
		for i := 0; i < 5; i++ {
			content := bytes.Replace(raw, []byte("Lamp Thing Model"), []byte("Lamp Thing Model"+strconv.Itoa(i)), 1)
			var err error
			id, err = c.PushFile(content, repo, "")
			assert.NoError(t, err)
		}
		entries, _ := os.ReadDir(filepath.Join(root, filepath.Dir(id)))
		assert.Len(t, entries, 8)
		var timestamps []string
		for _, e := range entries {
			b, _ := strings.CutSuffix(e.Name(), model.TMFileExtension)
			v, _ := model.ParseTMVersion(b)
			timestamps = append(timestamps, v.Timestamp)
		}
		assert.Equal(t, 8, len(timestamps))
		slices.Sort(timestamps)
		timestamps = slices.Compact(timestamps)
		assert.Equal(t, 8, len(timestamps))

	})
}
func TestPushToRepoVersioned(t *testing.T) {
	root, err := os.MkdirTemp(os.TempDir(), "tm-catalog")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(root) }()

	repo, err := repos.NewFileRepo(map[string]any{
		"type": "file",
		"loc":  root,
	}, model.EmptySpec)
	assert.NoError(t, err)

	c := NewPushCommand(time.Now)

	// write first TM
	_, raw, err := utils.ReadRequiredFile("../../test/data/push/omnilamp-versioned.json")
	assert.NoError(t, err)

	id, err := c.PushFile(raw, repo, "")
	assert.NoError(t, err)
	entries, _ := os.ReadDir(filepath.Join(root, filepath.Dir(id)))
	assert.Len(t, entries, 1)
	assert.True(t, strings.HasPrefix(entries[0].Name(), "v3.2.1"))

	// write a new version of ThingModel - saves new version
	time.Sleep(1050 * time.Millisecond)
	raw = bytes.Replace(raw, []byte("\"v3.2.1\""), []byte("\"v4.0.0\""), 1)
	id, err = c.PushFile(raw, repo, "")
	assert.NoError(t, err)
	entries, _ = os.ReadDir(filepath.Join(root, filepath.Dir(id)))
	assert.Len(t, entries, 2)
	assert.True(t, strings.HasPrefix(entries[1].Name(), "v4.0.0"))

	// change an older version and push - saves new version
	_, raw, err = utils.ReadRequiredFile("../../test/data/push/omnilamp-versioned.json")
	time.Sleep(1050 * time.Millisecond)
	raw = bytes.Replace(raw, []byte("Lamp Thing Model"), []byte("Lamp Thing"), 1)
	id, err = c.PushFile(raw, repo, "")
	assert.NoError(t, err)
	entries, _ = os.ReadDir(filepath.Join(root, filepath.Dir(id)))
	assert.Len(t, entries, 3)
	assert.True(t, strings.HasPrefix(entries[0].Name(), "v3.2.1"))
	assert.True(t, strings.HasPrefix(entries[1].Name(), "v3.2.1"))
	assert.True(t, strings.HasPrefix(entries[2].Name(), "v4.0.0"))

}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		in  string
		exp string
	}{
		{in: "", exp: ""},
		{in: "/", exp: ""},
		{in: "//", exp: ""},
		{in: "\\", exp: ""},
		{in: "\\\\", exp: ""},
		{in: "/a/", exp: "a"},
		{in: "/a/b/", exp: "a/b"},
		{in: "a\\b", exp: "a/b"},
		{in: "ä\\b/ôm/før mи", exp: "ae/b/om/foer-m"},
		{in: "\\a\\b\\", exp: "a/b"},
	}

	for i, test := range tests {
		out := sanitizePathForID(test.in)
		assert.Equal(t, test.exp, out, "failed for %s (test %d)", test.in, i)
	}
}
