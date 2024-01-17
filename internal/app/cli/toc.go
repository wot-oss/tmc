package cli

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

func UpdateToc(rm remotes.RemoteManager, spec remotes.RepoSpec, ids []string) error {
	remote, err := rm.Get(spec)
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
