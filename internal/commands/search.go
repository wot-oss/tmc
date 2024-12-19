package commands

import (
	"context"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func Search(ctx context.Context, rSpec model.RepoSpec, query string) (model.SearchResult, error, []*repos.RepoAccessError) {
	u, err := repos.GetUnion(rSpec)
	if err != nil {
		return model.SearchResult{}, err, nil
	}
	sr, errs := u.Search(ctx, query)
	return sr, nil, errs
}
