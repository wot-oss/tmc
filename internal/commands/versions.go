package commands

import (
	"errors"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

func ListVersions(remoteName, name string) (model.FoundEntry, error) {
	var rs []remotes.Remote
	if remoteName != "" {
		// get versions from a single remote
		remote, err := remotes.Get(remoteName)
		if err != nil {
			return model.FoundEntry{}, err
		}
		rs = []remotes.Remote{remote}
	} else {
		// get versions from all remotes
		var err error
		rs, err = remotes.All()
		if err != nil {
			return model.FoundEntry{}, err
		}
	}
	res := model.FoundEntry{}
	found := false
	for _, remote := range rs {
		toc, err := remote.Versions(name)
		if err != nil && errors.Is(err, remotes.ErrEntryNotFound) {
			continue
		}
		if err != nil {
			return model.FoundEntry{}, err
		}
		found = true
		res = res.Merge(model.NewFoundEntryFromTOCEntry(&toc, remote.Name()))
	}
	if !found {
		return model.FoundEntry{}, remotes.ErrEntryNotFound
	}
	return res, nil

}
