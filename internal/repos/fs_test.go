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

	descr := "test description"
	tests := []struct {
		strConf  string
		fileConf string
		expRoot  string
		expErr   bool
		expDescr string
	}{
		{"../dir/repoName", "", filepath.Join(filepath.Dir(wd), "/dir/repoName"), false, descr},
		{"./dir/repoName", "", filepath.Join(wd, "dir/repoName"), false, descr},
		{"dir/repoName", "", filepath.Join(wd, "dir/repoName"), false, descr},
		{"/dir/repoName", "", filepath.Join(filepath.VolumeName(wd), "/dir/repoName"), false, descr},
		{".", "", filepath.Join(wd), false, descr},
		{filepath.Join(wd, "dir/repoName"), "", filepath.Join(wd, "dir/repoName"), false, descr},
		{"~/dir/repoName", "", "~/dir/repoName", false, descr},
		{"", ``, "", true, ""},
		{"", `[]`, "", true, ""},
		{"", `{}`, "", true, ""},
		{"", `{"loc":{}}`, "", true, ""},
		{"", `{"loc":"dir/repoName"}`, filepath.Join(wd, "dir/repoName"), false, ""},
		{"", `{"loc":"/dir/repoName", "description": "some description"}`, filepath.Join(filepath.VolumeName(wd), "/dir/repoName"), false, "some description"},
		{"", `{"loc":"dir/repoName", "type":"http"}`, "", true, ""},
	}

	for i, test := range tests {
		cf, err := createFileRepoConfig(test.strConf, []byte(test.fileConf), descr)
		if test.expErr {
			assert.Error(t, err, "error expected in test %d for %s %s", i, test.strConf, test.fileConf)
			continue
		} else {
			assert.NoError(t, err, "no error expected in test %d for %s %s", i, test.strConf, test.fileConf)
		}
		assert.Equalf(t, "file", cf[KeyRepoType], "in test %d for %s %s", i, test.strConf, test.fileConf)
		assert.Equalf(t, test.expRoot, cf[KeyRepoLoc], "in test %d for %s %s", i, test.strConf, test.fileConf)
		if test.expDescr != "" {
			assert.Equal(t, test.expDescr, cf[KeyRepoDescription], "in test %d for %s %s", i, test.strConf, test.fileConf)
		} else {
			assert.Nil(t, cf[KeyRepoDescription], "in test %d for %s %s", i, test.strConf, test.fileConf)
		}
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
	assert.ErrorIs(t, err, model.ErrTMNotFound)
	assert.Equal(t, "", actId)

}

func TestFileRepo_Import(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRepo{
		root: temp,
		spec: model.NewRepoSpec("fr"),
	}
	tmName := "omnicorp-tm-department/omnicorp/omnilamp"
	id := tmName + "/v0.0.0-20231208142856-c49617d2e4fc.tm.json"
	_, err := r.Import(context.Background(), model.MustParseTMID(id), []byte{}, ImportOptions{})
	assert.Error(t, err)
	_, err = r.Import(context.Background(), model.MustParseTMID(id), []byte("{}"), ImportOptions{})
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(temp, id))

	_ = os.WriteFile(filepath.Join(temp, tmName, "v1.0.0-20231208142856-a49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
	_ = os.WriteFile(filepath.Join(temp, tmName, "v1.0.0-20231207142856-b49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
	_ = os.WriteFile(filepath.Join(temp, tmName, "v1.2.1-20231209142856-c49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)
	_ = os.WriteFile(filepath.Join(temp, tmName, "v0.0.1-20231208142856-d49617d2e4fc.tm.json"), []byte("{}"), defaultFilePermissions)

	id2 := "omnicorp-tm-department/omnicorp/omnilamp/v1.0.0-20231219123456-a49617d2e4fc.tm.json"
	res, err := r.Import(context.Background(), model.MustParseTMID(id2), []byte("{}"), ImportOptions{})
	expCErr := &ErrTMIDConflict{Type: IdConflictSameContent, ExistingId: "omnicorp-tm-department/omnicorp/omnilamp/v1.0.0-20231208142856-a49617d2e4fc.tm.json"}
	assert.Equal(t, expCErr, err)
	assert.Equal(t, ImportResult{Type: ImportResultError, Message: expCErr.Error(), Err: expCErr}, res)

	id3 := "omnicorp-tm-department/omnicorp/omnilamp/v1.0.0-20231219123456-f49617d2e4fc.tm.json"
	_, err = r.Import(context.Background(), model.MustParseTMID(id3), []byte("{}"), ImportOptions{})
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(temp, id3))

	id4 := "omnicorp-tm-department/omnicorp/omnilamp/v1.0.0-20231219123456-049617d2e4fc.tm.json"
	res, err = r.Import(context.Background(), model.MustParseTMID(id4), []byte("{\"val\":1}"), ImportOptions{})
	assert.NoError(t, err)
	expCErr = &ErrTMIDConflict{Type: IdConflictSameTimestamp, ExistingId: id3}
	assert.Equal(t, ImportResult{Type: ImportResultWarning, TmID: id4, Message: expCErr.Error(), Err: expCErr}, res)

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
	assert.ErrorIs(t, err, model.ErrTMNameNotFound)

	vers, err = r.Versions(context.Background(), "")
	assert.ErrorIs(t, err, model.ErrTMNameNotFound)
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
				assert.ErrorIs(t, err, model.ErrTMNotFound)
			})
			t.Run("hash matching id", func(t *testing.T) {
				err := r.Delete(context.Background(), "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20230101125023-be839ce9daf1.tm.json")
				assert.ErrorIs(t, err, model.ErrTMNotFound)
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
				id4 := "omnicorp-tm-department/omnicorp/omnilamp/v3.11.1-20240409155220-da7dbd7ed830.tm.json"
				assert.NoError(t, r.Delete(context.Background(), id1))
				assert.NoError(t, r.Delete(context.Background(), id2))
				_, err := os.Stat(filepath.Join(r.root, "omnicorp-tm-department/omnicorp/omnilamp/subfolder"))
				assert.True(t, os.IsNotExist(err))
				_, err = os.Stat(filepath.Join(r.root, "omnicorp-tm-department/omnicorp/omnilamp"))
				assert.NoError(t, err)
				assert.NoError(t, r.Delete(context.Background(), id3))
				assert.NoError(t, r.Delete(context.Background(), id4))
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

		entry := idx.FindByName("omnicorp-tm-department/omnicorp/omnilamp")
		assert.NotNil(t, entry)
		if assert.Len(t, entry.Versions, 3) {
			assert.Equal(t, []model.Attachment{{Name: "manual.txt", MediaType: "text/plain; charset=utf-8"}}, entry.Versions[2].Attachments)
		}
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

	t.Run("single tm id indexes tm name attachments", func(t *testing.T) {
		tmName := "omnicorp-tm-department/omnicorp/omnilamp/subfolder"
		attDir := filepath.Join(temp, tmName, model.AttachmentsDir)
		assert.NoError(t, os.MkdirAll(attDir, defaultDirPermissions))
		assert.NoError(t, os.WriteFile(filepath.Join(attDir, "README.txt"), []byte("Read This, or Else"), defaultFilePermissions))

		err := r.Index(context.Background(), "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v3.2.1-20240409155220-3f779458e453.tm.json")
		assert.NoError(t, err)

		idx, err := r.readIndex()
		assert.NoError(t, err)
		entry := idx.FindByName(tmName)
		assert.NotNil(t, entry)
		assert.Equal(t, []model.Attachment{{Name: "README.txt", MediaType: "text/plain; charset=utf-8"}}, entry.Attachments)
	})

	t.Run("single id's/index must be sorted", func(t *testing.T) {
		err := os.Remove(r.indexFilename())
		assert.NoError(t, err)
		assert.NoError(t, r.writeNamesFile(nil))

		tmName1 := "omnicorp-tm-department/omnicorp/omnilamp"
		tmId11 := "omnicorp-tm-department/omnicorp/omnilamp/v0.0.0-20240409155220-80424c65e4e6.tm.json"
		tmId12 := "omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20240409155220-3f779458e453.tm.json"
		tmId13 := "omnicorp-tm-department/omnicorp/omnilamp/v3.11.1-20240409155220-da7dbd7ed830.tm.json"

		tmName2 := "omnicorp-tm-department/omnicorp/omnilamp/subfolder"
		tmId21 := "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v0.0.0-20240409155220-80424c65e4e6.tm.json"
		tmId22 := "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v3.2.1-20240409155220-3f779458e453.tm.json"

		// update index with unordered ID's
		err = r.Index(context.Background(), tmId21, tmId12, tmId22, tmId13, tmId11)
		assert.NoError(t, err)

		idx, err := r.readIndex()
		assert.NoError(t, err)
		assert.Equal(t, 2, len(idx.Data))

		assert.Equal(t, tmName1, idx.Data[0].Name)
		assert.Equal(t, tmId13, idx.Data[0].Versions[0].TMID)
		assert.Equal(t, tmId12, idx.Data[0].Versions[1].TMID)
		assert.Equal(t, tmId11, idx.Data[0].Versions[2].TMID)
		assert.Equal(t, tmName2, idx.Data[1].Name)
		assert.Equal(t, tmId22, idx.Data[1].Versions[0].TMID)
		assert.Equal(t, tmId21, idx.Data[1].Versions[1].TMID)
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
				err := r.Index(context.Background())
				index, err := r.readIndex()
				assert.NoError(t, err)
				// then: nothing changes
				index.Filter(&model.SearchParams{Name: "omnicorp-tm-department/omnicorp/omnilamp"})
				if assert.Equal(t, 1, len(index.Data)) {
					assert.Equal(t, 3, len(index.Data[0].Versions))
				}
				names := r.readNamesFile()
				assert.Equal(t, []string{
					"omnicorp-tm-department/omnicorp/omnilamp",
					"omnicorp-tm-department/omnicorp/omnilamp/subfolder",
				}, names)
			})
			t.Run("existing id and file", func(t *testing.T) {
				// when: updating index for TM that has not been removed from disk
				err := r.Index(context.Background())
				assert.NoError(t, err)
				index, err := r.readIndex()
				assert.NoError(t, err)
				// then: nothing changes
				index.Filter(&model.SearchParams{Name: "omnicorp-tm-department/omnicorp/omnilamp"})
				if assert.Equal(t, 1, len(index.Data)) {
					assert.Equal(t, 3, len(index.Data[0].Versions))
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
				err := r.Index(context.Background())
				assert.NoError(t, err)
				index, err := r.readIndex()
				assert.NoError(t, err)
				// then: version is removed from index
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
			t.Run("last version", func(t *testing.T) {
				// given: last version of a TM is removed
				id := "omnicorp-tm-department/omnicorp/omnilamp/subfolder/v3.2.1-20240409155220-3f779458e453.tm.json"
				assert.NoError(t, os.Remove(filepath.Join(r.root, id)))
				// when: updating index for the id
				err = r.Index(context.Background())
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

func TestFileRepo_CheckIntegrity(t *testing.T) {

	t.Run("with only valid ThingModels", func(t *testing.T) {
		temp, _ := os.MkdirTemp("", "fr")
		defer os.RemoveAll(temp)
		assert.NoError(t, testutils.CopyDir("../../test/data/index", temp))
		spec := model.NewDirSpec(temp)

		// given: a clean repository with index
		r := &FileRepo{
			root: temp,
			spec: spec,
		}
		_ = r.Index(context.Background())
		// when checking the integrity
		res, err := r.CheckIntegrity(context.Background(), nil)

		// then: there is no total error
		assert.NoError(t, err)
		// and then: result list only contains OK results
		for _, result := range res {
			assert.Equal(t, model.CheckOK, result.Typ)
		}
	})

	t.Run("with some invalid ThingModels", func(t *testing.T) {
		temp, _ := os.MkdirTemp("", "fr")
		defer os.RemoveAll(temp)
		assert.NoError(t, testutils.CopyDir("../../test/data/integrity/faulty", temp))
		spec := model.NewDirSpec(temp)

		// given: a repository with unknown files
		r := &FileRepo{
			root: temp,
			spec: spec,
		}

		// when: checking the given ThingModels
		res, err := r.CheckIntegrity(context.Background(), nil)
		// then: there is no total error
		assert.NoError(t, err)
		// then: results include errors for invalid files
		assert.Contains(t, res, model.CheckResult{Typ: model.CheckErr, ResourceName: "mistake.md", Message: "file unknown"})
		assert.Contains(t, res, model.CheckResult{Typ: model.CheckErr, ResourceName: "omnicorp/omnicorp/lightall/.attachments/mistake.md", Message: "appears to be an attachment file which is not known to the repository. Make sure you import it using TMC CLI"})
		assert.Contains(t, res, model.CheckResult{Typ: model.CheckErr, ResourceName: "omnicorp/omnicorp/lightall/.attachments/v1.0.1-20240807094932-5a3840060b05/mistake.md", Message: "appears to be an attachment file which is not known to the repository. Make sure you import it using TMC CLI"})
		assert.Contains(t, res, model.CheckResult{Typ: model.CheckErr, ResourceName: "omnicorp/omnicorp/lightall/v1.0.2-20240819094932-5a3840060b05.tm.json", Message: "appears to be a TM file which is not known to the repository. Make sure you import it using TMC CLI"})
		assert.Contains(t, res, model.CheckResult{Typ: model.CheckErr, ResourceName: "vomnicorp/vomnicorp/mistake.md", Message: "file unknown"})
	})
}

func TestFileRepo_GetTMMetadata(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRepo{
		root: temp,
		spec: model.NewRepoSpec("fr"),
	}
	testutils.CopyFile("../../test/data/list/tm-catalog.toc.json", r.indexFilename())
	tmID := "omnicorp-r-d-research/omnicorp-gmbh-co-kg/senseall/v1.0.1-20231208142830-c49617d2e4fc.tm.json"
	testutils.CopyFile("../../test/data/repos/file/attachments/omnicorp-tm-department/omnicorp/omnilamp/v3.2.1-20240409155220-3f779458e453.tm.json", filepath.Join(temp, tmID))
	meta, err := r.GetTMMetadata(context.Background(), tmID)
	assert.NoError(t, err)
	assert.Equal(t, []model.Attachment{{Name: "firmware update notes.md"}}, meta[0].Attachments)
}

func TestFileRepo_FetchAttachment(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRepo{
		root: temp,
		spec: model.NewRepoSpec("fr"),
	}
	assert.NoError(t, testutils.CopyDir("../../test/data/repos/file/attachments", temp))
	tmName := "omnicorp-tm-department/omnicorp/omnilamp"
	fileA, _ := os.ReadFile("../../test/data/repos/file/attachments/omnicorp-tm-department/omnicorp/omnilamp/.attachments/README.md")
	fileB, _ := os.ReadFile("../../test/data/repos/file/attachments/omnicorp-tm-department/omnicorp/omnilamp/.attachments/v3.2.1-20240409155220-3f779458e453/cfg.json")
	idA := tmName + "/v3.2.1-20240409155220-3f779458e453.tm.json"
	baseNameA := "README.md"
	baseNameB := "cfg.json"

	t.Run("tm name attachment", func(t *testing.T) {
		content, err := r.FetchAttachment(context.Background(), model.NewTMNameAttachmentContainerRef(tmName), baseNameA)
		assert.NoError(t, err)
		assert.Equal(t, fileA, content)
	})
	t.Run("tm id attachment", func(t *testing.T) {
		content, err := r.FetchAttachment(context.Background(), model.NewTMIDAttachmentContainerRef(idA), baseNameB)
		assert.NoError(t, err)
		assert.Equal(t, fileB, content)
	})
	t.Run("non existent attachment", func(t *testing.T) {
		_, err := r.FetchAttachment(context.Background(), model.NewTMNameAttachmentContainerRef(tmName), "nothing-here")
		assert.ErrorIs(t, err, model.ErrAttachmentNotFound)
	})
	t.Run("non existent tm name", func(t *testing.T) {
		_, err := r.FetchAttachment(context.Background(), model.NewTMNameAttachmentContainerRef("omnicorp-tm-department/omnicorp/omnidarkness"), baseNameA)
		assert.ErrorIs(t, err, model.ErrTMNameNotFound)
	})
	t.Run("non existent tm id", func(t *testing.T) {
		_, err := r.FetchAttachment(context.Background(), model.NewTMIDAttachmentContainerRef(tmName+"/v1.2.3-20240409155220-3f779458e453.tm.json"), baseNameA)
		assert.ErrorIs(t, err, model.ErrTMNotFound)
	})
	t.Run("invalid tm name", func(t *testing.T) {
		_, err := r.FetchAttachment(context.Background(), model.NewTMNameAttachmentContainerRef("omnicorp-tm-departmentomnicorp/omnilamp"), baseNameA)
		assert.ErrorIs(t, err, model.ErrInvalidIdOrName)
	})
	t.Run("invalid tm id", func(t *testing.T) {
		_, err := r.FetchAttachment(context.Background(), model.NewTMIDAttachmentContainerRef(tmName+"/v1.2.3-20240409155220-3f779458e453"), baseNameA)
		assert.ErrorIs(t, err, model.ErrInvalidId)
	})
}

func TestFileRepo_ImportAttachment(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRepo{
		root: temp,
		spec: model.NewRepoSpec("fr"),
	}
	assert.NoError(t, testutils.CopyDir("../../test/data/repos/file/attachments", temp))
	tmName := "omnicorp-tm-department/omnicorp/omnilamp"
	ver := "v3.2.1-20240409155220-3f779458e453"
	id := tmName + "/" + ver + TMExt
	r2Name := "README2.md"
	r2Content := []byte("# read this, too")
	t.Run("tm name attachment without media type provided", func(t *testing.T) {
		ref := model.NewTMNameAttachmentContainerRef(tmName)
		err := r.ImportAttachment(context.Background(), ref, model.Attachment{Name: r2Name}, r2Content, false)
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(temp, tmName, model.AttachmentsDir, r2Name))
		index, err := r.readIndex()
		if assert.NoError(t, err) {
			c, _, _ := index.FindAttachmentContainer(ref)
			att, found := c.FindAttachment(r2Name)
			assert.True(t, found)
			assert.NotEmpty(t, att.MediaType)
		}
	})
	t.Run("tm name attachment with media type", func(t *testing.T) {
		ref := model.NewTMNameAttachmentContainerRef(tmName)
		err := r.ImportAttachment(context.Background(), ref, model.Attachment{Name: r2Name, MediaType: "text/html"}, r2Content, true)
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(temp, tmName, model.AttachmentsDir, r2Name))
		index, err := r.readIndex()
		if assert.NoError(t, err) {
			c, _, _ := index.FindAttachmentContainer(ref)
			att, found := c.FindAttachment(r2Name)
			assert.True(t, found)
			assert.Equal(t, "text/html", att.MediaType)
		}
	})
	t.Run("tm id attachment with media type provided by user", func(t *testing.T) {
		ref := model.NewTMIDAttachmentContainerRef(id)
		err := r.ImportAttachment(context.Background(), ref, model.Attachment{Name: r2Name, MediaType: "text/markdown"}, r2Content, false)
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(temp, tmName, model.AttachmentsDir, ver, r2Name))
		index, err := r.readIndex()
		if assert.NoError(t, err) {
			c, _, _ := index.FindAttachmentContainer(ref)
			att, found := c.FindAttachment(r2Name)
			assert.True(t, found)
			assert.Equal(t, "text/markdown", att.MediaType)
		}
	})
	t.Run("tm id attachment without media type", func(t *testing.T) {
		ref := model.NewTMIDAttachmentContainerRef(id)
		err := r.ImportAttachment(context.Background(), ref, model.Attachment{Name: r2Name, MediaType: "text/markdown"}, r2Content, true)
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(temp, tmName, model.AttachmentsDir, ver, r2Name))
		index, err := r.readIndex()
		if assert.NoError(t, err) {
			c, _, _ := index.FindAttachmentContainer(ref)
			att, found := c.FindAttachment(r2Name)
			assert.True(t, found)
			assert.Equal(t, "text/markdown", att.MediaType)
		}
	})
	t.Run("tm id attachment conflict", func(t *testing.T) {
		err := r.ImportAttachment(context.Background(), model.NewTMIDAttachmentContainerRef(id), model.Attachment{Name: r2Name}, r2Content, false)
		assert.ErrorIs(t, err, ErrAttachmentExists)
	})
	t.Run("non existent tm name", func(t *testing.T) {
		err := r.ImportAttachment(context.Background(), model.NewTMNameAttachmentContainerRef("omnicorp-tm-department/omnicorp/omnidarkness"), model.Attachment{Name: r2Name}, r2Content, false)
		assert.ErrorIs(t, err, model.ErrTMNameNotFound)
	})
	t.Run("non existent tm id", func(t *testing.T) {
		err := r.ImportAttachment(context.Background(), model.NewTMIDAttachmentContainerRef(tmName+"/v1.2.3-20240409155220-3f779458e453.tm.json"), model.Attachment{Name: r2Name}, r2Content, false)
		assert.ErrorIs(t, err, model.ErrTMNotFound)
	})
	t.Run("invalid tm name", func(t *testing.T) {
		err := r.ImportAttachment(context.Background(), model.NewTMNameAttachmentContainerRef("omnicorp-tm-departmentomnicorp/omnilamp"), model.Attachment{Name: r2Name}, r2Content, false)
		assert.ErrorIs(t, err, model.ErrInvalidIdOrName)
	})
	t.Run("invalid tm id", func(t *testing.T) {
		err := r.ImportAttachment(context.Background(), model.NewTMIDAttachmentContainerRef(tmName+"/v1.2.3-20240409155220-3f779458e453"), model.Attachment{Name: r2Name}, r2Content, false)
		assert.ErrorIs(t, err, model.ErrInvalidId)
	})
}

func TestFileRepo_DeleteAttachment(t *testing.T) {
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRepo{
		root: temp,
		spec: model.NewRepoSpec("fr"),
	}
	assert.NoError(t, testutils.CopyDir("../../test/data/repos/file/attachments", temp))
	tmName := "omnicorp-tm-department/omnicorp/omnilamp"
	ver := "v3.2.1-20240409155220-3f779458e453"
	idA := tmName + "/" + ver + TMExt
	attNameA := "README.md"
	attNameB := "cfg.json"

	t.Run("non existent attachment", func(t *testing.T) {
		err := r.DeleteAttachment(context.Background(), model.NewTMNameAttachmentContainerRef(tmName), "nothing-here")
		assert.ErrorIs(t, err, model.ErrAttachmentNotFound)
	})
	t.Run("non existent tm name", func(t *testing.T) {
		err := r.DeleteAttachment(context.Background(), model.NewTMNameAttachmentContainerRef("omnicorp-tm-department/omnicorp/omnidarkness"), attNameA)
		assert.ErrorIs(t, err, model.ErrTMNameNotFound)
	})
	t.Run("non existent tm id", func(t *testing.T) {
		err := r.DeleteAttachment(context.Background(), model.NewTMIDAttachmentContainerRef(tmName+"/v1.2.3-20240409155220-3f779458e453.tm.json"), attNameA)
		assert.ErrorIs(t, err, model.ErrTMNotFound)
	})
	t.Run("invalid tm name", func(t *testing.T) {
		err := r.DeleteAttachment(context.Background(), model.NewTMNameAttachmentContainerRef("omnicorp-tm-departmentomnicorp/omnilamp"), attNameA)
		assert.ErrorIs(t, err, model.ErrInvalidIdOrName)
	})
	t.Run("invalid tm id", func(t *testing.T) {
		err := r.DeleteAttachment(context.Background(), model.NewTMIDAttachmentContainerRef(tmName+"/v1.2.3-20240409155220-3f779458e453"), attNameA)
		assert.ErrorIs(t, err, model.ErrInvalidId)
	})
	t.Run("tm id attachment", func(t *testing.T) {
		err := r.DeleteAttachment(context.Background(), model.NewTMIDAttachmentContainerRef(idA), attNameB)
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(temp, tmName, model.AttachmentsDir, ver, attNameA))
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(filepath.Join(temp, tmName, model.AttachmentsDir, ver))
		assert.True(t, os.IsNotExist(err))
	})
	t.Run("tm name attachment", func(t *testing.T) {
		err := r.DeleteAttachment(context.Background(), model.NewTMNameAttachmentContainerRef(tmName), attNameA)
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(temp, tmName, model.AttachmentsDir, attNameA))
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(filepath.Join(temp, tmName, model.AttachmentsDir))
		assert.True(t, os.IsNotExist(err))
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
