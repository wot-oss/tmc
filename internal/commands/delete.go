package commands

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/repos"
)

type DeleteCommand struct {
}

func NewDeleteCommand() *DeleteCommand {
	return &DeleteCommand{}
}
func (c *DeleteCommand) Delete(rSpec model.RepoSpec, id string) error {
	r, err := repos.Get(rSpec)
	if err != nil {
		return err
	}
	err = r.Delete(id)
	if err != nil {
		return err
	}
	err = r.Index(id)
	return err
}
