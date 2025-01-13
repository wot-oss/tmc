package repos

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/config"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/testutils"
	"github.com/wot-oss/tmc/internal/utils"
)

func TestUnion_Search(t *testing.T) {
	t.Run("no bleve index", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "tmc-export")
		defer os.RemoveAll(tempDir)
		old := config.ConfigDir
		config.ConfigDir = filepath.Join(tempDir, "config")
		defer func() { config.ConfigDir = old }()
		repoRoot := filepath.Join(tempDir, "repo")
		r := &FileRepo{
			root: repoRoot,
			spec: model.NewRepoSpec("repo"),
		}
		u := NewUnion(r)
		err := testutils.CopyDir("../../test/data/repos/file/attachments", repoRoot)
		assert.NoError(t, err)

		_, errs := u.Search(context.Background(), "query")
		if assert.Len(t, errs, 1) {
			assert.ErrorIs(t, errs[0], model.ErrSearchIndexNotFound)
		}
	})
	t.Run("existing bleve index", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "tmc-export")
		defer os.RemoveAll(tempDir)
		old := config.ConfigDir
		config.ConfigDir = filepath.Join(tempDir, "config")
		defer func() { config.ConfigDir = old }()
		repoRoot := filepath.Join(tempDir, "repo")
		r := &FileRepo{
			root: repoRoot,
			spec: model.NewRepoSpec("repo"),
		}
		u := NewUnion(r)

		err := testutils.CopyDir("../../test/data/repos/file/attachments", repoRoot)
		assert.NoError(t, err)

		err = UpdateRepoIndex(context.Background(), r)
		assert.NoError(t, err)

		t.Run("with no match", func(t *testing.T) {
			res, errs := u.Search(context.Background(), "query")
			assert.Len(t, errs, 0)
			assert.Len(t, res.Entries, 0)
		})
		t.Run("with match", func(t *testing.T) {
			res, errs := u.Search(context.Background(), "\"Lamp reaches a critical temperature\"")
			assert.Len(t, errs, 0)
			assert.Len(t, res.Entries, 1)
		})

	})
	t.Run("outdated bleve index", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "tmc-export")
		defer os.RemoveAll(tempDir)
		old := config.ConfigDir
		config.ConfigDir = filepath.Join(tempDir, "config")
		defer func() { config.ConfigDir = old }()
		repoRoot := filepath.Join(tempDir, "repo")
		r := &FileRepo{
			root: repoRoot,
			spec: model.NewRepoSpec("repo"),
		}
		u := NewUnion(r)

		err := testutils.CopyDir("../../test/data/repos/file/attachments", repoRoot)
		assert.NoError(t, err)

		indexPath := BleveIndexPath(r)
		_ = os.MkdirAll(indexPath, defaultDirPermissions)
		_ = utils.WriteFileLines(
			[]string{time.Date(2024, 1, 1, 1, 1, 1, 0, time.UTC).Format(time.RFC3339Nano)},
			filepath.Join(indexPath, "updated"),
			defaultFilePermissions)

		res, errs := u.Search(context.Background(), "\"Lamp reaches a critical temperature\"")
		assert.Len(t, errs, 0)
		assert.Len(t, res.Entries, 1)

	})
}
