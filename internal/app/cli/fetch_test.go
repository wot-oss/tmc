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
		"", false)
	assert.NoError(t, err)
	os.Stdout = old
	_ = w.Close()
	stdout := <-outC
	assert.Equal(t, "{}\n", stdout)
}
func TestFetchExecutor_Fetch_To_OutputFile(t *testing.T) {
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

	e := NewFetchExecutor(rm)
	err = e.Fetch(remotes.NewRemoteSpec("remote"), tmid, "", true)

	fileTxt := filepath.Join(temp, "file.txt")
	_ = os.WriteFile(fileTxt, []byte("text"), 0660)
	err = e.Fetch(remotes.NewRemoteSpec("remote"), tmid, fileTxt, true)
	assert.Error(t, err)

	err = e.Fetch(remotes.NewRemoteSpec("remote"), tmid, fileTxt, false)
	assert.NoError(t, err)
	file, err := os.ReadFile(fileTxt)
	assert.NoError(t, err)
	assert.Equal(t, tm, file)

	err = e.Fetch(remotes.NewRemoteSpec("remote"), tmid, temp, false)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(temp, filepath.Base(aid)))
	file, err = os.ReadFile(filepath.Join(temp, filepath.Base(aid)))
	assert.NoError(t, err)
	assert.Equal(t, tm, file)

	err = e.Fetch(remotes.NewRemoteSpec("remote"), tmid, temp, true)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(temp, aid))
	file, err = os.ReadFile(filepath.Join(temp, aid))
	assert.NoError(t, err)
	assert.Equal(t, tm, file)

}
