package commands

import (
	"context"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

type VersionsCommand struct {
}

func NewVersionsCommand() *VersionsCommand {
	return &VersionsCommand{}
}
func (c *VersionsCommand) ListVersions(ctx context.Context, spec model.RepoSpec, name string) ([]model.FoundVersion, error, []*repos.RepoAccessError) {
	rs, err := repos.GetUnion(spec)
	if err != nil {
		return nil, err, nil
	}
	versions, errors := rs.Versions(ctx, name)
	return versions, nil, errors
}
