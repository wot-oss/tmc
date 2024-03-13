package cli

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

func Delete(remote model.RepoSpec, id string) error {
	err := commands.NewDeleteCommand().Delete(remote, id)
	if err != nil {
		Stderrf("Could not delete from remote: %v", err)
		return err
	}
	return nil
}
