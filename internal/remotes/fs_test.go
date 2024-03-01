package remotes

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"golang.org/x/exp/rand"
)

func TestNewFileRemote(t *testing.T) {
	root := "/tmp/tm-catalog1157316148"
	remote, err := NewFileRemote(map[string]any{
		"type": "file",
		"loc":  root,
	}, EmptySpec)
	assert.NoError(t, err)
	assert.Equal(t, root, remote.root)

	root = "/tmp/tm-catalog1157316148"
	remote, err = NewFileRemote(map[string]any{
		"type": "file",
		"loc":  root,
	}, EmptySpec)
	assert.NoError(t, err)
	assert.Equal(t, root, remote.root)

	root = "~/tm-catalog"
	remote, err = NewFileRemote(map[string]any{
		"type": "file",
		"loc":  root,
	}, EmptySpec)
	assert.NoError(t, err)
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, "tm-catalog"), remote.root)

	root = "~/tm-catalog"
	remote, err = NewFileRemote(map[string]any{
		"type": "file",
		"loc":  root,
	}, EmptySpec)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "tm-catalog"), remote.root)

	root = "~/tm-catalog"
	remote, err = NewFileRemote(map[string]any{
		"type": "file",
		"loc":  root,
	}, EmptySpec)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "tm-catalog"), remote.root)

	root = "c:\\Users\\user\\Desktop\\tm-catalog"
	remote, err = NewFileRemote(map[string]any{
		"type": "file",
		"loc":  root,
	}, EmptySpec)
	assert.NoError(t, err)
	assert.Equal(t, filepath.ToSlash("c:\\Users\\user\\Desktop\\tm-catalog"), filepath.ToSlash(remote.root))

	root = "C:\\Users\\user\\Desktop\\tm-catalog"
	remote, err = NewFileRemote(map[string]any{
		"type": "file",
		"loc":  root,
	}, EmptySpec)
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
		{"../dir/remoteName", "", filepath.Join(filepath.Dir(wd), "/dir/remoteName"), false},
		{"./dir/remoteName", "", filepath.Join(wd, "dir/remoteName"), false},
		{"dir/remoteName", "", filepath.Join(wd, "dir/remoteName"), false},
		{"/dir/remoteName", "", filepath.Join(filepath.VolumeName(wd), "/dir/remoteName"), false},
		{".", "", filepath.Join(wd), false},
		{filepath.Join(wd, "dir/remoteName"), "", filepath.Join(wd, "dir/remoteName"), false},
		{"~/dir/remoteName", "", "~/dir/remoteName", false},
		{"", ``, "", true},
		{"", `[]`, "", true},
		{"", `{}`, "", true},
		{"", `{"loc":{}}`, "", true},
		{"", `{"loc":"dir/remoteName"}`, filepath.Join(wd, "dir/remoteName"), false},
		{"", `{"loc":"/dir/remoteName"}`, filepath.Join(filepath.VolumeName(wd), "/dir/remoteName"), false},
		{"", `{"loc":"dir/remoteName", "type":"http"}`, "", true},
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
	}, EmptySpec)

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
		spec: NewRemoteSpec("fr"),
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
	assert.ErrorIs(t, err, ErrTmNotFound)
	assert.Equal(t, "", actId)

}

func TestFileRemote_Push(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRemote{
		root: temp,
		spec: NewRemoteSpec("fr"),
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
	assert.Equal(t, &ErrTMIDConflict{Type: IdConflictSameContent, ExistingId: "omnicorp-TM-department/omnicorp/omnilamp/v1.0.0-20231208142856-a49617d2e4fc.tm.json"}, err)

	id3 := "omnicorp-TM-department/omnicorp/omnilamp/v1.0.0-20231219123456-f49617d2e4fc.tm.json"
	err = r.Push(model.MustParseTMID(id3, false), []byte("{}"))
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(temp, id3))

	id4 := "omnicorp-TM-department/omnicorp/omnilamp/v1.0.0-20231219123456-b49617d2e4fc.tm.json"
	err = r.Push(model.MustParseTMID(id4, false), []byte("{\"val\":1}"))
	assert.Equal(t, &ErrTMIDConflict{Type: IdConflictSameTimestamp, ExistingId: id3}, err)

}

func TestFileRemote_List(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRemote{
		root: temp,
		spec: NewRemoteSpec("fr"),
	}
	copyFile("../../test/data/list/tm-catalog.toc.json", r.tocFilename())
	list, err := r.List(&model.SearchParams{})
	assert.NoError(t, err)
	assert.Len(t, list.Entries, 3)
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
	err = os.MkdirAll(filepath.Dir(to), defaultDirPermissions)
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
		spec: NewRemoteSpec("fr"),
	}
	copyFile("../../test/data/list/tm-catalog.toc.json", r.tocFilename())
	vers, err := r.Versions("omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall/a/b")
	assert.NoError(t, err)
	assert.Len(t, vers, 1)

	vers, err = r.Versions("omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall/subpath")
	assert.NoError(t, err)
	assert.Len(t, vers, 1)

	vers, err = r.Versions("omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/senseall")
	assert.NoError(t, err)
	assert.Len(t, vers, 1)

	vers, err = r.Versions("omnicorp-R-D-research/omnicorp-Gmbh-Co-KG/nothing-here")
	assert.ErrorIs(t, err, ErrTmNotFound)

	vers, err = r.Versions("")
	assert.ErrorIs(t, err, ErrTmNotFound)
}

func TestFileRemote_Delete(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	spec := NewRemoteSpec("fr")
	r := &FileRemote{
		root: temp,
		spec: spec,
	}
	err := copy.Copy("../../test/data/toc", temp)
	assert.NoError(t, err)

	t.Run("invalid id", func(t *testing.T) {
		err := r.Delete("invalid-id")
		assert.ErrorIs(t, err, model.ErrInvalidId)
	})
	t.Run("non-existent id", func(t *testing.T) {
		err := r.Delete("man/mpn/v1.0.1-20231024121314-abcd12345679.tm.json")
		assert.ErrorIs(t, err, ErrTmNotFound)
	})
	t.Run("hash matching id", func(t *testing.T) {
		err := r.Delete("omnicorp-TM-department/omnicorp/omnilamp/v0.0.0-20230101125023-be839ce9daf1.tm.json")
		assert.ErrorIs(t, err, ErrTmNotFound)
	})
	t.Run("existing id", func(t *testing.T) {
		id := "omnicorp-TM-department/omnicorp/omnilamp/v0.0.0-20240109125023-be839ce9daf1.tm.json"
		err := r.Delete(id)
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(r.root, id))
		assert.True(t, os.IsNotExist(err))
	})
	t.Run("cleans up empty folders", func(t *testing.T) {
		id1 := "omnicorp-TM-department/omnicorp/omnilamp/subfolder/v0.0.0-20240109125023-be839ce9daf1.tm.json"
		id2 := "omnicorp-TM-department/omnicorp/omnilamp/subfolder/v3.2.1-20240109125023-1e788769a659.tm.json"
		id3 := "omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20240109125023-1e788769a659.tm.json"
		assert.NoError(t, r.Delete(id1))
		assert.NoError(t, r.Delete(id2))
		_, err = os.Stat(filepath.Join(r.root, "omnicorp-TM-department/omnicorp/omnilamp/subfolder"))
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(filepath.Join(r.root, "omnicorp-TM-department/omnicorp/omnilamp"))
		assert.NoError(t, err)
		assert.NoError(t, r.Delete(id3))
		_, err = os.Stat(filepath.Join(r.root, "omnicorp-TM-department"))
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(r.root)
		assert.NoError(t, err)
	})
}

func TestFileRemote_UpdateTOC(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	spec := NewRemoteSpec("fr")
	r := &FileRemote{
		root: temp,
		spec: spec,
	}
	err := copy.Copy("../../test/data/toc", temp)
	assert.NoError(t, err)

	t.Run("single id/no toc file", func(t *testing.T) {
		err = r.UpdateToc("omnicorp-TM-department/omnicorp/omnilamp/subfolder/v0.0.0-20240109125023-be839ce9daf1.tm.json")
		assert.NoError(t, err)

		toc, err := r.readTOC()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(toc.Data))
		assert.Equal(t, "omnicorp-TM-department/omnicorp/omnilamp/subfolder", toc.Data[0].Name)
		assert.Equal(t, 1, len(toc.Data[0].Versions))
		assert.Equal(t, "omnicorp-TM-department/omnicorp/omnilamp/subfolder/v0.0.0-20240109125023-be839ce9daf1.tm.json", toc.Data[0].Versions[0].TMID)

		names := r.readNamesFile()
		assert.Equal(t, []string{"omnicorp-TM-department/omnicorp/omnilamp/subfolder"}, names)

	})
	t.Run("single id/existing toc file", func(t *testing.T) {
		err = r.UpdateToc("omnicorp-TM-department/omnicorp/omnilamp/subfolder/v3.2.1-20240109125023-1e788769a659.tm.json")
		assert.NoError(t, err)

		toc, err := r.readTOC()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(toc.Data))
		assert.Equal(t, "omnicorp-TM-department/omnicorp/omnilamp/subfolder", toc.Data[0].Name)
		assert.Equal(t, 2, len(toc.Data[0].Versions))
		names := r.readNamesFile()
		assert.Equal(t, []string{"omnicorp-TM-department/omnicorp/omnilamp/subfolder"}, names)
	})

	t.Run("full update/existing toc file", func(t *testing.T) {
		err = r.UpdateToc()
		assert.NoError(t, err)

		toc, err := r.readTOC()
		assert.NoError(t, err)
		assert.Equal(t, 2, len(toc.Data))
		names := r.readNamesFile()
		assert.Equal(t, []string{
			"omnicorp-TM-department/omnicorp/omnilamp",
			"omnicorp-TM-department/omnicorp/omnilamp/subfolder",
		}, names)
	})

	t.Run("full update/no toc file", func(t *testing.T) {
		err := os.Remove(r.tocFilename())
		assert.NoError(t, err)
		assert.NoError(t, r.writeNamesFile(nil))

		err = r.UpdateToc()
		assert.NoError(t, err)

		toc, err := r.readTOC()
		assert.NoError(t, err)
		assert.Equal(t, 2, len(toc.Data))
		names := r.readNamesFile()
		assert.Equal(t, []string{
			"omnicorp-TM-department/omnicorp/omnilamp",
			"omnicorp-TM-department/omnicorp/omnilamp/subfolder",
		}, names)
	})

	t.Run("remove id from toc", func(t *testing.T) {
		err := os.Remove(filepath.Join(r.root, "omnicorp-TM-department/omnicorp/omnilamp/v0.0.0-20240109125023-be839ce9daf1.tm.json"))
		assert.NoError(t, err)

		t.Run("non-existing id", func(t *testing.T) {
			err := r.UpdateToc("omnicorp-TM-department/omnicorp/omnilamp/v1.0.0-20240109125023-be839ce9daf1.tm.json")
			toc, err := r.readTOC()
			assert.NoError(t, err)
			toc.Filter(&model.SearchParams{Name: "omnicorp-TM-department/omnicorp/omnilamp"})
			if assert.Equal(t, 1, len(toc.Data)) {
				assert.Equal(t, 2, len(toc.Data[0].Versions))
			}
			names := r.readNamesFile()
			assert.Equal(t, []string{
				"omnicorp-TM-department/omnicorp/omnilamp",
				"omnicorp-TM-department/omnicorp/omnilamp/subfolder",
			}, names)
		})
		t.Run("existing id", func(t *testing.T) {
			err := r.UpdateToc("omnicorp-TM-department/omnicorp/omnilamp/v0.0.0-20240109125023-be839ce9daf1.tm.json")
			toc, err := r.readTOC()
			assert.NoError(t, err)
			toc.Filter(&model.SearchParams{Name: "omnicorp-TM-department/omnicorp/omnilamp"})
			if assert.Equal(t, 1, len(toc.Data)) {
				assert.Equal(t, 1, len(toc.Data[0].Versions))
			}
			names := r.readNamesFile()
			assert.Equal(t, []string{
				"omnicorp-TM-department/omnicorp/omnilamp",
				"omnicorp-TM-department/omnicorp/omnilamp/subfolder",
			}, names)
		})
		t.Run("last version", func(t *testing.T) {
			err := os.Remove(filepath.Join(r.root, "omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20240109125023-1e788769a659.tm.json"))
			assert.NoError(t, err)
			err = r.UpdateToc("omnicorp-TM-department/omnicorp/omnilamp/v3.2.1-20240109125023-1e788769a659.tm.json")
			toc, err := r.readTOC()
			assert.NoError(t, err)
			toc.Filter(&model.SearchParams{Name: "omnicorp-TM-department/omnicorp/omnilamp"})
			assert.Equal(t, 0, len(toc.Data))
			names := r.readNamesFile()
			assert.Equal(t, []string{
				"omnicorp-TM-department/omnicorp/omnilamp/subfolder",
			}, names)
		})
	})
}

func TestFileRemote_UpdateTOC_Parallel(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	templ := template.Must(template.New("tm").Parse(pTempl))
	mockReader := func(name string) ([]byte, error) {
		mpns, ver := filepath.Split(name)
		manufs, mpn := filepath.Split(filepath.Clean(mpns))
		auths, manuf := filepath.Split(filepath.Clean(manufs))
		auth := filepath.Base(filepath.Clean(auths))
		ids := fmt.Sprintf("%s/%s/%s/%s", auth, manuf, mpn, ver)
		id := model.MustParseTMID(ids, false)
		res := bytes.NewBuffer(nil)
		err := templ.Execute(res, map[string]any{
			"manufacturer": id.Manufacturer,
			"mpn":          id.Mpn,
			"author":       id.Author,
			"id":           ids,
		})
		assert.NoError(t, err)
		return res.Bytes(), nil
	}
	osReadFile = mockReader
	defer func() { osReadFile = os.ReadFile }()
	osStat = func(name string) (os.FileInfo, error) {
		return fakeFileInfo{name: name}, nil
	}
	defer func() { osStat = os.Stat }()
	spec := NewRemoteSpec("fr")
	r := &FileRemote{
		root: temp,
		spec: spec,
	}

	N := 50
	firstDate, _ := time.Parse(model.PseudoVersionTimestampFormat, "20231208142830")

	wg := sync.WaitGroup{}
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(v int) {
			date := firstDate.Add(time.Duration(v) * 1 * time.Second).Format(model.PseudoVersionTimestampFormat)
			b := make([]byte, 6)
			_, _ = rand.Read(b)
			err := r.UpdateToc(fmt.Sprintf("author/manuf/mpn/v1.0.%d-%s-%x.tm.json", v, date, b))
			assert.NoError(t, err)
			wg.Done()
		}(i)
	}
	wg.Wait()
	toc, err := r.readTOC()
	assert.NoError(t, err)

	assert.Equal(t, 1, len(toc.Data))
	assert.Equal(t, N, len(toc.Data[0].Versions))
	names := r.readNamesFile()
	assert.Equal(t, 1, len(names))
}

func TestFileRemote_ListCompletions(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRemote{
		root: temp,
		spec: NewRemoteSpec("fr"),
	}
	_ = os.MkdirAll(filepath.Join(temp, ".tmc"), defaultDirPermissions)

	t.Run("invalid", func(t *testing.T) {
		_, err := r.ListCompletions("invalid", "")
		assert.ErrorIs(t, err, ErrInvalidCompletionParams)
	})

	t.Run("no names file", func(t *testing.T) {
		names, err := r.ListCompletions(CompletionKindNames, "")
		assert.NoError(t, err)
		var exp []string
		assert.Equal(t, exp, names)
	})

	t.Run("names", func(t *testing.T) {
		_ = os.WriteFile(filepath.Join(temp, ".tmc", TmNamesFile), []byte("a/b/c\nd/e/f\n"), defaultFilePermissions)
		names, err := r.ListCompletions(CompletionKindNames, "")
		assert.NoError(t, err)
		assert.Equal(t, []string{"a/b/c", "d/e/f"}, names)
	})

	t.Run("fetchNames", func(t *testing.T) {
		tmName := "omnicorp-TM-department/omnicorp/omnilamp"
		_ = os.MkdirAll(filepath.Join(temp, tmName), defaultDirPermissions)
		_ = os.WriteFile(filepath.Join(temp, tmName, "v1.0.0-20231208142856-a49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
		_ = os.WriteFile(filepath.Join(temp, tmName, "v1.0.0-20231207142856-b49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
		_ = os.WriteFile(filepath.Join(temp, tmName, "v1.2.1-20231209142856-c49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
		_ = os.WriteFile(filepath.Join(temp, tmName, "v0.0.1-20231208142856-d49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
		fNames, err := r.ListCompletions(CompletionKindFetchNames, tmName)
		assert.NoError(t, err)
		assert.Equal(t, []string{"omnicorp-TM-department/omnicorp/omnilamp:v0.0.1", "omnicorp-TM-department/omnicorp/omnilamp:v1.0.0", "omnicorp-TM-department/omnicorp/omnilamp:v1.2.1"}, fNames)
	})
}

var pTempl = `{
  "@context": [
    "https://www.w3.org/2022/wot/td/v1.1",
    {
		"schema":"https://schema.org/"
	}
  ],
  "@type": "tm:ThingModel",
  "title": "Lamp Thing Model",
  "schema:manufacturer": {
    "schema:name": "{{.manufacturer}}"
  },
  "schema:mpn": "{{.mpn}}",
  "schema:author": {
    "schema:name": "{{.author}}"
  },
  "properties": {
    "status": {
      "description": "current status of the lamp (on|off)",
      "type": "string",
      "readOnly": true
    }
  },
  "actions": {
    "toggle": {
      "description": "Turn the lamp on or off"
    }
  },
  "events": {
    "overheating": {
      "description": "Lamp reaches a critical temperature (overheating)",
      "data": {
        "type": "string"
      }
    }
  },
  "version": {
    "model": "v1.0.{{.ver}}"
  }
,"id":"{{.id}}"}`

type fakeFileInfo struct {
	name string
}

func (f fakeFileInfo) Name() string {
	return f.name
}

func (f fakeFileInfo) Size() int64 {
	panic("implement me")
}

func (f fakeFileInfo) Mode() fs.FileMode {
	panic("implement me")
}

func (f fakeFileInfo) ModTime() time.Time {
	panic("implement me")
}

func (f fakeFileInfo) IsDir() bool {
	return false
}

func (f fakeFileInfo) Sys() any {
	return nil
}
