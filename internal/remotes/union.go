package remotes

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

type UnionRemote struct {
	rs []Remote
}

func NewUnionRemote(rs ...Remote) *UnionRemote {
	rs = slices.DeleteFunc(rs, func(remote Remote) bool {
		_, ok := remote.(*UnionRemote)
		return ok // paranoia calling: disallow union of union until it's necessary
	})
	return &UnionRemote{
		rs: rs,
	}
}
func (u *UnionRemote) Push(model.TMID, []byte) error {
	return ErrNotSupported
}

func (u *UnionRemote) Fetch(id string) (string, []byte, error) {
	for _, r := range u.rs {
		id, thing, err := r.Fetch(id)
		if err == nil {
			return id, thing, nil
		}
	}

	msg := fmt.Sprintf("No thing model found for %v", id)
	slog.Default().Error(msg)
	return "", nil, ErrTmNotFound
}

func (u *UnionRemote) UpdateToc(...string) error {
	return ErrNotSupported
}

func (u *UnionRemote) List(search *model.SearchParams) (model.SearchResult, error) {
	res := &model.SearchResult{}
	for _, remote := range u.rs {
		toc, err := remote.List(search)
		if err != nil {
			return model.SearchResult{}, fmt.Errorf("could not list %s: %w", remote.Spec(), err)
		}
		res.Merge(&toc)
	}
	return *res, nil
}

func (u *UnionRemote) Versions(name string) ([]model.FoundVersion, error) {
	var res []model.FoundVersion
	found := false
	for _, remote := range u.rs {
		vers, err := remote.Versions(name)
		if err != nil && errors.Is(err, ErrTmNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		found = true
		res = model.MergeFoundVersions(res, vers)
	}
	if !found {
		return nil, ErrTmNotFound
	}
	return res, nil
}

func (u *UnionRemote) Spec() RepoSpec {
	return EmptySpec
}
