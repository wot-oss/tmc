package cli

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos/mocks"
	"github.com/wot-oss/tmc/internal/testutils"
	rMocks "github.com/wot-oss/tmc/internal/testutils/reposmocks"
)

func TestAttachmentList(t *testing.T) {
	r := mocks.NewRepo(t)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewDirSpec("somewhere"), r, nil))
	ctx := context.Background()
	t.Run("with tmname", func(t *testing.T) {
		restore, getOutput := testutils.ReplaceStdout()
		defer restore()
		tmName := "author/manufacturer/mpn"
		r.On("List", ctx, &model.Filters{Name: tmName}).Return(
			model.SearchResult{
				Entries: []model.FoundEntry{
					{
						AttachmentContainer: model.AttachmentContainer{[]model.Attachment{
							{Name: "README.md", MediaType: "text/markdown"},
							{Name: "User Guide.pdf", MediaType: "application/pdf"},
						}},
					},
				},
			}, nil).Once()
		err := AttachmentList(ctx, model.NewDirSpec("somewhere"), tmName, OutputFormatPlain)
		assert.NoError(t, err)
		stdout := getOutput()
		assert.Equal(t, "NAME            MEDIATYPE        REPO\nREADME.md       text/markdown    \nUser Guide.pdf  application/pdf  \n", stdout)
	})
	t.Run("with resourceId", func(t *testing.T) {
		restore, getOutput := testutils.ReplaceStdout()
		defer restore()
		tmId := "author/manufacturer/mpn/v0.0.0-20240521143452-d662e089b3eb.tm.json"
		r.On("GetTMMetadata", ctx, tmId).Return([]model.FoundVersion{{
			IndexVersion: &model.IndexVersion{
				AttachmentContainer: model.AttachmentContainer{[]model.Attachment{
					{Name: "README.md", MediaType: "text/markdown"},
					{Name: "User Guide.pdf", MediaType: "application/pdf"},
				}},
			},
			FoundIn: model.FoundSource{},
		}}, nil).Once()
		err := AttachmentList(ctx, model.NewDirSpec("somewhere"), tmId, OutputFormatPlain)
		assert.NoError(t, err)
		stdout := getOutput()
		assert.Equal(t, "NAME            MEDIATYPE        REPO\nREADME.md       text/markdown    \nUser Guide.pdf  application/pdf  \n", stdout)
	})
	t.Run("with json output", func(t *testing.T) {
		restore, getOutput := testutils.ReplaceStdout()
		defer restore()
		tmName := "author/manufacturer/mpn"
		r.On("List", ctx, &model.Filters{Name: tmName}).Return(
			model.SearchResult{
				Entries: []model.FoundEntry{
					{
						AttachmentContainer: model.AttachmentContainer{[]model.Attachment{
							{Name: "README.md", MediaType: "text/markdown"},
							{Name: "User Guide.pdf", MediaType: "application/pdf"},
						}},
						FoundIn: model.FoundSource{
							RepoName: "r1",
						},
					},
				},
			}, nil).Once()
		err := AttachmentList(ctx, model.NewDirSpec("somewhere"), tmName, OutputFormatJSON)
		assert.NoError(t, err)
		stdout := getOutput()
		var actual any
		err = json.Unmarshal([]byte(stdout), &actual)
		assert.NoError(t, err)
		expected := []any{map[string]any{"name": "README.md", "mediaType": "text/markdown", "repo": "r1"}, map[string]any{"name": "User Guide.pdf", "mediaType": "application/pdf", "repo": "r1"}}
		assert.Equal(t, expected, actual)
	})

}

func TestAttachmentImport(t *testing.T) {
	r := mocks.NewRepo(t)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewDirSpec("somewhere"), r, nil))
	ctx := context.Background()
	tmNameOrId := "author/manufacturer/mpn"
	attName := "README.md"
	attFile := "../../../test/data/attachments/" + attName
	attContent, err := os.ReadFile(attFile)
	assert.NoError(t, err)
	t.Run("with original file name", func(t *testing.T) {
		r.On("ImportAttachment", ctx, model.NewTMNameAttachmentContainerRef(tmNameOrId), model.Attachment{Name: attName, MediaType: ""}, attContent, true).Return(nil).Once()
		err = AttachmentImport(ctx, model.NewDirSpec("somewhere"), tmNameOrId, attFile, "", "", true)
		assert.NoError(t, err)
	})

	t.Run("with overwritten file name", func(t *testing.T) {
		r.On("ImportAttachment", ctx, model.NewTMNameAttachmentContainerRef(tmNameOrId), model.Attachment{Name: "differentName.md", MediaType: ""}, attContent, true).Return(nil).Once()
		err = AttachmentImport(ctx, model.NewDirSpec("somewhere"), tmNameOrId, attFile, "differentName.md", "", true)
		assert.NoError(t, err)
	})

}

func TestAttachmentFetch(t *testing.T) {
	restore, getOutput := testutils.ReplaceStdout()
	defer restore()

	r := mocks.NewRepo(t)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewDirSpec("somewhere"), r, nil))
	ctx := context.Background()
	tmNameOrId := "author/manufacturer/mpn"
	attName := "README.md"
	attContent := []byte("attachment content")
	r.On("FetchAttachment", ctx, model.NewTMNameAttachmentContainerRef(tmNameOrId), attName).Return(attContent, nil).Once()
	err := AttachmentFetch(ctx, model.NewDirSpec("somewhere"), tmNameOrId, attName, false, "")
	assert.NoError(t, err)

	stdout := getOutput()
	assert.Equal(t, string(attContent), stdout)
}

func TestAttachmentDelete(t *testing.T) {
	r := mocks.NewRepo(t)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewDirSpec("somewhere"), r, nil))
	ctx := context.Background()
	tmNameOrId := "author/manufacturer/mpn"
	attName := "README.md"
	r.On("DeleteAttachment", ctx, model.NewTMNameAttachmentContainerRef(tmNameOrId), attName).Return(nil).Once()
	err := AttachmentDelete(ctx, model.NewDirSpec("somewhere"), tmNameOrId, attName)
	assert.NoError(t, err)
}
