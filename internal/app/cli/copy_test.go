package cli

import (
	"context"
	"os"
	"testing"

	"github.com/wot-oss/tmc/internal/utils"

	"github.com/wot-oss/tmc/internal/repos"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos/mocks"
	rMocks "github.com/wot-oss/tmc/internal/testutils/reposmocks"
)

var copyListRes = model.SearchResult{
	Entries: []model.FoundEntry{
		{
			Name:         "omnicorp-tm-department/omnicorp/omnilamp",
			Author:       model.SchemaAuthor{Name: "omnicorp-tm-department"},
			Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
			Mpn:          "omnilamp",
			FoundIn:      model.FoundSource{RepoName: "r1"},
			Versions: []model.FoundVersion{
				{
					IndexVersion: &model.IndexVersion{
						TMID:        "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20240409155220-80424c65e4e6.tm.json",
						Description: "desc version v0.0.0",
						Version:     model.Version{Model: "0.0.0"},
						Digest:      "80424c65e4e6",
						TimeStamp:   "20240409155220",
						ExternalID:  "ext-2",
					},
					FoundIn: model.FoundSource{RepoName: "r1"},
				},
				{
					IndexVersion: &model.IndexVersion{
						TMID:        "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20240409155220-3f779458e453.tm.json",
						Description: "desc version v3.2.1",
						Version:     model.Version{Model: "3.2.1"},
						Digest:      "3f779458e453",
						TimeStamp:   "20240409155220",
						ExternalID:  "ext-2",
					},
					FoundIn: model.FoundSource{RepoName: "r1"},
				},
				{
					IndexVersion: &model.IndexVersion{
						TMID:        "omnicorp-tm-department/omnicorp/omnilamp/v3.11.1-20240409155220-da7dbd7ed830.tm.json",
						Description: "desc version v3.11.1",
						Version:     model.Version{Model: "3.11.1"},
						Digest:      "da7dbd7ed830",
						TimeStamp:   "20240409155220",
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
	},
}

var copySingleListRes model.SearchResult = model.SearchResult{
	Entries: []model.FoundEntry{
		{
			Name:         "omnicorp-tm-department/omnicorp/omnilamp",
			Author:       model.SchemaAuthor{Name: "omnicorp-tm-department"},
			Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
			Mpn:          "omnilamp",
			FoundIn:      model.FoundSource{RepoName: "r1"},
			Versions: []model.FoundVersion{
				{
					IndexVersion: &model.IndexVersion{
						TMID:        "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20240409155220-80424c65e4e6.tm.json",
						Description: "desc version v0.0.0",
						Version:     model.Version{Model: "0.0.0"},
						Digest:      "80424c65e4e6",
						TimeStamp:   "20240409155220",
						ExternalID:  "ext-2",
						AttachmentContainer: model.AttachmentContainer{
							Attachments: []model.Attachment{{
								Name:      "README.md",
								MediaType: "text/markdown",
							}},
						},
					},
					FoundIn: model.FoundSource{RepoName: "r1"},
				},
			},
		},
	},
}

func TestCopy(t *testing.T) {

	t.Run("with ok", func(t *testing.T) {
		// given: a repo having 3 ThingModels and 2 attachments and a target repo
		sourceSpec := model.NewRepoSpec("r1")
		targetSpec := model.NewRepoSpec("target")
		source := mocks.NewRepo(t)
		target := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunctionFromList(t, []model.RepoSpec{sourceSpec, targetSpec}, []repos.Repo{source, target}, []error{nil, nil}))

		tmID_1 := copyListRes.Entries[0].Versions[0].TMID
		tmID_2 := copyListRes.Entries[0].Versions[1].TMID
		tmID_3 := copyListRes.Entries[0].Versions[2].TMID
		_, tmContent1, _ := utils.ReadRequiredFile("../../../test/data/index/omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20240409155220-80424c65e4e6.tm.json")
		_, tmContent2, _ := utils.ReadRequiredFile("../../../test/data/index/omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20240409155220-3f779458e453.tm.json")
		_, tmContent3, _ := utils.ReadRequiredFile("../../../test/data/index/omnicorp-tm-department/omnicorp/omnilamp/v3.11.1-20240409155220-da7dbd7ed830.tm.json")
		readmeContent := []byte("# Read This First")
		changelogContent := []byte("# CHANGELOG")
		var sp *model.SearchParams
		source.On("List", mock.Anything, sp).Return(copyListRes, nil).Once()
		source.On("Fetch", mock.Anything, tmID_1).Return(tmID_1, tmContent1, nil).Once()
		source.On("Fetch", mock.Anything, tmID_2).Return(tmID_2, tmContent2, nil).Once()
		source.On("Fetch", mock.Anything, tmID_3).Return(tmID_3, tmContent3, nil).Once()
		source.On("FetchAttachment", mock.Anything, model.NewTMNameAttachmentContainerRef(copyListRes.Entries[0].Name), "README.md").Return(readmeContent, nil).Once()
		source.On("FetchAttachment", mock.Anything, model.NewTMIDAttachmentContainerRef(tmID_3), "CHANGELOG.md").Return(changelogContent, nil).Once()
		target.On("Import", mock.Anything, model.MustParseTMID(tmID_1), utils.NormalizeLineEndings(tmContent1), repos.ImportOptions{Force: true}).
			Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmID_1, Message: "", Err: nil}, nil).Once()
		target.On("Import", mock.Anything, model.MustParseTMID(tmID_2), utils.NormalizeLineEndings(tmContent2), repos.ImportOptions{Force: true}).
			Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmID_2, Message: "", Err: nil}, nil).Once()
		target.On("Import", mock.Anything, model.MustParseTMID(tmID_3), utils.NormalizeLineEndings(tmContent3), repos.ImportOptions{Force: true}).
			Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmID_3, Message: "", Err: nil}, nil).Once()
		target.On("ImportAttachment", mock.Anything, model.NewTMNameAttachmentContainerRef(copyListRes.Entries[0].Name), model.Attachment{Name: "README.md"}, readmeContent, true).Return(nil).Once()
		target.On("ImportAttachment", mock.Anything, model.NewTMIDAttachmentContainerRef(tmID_3), model.Attachment{Name: "CHANGELOG.md"}, changelogContent, true).Return(nil).Once()
		target.On("Index", mock.Anything, tmID_1, tmID_2, tmID_3).Return(nil)
		target.On("Index", mock.Anything, tmID_1).Return(nil)
		target.On("Index", mock.Anything, tmID_2).Return(nil)
		target.On("Index", mock.Anything, tmID_3).Return(nil)

		// when: copying from repo
		err := Copy(context.Background(), sourceSpec, targetSpec, nil, repos.ImportOptions{Force: true})

		// then: there is no error
		assert.NoError(t, err)
		// and then: all expectations on target mock are met

	})
	t.Run("with only attachment updates", func(t *testing.T) {
		// given: a repo having 3 ThingModels and 2 attachments and a target repo
		sourceSpec := model.NewRepoSpec("r1")
		targetSpec := model.NewRepoSpec("target")
		source := mocks.NewRepo(t)
		target := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunctionFromList(t, []model.RepoSpec{sourceSpec, targetSpec}, []repos.Repo{source, target}, []error{nil, nil}))

		tmID_1 := copyListRes.Entries[0].Versions[0].TMID
		tmID_2 := copyListRes.Entries[0].Versions[1].TMID
		tmID_3 := copyListRes.Entries[0].Versions[2].TMID
		_, tmContent1, _ := utils.ReadRequiredFile("../../../test/data/index/omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20240409155220-80424c65e4e6.tm.json")
		_, tmContent2, _ := utils.ReadRequiredFile("../../../test/data/index/omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20240409155220-3f779458e453.tm.json")
		_, tmContent3, _ := utils.ReadRequiredFile("../../../test/data/index/omnicorp-tm-department/omnicorp/omnilamp/v3.11.1-20240409155220-da7dbd7ed830.tm.json")
		readmeContent := []byte("# Read This First")
		changelogContent := []byte("# CHANGELOG")
		var sp *model.SearchParams
		source.On("List", mock.Anything, sp).Return(copyListRes, nil).Once()
		source.On("Fetch", mock.Anything, tmID_1).Return(tmID_1, tmContent1, nil).Once()
		source.On("Fetch", mock.Anything, tmID_2).Return(tmID_2, tmContent2, nil).Once()
		source.On("Fetch", mock.Anything, tmID_3).Return(tmID_3, tmContent3, nil).Once()
		source.On("FetchAttachment", mock.Anything, model.NewTMNameAttachmentContainerRef(copyListRes.Entries[0].Name), "README.md").Return(readmeContent, nil).Once()
		source.On("FetchAttachment", mock.Anything, model.NewTMIDAttachmentContainerRef(tmID_3), "CHANGELOG.md").Return(changelogContent, nil).Once()
		expRes, impErr := repos.ImportResultFromError(&repos.ErrTMIDConflict{Type: repos.IdConflictSameContent, ExistingId: tmID_1})
		target.On("Import", mock.Anything, model.MustParseTMID(tmID_1), utils.NormalizeLineEndings(tmContent1), repos.ImportOptions{Force: true}).
			Return(expRes, impErr).Once()
		target.On("Import", mock.Anything, model.MustParseTMID(tmID_2), utils.NormalizeLineEndings(tmContent2), repos.ImportOptions{Force: true}).
			Return(repos.ImportResultFromError(&repos.ErrTMIDConflict{Type: repos.IdConflictSameContent, ExistingId: tmID_2})).Once()
		target.On("Import", mock.Anything, model.MustParseTMID(tmID_3), utils.NormalizeLineEndings(tmContent3), repos.ImportOptions{Force: true}).
			Return(repos.ImportResultFromError(&repos.ErrTMIDConflict{Type: repos.IdConflictSameContent, ExistingId: tmID_3})).Once()
		target.On("ImportAttachment", mock.Anything, model.NewTMNameAttachmentContainerRef(copyListRes.Entries[0].Name), model.Attachment{Name: "README.md"}, readmeContent, true).Return(nil).Once()
		target.On("ImportAttachment", mock.Anything, model.NewTMIDAttachmentContainerRef(tmID_3), model.Attachment{Name: "CHANGELOG.md"}, changelogContent, true).Return(nil).Once()

		// when: copying from repo
		err := Copy(context.Background(), sourceSpec, targetSpec, nil, repos.ImportOptions{Force: true})

		// then: there is a total error equal to the first failure
		assert.ErrorIs(t, err, impErr)
		// and then: all expectations on target mock are met

	})
	t.Run("with ignore-existing option", func(t *testing.T) {
		// given: a repo having 3 ThingModels and 2 attachments and a target repo
		sourceSpec := model.NewRepoSpec("r1")
		targetSpec := model.NewRepoSpec("target")
		source := mocks.NewRepo(t)
		target := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunctionFromList(t, []model.RepoSpec{sourceSpec, targetSpec}, []repos.Repo{source, target}, []error{nil, nil}))

		tmID_1 := copyListRes.Entries[0].Versions[0].TMID
		tmID_2 := copyListRes.Entries[0].Versions[1].TMID
		tmID_3 := copyListRes.Entries[0].Versions[2].TMID
		_, tmContent1, _ := utils.ReadRequiredFile("../../../test/data/index/omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20240409155220-80424c65e4e6.tm.json")
		_, tmContent2, _ := utils.ReadRequiredFile("../../../test/data/index/omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20240409155220-3f779458e453.tm.json")
		_, tmContent3, _ := utils.ReadRequiredFile("../../../test/data/index/omnicorp-tm-department/omnicorp/omnilamp/v3.11.1-20240409155220-da7dbd7ed830.tm.json")
		readmeContent := []byte("# Read This First")
		changelogContent := []byte("# CHANGELOG")
		var sp *model.SearchParams
		source.On("List", mock.Anything, sp).Return(copyListRes, nil).Once()
		source.On("Fetch", mock.Anything, tmID_1).Return(tmID_1, tmContent1, nil).Once()
		source.On("Fetch", mock.Anything, tmID_2).Return(tmID_2, tmContent2, nil).Once()
		source.On("Fetch", mock.Anything, tmID_3).Return(tmID_3, tmContent3, nil).Once()
		source.On("FetchAttachment", mock.Anything, model.NewTMNameAttachmentContainerRef(copyListRes.Entries[0].Name), "README.md").Return(readmeContent, nil).Once()
		source.On("FetchAttachment", mock.Anything, model.NewTMIDAttachmentContainerRef(tmID_3), "CHANGELOG.md").Return(changelogContent, nil).Once()
		expRes, impErr := repos.ImportResultFromError(&repos.ErrTMIDConflict{Type: repos.IdConflictSameContent, ExistingId: tmID_1})
		target.On("Import", mock.Anything, model.MustParseTMID(tmID_1), utils.NormalizeLineEndings(tmContent1), repos.ImportOptions{IgnoreExisting: true}).
			Return(expRes, impErr).Once()
		target.On("Import", mock.Anything, model.MustParseTMID(tmID_2), utils.NormalizeLineEndings(tmContent2), repos.ImportOptions{IgnoreExisting: true}).
			Return(repos.ImportResultFromError(&repos.ErrTMIDConflict{Type: repos.IdConflictSameContent, ExistingId: tmID_2})).Once()
		target.On("Import", mock.Anything, model.MustParseTMID(tmID_3), utils.NormalizeLineEndings(tmContent3), repos.ImportOptions{IgnoreExisting: true}).
			Return(repos.ImportResultFromError(&repos.ErrTMIDConflict{Type: repos.IdConflictSameContent, ExistingId: tmID_3})).Once()
		target.On("ImportAttachment", mock.Anything, model.NewTMNameAttachmentContainerRef(copyListRes.Entries[0].Name), model.Attachment{Name: "README.md"}, readmeContent, false).Return(repos.ErrAttachmentExists).Once()
		target.On("ImportAttachment", mock.Anything, model.NewTMIDAttachmentContainerRef(tmID_3), model.Attachment{Name: "CHANGELOG.md"}, changelogContent, false).Return(repos.ErrAttachmentExists).Once()

		// when: copying from repo
		err := Copy(context.Background(), sourceSpec, targetSpec, nil, repos.ImportOptions{IgnoreExisting: true})

		// then: there is no error
		assert.NoError(t, err)
		// and then: all expectations on target mock are met

	})

	t.Run("with empty source spec", func(t *testing.T) {
		sourceSpec := model.NewRepoSpec("r1")
		targetSpec := model.NewRepoSpec("target")
		source := mocks.NewRepo(t)
		target := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunctionFromList(t, []model.RepoSpec{sourceSpec, targetSpec, model.EmptySpec}, []repos.Repo{source, target, nil}, []error{nil, nil, repos.ErrAmbiguous}))
		//rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, source, target))
		err := Copy(context.Background(), model.EmptySpec, model.NewRepoSpec("r1"), nil, repos.ImportOptions{})
		assert.ErrorIs(t, err, repos.ErrAmbiguous)
	})

	t.Run("with error fetching a ThingModel", func(t *testing.T) {
		// given: a repo having 1 ThingModel and 1 attachments and a target repo
		sourceSpec := model.NewRepoSpec("r1")
		targetSpec := model.NewRepoSpec("target")
		source := mocks.NewRepo(t)
		target := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunctionFromList(t, []model.RepoSpec{sourceSpec, targetSpec}, []repos.Repo{source, target}, []error{nil, nil}))

		tmid := copySingleListRes.Entries[0].Versions[0].TMID
		var sp *model.SearchParams
		source.On("List", mock.Anything, sp).Return(copySingleListRes, nil).Once()
		source.On("Fetch", mock.Anything, tmid).Return(tmid, nil, model.ErrTMNotFound).Once()

		// when: copying from repo
		err := Copy(context.Background(), sourceSpec, targetSpec, nil, repos.ImportOptions{})

		// then: there is a total error
		assert.ErrorIs(t, err, model.ErrTMNotFound)
		// and then: all expectations on target mock are met
	})

	t.Run("with error fetching an attachment", func(t *testing.T) {
		// given: a repo having 1 ThingModel and 1 attachments and a target repo
		sourceSpec := model.NewRepoSpec("r1")
		targetSpec := model.NewRepoSpec("target")
		source := mocks.NewRepo(t)
		target := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunctionFromList(t, []model.RepoSpec{sourceSpec, targetSpec}, []repos.Repo{source, target}, []error{nil, nil}))

		tmid := copySingleListRes.Entries[0].Versions[0].TMID
		_, tmContent1, _ := utils.ReadRequiredFile("../../../test/data/index/omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20240409155220-80424c65e4e6.tm.json")
		var sp *model.SearchParams
		source.On("List", mock.Anything, sp).Return(copySingleListRes, nil).Once()
		source.On("Fetch", mock.Anything, tmid).Return(tmid, tmContent1, nil).Once()
		source.On("FetchAttachment", mock.Anything, model.NewTMIDAttachmentContainerRef(tmid), "README.md").Return(nil, model.ErrAttachmentNotFound).Once()
		target.On("Import", mock.Anything, model.MustParseTMID(tmid), utils.NormalizeLineEndings(tmContent1), repos.ImportOptions{}).
			Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid, Message: "", Err: nil}, nil).Once()
		target.On("Index", mock.Anything, tmid).Return(nil).Twice()

		// when: copying from repo
		err := Copy(context.Background(), sourceSpec, targetSpec, nil, repos.ImportOptions{})

		// then: there is a total error
		assert.ErrorIs(t, err, model.ErrAttachmentNotFound)
		// and then: all expectations on target mock are met
	})

	t.Run("with error importing a ThingModel", func(t *testing.T) {
		// given: a repo having 1 ThingModel and 1 attachments and a target repo
		sourceSpec := model.NewRepoSpec("r1")
		targetSpec := model.NewRepoSpec("target")
		source := mocks.NewRepo(t)
		target := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunctionFromList(t, []model.RepoSpec{sourceSpec, targetSpec}, []repos.Repo{source, target}, []error{nil, nil}))

		tmid := copySingleListRes.Entries[0].Versions[0].TMID
		_, tmContent1, _ := utils.ReadRequiredFile("../../../test/data/index/omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20240409155220-80424c65e4e6.tm.json")
		var sp *model.SearchParams
		source.On("List", mock.Anything, sp).Return(copySingleListRes, nil).Once()
		source.On("Fetch", mock.Anything, tmid).Return(tmid, tmContent1, nil).Once()
		res, resErr := repos.ImportResultFromError(repos.ErrNotSupported)
		target.On("Import", mock.Anything, model.MustParseTMID(tmid), utils.NormalizeLineEndings(tmContent1), repos.ImportOptions{}).
			Return(res, resErr).Once()

		// when: copying from repo
		err := Copy(context.Background(), sourceSpec, targetSpec, nil, repos.ImportOptions{})

		// then: there is a total error
		assert.ErrorIs(t, err, repos.ErrNotSupported)
		// and then: all expectations on target mock are met
	})

	t.Run("with error importing an attachment", func(t *testing.T) {
		// given: a repo having 1 ThingModel and 1 attachments and a target repo
		sourceSpec := model.NewRepoSpec("r1")
		targetSpec := model.NewRepoSpec("target")
		source := mocks.NewRepo(t)
		target := mocks.NewRepo(t)
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunctionFromList(t, []model.RepoSpec{sourceSpec, targetSpec}, []repos.Repo{source, target}, []error{nil, nil}))

		tmid := copySingleListRes.Entries[0].Versions[0].TMID
		_, tmContent1, _ := utils.ReadRequiredFile("../../../test/data/index/omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20240409155220-80424c65e4e6.tm.json")
		readmeContent := []byte("# Read This First")
		var sp *model.SearchParams
		source.On("List", mock.Anything, sp).Return(copySingleListRes, nil).Once()
		source.On("Fetch", mock.Anything, tmid).Return(tmid, tmContent1, nil).Once()
		source.On("FetchAttachment", mock.Anything, model.NewTMIDAttachmentContainerRef(tmid), "README.md").Return(readmeContent, nil).Once()
		target.On("Import", mock.Anything, model.MustParseTMID(tmid), utils.NormalizeLineEndings(tmContent1), repos.ImportOptions{}).
			Return(repos.ImportResult{Type: repos.ImportResultOK, TmID: tmid, Message: "", Err: nil}, nil).Once()
		target.On("ImportAttachment", mock.Anything, model.NewTMIDAttachmentContainerRef(tmid), model.Attachment{Name: "README.md", MediaType: "text/markdown"}, readmeContent, false).Return(os.ErrPermission).Once()
		target.On("Index", mock.Anything, tmid).Return(nil).Twice()

		// when: copying from repo
		err := Copy(context.Background(), sourceSpec, targetSpec, nil, repos.ImportOptions{})

		// then: there is a total error
		assert.ErrorIs(t, err, os.ErrPermission)
		// and then: all expectations on target mock are met
	})
}
