package commands

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

type DeleteCommand struct {
	remoteMgr remotes.RemoteManager
}

func NewDeleteCommand(m remotes.RemoteManager) *DeleteCommand {
	return &DeleteCommand{
		remoteMgr: m,
	}
}
func (c *DeleteCommand) Delete(rSpec remotes.RepoSpec, id string) error {
	r, err := c.remoteMgr.Get(rSpec)
	if err != nil {
		return err
	}
	err = r.Delete(id)
	if err != nil {
		return err
	}
	err = r.UpdateToc(id)
	return err
}
