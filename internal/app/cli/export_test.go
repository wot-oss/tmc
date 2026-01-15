package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos/mocks"
	rMocks "github.com/wot-oss/tmc/internal/testutils/reposmocks"
)

var exportListRes = model.SearchResult{
	Entries: []model.FoundEntry{
		{
			Name:         "a-corp/eagle/bt2000",
			Author:       model.SchemaAuthor{Name: "a-corp"},
			Manufacturer: model.SchemaManufacturer{Name: "eagle"},
			Mpn:          "bt2000",
			FoundIn:      model.FoundSource{RepoName: "r1"},
			Versions: []model.FoundVersion{
				{
					IndexVersion: &model.IndexVersion{
						TMID:        "a-corp/eagle/bt2000/v1.0.0-20240108140117-243d1b462ccc.tm.json",
						Description: "desc version v1.0.0",
						Version:     model.Version{Model: "1.0.0"},
						Digest:      "243d1b462ccc",
						TimeStamp:   "20240108140117",
						ExternalID:  "ext-2",
					},
					FoundIn: model.FoundSource{RepoName: "r1"},
				},
				{
					IndexVersion: &model.IndexVersion{
						TMID:        "a-corp/eagle/bt2000/v1.0.0-20231231153548-243d1b462ddd.tm.json",
						Description: "desc version v0.0.0",
						Version:     model.Version{Model: "0.0.0"},
						Digest:      "243d1b462ddd",
						TimeStamp:   "20231231153548",
						ExternalID:  "ext-1",
						AttachmentContainer: model.AttachmentContainer{
							Attachments: []model.Attachment{{
								Name: "CHANGELOG.md",
							}},
						},
					},
					FoundIn: model.FoundSource{RepoName: "r1"},
				},
			},
			AttachmentContainer: model.AttachmentContainer{
				Attachments: []model.Attachment{{
					Name: "README.md",
				}},
			},
		},
		{
			Name:         "b-corp/frog/bt3000",
			Author:       model.SchemaAuthor{Name: "b-corp"},
			Manufacturer: model.SchemaManufacturer{Name: "frog"},
			Mpn:          "bt3000",
			FoundIn:      model.FoundSource{RepoName: "r1"},
			Versions: []model.FoundVersion{
				{
					IndexVersion: &model.IndexVersion{
						TMID:        "b-corp/frog/bt3000/v1.0.0-20240108140117-743d1b462uuu.tm.json",
						Description: "desc version v1.0.0",
						Version:     model.Version{Model: "1.0.0"},
						Digest:      "743d1b462uuu",
						TimeStamp:   "20240108140117",
						ExternalID:  "ext-3",
					},
					FoundIn: model.FoundSource{RepoName: "r1"},
				},
			},
		},
	},
}

var exportSingleListRes model.SearchResult = model.SearchResult{
	Entries: []model.FoundEntry{
		{
			Name:         "b-corp/frog/bt3000",
			Author:       model.SchemaAuthor{Name: "b-corp"},
			Manufacturer: model.SchemaManufacturer{Name: "frog"},
			Mpn:          "bt3000",
			Versions: []model.FoundVersion{
				{
					IndexVersion: &model.IndexVersion{
						TMID:        "b-corp/frog/bt3000/v1.0.0-20240108140117-743d1b462uuu.tm.json",
						Description: "desc version v1.0.0",
						Version:     model.Version{Model: "1.0.0"},
						Digest:      "743d1b462uuu",
						TimeStamp:   "20240108140117",
						ExternalID:  "ext-3",
						AttachmentContainer: model.AttachmentContainer{
							Attachments: []model.Attachment{{
								Name: "README.md",
							}},
						},
					},
					FoundIn: model.FoundSource{RepoName: "r1"},
				},
			},
		},
	},
}

func TestExport(t *testing.T) {

	t.Run("with ok", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "tmc-export")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// given: a RepoManager and a repo having 3 ThingModels and 2 attachments
		repoSpec := model.NewRepoSpec("r1")
		r := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repoSpec, r, nil))

		tmID_1 := exportListRes.Entries[0].Versions[0].TMID
		tmID_2 := exportListRes.Entries[0].Versions[1].TMID
		tmID_3 := exportListRes.Entries[1].Versions[0].TMID
		tmContent1 := []byte("some TM content 1")
		tmContent2 := []byte("some TM content 2")
		tmContent3 := []byte("some TM content 3")
		readmeContent := []byte("# Read This First")
		changelogContent := []byte("# CHANGELOG")
		var sp *model.Filters
		r.On("List", mock.Anything, sp).Return(exportListRes, nil).Once()
		r.On("Fetch", mock.Anything, tmID_1).Return(tmID_1, tmContent1, nil).Once()
		r.On("Fetch", mock.Anything, tmID_2).Return(tmID_2, tmContent2, nil).Once()
		r.On("Fetch", mock.Anything, tmID_3).Return(tmID_3, tmContent3, nil).Once()
		r.On("FetchAttachment", mock.Anything, model.NewTMNameAttachmentContainerRef(exportListRes.Entries[0].Name), "README.md").Return(readmeContent, nil).Once()
		r.On("FetchAttachment", mock.Anything, model.NewTMIDAttachmentContainerRef(tmID_2), "CHANGELOG.md").Return(changelogContent, nil).Once()

		// when: exporting from repo
		err = Export(context.Background(), repoSpec, nil, tempDir, false, true, OutputFormatPlain)

		// then: there is no error
		assert.NoError(t, err)
		// and then: the exported ThingModels are written to the output path
		assertFile(t, filepath.Join(tempDir, tmID_1), tmContent1)
		assertFile(t, filepath.Join(tempDir, tmID_2), tmContent2)
		assertFile(t, filepath.Join(tempDir, tmID_3), tmContent3)
		ver, _ := strings.CutSuffix(filepath.Base(tmID_2), ".tm.json")
		assertFile(t, filepath.Join(tempDir, exportListRes.Entries[0].Name, model.AttachmentsDir, "README.md"), readmeContent)
		assertFile(t, filepath.Join(tempDir, exportListRes.Entries[0].Name, model.AttachmentsDir, ver, "CHANGELOG.md"), changelogContent)
	})
	// t.Run("with ok with json output", func(t *testing.T) {
	// 	tempDir, err := os.MkdirTemp("", "tmc-export")
	// 	assert.NoError(t, err)
	// 	defer os.RemoveAll(tempDir)
	// 	restore, getStdout := testutils.ReplaceStdout()
	// 	defer restore()

	// 	// given: a RepoManager and a repo having 3 ThingModels and 2 attachments
	// 	repoSpec := model.NewRepoSpec("r1")
	// 	r := mocks.NewRepo(t)
	// 	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repoSpec, r, nil))

	// 	tmID_1 := exportListRes.Entries[0].Versions[0].TMID
	// 	tmID_2 := exportListRes.Entries[0].Versions[1].TMID
	// 	tmID_3 := exportListRes.Entries[1].Versions[0].TMID
	// 	tmContent1 := []byte("some TM content 1")
	// 	tmContent2 := []byte("some TM content 2")
	// 	tmContent3 := []byte("some TM content 3")
	// 	readmeContent := []byte("# Read This First")
	// 	changelogContent := []byte("# CHANGELOG")
	// 	var sp *model.Filters
	// 	r.On("List", mock.Anything, sp).Return(exportListRes, nil).Once()
	// 	r.On("Fetch", mock.Anything, tmID_1).Return(tmID_1, tmContent1, nil).Once()
	// 	r.On("Fetch", mock.Anything, tmID_2).Return(tmID_2, tmContent2, nil).Once()
	// 	r.On("Fetch", mock.Anything, tmID_3).Return(tmID_3, tmContent3, nil).Once()
	// 	r.On("FetchAttachment", mock.Anything, model.NewTMNameAttachmentContainerRef(exportListRes.Entries[0].Name), "README.md").Return(readmeContent, nil).Once()
	// 	r.On("FetchAttachment", mock.Anything, model.NewTMIDAttachmentContainerRef(tmID_2), "CHANGELOG.md").Return(changelogContent, nil).Once()

	// 	// when: exporting from repo
	// 	err = Export(context.Background(), repoSpec, nil, tempDir, false, true, OutputFormatJSON)

	// 	// then: there is no error
	// 	assert.NoError(t, err)
	// 	// and then: the exported ThingModels are written to the output path
	// 	assertFile(t, filepath.Join(tempDir, tmID_1), tmContent1)
	// 	assertFile(t, filepath.Join(tempDir, tmID_2), tmContent2)
	// 	assertFile(t, filepath.Join(tempDir, tmID_3), tmContent3)
	// 	ver, _ := strings.CutSuffix(filepath.Base(tmID_2), ".tm.json")
	// 	assertFile(t, filepath.Join(tempDir, exportListRes.Entries[0].Name, model.AttachmentsDir, "README.md"), readmeContent)
	// 	assertFile(t, filepath.Join(tempDir, exportListRes.Entries[0].Name, model.AttachmentsDir, ver, "CHANGELOG.md"), changelogContent)

	// 	stdout := getStdout()
	// 	var actual any
	// 	err = json.Unmarshal([]byte(stdout), &actual)
	// 	assert.NoError(t, err)
	// 	expected := []any{map[string]any{"resourceId": "a-corp/eagle/bt2000/.attachments/README.md", "type": "OK"}, map[string]any{"resourceId": "a-corp/eagle/bt2000/v1.0.0-20240108140117-243d1b462ccc.tm.json", "type": "OK"}, map[string]any{"resourceId": "a-corp/eagle/bt2000/v1.0.0-20231231153548-243d1b462ddd.tm.json", "type": "OK"}, map[string]any{"resourceId": "a-corp/eagle/bt2000/.attachments/v1.0.0-20231231153548-243d1b462ddd/CHANGELOG.md", "type": "OK"}, map[string]any{"resourceId": "b-corp/frog/bt3000/v1.0.0-20240108140117-743d1b462uuu.tm.json", "type": "OK"}}
	// 	assert.Equal(t, expected, actual)

	// })

	t.Run("with empty output path", func(t *testing.T) {
		// given: a RepoManager and a repo
		repoSpec := model.NewRepoSpec("r1")
		r := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repoSpec, r, nil))

		// and given: an empty output path
		outputPath := ""

		// when: exporting from repo
		err := Export(context.Background(), repoSpec, nil, outputPath, false, false, OutputFormatPlain)

		// then: there is an error
		assert.Error(t, err)
		// and then: there are no calls on the repo
		r.AssertNotCalled(t, "List", mock.Anything, mock.Anything)
		r.AssertNotCalled(t, "Fetch", mock.Anything, mock.Anything)
	})

	t.Run("with output path is a file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "tmc-export")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// given: a RepoManager and a repo
		repoSpec := model.NewRepoSpec("r1")
		r := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repoSpec, r, nil))

		// and given: an output path that is actually a file
		outputPath := filepath.Join(tempDir, "foo.bar")
		_ = os.WriteFile(outputPath, []byte("foobar"), 0660)

		// when: exporting from repo
		err = Export(context.Background(), repoSpec, nil, outputPath, false, false, OutputFormatPlain)

		// then: there is an error
		assert.Error(t, err)
		// and then: there are no calls on the repo
		r.AssertNotCalled(t, "List", mock.Anything, mock.Anything)
		r.AssertNotCalled(t, "Fetch", mock.Anything, mock.Anything)
	})

	// t.Run("with error accessing a repo", func(t *testing.T) {
	// 	tempDir, err := os.MkdirTemp("", "tmc-export")
	// 	assert.NoError(t, err)
	// 	defer os.RemoveAll(tempDir)

	// 	restoreStdout, getStdout := testutils.ReplaceStdout()
	// 	restoreStderr, getStderr := testutils.ReplaceStderr()
	// 	defer restoreStdout()
	// 	defer restoreStderr()

	// 	// given: a RepoManager and 2 repos
	// 	repoSpec1 := model.NewRepoSpec("r1")
	// 	repoSpec2 := model.NewRepoSpec("r2")
	// 	r1 := mocks.NewRepo(t)
	// 	r2 := mocks.NewRepo(t)

	// 	repoMap := map[string]repos.Repo{repoSpec1.RepoName(): r1, repoSpec2.RepoName(): r2}
	// 	rMocks.MockReposGet(t, func(s model.RepoSpec) (repos.Repo, error) {
	// 		return repoMap[s.RepoName()], nil
	// 	})
	// 	rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r1, r2))

	// 	var sp *model.Filters

	// 	// and given: repo 1 returns a ThingModel that can be fetched
	// 	tmID := exportSingleListRes.Entries[0].Versions[0].TMID
	// 	tmContent := []byte("some content")
	// 	r1.On("Spec").Return(repoSpec1).Maybe()
	// 	r1.On("List", mock.Anything, sp).Return(exportSingleListRes, nil).Once()
	// 	r1.On("Fetch", mock.Anything, tmID).Return(tmID, tmContent, nil).Once()

	// 	// and given: repo 2 returns an error when accessing
	// 	accessError := errors.New("some repo access error")
	// 	r2.On("Spec").Return(repoSpec2).Maybe()
	// 	r2.On("List", mock.Anything, sp).Return(model.SearchResult{}, accessError).Once()

	// 	// when: exporting from both repos
	// 	err = Export(context.Background(), model.EmptySpec, nil, tempDir, false, false, OutputFormatPlain)
	// 	stdout := getStdout()
	// 	stderr := getStderr()

	// 	// then: there is a total error
	// 	assert.Error(t, err)
	// 	// and then: the exported ThingModels from repo 1 is written to the output path
	// 	assertFile(t, filepath.Join(tempDir, tmID), tmContent)
	// 	// and then: there are no fetch calls on repo 2
	// 	r2.AssertNotCalled(t, "Fetch", mock.Anything, mock.Anything)
	// 	// and then: stdout outputs the exported ThingModel
	// 	assert.Contains(t, stdout, tmID)
	// 	// and then: stderr outputs errors for the repo that could not be accessed
	// 	assert.Contains(t, stderr, accessError.Error())
	// })

	t.Run("with error fetching a ThingModel", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "tmc-export")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// given: a RepoManager and a repo having one ThingModel
		repoSpec := model.NewRepoSpec("r1")
		r := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repoSpec, r, nil))

		tmID := exportSingleListRes.Entries[0].Versions[0].TMID
		tmContent := []byte("some TM content 1")
		var sp *model.Filters
		r.On("Spec").Return(repoSpec).Once()
		r.On("List", mock.Anything, sp).Return(exportSingleListRes, nil).Once()

		// and given: repo returns an error when fetching the ThingModel
		fetchError := errors.New("some fetch error")
		r.On("Fetch", mock.Anything, tmID).Return(tmID, tmContent, fetchError).Once()

		// when: exporting from repo
		err = Export(context.Background(), repoSpec, nil, tempDir, false, false, OutputFormatPlain)

		// then: there is a total error
		assert.Error(t, err)
	})
	t.Run("with error fetching an attachment", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "tmc-export")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// given: a RepoManager and a repo having one ThingModel
		repoSpec := model.NewRepoSpec("r1")
		r := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, repoSpec, r, nil))

		tmID := exportSingleListRes.Entries[0].Versions[0].TMID
		tmContent := []byte("some TM content 1")
		var sp *model.Filters
		r.On("List", mock.Anything, sp).Return(exportSingleListRes, nil).Once()
		r.On("Fetch", mock.Anything, tmID).Return(tmID, tmContent, nil).Once()

		// and given: repo returns an error when fetching an Attachment
		r.On("FetchAttachment", mock.Anything, model.NewTMIDAttachmentContainerRef(tmID), "README.md").Return(nil, errors.New("no attachment for you")).Once()

		// when: exporting from repo with attachments
		err = Export(context.Background(), repoSpec, nil, tempDir, false, true, OutputFormatPlain)

		// then: there is a total error
		assert.Error(t, err)
	})
}

// func TestExport_exportThingModel(t *testing.T) {
// 	tempDir, err := os.MkdirTemp("", "tmc-export")
// 	assert.NoError(t, err)
// 	defer os.RemoveAll(tempDir)

// 	// given: a Repo
// 	repoName := "r1"
// 	r := mocks.NewRepo(t)
// 	spec := model.NewRepoSpec(repoName)
// 	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec(repoName), r, nil))
// 	r.On("Spec").Return(spec)

// 	tmID := exportListRes.Entries[0].Versions[0].TMID

// 	t.Run("result with success", func(t *testing.T) {
// 		// given: ThingModel can be fetched successfully
// 		r.On("Fetch", mock.Anything, tmID).Return(tmID, []byte("some TM content"), nil).Once()
// 		// when: exporting from repo
// 		res, err := exportThingModel(context.Background(), tempDir, exportListRes.Entries[0].Versions[0], false)
// 		// then: there is no error
// 		assert.NoError(t, err)
// 		// and then: the result is opResultOK
// 		assert.Equal(t, opResultOK, res.Type)
// 		assert.Equal(t, tmID, res.ResourceId)
// 		assert.Equal(t, "", res.Text)
// 	})

// 	t.Run("result with error", func(t *testing.T) {
// 		// given: ThingModel cannot be fetched successfully
// 		r.On("Fetch", mock.Anything, tmID).Return(tmID, nil, errors.New("fetch failed")).Once()
// 		// when: exporting from repo
// 		res, err := exportThingModel(context.Background(), tempDir, exportListRes.Entries[0].Versions[0], false)
// 		// then: there is an error
// 		assert.Error(t, err)
// 		// and then: the result is opResultErr
// 		assert.Equal(t, opResultErr, res.Type)
// 		assert.NotEmpty(t, res.Text)
// 		assert.Equal(t, tmID, res.ResourceId)
// 	})
// }

func assertFile(t *testing.T, fileName string, fileContent []byte) {
	assert.FileExists(t, fileName)
	file, err := os.ReadFile(fileName)
	assert.NoError(t, err)
	assert.Equal(t, fileContent, file)
}
