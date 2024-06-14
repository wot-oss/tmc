package commands

import (
	"context"
	"fmt"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func GetTMMetadata(ctx context.Context, spec model.RepoSpec, tmID string) (*model.FoundVersion, error) {
	repo, err := repos.Get(spec)
	if err != nil {
		return nil, fmt.Errorf("Could not ìnitialize a repo instance for %s: %w\ncheck config", spec, err)
	}

	sr, err := repo.GetTMMetadata(ctx, tmID)
	return sr, err
}
