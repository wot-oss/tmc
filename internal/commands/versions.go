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
func (c *VersionsCommand) ListVersions(spec remotes.RepoSpec, name string) ([]model.FoundVersion, error) {
	rs, err := remotes.GetSpecdOrAll(c.remoteMgr, spec)
	if err != nil {
		return nil, err
	}
	var res []model.FoundVersion
	found := false
	for _, remote := range rs {
		vers, err := remote.Versions(name)
		if err != nil && errors.Is(err, remotes.ErrEntryNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		found = true
		res = model.MergeFoundVersions(res, vers)
	}
	if !found {
		return nil, remotes.ErrEntryNotFound
	}
	return res, nil

}
