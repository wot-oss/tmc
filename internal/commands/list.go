package commands

import (
	"fmt"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

type ListCommand struct {
	remoteMgr remotes.RemoteManager
}

func NewListCommand(m remotes.RemoteManager) *ListCommand {
	return &ListCommand{
		remoteMgr: m,
	}
}
func (c *ListCommand) List(rSpec remotes.RepoSpec, search *model.SearchParams) (model.SearchResult, error) {
	rs, err := remotes.GetSpecdOrAll(c.remoteMgr, rSpec)
	if err != nil {
		return model.SearchResult{}, err
	}

	res := &model.SearchResult{}
	for _, remote := range rs {
		toc, err := remote.List(search)
		if err != nil {
			return model.SearchResult{}, fmt.Errorf("could not list %s: %w", remote.Spec(), err)
		}
		res.Merge(&toc)
	}
	return *res, nil
}
