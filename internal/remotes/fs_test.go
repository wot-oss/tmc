package remotes

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

func TestNewFileRemote(t *testing.T) {
	root := "/tmp/tm-catalog1157316148"
	remote, err := NewFileRemote(map[string]any{
		"type": "file",
		"loc":  root,
	}, "")
	assert.NoError(t, err)
	assert.Equal(t, root, remote.root)

	root = "/tmp/tm-catalog1157316148"
	remote, err = NewFileRemote(map[string]any{
		"type": "file",
		"loc":  root,
	}, "")
	assert.NoError(t, err)
	assert.Equal(t, root, remote.root)

	root = "~/tm-catalog"
	remote, err = NewFileRemote(map[string]any{
		"type": "file",
		"loc":  root,
	}, "")
	assert.NoError(t, err)
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, "tm-catalog"), remote.root)

	root = "~/tm-catalog"
	remote, err = NewFileRemote(map[string]any{
		"type": "file",
		"loc":  root,
	}, "")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "tm-catalog"), remote.root)

	root = "~/tm-catalog"
	remote, err = NewFileRemote(map[string]any{
		"type": "file",
		"loc":  root,
	}, "")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "tm-catalog"), remote.root)

	root = "c:\\Users\\user\\Desktop\\tm-catalog"
	remote, err = NewFileRemote(map[string]any{
		"type": "file",
		"loc":  root,
	}, "")
	assert.NoError(t, err)
	assert.Equal(t, filepath.ToSlash("c:\\Users\\user\\Desktop\\tm-catalog"), filepath.ToSlash(remote.root))

	root = "C:\\Users\\user\\Desktop\\tm-catalog"
	remote, err = NewFileRemote(map[string]any{
		"type": "file",
		"loc":  root,
	}, "")
	assert.NoError(t, err)
	assert.Equal(t, filepath.ToSlash("C:\\Users\\user\\Desktop\\tm-catalog"), filepath.ToSlash(remote.root))

}

func TestCreateFileRemoteConfig(t *testing.T) {
	wd, _ := os.Getwd()

	tests := []struct {
		strConf  string
		fileConf string
		expRoot  string
		expErr   bool
	}{
		{"../dir/name", "", filepath.Join(filepath.Dir(wd), "/dir/name"), false},
		{"./dir/name", "", filepath.Join(wd, "dir/name"), false},
		{"dir/name", "", filepath.Join(wd, "dir/name"), false},
		{"/dir/name", "", filepath.Join(filepath.VolumeName(wd), "/dir/name"), false},
		{".", "", filepath.Join(wd), false},
		{filepath.Join(wd, "dir/name"), "", filepath.Join(wd, "dir/name"), false},
		{"~/dir/name", "", "~/dir/name", false},
		{"", ``, "", true},
		{"", `[]`, "", true},
		{"", `{}`, "", true},
		{"", `{"loc":{}}`, "", true},
		{"", `{"loc":"dir/name"}`, filepath.Join(wd, "dir/name"), false},
		{"", `{"loc":"/dir/name"}`, filepath.Join(filepath.VolumeName(wd), "/dir/name"), false},
		{"", `{"loc":"dir/name", "type":"http"}`, "", true},
	}

	for i, test := range tests {
		cf, err := createFileRemoteConfig(test.strConf, []byte(test.fileConf))
		if test.expErr {
			assert.Error(t, err, "error expected in test %d for %s %s", i, test.strConf, test.fileConf)
			continue
		} else {
			assert.NoError(t, err, "no error expected in test %d for %s %s", i, test.strConf, test.fileConf)
		}
		assert.Equalf(t, "file", cf[KeyRemoteType], "in test %d for %s %s", i, test.strConf, test.fileConf)
		assert.Equalf(t, test.expRoot, cf[KeyRemoteLoc], "in test %d for %s %s", i, test.strConf, test.fileConf)

	}
}

func TestValidatesRoot(t *testing.T) {
	remote, _ := NewFileRemote(map[string]any{
		"type": "file",
		"loc":  "/temp/surely-does-not-exist-5245874598745",
	}, "")

	_, err := remote.List(&model.SearchParams{Query: ""})
	assert.ErrorIs(t, err, ErrRootInvalid)
	_, err = remote.Versions("manufacturer/mpn")
	assert.ErrorIs(t, err, ErrRootInvalid)
	_, _, err = remote.Fetch("manufacturer/mpn")
	assert.ErrorIs(t, err, ErrRootInvalid)

}

func TestFileRemote_Fetch(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRemote{
		root: temp,
		name: "fr",
	}
	tmName := "omnicorp-TM-department/omnicorp/omnilamp"
	fileA := []byte("{\"ver\":\"a\"}")
	fileB := []byte("{\"ver\":\"b\"}")
	fileC := []byte("{\"ver\":\"c\"}")
	fileD := []byte("{\"ver\":\"d\"}")
	idA := filepath.Join(tmName, "v1.0.0-20231208142856-a49617d2e4fc.tm.json")
	idB := filepath.Join(tmName, "v1.0.0-20231207142856-b49617d2e4fc.tm.json")
	idC := filepath.Join(tmName, "v1.2.1-20231209142856-c49617d2e4fc.tm.json")
	idD := filepath.Join(tmName, "v0.0.1-20231208142856-d49617d2e4fc.tm.json")
	nameA := filepath.Join(temp, idA)
	nameB := filepath.Join(temp, idB)
	nameC := filepath.Join(temp, idC)
	nameD := filepath.Join(temp, idD)
	_ = os.MkdirAll(filepath.Join(temp, tmName), defaultDirPermissions)
	_ = os.WriteFile(nameA, fileA, defaultFilePermissions)
	_ = os.WriteFile(nameB, fileB, defaultFilePermissions)
	_ = os.WriteFile(nameC, fileC, defaultFilePermissions)
	_ = os.WriteFile(nameD, fileD, defaultFilePermissions)

	actId, bytes, err := r.Fetch(idA)
	assert.NoError(t, err)
	assert.Equal(t, idA, actId)
	assert.Equal(t, fileA, bytes)

	actId, bytes, err = r.Fetch(idB)
	assert.NoError(t, err)
	assert.Equal(t, idB, actId)
	assert.Equal(t, fileB, bytes)

	actId, bytes, err = r.Fetch(filepath.Join(tmName, "v1.0.0-20231212142856-a49617d2e4fc.tm.json"))
	assert.NoError(t, err)
	assert.Equal(t, idA, actId)
	assert.Equal(t, fileA, bytes)

	actId, bytes, err = r.Fetch(filepath.Join(tmName, "v1.0.0-20231212142856-e49617d2e4fc.tm.json"))
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.Equal(t, "", actId)

}

func TestFileRemote_Push(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRemote{
		root: temp,
		name: "fr",
	}
	tmName := "omnicorp-TM-department/omnicorp/omnilamp"
	id := tmName + "/v0.0.0-20231208142856-c49617d2e4fc.tm.json"
	err := r.Push(model.MustParseTMID(id, false), []byte{})
	assert.Error(t, err)
	err = r.Push(model.MustParseTMID(id, false), []byte("{}"))
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(temp, id))

	_ = os.WriteFile(filepath.Join(temp, tmName, "v1.0.0-20231208142856-a49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
	_ = os.WriteFile(filepath.Join(temp, tmName, "v1.0.0-20231207142856-b49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
	_ = os.WriteFile(filepath.Join(temp, tmName, "v1.2.1-20231209142856-c49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
	_ = os.WriteFile(filepath.Join(temp, tmName, "v0.0.1-20231208142856-d49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)

	id2 := "omnicorp-TM-department/omnicorp/omnilamp/v1.0.0-20231219123456-a49617d2e4fc.tm.json"
	err = r.Push(model.MustParseTMID(id2, false), []byte("{}"))
	assert.Equal(t, &ErrTMExists{ExistingId: "omnicorp-TM-department/omnicorp/omnilamp/v1.0.0-20231208142856-a49617d2e4fc.tm.json"}, err)

	id3 := "omnicorp-TM-department/omnicorp/omnilamp/v1.0.0-20231219123456-f49617d2e4fc.tm.json"
	err = r.Push(model.MustParseTMID(id3, false), []byte("{}"))
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(temp, id3))
}

func TestFileRemote_List(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRemote{
		root: temp,
		name: "fr",
	}
	copyFile("../../test/data/list/tm-catalog.toc.json", filepath.Join(temp, TOCFilename))
	list, err := r.List(&model.SearchParams{})
	assert.NoError(t, err)
	assert.Len(t, list.Entries, 4)
}

func copyFile(from, to string) {
	from, err := filepath.Abs(from)
	if err != nil {
		fmt.Println(err)
	}
	to, err = filepath.Abs(to)
	if err != nil {
		fmt.Println(err)
	}
	fromF, err := os.OpenFile(from, os.O_RDONLY, 0)
	if err != nil {
		fmt.Println(err)
	}
	toF, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE, 0700)
	if err != nil {
		fmt.Println(err)
	}
	_, err = io.Copy(toF, fromF)
	if err != nil {
		fmt.Println(err)
	}
}

func TestFileRemote_Versions(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRemote{
		root: temp,
		name: "fr",
	}
	copyFile("../../test/data/list/tm-catalog.toc.json", filepath.Join(temp, TOCFilename))
	vers, err := r.Versions("omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall/a/b")
	assert.NoError(t, err)
	assert.Len(t, vers.Versions, 1)

	vers, err = r.Versions("omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall/subpath")
	assert.NoError(t, err)
	assert.Len(t, vers.Versions, 1)

	vers, err = r.Versions("omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall")
	assert.NoError(t, err)
	assert.Len(t, vers.Versions, 1)

	vers, err = r.Versions("omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/nothing-here")
	assert.ErrorIs(t, err, ErrEntryNotFound)

	vers, err = r.Versions("systemx/siemens/AQualSenDev-virtual")
	assert.NoError(t, err)
	assert.Len(t, vers.Versions, 1)

	vers, err = r.Versions("")
	assert.ErrorContains(t, err, "specify a name")
}
