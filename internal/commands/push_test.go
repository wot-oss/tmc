package commands

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"testing"
	"time"
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
	}, []byte("{}"))

	assert.Equal(t, "author/omnicorp/senseall/v3.2.1-20231110123243-bf21a9e8fbc5.tm.json", id.String())
}
