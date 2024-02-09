package commands

import (
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
func (c *ListCommand) List(rSpec remotes.RepoSpec, search *model.SearchParams) (model.SearchResult, error, []remotes.RepoAccessError) {
	rs, err := remotes.GetSpecdOrAll(c.remoteMgr, rSpec)
	if err != nil {
		return model.SearchResult{}, err, nil
	}
	sr, errs := rs.List(search)
	return sr, nil, errs
}
