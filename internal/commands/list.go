package commands

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

func List(remoteName, filter string) (model.SearchResult, error) {
	var rs []remotes.Remote
	if remoteName != "" {
		// get list from a single remote
		remote, err := remotes.Get(remoteName)
		if err != nil {
			return model.SearchResult{}, err
		}
		rs = []remotes.Remote{remote}
	} else {
		// get list from all remotes
		var err error
		rs, err = remotes.All()
		if err != nil {
			return model.SearchResult{}, err
		}
	}
	res := model.SearchResult{}
	for _, remote := range rs {
		toc, err := remote.List(filter)
		if err != nil {
			return model.SearchResult{}, err
		}
		res = res.Merge(model.NewSearchResultFromTOC(toc, remote.Name()))
	}
	return res, nil
}
