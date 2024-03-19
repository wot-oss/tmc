package commands

import (
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
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
