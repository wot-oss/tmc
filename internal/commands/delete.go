package commands

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

type DeleteCommand struct {
}

func NewDeleteCommand() *DeleteCommand {
	return &DeleteCommand{}
}
func (c *DeleteCommand) Delete(rSpec model.RepoSpec, id string) error {
	r, err := remotes.Get(rSpec)
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
