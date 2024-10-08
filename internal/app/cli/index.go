package cli

import (
	"context"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func Index(ctx context.Context, spec model.RepoSpec) error {
	repo, err := repos.Get(spec)
	if err != nil {
		Stderrf("could not initialize a repo instance for %v: %v. check config", spec, err)
		return err
	}

	err = repo.Index(ctx)

	if err != nil {
		Stderrf("could not create Index: %v", err)
		return err
	}
	return nil
}
