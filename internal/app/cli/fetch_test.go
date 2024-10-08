package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos/mocks"
	"github.com/wot-oss/tmc/internal/testutils"
	rMocks "github.com/wot-oss/tmc/internal/testutils/reposmocks"
)

func TestFetchExecutor_Fetch_To_Stdout(t *testing.T) {
	restore, getOutput := testutils.ReplaceStdout()
	defer restore()

	r := mocks.NewRepo(t)

	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("repo"), r, nil))
	r.On("Fetch", mock.Anything, "author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json").
		Return("author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json", []byte("{}"), nil)

	err := Fetch(context.Background(), model.NewRepoSpec("repo"), "author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json", "", false)
	assert.NoError(t, err)
	stdout := getOutput()
	assert.Equal(t, "{}\n", stdout)
}

func TestFetchExecutor_Fetch_To_OutputFolder(t *testing.T) {
	temp, err := os.MkdirTemp("", "fs")
	assert.NoError(t, err)
	defer os.RemoveAll(temp)

	const tmid = "author/manufacturer/mpn/folder/sub/v1.0.0-20220212123243-c49617d2e4fc.tm.json"
	const aid = "author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json"
	var tm = []byte("{}")

	r := mocks.NewRepo(t)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("repo"), r, nil))
	r.On("Fetch", mock.Anything, tmid).Return(aid, tm, nil)

	// when: fetching to output folder
	err = Fetch(context.Background(), model.NewRepoSpec("repo"), tmid, temp, false)
	// then: the file exists below the output folder with tree structure given by the ID
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(temp, aid))
	s, err := os.Stat(filepath.Join(temp, aid))
	modTimeOld := s.ModTime()

	// when: fetching again the ID to same output folder
	time.Sleep(time.Millisecond * 200)
	err = Fetch(context.Background(), model.NewRepoSpec("repo"), tmid, temp, false)
	// then: the file has been overwritten and has a newer mod time
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(temp, aid))
	s, err = os.Stat(filepath.Join(temp, aid))
	modTimeNew := s.ModTime()
	assert.Greater(t, modTimeNew, modTimeOld)

	// given: output folder that is actually a file
	fileNoDir := filepath.Join(temp, "file.txt")
	_ = os.WriteFile(fileNoDir, []byte("text"), 0660)
	// when: fetching to output folder
	err = Fetch(context.Background(), model.NewRepoSpec("repo"), tmid, fileNoDir, false)
	// then: an error is returned
	assert.Error(t, err)
}
