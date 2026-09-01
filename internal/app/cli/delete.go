package cli

import (
	"context"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
)

func Delete(ctx context.Context, repo model.RepoSpec, id string) error {
	err := commands.Delete(ctx, repo, id)
	if err != nil {
		Stderrf("Could not delete from repo: %v", err)
		return err
	}
	return nil
}
