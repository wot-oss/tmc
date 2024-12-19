package cli

import (
	"context"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func CreateSearchIndex(ctx context.Context, spec model.RepoSpec) error {

	var rs []repos.Repo
	if spec.IsEmpty() {
		var err error
		rs, err = repos.All()
		if err != nil {
			Stderrf("couldn't get repositories to index: %v", err)
			return err
		}
	} else {
		r, err := repos.Get(spec)
		if err != nil {
			Stderrf("couldn't get repository to index: %v", err)
			return err
		}
		rs = append(rs, r)
	}

	for _, repo := range rs {
		err := repos.UpdateRepoIndex(ctx, repo)
		if err != nil {
			Stderrf("couldn't create search index: %v", err)
			return err
		}
	}
	return nil
}
