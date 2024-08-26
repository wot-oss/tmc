package commands

import (
	"context"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func List(ctx context.Context, rSpec model.RepoSpec, search *model.SearchParams) (model.SearchResult, error, []*repos.RepoAccessError) {
	u, err := repos.GetUnion(rSpec)
	if err != nil {
		return model.SearchResult{}, err, nil
	}
	sr, errs := u.List(ctx, search)
	return sr, nil, errs
}
