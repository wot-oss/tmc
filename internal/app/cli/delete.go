package cli

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

func Delete(repo model.RepoSpec, id string) error {
	err := commands.NewDeleteCommand().Delete(repo, id)
	if err != nil {
		Stderrf("Could not delete from repo: %v", err)
		return err
	}
	return nil
}
