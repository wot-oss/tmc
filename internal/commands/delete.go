package commands

import (
	"context"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func Delete(ctx context.Context, rSpec model.RepoSpec, id string) error {
	r, err := repos.Get(rSpec)
	if err != nil {
		return err
	}
	err = r.Delete(ctx, id)
	if err != nil {
		return err
	}
	err = r.Index(ctx, id)
	return err
}
