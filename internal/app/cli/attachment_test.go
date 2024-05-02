package cli

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos/mocks"
	"github.com/wot-oss/tmc/internal/testutils"
	rMocks "github.com/wot-oss/tmc/internal/testutils/reposmocks"
)

func TestAttachmentList(t *testing.T) {
	restore, getOutput := testutils.ReplaceStdout()
	defer restore()

	r := mocks.NewRepo(t)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewDirSpec("somewhere"), r, nil))
	ctx := context.Background()
	tmNameOrId := "author/manufacturer/mpn"
	r.On("ListAttachments", ctx, tmNameOrId).Return([]string{"README.md", "User Guide.pdf"}, nil).Once()
	err := AttachmentList(ctx, model.NewDirSpec("somewhere"), tmNameOrId)
	assert.NoError(t, err)
	stdout := getOutput()
	assert.Equal(t, "README.md\nUser Guide.pdf\n", stdout)
}

func TestAttachmentPush(t *testing.T) {
	r := mocks.NewRepo(t)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewDirSpec("somewhere"), r, nil))
	ctx := context.Background()
	tmNameOrId := "author/manufacturer/mpn"
	attName := "README.md"
	attFile := "../../../test/data/attachments/" + attName
	attContent, err := os.ReadFile(attFile)
	assert.NoError(t, err)
	r.On("PushAttachment", ctx, tmNameOrId, attName, attContent).Return(nil).Once()
	r.On("Index", ctx, tmNameOrId).Return(nil).Once()
	err = AttachmentPush(ctx, model.NewDirSpec("somewhere"), tmNameOrId, attFile)
	assert.NoError(t, err)
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
	r.On("FetchAttachment", ctx, tmNameOrId, attName).Return(attContent, nil).Once()
	err := AttachmentFetch(ctx, model.NewDirSpec("somewhere"), tmNameOrId, attName)
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
	r.On("DeleteAttachment", ctx, tmNameOrId, attName).Return(nil).Once()
	r.On("Index", ctx, tmNameOrId).Return(nil).Once()
	err := AttachmentDelete(ctx, model.NewDirSpec("somewhere"), tmNameOrId, attName)
	assert.NoError(t, err)
}
