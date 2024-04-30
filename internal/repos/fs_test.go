package repos

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/wot-oss/tmc/internal/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/model"
	"golang.org/x/exp/rand"
)

func TestNewFileRepo(t *testing.T) {
	root := "/tmp/tm-catalog1157316148"
	repo, err := NewFileRepo(map[string]any{
		"type": "file",
		"loc":  root,
	}, model.EmptySpec)
	assert.NoError(t, err)
	assert.Equal(t, root, repo.root)

	root = "/tmp/tm-catalog1157316148"
	repo, err = NewFileRepo(map[string]any{
		"type": "file",
		"loc":  root,
	}, model.EmptySpec)
	assert.NoError(t, err)
	assert.Equal(t, root, repo.root)

	root = "~/tm-catalog"
	repo, err = NewFileRepo(map[string]any{
		"type": "file",
		"loc":  root,
	}, model.EmptySpec)
	assert.NoError(t, err)
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, "tm-catalog"), repo.root)

	root = "~/tm-catalog"
	repo, err = NewFileRepo(map[string]any{
		"type": "file",
		"loc":  root,
	}, model.EmptySpec)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "tm-catalog"), repo.root)

	root = "~/tm-catalog"
	repo, err = NewFileRepo(map[string]any{
		"type": "file",
		"loc":  root,
	}, model.EmptySpec)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "tm-catalog"), repo.root)

	root = "c:\\Users\\user\\Desktop\\tm-catalog"
	repo, err = NewFileRepo(map[string]any{
		"type": "file",
		"loc":  root,
	}, model.EmptySpec)
	assert.NoError(t, err)
	assert.Equal(t, filepath.ToSlash("c:\\Users\\user\\Desktop\\tm-catalog"), filepath.ToSlash(repo.root))

	root = "C:\\Users\\user\\Desktop\\tm-catalog"
	repo, err = NewFileRepo(map[string]any{
		"type": "file",
		"loc":  root,
	}, model.EmptySpec)
	assert.NoError(t, err)
	assert.Equal(t, filepath.ToSlash("C:\\Users\\user\\Desktop\\tm-catalog"), filepath.ToSlash(repo.root))

}

func TestCreateFileRepoConfig(t *testing.T) {
	wd, _ := os.Getwd()

	tests := []struct {
		strConf  string
		fileConf string
		expRoot  string
		expErr   bool
	}{
		{"../dir/repoName", "", filepath.Join(filepath.Dir(wd), "/dir/repoName"), false},
		{"./dir/repoName", "", filepath.Join(wd, "dir/repoName"), false},
		{"dir/repoName", "", filepath.Join(wd, "dir/repoName"), false},
		{"/dir/repoName", "", filepath.Join(filepath.VolumeName(wd), "/dir/repoName"), false},
		{".", "", filepath.Join(wd), false},
		{filepath.Join(wd, "dir/repoName"), "", filepath.Join(wd, "dir/repoName"), false},
		{"~/dir/repoName", "", "~/dir/repoName", false},
		{"", ``, "", true},
		{"", `[]`, "", true},
		{"", `{}`, "", true},
		{"", `{"loc":{}}`, "", true},
		{"", `{"loc":"dir/repoName"}`, filepath.Join(wd, "dir/repoName"), false},
		{"", `{"loc":"/dir/repoName"}`, filepath.Join(filepath.VolumeName(wd), "/dir/repoName"), false},
		{"", `{"loc":"dir/repoName", "type":"http"}`, "", true},
	}

	for i, test := range tests {
		cf, err := createFileRepoConfig(test.strConf, []byte(test.fileConf))
		if test.expErr {
			assert.Error(t, err, "error expected in test %d for %s %s", i, test.strConf, test.fileConf)
			continue
		} else {
			assert.NoError(t, err, "no error expected in test %d for %s %s", i, test.strConf, test.fileConf)
		}
		assert.Equalf(t, "file", cf[KeyRepoType], "in test %d for %s %s", i, test.strConf, test.fileConf)
		assert.Equalf(t, test.expRoot, cf[KeyRepoLoc], "in test %d for %s %s", i, test.strConf, test.fileConf)

	}
}

func TestValidatesRoot(t *testing.T) {
	repo, _ := NewFileRepo(map[string]any{
		"type": "file",
		"loc":  "/temp/surely-does-not-exist-5245874598745",
	}, model.EmptySpec)

	_, err := repo.List(context.Background(), nil)
	assert.ErrorIs(t, err, ErrRootInvalid)
	_, err = repo.Versions(context.Background(), "manufacturer/mpn")
	assert.ErrorIs(t, err, ErrRootInvalid)
	_, _, err = repo.Fetch(context.Background(), "manufacturer/mpn")
	assert.ErrorIs(t, err, ErrRootInvalid)

}

func TestFileRepo_Fetch(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRepo{
		root: temp,
		spec: model.NewRepoSpec("fr"),
	}
	tmName := "omnicorp-tm-department/omnicorp/omnilamp"
	fileA := []byte("{\"ver\":\"a\"}")
	fileB := []byte("{\"ver\":\"b\"}")
	fileC := []byte("{\"ver\":\"c\"}")
	fileD := []byte("{\"ver\":\"d\"}")
	idA := tmName + "/v1.0.0-20231208142856-a49617d2e4fc.tm.json"
	idB := tmName + "/v1.0.0-20231207142856-b49617d2e4fc.tm.json"
	idC := tmName + "/v1.2.1-20231209142856-c49617d2e4fc.tm.json"
	idD := tmName + "/v0.0.1-20231208142856-d49617d2e4fc.tm.json"
	fNameA := filepath.Join(temp, idA)
	fNameB := filepath.Join(temp, idB)
	fNameC := filepath.Join(temp, idC)
	fNameD := filepath.Join(temp, idD)
	_ = os.MkdirAll(filepath.Join(temp, tmName), defaultDirPermissions)
	_ = os.WriteFile(fNameA, fileA, defaultFilePermissions)
	_ = os.WriteFile(fNameB, fileB, defaultFilePermissions)
	_ = os.WriteFile(fNameC, fileC, defaultFilePermissions)
	_ = os.WriteFile(fNameD, fileD, defaultFilePermissions)

	actId, content, err := r.Fetch(context.Background(), idA)
	assert.NoError(t, err)
	assert.Equal(t, idA, actId)
	assert.Equal(t, fileA, content)

	actId, content, err = r.Fetch(context.Background(), idB)
	assert.NoError(t, err)
	assert.Equal(t, idB, actId)
	assert.Equal(t, fileB, content)

	actId, content, err = r.Fetch(context.Background(), tmName+"/v1.0.0-20231212142856-a49617d2e4fc.tm.json")
	assert.NoError(t, err)
	assert.Equal(t, idA, actId)
	assert.Equal(t, fileA, content)

	actId, content, err = r.Fetch(context.Background(), tmName+"/v1.0.0-20231212142856-e49617d2e4fc.tm.json")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Equal(t, "", actId)

}

func TestFileRepo_Push(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRepo{
		root: temp,
		spec: model.NewRepoSpec("fr"),
	}
	tmName := "omnicorp-tm-department/omnicorp/omnilamp"
	id := tmName + "/v0.0.0-20231208142856-c49617d2e4fc.tm.json"
	err := r.Push(context.Background(), model.MustParseTMID(id), []byte{})
	assert.Error(t, err)
	err = r.Push(context.Background(), model.MustParseTMID(id), []byte("{}"))
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(temp, id))

	_ = os.WriteFile(filepath.Join(temp, tmName, "v1.0.0-20231208142856-a49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
	_ = os.WriteFile(filepath.Join(temp, tmName, "v1.0.0-20231207142856-b49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
	_ = os.WriteFile(filepath.Join(temp, tmName, "v1.2.1-20231209142856-c49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
	_ = os.WriteFile(filepath.Join(temp, tmName, "v0.0.1-20231208142856-d49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)

	id2 := "omnicorp-tm-department/omnicorp/omnilamp/v1.0.0-20231219123456-a49617d2e4fc.tm.json"
	err = r.Push(context.Background(), model.MustParseTMID(id2), []byte("{}"))
	assert.Equal(t, &ErrTMIDConflict{Type: IdConflictSameContent, ExistingId: "omnicorp-tm-department/omnicorp/omnilamp/v1.0.0-20231208142856-a49617d2e4fc.tm.json"}, err)

	id3 := "omnicorp-tm-department/omnicorp/omnilamp/v1.0.0-20231219123456-f49617d2e4fc.tm.json"
	err = r.Push(context.Background(), model.MustParseTMID(id3), []byte("{}"))
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(temp, id3))

	id4 := "omnicorp-tm-department/omnicorp/omnilamp/v1.0.0-20231219123456-b49617d2e4fc.tm.json"
	err = r.Push(context.Background(), model.MustParseTMID(id4), []byte("{\"val\":1}"))
	assert.Equal(t, &ErrTMIDConflict{Type: IdConflictSameTimestamp, ExistingId: id3}, err)

}

func TestFileRepo_List(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRepo{
		root: temp,
		spec: model.NewRepoSpec("fr"),
	}
	testutils.CopyFile("../../test/data/list/tm-catalog.toc.json", r.indexFilename())
	list, err := r.List(context.Background(), &model.SearchParams{})
	assert.NoError(t, err)
	assert.Len(t, list.Entries, 3)
}

func TestFileRepo_Versions(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRepo{
		root: temp,
		spec: model.NewRepoSpec("fr"),
	}
	testutils.CopyFile("../../test/data/list/tm-catalog.toc.json", r.indexFilename())
	vers, err := r.Versions(context.Background(), "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/a/b")
	assert.NoError(t, err)
	assert.Len(t, vers, 1)

	vers, err = r.Versions(context.Background(), "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/subpath")
	assert.NoError(t, err)
	assert.Len(t, vers, 1)

	vers, err = r.Versions(context.Background(), "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall")
	assert.NoError(t, err)
	assert.Len(t, vers, 1)

	vers, err = r.Versions(context.Background(), "omnicorp-r-d-research/omnicorp-gmbh-co-kg/nothing-here")
	assert.ErrorIs(t, err, ErrNotFound)

	vers, err = r.Versions(context.Background(), "")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestFileRepo_Delete(t *testing.T) {
	tests := []struct {
		name  string
		root  string
		setup func(string) (string, string)
	}{
		{"current", ".", func(temp string) (catalog string, workdir string) {
			return temp, temp
		}},
		{"sibling", "../sibling", func(temp string) (catalog string, workdir string) {
			catalogDir := filepath.Join(temp, "sibling")
			_ = os.Mkdir(catalogDir, defaultDirPermissions)
			workDir := filepath.Join(temp, "workdir")
			_ = os.Mkdir(workDir, defaultDirPermissions)
			return catalogDir, workDir
		}},
		{"below", "./below", func(temp string) (catalog string, workdir string) {
			catalogDir := filepath.Join(temp, "below")
			_ = os.Mkdir(catalogDir, defaultDirPermissions)
			return catalogDir, temp
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			temp, _ := os.MkdirTemp("", "fr")
			defer os.RemoveAll(temp)

			wd, _ := os.Getwd()
			catalog, newWd := test.setup(temp)
			assert.NoError(t, testutils.CopyDir("../../test/data/index", catalog))
			_ = os.Chdir(newWd)
			defer os.Chdir(wd)

			spec := model.NewDirSpec(test.root)
			r := &FileRepo{
				root: test.root,
				spec: spec,
			}
			assert.NoError(t, r.Index(context.Background()))

			t.Run("invalid id", func(t *testing.T) {
				err := r.Delete(context.Background(), "invalid-id")
				assert.ErrorIs(t, err, model.ErrInvalidId)
			})
			t.Run("non-existent id", func(t *testing.T) {
				err := r.Delete(context.Background(), "auth/man/mpn/v1.0.1-20231024121314-abcd12345679.tm.json")
				assert.ErrorIs(t, err, ErrNotFound)
			})
			t.Run("hash matching id", func(t *testing.T) {
				err := r.Delete(context.Background(), "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20230101125023-be839ce9daf1.tm.json")
				assert.ErrorIs(t, err, ErrNotFound)
			})
			t.Run("existing id", func(t *testing.T) {
				id := "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20240409155220-80424c65e4e6.tm.json"
				err := r.Delete(context.Background(), id)
				assert.NoError(t, err)
				_, err = os.Stat(filepath.Join(r.root, id))
				assert.True(t, os.IsNotExist(err))
			})
			t.Run("cleans up empty folders", func(t *testing.T) {
				id1 := "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v0.0.0-20240409155220-80424c65e4e6.tm.json"
				id2 := "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v3.2.1-20240409155220-3f779458e453.tm.json"
				id3 := "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20240409155220-3f779458e453.tm.json"
				assert.NoError(t, r.Delete(context.Background(), id1))
				assert.NoError(t, r.Delete(context.Background(), id2))
				_, err := os.Stat(filepath.Join(r.root, "omnicorp-tm-department/omnicorp/omnilamp/subfolder"))
				assert.True(t, os.IsNotExist(err))
				_, err = os.Stat(filepath.Join(r.root, "omnicorp-tm-department/omnicorp/omnilamp"))
				assert.NoError(t, err)
				assert.NoError(t, r.Delete(context.Background(), id3))
				_, err = os.Stat(filepath.Join(r.root, "omnicorp-tm-department"))
				assert.True(t, os.IsNotExist(err))
				_, err = os.Stat(r.root)
				assert.NoError(t, err)
			})
		})
	}
}

func TestFileRepo_Index(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	assert.NoError(t, testutils.CopyDir("../../test/data/index", temp))
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	_ = os.Chdir(temp)
	spec := model.NewDirSpec(temp)
	r := &FileRepo{
		root: temp,
		spec: spec,
	}

	t.Run("single id/no index file", func(t *testing.T) {
		err := r.Index(context.Background(), "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v0.0.0-20240409155220-80424c65e4e6.tm.json")
		assert.NoError(t, err)

		idx, err := r.readIndex()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(idx.Data))
		assert.Equal(t, "omnicorp-tm-department/omnicorp/omnilamp/subfolder", idx.Data[0].Name)
		assert.Equal(t, 1, len(idx.Data[0].Versions))
		assert.Equal(t, "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v0.0.0-20240409155220-80424c65e4e6.tm.json", idx.Data[0].Versions[0].TMID)

		names := r.readNamesFile()
		assert.Equal(t, []string{"omnicorp-tm-department/omnicorp/omnilamp/subfolder"}, names)

	})
	t.Run("single id/existing index file", func(t *testing.T) {
		err := r.Index(context.Background(), "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v3.2.1-20240409155220-3f779458e453.tm.json")
		assert.NoError(t, err)

		idx, err := r.readIndex()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(idx.Data))
		assert.Equal(t, "omnicorp-tm-department/omnicorp/omnilamp/subfolder", idx.Data[0].Name)
		assert.Equal(t, 2, len(idx.Data[0].Versions))
		names := r.readNamesFile()
		assert.Equal(t, []string{"omnicorp-tm-department/omnicorp/omnilamp/subfolder"}, names)
	})

	t.Run("full update/existing index file", func(t *testing.T) {
		err := r.Index(context.Background())
		assert.NoError(t, err)

		idx, err := r.readIndex()
		assert.NoError(t, err)
		assert.Equal(t, 2, len(idx.Data))
		names := r.readNamesFile()
		assert.Equal(t, []string{
			"omnicorp-tm-department/omnicorp/omnilamp",
			"omnicorp-tm-department/omnicorp/omnilamp/subfolder",
		}, names)
	})

	t.Run("full update/no index file", func(t *testing.T) {
		err := os.Remove(r.indexFilename())
		assert.NoError(t, err)
		assert.NoError(t, r.writeNamesFile(nil))

		err = r.Index(context.Background())
		assert.NoError(t, err)

		idx, err := r.readIndex()
		assert.NoError(t, err)
		assert.Equal(t, 2, len(idx.Data))
		names := r.readNamesFile()
		assert.Equal(t, []string{
			"omnicorp-tm-department/omnicorp/omnilamp",
			"omnicorp-tm-department/omnicorp/omnilamp/subfolder",
		}, names)
	})
}

func TestFileRepo_UpdateIndex_RemoveId(t *testing.T) {
	tests := []struct {
		name  string
		root  string
		setup func(string) (string, string)
	}{
		{"current", ".", func(temp string) (catalog string, workdir string) {
			return temp, temp
		}},
		{"sibling", "../sibling", func(temp string) (catalog string, workdir string) {
			catalogDir := filepath.Join(temp, "sibling")
			_ = os.Mkdir(catalogDir, defaultDirPermissions)
			workDir := filepath.Join(temp, "workdir")
			_ = os.Mkdir(workDir, defaultDirPermissions)
			return catalogDir, workDir
		}},
		{"below", "./below", func(temp string) (catalog string, workdir string) {
			catalogDir := filepath.Join(temp, "below")
			_ = os.Mkdir(catalogDir, defaultDirPermissions)
			return catalogDir, temp
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			temp, _ := os.MkdirTemp("", "fr")
			defer os.RemoveAll(temp)

			wd, _ := os.Getwd()
			catalogDir, newWd := test.setup(temp)
			assert.NoError(t, testutils.CopyDir("../../test/data/index", catalogDir))
			_ = os.Chdir(newWd)
			defer os.Chdir(wd)
			assert.NoError(t, os.Remove(filepath.Join(catalogDir, "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v0.0.0-20240409155220-80424c65e4e6.tm.json")))

			spec := model.NewDirSpec(test.root)
			r := &FileRepo{
				root: test.root,
				spec: spec,
			}
			err := r.Index(context.Background())
			assert.NoError(t, err)

			t.Run("non-existing id", func(t *testing.T) {
				// when: deleting a non-existing id from index
				err := r.Index(context.Background(), "omnicorp-tm-department/omnicorp/omnilamp/v9.9.9-20240109125023-be839ce9daf1.tm.json")
				index, err := r.readIndex()
				assert.NoError(t, err)
				// then: nothing changes
				index.Filter(&model.SearchParams{Name: "omnicorp-tm-department/omnicorp/omnilamp"})
				if assert.Equal(t, 1, len(index.Data)) {
					assert.Equal(t, 2, len(index.Data[0].Versions))
				}
				names := r.readNamesFile()
				assert.Equal(t, []string{
					"omnicorp-tm-department/omnicorp/omnilamp",
					"omnicorp-tm-department/omnicorp/omnilamp/subfolder",
				}, names)
			})
			t.Run("existing id and file", func(t *testing.T) {
				// when: updating index for TM that has not been removed from disk
				err := r.Index(context.Background(), "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20240409155220-80424c65e4e6.tm.json")
				assert.NoError(t, err)
				index, err := r.readIndex()
				assert.NoError(t, err)
				// then: nothing changes
				index.Filter(&model.SearchParams{Name: "omnicorp-tm-department/omnicorp/omnilamp"})
				if assert.Equal(t, 1, len(index.Data)) {
					assert.Equal(t, 2, len(index.Data[0].Versions))
				}
				names := r.readNamesFile()
				assert.Equal(t, []string{
					"omnicorp-tm-department/omnicorp/omnilamp",
					"omnicorp-tm-department/omnicorp/omnilamp/subfolder",
				}, names)
			})
			t.Run("existing id of deleted file", func(t *testing.T) {
				// given: a deleted TM file
				_ = os.Remove(filepath.Join(r.root, "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20240409155220-80424c65e4e6.tm.json"))
				// when: updating index for the TM
				err := r.Index(context.Background(), "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20240409155220-80424c65e4e6.tm.json")
				assert.NoError(t, err)
				index, err := r.readIndex()
				assert.NoError(t, err)
				// then: version is removed from index
				index.Filter(&model.SearchParams{Name: "omnicorp-tm-department/omnicorp/omnilamp"})
				if assert.Equal(t, 1, len(index.Data)) {
					assert.Equal(t, 1, len(index.Data[0].Versions))
				}
				names := r.readNamesFile()
				assert.Equal(t, []string{
					"omnicorp-tm-department/omnicorp/omnilamp",
					"omnicorp-tm-department/omnicorp/omnilamp/subfolder",
				}, names)
			})
			t.Run("last version", func(t *testing.T) {
				// given: last version of a TM is removed
				id := "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v3.2.1-20240409155220-3f779458e453.tm.json"
				assert.NoError(t, os.Remove(filepath.Join(r.root, id)))
				// when: updating index for the id
				err = r.Index(context.Background(), id)
				assert.NoError(t, err)
				index, err := r.readIndex()
				assert.NoError(t, err)
				// then: name is removed from index
				index.Filter(&model.SearchParams{Name: "omnicorp-tm-department/omnicorp/omnilamp/subfolder"})
				assert.Equal(t, 0, len(index.Data))
				names := r.readNamesFile()
				// then: name is removed from names file
				assert.Equal(t, []string{
					"omnicorp-tm-department/omnicorp/omnilamp",
				}, names)
			})
		})
	}

}

func TestFileRepo_Index_Parallel(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	templ := template.Must(template.New("tm").Parse(pTempl))
	mockReader := func(name string) ([]byte, error) {
		ids, _ := strings.CutPrefix(name, temp+string(filepath.Separator))
		ids = filepath.ToSlash(ids)
		parts := strings.Split(ids, "/")
		assert.Len(t, parts, 4)
		auths := parts[0]
		manufs := parts[1]
		mpns := parts[2]
		res := bytes.NewBuffer(nil)
		err := templ.Execute(res, map[string]any{
			"manufacturer": manufs,
			"mpn":          mpns,
			"author":       auths,
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
	spec := model.NewRepoSpec("fr")
	r := &FileRepo{
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
			err := r.Index(context.Background(), fmt.Sprintf("author/manuf/mpn/v1.0.%d-%s-%x.tm.json", v, date, b))
			assert.NoError(t, err)
			wg.Done()
		}(i)
	}
	wg.Wait()
	idx, err := r.readIndex()
	assert.NoError(t, err)

	assert.Equal(t, 1, len(idx.Data))
	assert.Equal(t, N, len(idx.Data[0].Versions))
	names := r.readNamesFile()
	assert.Equal(t, 1, len(names))
}

func TestFileRepo_ListCompletions(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRepo{
		root: temp,
		spec: model.NewRepoSpec("fr"),
	}
	_ = os.MkdirAll(filepath.Join(temp, ".tmc"), defaultDirPermissions)

	t.Run("invalid", func(t *testing.T) {
		_, err := r.ListCompletions(context.Background(), "invalid", nil, "")
		assert.ErrorIs(t, err, ErrInvalidCompletionParams)
	})

	t.Run("no names file", func(t *testing.T) {
		names, err := r.ListCompletions(context.Background(), CompletionKindNames, nil, "")
		assert.NoError(t, err)
		var exp []string
		assert.Equal(t, exp, names)
	})

	t.Run("names", func(t *testing.T) {
		_ = os.WriteFile(filepath.Join(temp, ".tmc", TmNamesFile), []byte("omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall\n"+
			"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/a/b\n"+
			"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/subpath\n"), defaultFilePermissions)
		t.Run("empty", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, nil, "")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/"}, completions)
		})
		t.Run("some letters", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, nil, "om")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/"}, completions)
		})
		t.Run("some letters non existing", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, nil, "aaa")
			assert.NoError(t, err)
			var expRes []string
			assert.Equal(t, expRes, completions)
		})
		t.Run("full first name part", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, nil, "omnicorp-r-d-research/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/"}, completions)
		})
		t.Run("some letters second part", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, nil, "omnicorp-r-d-research/omnicorp")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/"}, completions)
		})
		t.Run("full second part", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, nil, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall", "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/"}, completions)
		})
		t.Run("full third part", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, nil, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/a/", "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/subpath"}, completions)
		})
		t.Run("full fourth part", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, nil, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/a/")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/a/b"}, completions)
		})
		t.Run("full name", func(t *testing.T) {
			completions, err := r.ListCompletions(context.Background(), CompletionKindNames, nil, "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/subpath")
			assert.NoError(t, err)
			assert.Equal(t, []string{"omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/subpath"}, completions)
		})
	})

	t.Run("fetchNames", func(t *testing.T) {
		tmName := "omnicorp-tm-department/omnicorp/omnilamp"
		_ = os.MkdirAll(filepath.Join(temp, tmName), defaultDirPermissions)
		_ = os.WriteFile(filepath.Join(temp, tmName, "v1.0.0-20231208142856-a49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
		_ = os.WriteFile(filepath.Join(temp, tmName, "v1.0.0-20231207142856-b49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
		_ = os.WriteFile(filepath.Join(temp, tmName, "v1.2.1-20231209142856-c49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
		_ = os.WriteFile(filepath.Join(temp, tmName, "v0.0.1-20231208142856-d49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
		fNames, err := r.ListCompletions(context.Background(), CompletionKindFetchNames, nil, tmName)
		assert.NoError(t, err)
		assert.Equal(t, []string{"omnicorp-tm-department/omnicorp/omnilamp:v0.0.1", "omnicorp-tm-department/omnicorp/omnilamp:v1.0.0", "omnicorp-tm-department/omnicorp/omnilamp:v1.2.1"}, fNames)
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
