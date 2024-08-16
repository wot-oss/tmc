package commands

import (
	"context"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func GetTMMetadata(ctx context.Context, spec model.RepoSpec, tmID string) ([]model.FoundVersion, error, []*repos.RepoAccessError) {
	rs, err := repos.GetUnion(spec)
	if err != nil {
		return nil, err, nil
	}

	sr, errs := rs.GetTMMetadata(ctx, tmID)
	return sr, nil, errs
}
