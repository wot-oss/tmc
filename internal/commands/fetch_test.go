package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
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
	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)
	rm.On("All").Return([]remotes.Remote{r}, nil)
	rm.On("Get", remotes.NewRemoteSpec("r1")).Return(r, nil)
	setUpVersionsForFetchByTMIDOrName(r)

	r.On("Fetch", "manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json").Return("manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json", []byte("{\"ver\":\"v1.0.0\"}"), nil)
	r.On("Fetch", "manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json").Return("manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json", []byte("{\"ver\":\"v1.0.0\"}"), nil)
	r.On("Fetch", "author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json").Return("author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json", []byte("{\"ver\":\"v1.0.0\"}"), nil)
	r.On("Fetch", "author/manufacturer/mpn/v1.0.4-20231206123243-d49617d2e4fc.tm.json").Return("author/manufacturer/mpn/v1.0.4-20231206123243-d49617d2e4fc.tm.json", []byte("{\"ver\":\"v1.0.4\"}"), nil)
	r.On("Fetch", "author/manufacturer/mpn/v1.2.3-20231207153243-e49617d2e4ff.tm.json").Return("author/manufacturer/mpn/v1.2.3-20231207153243-e49617d2e4ff.tm.json", []byte("{\"ver\":\"v1.2.3\"}"), nil)
	r.On("Fetch", "author/manufacturer/mpn/v2.0.0-20231208123243-f49617d2e4fc.tm.json").Return("author/manufacturer/mpn/v2.0.0-20231208123243-f49617d2e4fc.tm.json", []byte("{\"ver\":\"v2.0.0\"}"), nil)
	r.On("Fetch", "author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json").Return("author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json", []byte("{\"ver\":\"v1.0.0\"}"), nil)

	f := NewFetchCommand(rm)

	tests := []struct {
		in         string
		expErr     bool
		expErrText string
		expVer     string
	}{
		{"", true, "Invalid name format:  - Must be NAME[:SEMVER]", ""},
		{"manufacturer", true, "Invalid name format: manufacturer - Must be NAME[:SEMVER]", ""},
		{"manufacturer/mpn", false, "", ""},
		{"manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json", false, "", "v1.0.0"},
		{"manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json", false, "", "v1.0.0"},
		{"author/manufacturer/mpn", false, "", "v2.0.0"},
		{"author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json", false, "", "v1.0.0"},
		{"author/manufacturer/mpn:v1.0.0", false, "", "v1.0.0"},
		{"author/manufacturer/mpn:1.0.0", false, "", "v1.0.0"},
		{"author/manufacturer/mpn:1.a.0", true, "Invalid Semantic Version", ""},
		{"author/manufacturer/mpn:v1.0", false, "", "v1.0.4"},
		{"author/manufacturer/mpn:1.3", true, "No version 1.3 found", ""},
		{"author/manufacturer/mpn:1.1", true, "No version 1.1 found", ""},
		{"author/manufacturer/mpn:1.2", false, "", "v1.2.3"},
		{"author/manufacturer/mpn:3", true, "No version 3 found", ""},
		{"author/manufacturer/mpn:v1", false, "", "v1.2.3"},
		{"author/manufacturer/mpn/folder/sub", false, "", "v1.0.0"},
		{"author/manufacturer/mpn/folder/sub:v1.0.0", false, "", "v1.0.0"},
		{"author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json", false, "", "v1.0.0"},
	}

	for _, test := range tests {
		_, b, err := f.FetchByTMIDOrName(remotes.EmptySpec, test.in)
		if test.expErr {
			assert.Error(t, err, "Expected error in FetchByTMIDOrName(%s), but got nil", test.in)
			assert.ErrorContains(t, err, test.expErrText, "Unexpected error in FetchByTMIDOrName(%s)", test.in)
		} else {
			assert.NoError(t, err, "Expected no error in FetchByTMIDOrName(%s)", test.in)
			assert.True(t, bytes.Contains(b, []byte(test.expVer)), "FetchByTMIDOrName(%s) result does not contain %s. Got: %s", test.in, test.expVer, string(b))
		}
	}
}

func setUpVersionsForFetchByTMIDOrName(r *remotes.MockRemote) {
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
				FoundIn: model.FoundSource{RemoteName: "r1"},
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
				FoundIn: model.FoundSource{RemoteName: "r1"},
			},
			{
				TOCVersion: model.TOCVersion{
					Version:   model.Version{Model: "v1.0.4"},
					TMID:      "author/manufacturer/mpn/v1.0.4-20231206123243-d49617d2e4fc.tm.json",
					Digest:    "d49617d2e4fc",
					TimeStamp: "20231206123243",
				},
				FoundIn: model.FoundSource{RemoteName: "r1"},
			},
			{
				TOCVersion: model.TOCVersion{
					Version:   model.Version{Model: "v1.2.0"},
					TMID:      "author/manufacturer/mpn/v1.2.0-20231207163243-e49617d2e4fc.tm.json",
					Digest:    "e49617d2e4fc",
					TimeStamp: "20231207163243", // this is on purpose more recent by timestamp than the latest semver (v.1.2.3)
				},
				FoundIn: model.FoundSource{RemoteName: "r1"},
			},
			{
				TOCVersion: model.TOCVersion{
					Version:   model.Version{Model: "v1.2.1"},
					TMID:      "author/manufacturer/mpn/v1.2.1-20231207133243-e49617d2e4fd.tm.json",
					Digest:    "e49617d2e4fd",
					TimeStamp: "20231207133243",
				},
				FoundIn: model.FoundSource{RemoteName: "r1"},
			},
			{
				TOCVersion: model.TOCVersion{
					Version:   model.Version{Model: "v1.2.2"},
					TMID:      "author/manufacturer/mpn/v1.2.2-20231207143243-e49617d2e4fe.tm.json",
					Digest:    "e49617d2e4fe",
					TimeStamp: "20231207143243",
				},
				FoundIn: model.FoundSource{RemoteName: "r1"},
			},
			{
				TOCVersion: model.TOCVersion{
					Version:   model.Version{Model: "v1.2.3"},
					TMID:      "author/manufacturer/mpn/v1.2.3-20231207153243-e49617d2e4ff.tm.json",
					Digest:    "e49617d2e4ff",
					TimeStamp: "20231207153243",
				},
				FoundIn: model.FoundSource{RemoteName: "r1"},
			},
			{
				TOCVersion: model.TOCVersion{
					Version:   model.Version{Model: "v2.0.0"},
					TMID:      "author/manufacturer/mpn/v2.0.0-20231208123243-f49617d2e4fc.tm.json",
					Digest:    "f49617d2e4fc",
					TimeStamp: "20231205123243",
				},
				FoundIn: model.FoundSource{RemoteName: "r1"},
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
				FoundIn: model.FoundSource{RemoteName: "r1"},
			},
		},
	}, nil)
}

func TestFetchCommand_FetchByTMIDOrName_MultipleRemotes(t *testing.T) {
	rm := remotes.NewMockRemoteManager(t)
	r1 := remotes.NewMockRemote(t)
	r2 := remotes.NewMockRemote(t)
	rm.On("All").Return([]remotes.Remote{r1, r2}, nil)
	rm.On("Get", remotes.NewRemoteSpec("r1")).Return(r1, nil)
	rm.On("Get", remotes.NewRemoteSpec("r2")).Return(r2, nil)
	r1.On("Fetch", "author/manufacturer/mpn/v1.0.0-20231005123243-a49617d2e4fc.tm.json").Return("author/manufacturer/mpn/v1.0.0-20231005123243-a49617d2e4fc.tm.json", []byte("{\"src\": \"r1\"}"), nil)
	r1.On("Fetch", "author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json").Return("", []byte{}, ErrTmNotFound)
	r2.On("Fetch", "author/manufacturer/mpn/v1.0.0-20231005123243-a49617d2e4fc.tm.json").Return("", []byte{}, ErrTmNotFound)
	r2.On("Fetch", "author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json").Return("author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json", []byte("{\"src\": \"r2\"}"), nil)
	r1.On("Versions", "author/manufacturer/mpn").Return(model.FoundEntry{
		Name:         "author/manufacturer/mpn",
		Manufacturer: model.SchemaManufacturer{Name: "manufacturer"},
		Mpn:          "mpn",
		Author:       model.SchemaAuthor{Name: "author"},
		Versions: []model.FoundVersion{
			{
				TOCVersion: model.TOCVersion{
					Version:   model.Version{Model: "v1.0.0"},
					TMID:      "author/manufacturer/mpn/v1.0.0-20231005123243-a49617d2e4fc.tm.json",
					Digest:    "a49617d2e4fc",
					TimeStamp: "20231005123243",
				},
				FoundIn: model.FoundSource{RemoteName: "r1"},
			},
		},
	}, nil)
	r2.On("Versions", "author/manufacturer/mpn").Return(model.FoundEntry{
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
				FoundIn: model.FoundSource{RemoteName: "r2"},
			},
		},
	}, nil)

	f := NewFetchCommand(rm)
	var id string
	var b []byte
	var err error
	t.Run("fetch from unspecified by id", func(t *testing.T) {
		id, b, err = f.FetchByTMIDOrName(remotes.EmptySpec, "author/manufacturer/mpn/v1.0.0-20231005123243-a49617d2e4fc.tm.json")
		assert.NoError(t, err)
		assert.Equal(t, "author/manufacturer/mpn/v1.0.0-20231005123243-a49617d2e4fc.tm.json", id)
		assert.True(t, bytes.Contains(b, []byte("r1")))

		id, b, err = f.FetchByTMIDOrName(remotes.EmptySpec, "author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json")
		assert.NoError(t, err)
		assert.Equal(t, "author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json", id)
		assert.True(t, bytes.Contains(b, []byte("r2")))
	})
	t.Run("fetch from named by id", func(t *testing.T) {
		id, b, err = f.FetchByTMIDOrName(remotes.NewRemoteSpec("r1"), "author/manufacturer/mpn/v1.0.0-20231005123243-a49617d2e4fc.tm.json")
		assert.NoError(t, err)
		assert.Equal(t, "author/manufacturer/mpn/v1.0.0-20231005123243-a49617d2e4fc.tm.json", id)
		assert.True(t, bytes.Contains(b, []byte("r1")))

		id, b, err = f.FetchByTMIDOrName(remotes.NewRemoteSpec("r1"), "author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json")
		assert.Error(t, err)

		id, b, err = f.FetchByTMIDOrName(remotes.NewRemoteSpec("r2"), "author/manufacturer/mpn/v1.0.0-20231005123243-a49617d2e4fc.tm.json")
		assert.Error(t, err)

		id, b, err = f.FetchByTMIDOrName(remotes.NewRemoteSpec("r2"), "author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json")
		assert.NoError(t, err)
		assert.Equal(t, "author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json", id)
		assert.True(t, bytes.Contains(b, []byte("r2")))
	})
	t.Run("fetch from unspecified by name", func(t *testing.T) {
		id, b, err = f.FetchByTMIDOrName(remotes.EmptySpec, "author/manufacturer/mpn")
		assert.NoError(t, err)
		assert.Equal(t, "author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json", id)
		assert.True(t, bytes.Contains(b, []byte("r2")))

	})
	t.Run("fetch from named by name", func(t *testing.T) {
		id, b, err = f.FetchByTMIDOrName(remotes.NewRemoteSpec("r1"), "author/manufacturer/mpn")
		assert.NoError(t, err)
		assert.Equal(t, "author/manufacturer/mpn/v1.0.0-20231005123243-a49617d2e4fc.tm.json", id)
		assert.True(t, bytes.Contains(b, []byte("r1")))

		id, b, err = f.FetchByTMIDOrName(remotes.NewRemoteSpec("r2"), "author/manufacturer/mpn")
		assert.NoError(t, err)
		assert.Equal(t, "author/manufacturer/mpn/v1.0.0-20231205123243-c49617d2e4fc.tm.json", id)
		assert.True(t, bytes.Contains(b, []byte("r2")))

	})
}
