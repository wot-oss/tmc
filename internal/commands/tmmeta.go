package commands

import (
	"context"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func GetTMMetadata(ctx context.Context, spec model.RepoSpec, tmID string) ([]model.FoundVersion, error, []*repos.RepoAccessError) {
	u, err := repos.GetUnion(spec)
	if err != nil {
		return nil, err, nil
	}

	sr, errs := u.GetTMMetadata(ctx, tmID)
	return sr, nil, errs
}
