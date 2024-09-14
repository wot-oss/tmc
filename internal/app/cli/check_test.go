package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/repos/mocks"
	"github.com/wot-oss/tmc/internal/testutils"
	rMocks "github.com/wot-oss/tmc/internal/testutils/reposmocks"
)

func TestCheckIntegrity_IndexedResources(t *testing.T) {

	t.Run("with repository not found", func(t *testing.T) {
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r1"), nil, repos.ErrRepoNotFound))
		err := CheckIntegrity(context.Background(), model.NewRepoSpec("r1"), nil)
		assert.ErrorIs(t, err, repos.ErrRepoNotFound)
	})

	t.Run("with not a repository", func(t *testing.T) {
		r := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r1"), r, nil))
		r.On("List", mock.Anything, mock.Anything).Return(model.SearchResult{}, repos.ErrNoIndex)
		err := CheckIntegrity(context.Background(), model.NewRepoSpec("r1"), nil)
		assert.NoError(t, err)
	})

	t.Run("without error", func(t *testing.T) {
		r := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r1"), r, nil))
		sr := model.SearchResult{Entries: []model.FoundEntry{
			{
				Name: "mycompany/bartech/bazlamp",
				Manufacturer: model.SchemaManufacturer{
					Name: "bartech",
				},
				Mpn: "bazlamp",
				Author: model.SchemaAuthor{
					Name: "mycompany",
				},
				Versions: []model.FoundVersion{
					{
						IndexVersion: &model.IndexVersion{
							TMID: "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json",
							AttachmentContainer: model.AttachmentContainer{
								Attachments: []model.Attachment{{Name: "CHANGELOG.md"}},
							},
						},
					},
				},
				AttachmentContainer: model.AttachmentContainer{
					Attachments: []model.Attachment{{Name: "README.md"}},
				},
			},
			{
				Name: "yourcompany/bartech/bazlamp",
				Manufacturer: model.SchemaManufacturer{
					Name: "bartech",
				},
				Mpn: "bazlamp",
				Author: model.SchemaAuthor{
					Name: "yourcompany",
				},
				Versions: []model.FoundVersion{
					{
						IndexVersion: &model.IndexVersion{
							TMID: "yourcompany/bartech/bazlamp/v0.0.1-20240101120000-35afe53c124a.tm.json",
						},
					},
				},
			},
		}}
		r.On("List", mock.Anything, mock.Anything).Return(sr, nil)
		r.On("Fetch", mock.Anything, "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json").Return("mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json", []byte(tm1), nil)
		r.On("Fetch", mock.Anything, "yourcompany/bartech/bazlamp/v0.0.1-20240101120000-35afe53c124a.tm.json").Return("yourcompany/bartech/bazlamp/v0.0.1-20240101120000-35afe53c124a.tm.json", []byte(tm6), nil)
		r.On("FetchAttachment", mock.Anything, model.NewTMNameAttachmentContainerRef("mycompany/bartech/bazlamp"), "README.md").Return([]byte("# READ THIS"), nil)
		r.On("FetchAttachment", mock.Anything, model.NewTMIDAttachmentContainerRef("mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json"), "CHANGELOG.md").Return([]byte("# THIS HAS CHANGED"), nil)
		r.On("CheckIntegrity", mock.Anything, mock.Anything).Return(nil, nil).Once()
		err := CheckIntegrity(context.Background(), model.NewRepoSpec("r1"), nil)
		assert.NoError(t, err)
	})
	t.Run("with attachment not found", func(t *testing.T) {
		restore, getStdout := testutils.ReplaceStdout()
		defer restore()

		r := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r1"), r, nil))
		sr := model.SearchResult{Entries: []model.FoundEntry{
			{
				Name: "mycompany/bartech/bazlamp",
				Manufacturer: model.SchemaManufacturer{
					Name: "bartech",
				},
				Mpn: "bazlamp",
				Author: model.SchemaAuthor{
					Name: "mycompany",
				},
				Versions: []model.FoundVersion{
					{
						IndexVersion: &model.IndexVersion{
							TMID: "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json",
						},
					},
				},
				AttachmentContainer: model.AttachmentContainer{
					Attachments: []model.Attachment{{Name: "README.md"}},
				},
			},
		}}
		r.On("List", mock.Anything, mock.Anything).Return(sr, nil)
		r.On("Fetch", mock.Anything, "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json").Return("mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json", []byte(tm1), nil)
		// when: FetchAttachments returns an error
		r.On("FetchAttachment", mock.Anything, model.NewTMNameAttachmentContainerRef("mycompany/bartech/bazlamp"), "README.md").Return(nil, model.ErrAttachmentNotFound)
		r.On("CheckIntegrity", mock.Anything, mock.Anything).Return(nil, nil).Once()
		err := CheckIntegrity(context.Background(), model.NewRepoSpec("r1"), nil)
		// then: there is an error
		assert.Error(t, err)
		// and then: stdout contains the correct error message
		assert.Contains(t, getStdout(), "attachment not found")
	})
	t.Run("with TM file not found", func(t *testing.T) {
		r := mocks.NewRepo(t)
		restore, getStdout := testutils.ReplaceStdout()
		defer restore()
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r1"), r, nil))
		r.On("List", mock.Anything, mock.Anything).Return(sr1, nil)
		// when: fetch returns an error
		r.On("Fetch", mock.Anything, "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json").Return("", nil, model.ErrTMNotFound)
		r.On("CheckIntegrity", mock.Anything, mock.Anything).Return(nil, nil).Once()
		err := CheckIntegrity(context.Background(), model.NewRepoSpec("r1"), nil)
		// then: there is an error
		assert.Error(t, err)
		// and then: stdout contains the correct error message
		assert.Contains(t, getStdout(), "could not fetch the TM file to verify integrity")
	})
	t.Run("with TM file invalid", func(t *testing.T) {
		r := mocks.NewRepo(t)
		restore, getStdout := testutils.ReplaceStdout()
		defer restore()
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r1"), r, nil))
		r.On("List", mock.Anything, mock.Anything).Return(sr1, nil)
		// when: fetch returns an TM without MPN
		r.On("Fetch", mock.Anything, "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json").Return("mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json", []byte(tm4), nil)
		r.On("CheckIntegrity", mock.Anything, mock.Anything).Return(nil, nil).Once()
		err := CheckIntegrity(context.Background(), model.NewRepoSpec("r1"), nil)
		// then: there is an error
		assert.Error(t, err)
		// and then: stdout contains the correct error message
		assert.Contains(t, getStdout(), "invalid TM content")
	})
	t.Run("with missing id in TM file", func(t *testing.T) {
		r := mocks.NewRepo(t)
		restore, getStdout := testutils.ReplaceStdout()
		defer restore()
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r1"), r, nil))
		r.On("List", mock.Anything, mock.Anything).Return(sr1, nil)
		// when: fetch returns an TM without id
		r.On("Fetch", mock.Anything, "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json").Return("mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json", []byte(tm2), nil)
		r.On("CheckIntegrity", mock.Anything, mock.Anything).Return(nil, nil).Once()
		err := CheckIntegrity(context.Background(), model.NewRepoSpec("r1"), nil)
		// then: there is an error
		assert.Error(t, err)
		// and then: stdout contains the correct error message
		assert.Contains(t, getStdout(), "TM id is missing in the file")
	})
	t.Run("with invalid id in TM file", func(t *testing.T) {
		r := mocks.NewRepo(t)
		restore, getStdout := testutils.ReplaceStdout()
		defer restore()
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r1"), r, nil))
		r.On("List", mock.Anything, mock.Anything).Return(sr1, nil)
		// when: fetch returns an TM with invalid id
		r.On("Fetch", mock.Anything, "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json").Return("mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json", []byte(tm3), nil)
		r.On("CheckIntegrity", mock.Anything, mock.Anything).Return(nil, nil).Once()
		err := CheckIntegrity(context.Background(), model.NewRepoSpec("r1"), nil)
		// then: there is an error
		assert.Error(t, err)
		// and then: stdout contains the correct error message
		assert.Contains(t, getStdout(), "TM id in the file is invalid")
	})
	t.Run("with wrong TM location", func(t *testing.T) {
		r := mocks.NewRepo(t)
		restore, getStdout := testutils.ReplaceStdout()
		defer restore()
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r1"), r, nil))
		r.On("List", mock.Anything, mock.Anything).Return(sr1, nil)
		// when: fetch returns incorrect file location
		r.On("Fetch", mock.Anything, "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json").Return("mycompany/bartech/bazlamp/v0.0.1-20240101120000-35afe53c124a.tm.json", []byte(tm1), nil)
		r.On("CheckIntegrity", mock.Anything, mock.Anything).Return(nil, nil).Once()
		err := CheckIntegrity(context.Background(), model.NewRepoSpec("r1"), nil)
		// then: there is an error
		assert.Error(t, err)
		// and then: stdout contains the correct error message
		assert.Contains(t, getStdout(), "TM id does not match the file location")
	})
	t.Run("with wrong TM content", func(t *testing.T) {
		r := mocks.NewRepo(t)
		restore, getStdout := testutils.ReplaceStdout()
		defer restore()
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r1"), r, nil))
		r.On("List", mock.Anything, mock.Anything).Return(sr1, nil)
		// when: fetch returns file with incorrect hash
		r.On("Fetch", mock.Anything, "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json").Return("mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json", []byte(tm5), nil)
		r.On("CheckIntegrity", mock.Anything, mock.Anything).Return(nil, nil).Once()
		err := CheckIntegrity(context.Background(), model.NewRepoSpec("r1"), nil)
		// then: there is an error
		assert.Error(t, err)
		// and then: stdout contains the correct error message
		assert.Contains(t, getStdout(), "file content does not match the digest in ID")
	})
	t.Run("with CheckIntegrity error", func(t *testing.T) {
		restore, getStdout := testutils.ReplaceStdout()
		defer restore()

		r := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r1"), r, nil))
		sr := model.SearchResult{Entries: []model.FoundEntry{
			{
				Name: "mycompany/bartech/bazlamp",
				Manufacturer: model.SchemaManufacturer{
					Name: "bartech",
				},
				Mpn: "bazlamp",
				Author: model.SchemaAuthor{
					Name: "mycompany",
				},
				Versions: []model.FoundVersion{
					{
						IndexVersion: &model.IndexVersion{
							TMID: "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json",
						},
					},
				},
			},
		}}
		r.On("List", mock.Anything, mock.Anything).Return(sr, nil)
		r.On("Fetch", mock.Anything, "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json").Return("mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json", []byte(tm1), nil)
		// when: CheckIntegrity returns an error
		r.On("CheckIntegrity", mock.Anything, mock.Anything).Return([]model.CheckResult{
			{model.CheckErr, "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json", "something unexpected"},
		}, errors.New("something")).Once()
		err := CheckIntegrity(context.Background(), model.NewRepoSpec("r1"), nil)
		// then: there is an error
		assert.Error(t, err)
		// and then: stdout contains the correct error message
		assert.Contains(t, getStdout(), "something unexpected")
	})
}

// correct TM
var tm1 = `{
  "@context": [ "https://www.w3.org/2022/wot/td/v1.1", { "schema":"https://schema.org/" }],
  "@type": "tm:ThingModel",
  "id": "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json",
  "title": "Lamp Thing Model",
  "schema:author": { "schema:name": "MyCompany" },
  "schema:manufacturer": { "schema:name": "BarTech" },
  "schema:mpn": "BazLamp}",  
  "version": { "model": "v.0.0.1" }
}`

// id missing
var tm2 = `{
  "@context": [ "https://www.w3.org/2022/wot/td/v1.1", { "schema":"https://schema.org/" }],
  "@type": "tm:ThingModel", 
  "title": "Lamp Thing Model",
  "schema:author": { "schema:name": "MyCompany" },
  "schema:manufacturer": { "schema:name": "BarTech" },
  "schema:mpn": "BazLamp}",  
  "version": { "model": "v.0.0.2" }
}`

// invalid id
var tm3 = `{
  "@context": [ "https://www.w3.org/2022/wot/td/v1.1", { "schema":"https://schema.org/" }],
  "@type": "tm:ThingModel", 
  "id": "my-custom-id",
  "title": "Lamp Thing Model",
  "schema:author": { "schema:name": "MyCompany" },
  "schema:manufacturer": { "schema:name": "BarTech" },
  "schema:mpn": "BazLamp}",  
  "version": { "model": "v.0.0.3" }
}`

// missing MPN
var tm4 = `{
  "@context": [ "https://www.w3.org/2022/wot/td/v1.1", { "schema":"https://schema.org/" }],
  "@type": "tm:ThingModel",
  "id": "mycompany/bartech/bazlamp/v0.0.4-20240206142430-2cc13316b7d8.tm.json",
  "title": "Lamp Thing Model",
  "schema:author": { "schema:name": "MyCompany" },
  "schema:manufacturer": { "schema:name": "BarTech" }, 
  "version": { "model": "v.0.0.4" }
}`

// invalid hash
var tm5 = `{
  "@context": [ "https://www.w3.org/2022/wot/td/v1.1", { "schema":"https://schema.org/" }],
  "@type": "tm:ThingModel",
  "id": "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json",
  "title": "Lamp Thing Model",
  "schema:author": { "schema:name": "MyCompany" },
  "schema:manufacturer": { "schema:name": "BarTech" },
  "schema:mpn": "BazLamp}",  
  "version": { "model": "v0.0.1" }
}`

// another correct TM
var tm6 = `{
  "@context": [ "https://www.w3.org/2022/wot/td/v1.1", { "schema":"https://schema.org/" }],
  "@type": "tm:ThingModel",
  "id": "yourcompany/bartech/bazlamp/v0.0.1-20240101120000-35afe53c124a.tm.json",
  "title": "Lamp Thing Model",
  "schema:author": { "schema:name": "YourCompany" },
  "schema:manufacturer": { "schema:name": "BarTech" },
  "schema:mpn": "BazLamp}",  
  "version": { "model": "v.0.0.1" }
}`

var sr1 = model.SearchResult{Entries: []model.FoundEntry{
	{
		Name: "mycompany/bartech/bazlamp",
		Manufacturer: model.SchemaManufacturer{
			Name: "bartech",
		},
		Mpn: "bazlamp",
		Author: model.SchemaAuthor{
			Name: "mycompany",
		},
		Versions: []model.FoundVersion{
			{
				IndexVersion: &model.IndexVersion{
					TMID: "mycompany/bartech/bazlamp/v0.0.1-20240101120000-78ff2e36fe32.tm.json",
				},
			},
		},
	},
}}
