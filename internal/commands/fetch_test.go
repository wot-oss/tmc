package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes/mocks"
)

func TestParseFetchName(t *testing.T) {
	tests := []struct {
		in      string
		expErr  bool
		expName string
		expSD   string
	}{
		{"", true, "", ""},
		{"manufacturer", true, "", ""},
		{"manufacturer\\mpn", true, "", ""},
		{"manu-facturer/mpn", false, "manu-facturer/mpn", ""},
		{"manufacturer/mpn:1.2.3", false, "manufacturer/mpn", "1.2.3"},
		{"manufacturer/mpn:v1.2.3", false, "manufacturer/mpn", "v1.2.3"},
		{"manufacturer/mpn:43748209adcb", false, "manufacturer/mpn", "43748209adcb"},
		{"author/manufacturer/mpn:1.2.3", false, "author/manufacturer/mpn", "1.2.3"},
		{"author/manufacturer/mpn:v1.2.3", false, "author/manufacturer/mpn", "v1.2.3"},
		{"author/manufacturer/mpn:43748209adcb", false, "author/manufacturer/mpn", "43748209adcb"},
		{"author/manufacturer/mpn/folder/structure:1.2.3", false, "author/manufacturer/mpn/folder/structure", "1.2.3"},
		{"author/manufacturer/mpn/folder/structure:v1.2.3-alpha1", false, "author/manufacturer/mpn/folder/structure", "v1.2.3-alpha1"},
		{"author/manufacturer/mpn/folder/structure:43748209adcb", false, "author/manufacturer/mpn/folder/structure", "43748209adcb"},
	}

	for _, test := range tests {
		out, err := ParseFetchName(test.in)
		if test.expErr {
			assert.Error(t, err, "Want: error in ParseFetchName(%s). Got: nil", test.in)
		} else {
			assert.NoError(t, err, "Want: no error in ParseFetchName(%s). Got: %v", test.in, err)
			exp := FetchName{test.expName, test.expSD}
			assert.Equal(t, exp, out, "Want: ParseFetchName(%s) = %v. Got: %v", test.in, exp, out)
		}
	}
}

func TestFetchCommand_FetchByTMIDOrName(t *testing.T) {
	rm := mocks.NewRemoteManager(t)
	r := mocks.NewRemote(t)
	rm.On("All").Return([]remotes.Remote{r}, nil)
	rm.On("Get", "").Return(r, nil)
	setUpVersionsForFetchByTMIDOrName(r)

	r.On("Fetch", "manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json").Return("manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json", []byte("{}"), nil)
	r.On("Fetch", "manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json").Return("manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json", []byte("{}"), nil)
	r.On("Fetch", "author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json").Return("author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json", []byte("{}"), nil)
	r.On("Fetch", "author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json").Return("author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json", []byte("{}"), nil)

	f := NewFetchCommand(rm)

	tests := []struct {
		in         string
		expErr     bool
		expErrText string
		expBLen    int
	}{
		{"", true, "Invalid name format:  - Must be NAME[:SEMVER|DIGEST]", 0},
		{"manufacturer", true, "Invalid name format: manufacturer - Must be NAME[:SEMVER|DIGEST]", 0},
		{"manufacturer/mpn", false, "", 2},
		{"manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json", false, "", 2},
		{"manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json", false, "", 2},
		{"author/manufacturer/mpn", false, "", 2},
		{"author/manufacturer/mpn:v1.0.0", false, "", 2},
		{"author/manufacturer/mpn:1.0.0", false, "", 2},
		{"author/manufacturer/mpn:c49617d2e4fc", false, "", 2},
		{"author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json", false, "", 2},
		{"author/manufacturer/mpn/folder/sub", false, "", 2},
		{"author/manufacturer/mpn/folder/sub:v1.0.0", false, "", 2},
		{"author/manufacturer/mpn/folder/sub:c49617d2e4fc", false, "", 2},
		{"author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json", false, "", 2},
	}

	for _, test := range tests {
		_, b, err := f.FetchByTMIDOrName("", test.in)
		if test.expErr {
			assert.Error(t, err, "Expected error in FetchByTMIDOrName(%s), but got nil", test.in)
			assert.ErrorContains(t, err, test.expErrText, "Unexpected error in FetchByTMIDOrName(%s)", test.in)
		} else {
			assert.NoError(t, err, "Expected no error in FetchByTMIDOrName(%s)", test.in)
			assert.Equal(t, test.expBLen, len(b), "Unexpected result length in FetchByTMIDOrName(%s)", test.in)
		}
	}
}

func setUpVersionsForFetchByTMIDOrName(r *mocks.Remote) {
	r.On("Versions", "manufacturer/mpn").Return(model.FoundEntry{
		Name:         "manufacturer/mpn",
		Manufacturer: model.SchemaManufacturer{Name: "manufacturer"},
		Mpn:          "mpn",
		Author:       model.SchemaAuthor{Name: "manufacturer"},
		Versions: []model.FoundVersion{
			{
				TOCVersion: model.TOCVersion{
					Version:   model.Version{Model: "v1.0.0"},
					TMID:      "manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json",
					Digest:    "c49617d2e4fc",
					TimeStamp: "20231205123243",
				},
				FoundIn: "r1",
			},
		},
	}, nil)
	r.On("Versions", "author/manufacturer/mpn").Return(model.FoundEntry{
		Name:         "author/manufacturer/mpn",
		Manufacturer: model.SchemaManufacturer{Name: "manufacturer"},
		Mpn:          "mpn",
		Author:       model.SchemaAuthor{Name: "author"},
		Versions: []model.FoundVersion{
			{
				TOCVersion: model.TOCVersion{
					Version:   model.Version{Model: "v1.0.0"},
					TMID:      "author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json",
					Digest:    "c49617d2e4fc",
					TimeStamp: "20231205123243",
				},
				FoundIn: "r1",
			},
		},
	}, nil)
	r.On("Versions", "author/manufacturer/mpn/folder/sub").Return(model.FoundEntry{
		Name:         "author/manufacturer/mpn/folder/sub",
		Manufacturer: model.SchemaManufacturer{Name: "manufacturer"},
		Mpn:          "mpn",
		Author:       model.SchemaAuthor{Name: "author"},
		Versions: []model.FoundVersion{
			{
				TOCVersion: model.TOCVersion{
					Version:   model.Version{Model: "v1.0.0"},
					TMID:      "author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json",
					Digest:    "c49617d2e4fc",
					TimeStamp: "20231205123243",
				},
				FoundIn: "r1",
			},
		},
	}, nil)
}
