package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes/mocks"
)

var listResult = model.SearchResult{
	Entries: []model.FoundEntry{
		{
			Name:         "a-corp/eagle/BT2000",
			Author:       model.SchemaAuthor{Name: "a-corp"},
			Manufacturer: model.SchemaManufacturer{Name: "eagle"},
			Mpn:          "BT2000",
			Versions: []model.FoundVersion{
				{
					TOCVersion: model.TOCVersion{
						TMID:        "a-corp/eagle/BT2000/v1.0.0-20240108140117-243d1b462ccc.tm.json",
						Description: "desc version v1.0.0",
						Version:     model.Version{Model: "1.0.0"},
						Digest:      "243d1b462ccc",
						TimeStamp:   "20240108140117",
						ExternalID:  "ext-2",
					},
					FoundIn: model.FoundSource{RemoteName: "r1"},
				},
				{
					TOCVersion: model.TOCVersion{
						TMID:        "a-corp/eagle/BT2000/v1.0.0-20231231153548-243d1b462ddd.tm.json",
						Description: "desc version v0.0.0",
						Version:     model.Version{Model: "0.0.0"},
						Digest:      "243d1b462ddd",
						TimeStamp:   "20231231153548",
						ExternalID:  "ext-1",
					},
					FoundIn: model.FoundSource{RemoteName: "r1"},
				},
			},
		},
		{
			Name:         "b-corp/frog/BT3000",
			Author:       model.SchemaAuthor{Name: "b-corp"},
			Manufacturer: model.SchemaManufacturer{Name: "frog"},
			Mpn:          "BT3000",
			Versions: []model.FoundVersion{
				{
					TOCVersion: model.TOCVersion{
						TMID:        "b-corp/frog/BT3000/v1.0.0-20240108140117-743d1b462uuu.tm.json",
						Description: "desc version v1.0.0",
						Version:     model.Version{Model: "1.0.0"},
						Digest:      "743d1b462uuu",
						TimeStamp:   "20240108140117",
						ExternalID:  "ext-3",
					},
					FoundIn: model.FoundSource{RemoteName: "r1"},
				},
			},
		},
	},
}

func TestPullExecutor_Pull(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tmc-pull")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// given: a RemoteManager and a Remote having 3 ThingModels
	r := mocks.NewRemote(t)
	remotes.MockRemotesGet(t, func(s model.RepoSpec) (remotes.Remote, error) {
		if assert.Equal(t, model.NewRemoteSpec("r1"), s) {
			return r, nil
		}
		err := fmt.Errorf("unexpected spec in mock: %v", s)
		remotes.MockFail(t, err)
		return nil, err

	})

	tmID_1 := listResult.Entries[0].Versions[0].TMID
	tmID_2 := listResult.Entries[0].Versions[1].TMID
	tmID_3 := listResult.Entries[1].Versions[0].TMID
	tmContent1 := []byte("some TM content 1")
	tmContent2 := []byte("some TM content 2")
	tmContent3 := []byte("some TM content 3")
	search := &model.SearchParams{}

	r.On("List", search).Return(listResult, nil).Once()
	r.On("Fetch", tmID_1).Return(tmID_1, tmContent1, nil).Once()
	r.On("Fetch", tmID_2).Return(tmID_2, tmContent2, nil).Once()
	r.On("Fetch", tmID_3).Return(tmID_3, tmContent3, nil).Once()

	// and given: a PullExecutor under test
	underTest := NewPullExecutor()
	// when: pulling from remote
	err = underTest.Pull(model.NewRemoteSpec("r1"), search, tempDir, false)
	// then: there is no error
	assert.NoError(t, err)
	// and then: the pulled ThingModels are written to the output path
	assertFile(t, filepath.Join(tempDir, tmID_1), tmContent1)
	assertFile(t, filepath.Join(tempDir, tmID_2), tmContent2)
	assertFile(t, filepath.Join(tempDir, tmID_3), tmContent3)
}

func TestPullExecutor_pullThingModel(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tmc-pull")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// given: a Remote
	r := mocks.NewRemote(t)
	spec := model.NewRemoteSpec("r1")
	remotes.MockRemotesGet(t, func(s model.RepoSpec) (remotes.Remote, error) {
		if assert.Equal(t, spec, s) {
			return r, nil
		}
		err := fmt.Errorf("unexpected spec in mock: %v", s)
		remotes.MockFail(t, err)
		return nil, err

	})
	r.On("Spec").Return(spec)

	fc := commands.NewFetchCommand()
	tmID := listResult.Entries[0].Versions[0].TMID

	// and given: a PullExecutor under test
	underTest := NewPullExecutor()

	t.Run("result with success", func(t *testing.T) {
		// given: ThingModel can be fetched successfully
		r.On("Fetch", tmID).Return(tmID, []byte("some TM content"), nil).Once()
		// when: pulling from remote
		res, err := underTest.pullThingModel(fc, tempDir, listResult.Entries[0].Versions[0], false)
		// then: there is no error
		assert.NoError(t, err)
		// and then: the result is PullOK
		assert.Equal(t, PullOK, res.typ)
		assert.Equal(t, tmID, res.tmid)
		assert.Equal(t, "", res.text)
	})

	t.Run("result with error", func(t *testing.T) {
		// given: ThingModel cannot be fetched successfully
		r.On("Fetch", tmID).Return(tmID, nil, errors.New("fetch failed")).Once()
		// when: pulling from remote
		res, err := underTest.pullThingModel(fc, tempDir, listResult.Entries[0].Versions[0], false)
		// then: there is an error
		assert.Error(t, err)
		// and then: the result is PullErr
		assert.Equal(t, PullErr, res.typ)
		assert.NotEmpty(t, res.text)
		assert.Equal(t, tmID, res.tmid)
	})
}

func TestPullExecutor_Pull_InvalidOutputPath(t *testing.T) {
	// given: a Remote having 3 ThingModels
	r := mocks.NewRemote(t)
	remotes.MockRemotesGet(t, func(s model.RepoSpec) (remotes.Remote, error) {
		return r, nil
	})
	r.On("List", mock.Anything).Return(listResult, nil).Maybe()
	search := &model.SearchParams{}

	// and given: a PullExecutor under test
	underTest := NewPullExecutor()

	t.Run("with empty output path", func(t *testing.T) {
		// given: an empty output path
		outputPath := ""
		// when: pulling from remote
		err := underTest.Pull(model.NewRemoteSpec("r1"), search, outputPath, false)
		// then: there is an error
		assert.Error(t, err)
		// and then: there are no calls on Remote
		r.AssertNotCalled(t, "List", mock.Anything)
		r.AssertNotCalled(t, "Fetch", mock.Anything)
	})

	t.Run("with output path is a file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "tmc-pull")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// given: an output path that is actually a file
		outputPath := filepath.Join(tempDir, "foo.bar")
		_ = os.WriteFile(outputPath, []byte("foobar"), 0660)
		// when: pulling from remote
		err = underTest.Pull(model.NewRemoteSpec("r1"), search, outputPath, false)
		// then: there is an error
		assert.Error(t, err)
		// and then: there are no calls on Remote
		r.AssertNotCalled(t, "List", mock.Anything)
		r.AssertNotCalled(t, "Fetch", mock.Anything)
	})
}

func assertFile(t *testing.T, fileName string, fileContent []byte) {
	assert.FileExists(t, fileName)
	file, err := os.ReadFile(fileName)
	assert.NoError(t, err)
	assert.Equal(t, fileContent, file)
}
