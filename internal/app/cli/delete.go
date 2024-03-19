package cli

import (
	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
)

func Delete(repo model.RepoSpec, id string) error {
	err := commands.NewDeleteCommand().Delete(repo, id)
	if err != nil {
		Stderrf("Could not delete from repo: %v", err)
		return err
	}
	return nil
}
