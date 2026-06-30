package commands

import (
	"context"
	"slices"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

type DeleteOptions struct {
	WithVariants bool
}

func Delete(ctx context.Context, rSpec model.RepoSpec, id string, opts DeleteOptions) error {
	r, err := repos.Get(rSpec)
	if err != nil {
		return err
	}

	if opts.WithVariants {
		variantIDs, err := findParentVariantIDs(ctx, r, id)
		if err != nil {
			return err
		}
		for _, variantID := range variantIDs {
			if variantID == id {
				continue
			}
			err = r.Delete(ctx, variantID)
			if err != nil {
				return err
			}
		}
	}

	err = r.Delete(ctx, id)
	return err
}

func findParentVariantIDs(ctx context.Context, r repos.Repo, parentTMID string) ([]string, error) {
	parsedTMID, err := model.ParseTMID(parentTMID)
	if err != nil {
		return nil, err
	}

	searchResult, err := r.List(ctx, &model.Filters{
		Name: parsedTMID.Name,
		Options: model.FilterOptions{
			NameFilterType: model.FullMatch,
		},
	})
	if err != nil {
		return nil, err
	}

	for _, entry := range searchResult.Entries {
		hasParentVersion := slices.ContainsFunc(entry.Versions, func(v model.FoundVersion) bool {
			return v.TMID == parentTMID
		})
		if !hasParentVersion {
			continue
		}
		variantIDs := make([]string, 0, len(entry.Variants))
		for _, variant := range entry.Variants {
			variantIDs = append(variantIDs, variant.VariantID)
		}
		return variantIDs, nil
	}

	return nil, nil
}
