package commands

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

type VersionsCommand struct {
}

func NewVersionsCommand() *VersionsCommand {
	return &VersionsCommand{}
}
func (c *VersionsCommand) ListVersions(spec model.RepoSpec, name string) ([]model.FoundVersion, error, []*remotes.RepoAccessError) {
	rs, err := remotes.GetSpecdOrAll(spec)
	if err != nil {
		return nil, err, nil
	}
	versions, errors := rs.Versions(name)
	return versions, nil, errors
}
