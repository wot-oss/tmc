package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

func TestFetchExecutor_Fetch_To_Stdout(t *testing.T) {
	old := os.Stdout

	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)

	rm.On("Get", remotes.NewRemoteSpec("remote")).Return(r, nil)
	r.On("Fetch", "author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json").
		Return("author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json", []byte("{}"), nil)

	e := NewFetchExecutor(rm)
	rr, w, _ := os.Pipe()
	os.Stdout = w
	outC := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rr)
		outC <- buf.String()
	}()
	err := e.Fetch(remotes.NewRemoteSpec("remote"), "author/manufacturer/mpn/folder/sub/v1.0.0-20231205123243-c49617d2e4fc.tm.json",
		"")
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

	rm := remotes.NewMockRemoteManager(t)
	r := remotes.NewMockRemote(t)
	rm.On("Get", remotes.NewRemoteSpec("remote")).Return(r, nil)
	r.On("Fetch", tmid).Return(aid, tm, nil)

	// given: a fetch executor
	e := NewFetchExecutor(rm)

	// when: fetching to output folder
	err = e.Fetch(remotes.NewRemoteSpec("remote"), tmid, temp)
	// then: the file exists below the output folder with tree structure given by the ID
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(temp, aid))
	s, err := os.Stat(filepath.Join(temp, aid))
	modTimeOld := s.ModTime()

	// when: fetching again the ID to same output folder
	err = e.Fetch(remotes.NewRemoteSpec("remote"), tmid, temp)
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
	err = e.Fetch(remotes.NewRemoteSpec("remote"), tmid, fileNoDir)
	// then: an error is returned
	assert.Error(t, err)
}
