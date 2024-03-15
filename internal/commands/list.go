package commands

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/repos"
)

func List(rSpec model.RepoSpec, search *model.SearchParams) (model.SearchResult, error, []*repos.RepoAccessError) {
	rs, err := repos.GetSpecdOrAll(rSpec)
	if err != nil {
		return model.SearchResult{}, err, nil
	}
	sr, errs := rs.List(search)
	return sr, nil, errs
}
