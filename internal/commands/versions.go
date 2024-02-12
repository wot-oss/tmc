package commands

import (
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
func (c *VersionsCommand) ListVersions(spec remotes.RepoSpec, name string) ([]model.FoundVersion, error, []*remotes.RepoAccessError) {
	rs, err := remotes.GetSpecdOrAll(c.remoteMgr, spec)
	if err != nil {
		return nil, err, nil
	}
	versions, errors := rs.Versions(name)
	return versions, nil, errors
}
