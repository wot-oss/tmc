package commands

import (
	"context"
	"errors"
	"os"
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

func TestExport_exportThingModel(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tmc-export")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// given: a Repo
	repoName := "r1"
	r := mocks.NewRepo(t)
	spec := model.NewRepoSpec(repoName)
	fsTarget, err := NewFileSystemExportTarget(tempDir)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec(repoName), r, nil))
	r.On("Spec").Return(spec)

	tmID := exportListRes.Entries[0].Versions[0].TMID

	t.Run("result with success", func(t *testing.T) {
		// given: ThingModel can be fetched successfully
		r.On("Fetch", mock.Anything, tmID).Return(tmID, []byte("some TM content"), nil).Once()
		// when: exporting from repo
		res, err := exportThingModel(context.Background(), spec, fsTarget, exportListRes.Entries[0].Versions[0], false)
		// then: there is no error
		assert.NoError(t, err)
		// and then: the result is opResultOK
		// assert.Equal(t, opResultOK, res.Type)
		assert.Equal(t, tmID, res.ResourceId)
		assert.Equal(t, nil, res.Error)
	})

	t.Run("result with error", func(t *testing.T) {
		// given: ThingModel cannot be fetched successfully
		r.On("Fetch", mock.Anything, tmID).Return(tmID, nil, errors.New("fetch failed")).Once()
		// when: exporting from repo
		res, err := exportThingModel(context.Background(), spec, fsTarget, exportListRes.Entries[0].Versions[0], false)
		// then: there is an error
		assert.Error(t, err)
		// and then: the result is opResultErr
		// assert.Equal(t, opResultErr, res.Type)
		assert.NotEmpty(t, res.Error)
		assert.Equal(t, tmID, res.ResourceId)
	})
}
