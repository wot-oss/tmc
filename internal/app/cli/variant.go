package cli

import (
	"context"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
)

func VariantAdd(ctx context.Context, spec model.RepoSpec, tmID string, opts commands.AddVariantOptions) error {
	err := commands.AddVariant(ctx, spec, tmID, opts)
	if err != nil {
		Stderrf("Could not add variant to %s: %v", tmID, err)
	}
	return err
}
