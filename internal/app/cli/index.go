package cli

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/repos"
)

func Index(spec model.RepoSpec, ids []string) error {
	repo, err := repos.Get(spec)
	if err != nil {
		Stderrf("could not initialize a repo instance for %v: %v. check config", spec, err)
		return err
	}

	err = repo.Index(ids...)

	if err != nil {
		Stderrf("could not create Index: %v", err)
		return err
	}
	return nil
}
