package commands

import (
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func List(rSpec model.RepoSpec, search *model.SearchParams) (model.SearchResult, error, []*repos.RepoAccessError) {
	rs, err := repos.GetSpecdOrAll(rSpec)
	if err != nil {
		return model.SearchResult{}, err, nil
	}
	sr, errs := rs.List(search)
	return sr, nil, errs
}
