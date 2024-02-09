package remotes

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

type Union struct {
	rs []Remote
}

type RepoAccessError interface {
	error
	Unwrap() error
	Spec() RepoSpec
}
type remoteAccessError struct {
	spec RepoSpec
	err  error
}

func (e *remoteAccessError) Error() string {
	return fmt.Sprintf("%s returned from %v", e.err.Error(), e.spec)
}

func (e *remoteAccessError) Unwrap() error {
	return e.err
}
func (e *remoteAccessError) Spec() RepoSpec {
	return e.spec
}

func NewUnion(rs ...Remote) *Union {
	// paranoia calling: flatten the list to disallow union of union until it's necessary
	var ers []Remote
	for _, r := range rs {
		ers = append(ers, r)
	}

	return &Union{
		rs: ers,
	}
}

func (u *Union) Fetch(id string) (string, []byte, error, []RepoAccessError) {
	var errs []RepoAccessError
	for _, r := range u.rs {
		id, thing, err := r.Fetch(id)
		if err == nil {
			return id, thing, nil, nil
		} else {
			if !errors.Is(err, ErrTmNotFound) {
				errs = append(errs, &remoteAccessError{
					spec: r.Spec(),
					err:  err,
				})
			}
		}
	}
	msg := fmt.Sprintf("No thing model found for %v", id)
	slog.Default().Error(msg)
	return "", nil, ErrTmNotFound, errs
}

func (u *Union) List(search *model.SearchParams) (model.SearchResult, []RepoAccessError) {
	var errs []RepoAccessError
	res := &model.SearchResult{}
	for _, remote := range u.rs {
		toc, err := remote.List(search)
		if err != nil {
			errs = append(errs, &remoteAccessError{
				spec: remote.Spec(),
				err:  err,
			})
			continue
		}
		res.Merge(&toc)
	}
	return *res, errs
}

func (u *Union) Versions(name string) ([]model.FoundVersion, []RepoAccessError) {
	var errs []RepoAccessError
	var res []model.FoundVersion
	for _, remote := range u.rs {
		vers, err := remote.Versions(name)
		if err != nil {
			if !errors.Is(err, ErrTmNotFound) {
				errs = append(errs, &remoteAccessError{
					spec: remote.Spec(),
					err:  err,
				})
			}
			continue
		}
		res = model.MergeFoundVersions(res, vers)
	}
	return res, errs
}

func (u *Union) ListCompletions(kind string, toComplete string) []string {
	var cs []string
	for _, r := range u.rs {
		rcs, err := r.ListCompletions(kind, toComplete)
		if err == nil {
			cs = append(cs, rcs...)
		}
	}
	slices.Sort(cs)
	return slices.Compact(cs)
}
