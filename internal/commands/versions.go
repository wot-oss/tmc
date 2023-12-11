package commands

import (
	"errors"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

type VersionsCommand struct {
	remoteMgr remotes.RemoteManager
}

func NewVersionsCommand(manager remotes.RemoteManager) *VersionsCommand {
	return &VersionsCommand{
		remoteMgr: manager,
	}
}
func (c *VersionsCommand) ListVersions(remoteName, name string) (model.FoundEntry, error) {
	rs, err := remotes.GetNamedOrAll(c.remoteMgr, remoteName)
	if err != nil {
		return model.FoundEntry{}, err
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
