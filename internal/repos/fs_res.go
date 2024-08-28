package repos

import (
	"context"
	"os"
	"path/filepath"

	"github.com/wot-oss/tmc/internal/model"
)

func readResource(path string) (stat os.FileInfo, data []byte, err error) {
	stat, statErr := osStat(path)
	if statErr != nil {
		return stat, nil, statErr
	}
	if stat != nil && !stat.IsDir() {
		data, err = osReadFile(path)
	}
	return stat, data, err
}

func (f *FileRepo) verifyAllFilesAreIndexed(ctx context.Context, idx *model.Index) ([]model.CheckResult, error) {

	var results []model.CheckResult

	err := filepath.Walk(f.root, func(path string, info os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(f.root, path)
		if err != nil {
			return err
		}
		checkResult := f.verifyFileIsIndexed(rel, idx)
		results = append(results, checkResult)

		return nil
	})

	return results, err

}
