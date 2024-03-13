package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes/mocks"
)

func TestFetchExecutor_Fetch_To_Stdout(t *testing.T) {
	old := os.Stdout

	r := mocks.NewRemote(t)

	remotes.MockRemotesGet(t, func(s model.RepoSpec) (remotes.Remote, error) {
		if reflect.DeepEqual(model.NewRemoteSpec("remote"), s) {
			return r, nil
		}
		err := fmt.Errorf("unexpected spec in mock: %v", s)
		remotes.MockFail(t, err)
		return nil, err

	})
	r.On("Fetch", "author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json").
		Return("author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json", []byte("{}"), nil)

	e := NewFetchExecutor()
	rr, w, _ := os.Pipe()
	os.Stdout = w
	outC := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rr)
		outC <- buf.String()
	}()
	err := e.Fetch(model.NewRemoteSpec("remote"), "author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json", "", false)
	assert.NoError(t, err)
	os.Stdout = old
	_ = w.Close()
	stdout := <-outC
	assert.Equal(t, "{}\n", stdout)
}

func TestFetchExecutor_Fetch_To_OutputFolder(t *testing.T) {
	temp, err := os.MkdirTemp("", "fs")
	assert.NoError(t, err)
	defer os.RemoveAll(temp)

	const tmid = "author/manufacturer/mpn/folder/sub/v1.0.0-20220212123243-c49617d2e4fc.tm.json"
	const aid = "author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json"
	var tm = []byte("{}")

	r := mocks.NewRemote(t)
	remotes.MockRemotesGet(t, func(s model.RepoSpec) (remotes.Remote, error) {
		if reflect.DeepEqual(model.NewRemoteSpec("remote"), s) {
			return r, nil
		}
		err := fmt.Errorf("unexpected spec in mock: %v", s)
		remotes.MockFail(t, err)
		return nil, err

	})
	r.On("Fetch", tmid).Return(aid, tm, nil)

	// given: a fetch executor
	e := NewFetchExecutor()

	// when: fetching to output folder
	err = e.Fetch(model.NewRemoteSpec("remote"), tmid, temp, false)
	// then: the file exists below the output folder with tree structure given by the ID
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(temp, aid))
	s, err := os.Stat(filepath.Join(temp, aid))
	modTimeOld := s.ModTime()

	// when: fetching again the ID to same output folder
	time.Sleep(time.Millisecond * 200)
	err = e.Fetch(model.NewRemoteSpec("remote"), tmid, temp, false)
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
	err = e.Fetch(model.NewRemoteSpec("remote"), tmid, fileNoDir, false)
	// then: an error is returned
	assert.Error(t, err)
}
