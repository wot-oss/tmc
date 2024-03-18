package cli

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

func UpdateToc(spec model.RepoSpec, ids []string) error {
	remote, err := remotes.Get(spec)
	if err != nil {
		Stderrf("could not initialize a remote instance for %v: %v. check config", spec, err)
		return err
	}

	err = remote.UpdateToc(ids...)

	if err != nil {
		Stderrf("could not create TOC: %v", err)
		return err
	}
	return nil
}
