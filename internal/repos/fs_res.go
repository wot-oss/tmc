package repos

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/wot-oss/tmc/internal/model"
)

func (f *FileRepo) RangeResources(ctx context.Context, filter model.ResourceFilter,
	visit func(model.Resource, error) bool) error {

	err := f.checkRootValid()
	if err != nil {
		return err
	}
	var log = slog.Default()

	var resources []model.Resource
	if len(filter.Names) == 0 {
		err := filepath.Walk(f.root, func(path string, info os.FileInfo, err error) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if !info.IsDir() {
				resType := resourceType(info.Name())

				if slices.Contains(filter.Types, resType) {
					id, _ := strings.CutPrefix(filepath.ToSlash(filepath.Clean(path)), filepath.ToSlash(filepath.Clean(f.root)))
					id, _ = strings.CutPrefix(id, "/")
					resources = append(resources, model.Resource{
						Name:    id,
						RelPath: id,
						Typ:     resType,
					})
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		for _, name := range filter.Names {
			resType := resourceType(name)

			if slices.Contains(filter.Types, resType) {
				resources = append(resources, model.Resource{
					Name:    name,
					RelPath: name,
					Typ:     resType,
				})
			}
		}
	}

	for _, res := range resources {
		log.Info("read resource", "name", res.Name)
		path := filepath.Join(f.root, res.Name)

		stat, data, err := readResource(path)

		if err != nil {
			log.Error(err.Error(), "resource", res.Name)
		}

		if errors.Is(err, os.ErrNotExist) {
			err = ErrResourceNotExists
		} else if stat != nil && stat.IsDir() {
			err = ErrResourceInvalid
		} else if err != nil {
			err = ErrResourceAccess
		}

		res.Raw = data

		ret := visit(res, err)
		if !ret {
			break
		}
	}
	return nil
}

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

func resourceType(resName string) model.ResourceType {
	if strings.HasSuffix(resName, TMExt) {
		return model.ResTypeTM
	}
	return model.ResTypeUnknown
}
