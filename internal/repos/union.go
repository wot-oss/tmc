package repos

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/wot-oss/tmc/internal/model"
)

type Union struct {
	rs []Repo
}

type mapResult[T any] struct {
	res T
	err *RepoAccessError
}

type RepoAccessError struct {
	spec model.RepoSpec
	err  error
}

func NewRepoAccessError(spec model.RepoSpec, err error) *RepoAccessError {
	if err == nil {
		return nil
	}
	return &RepoAccessError{spec: spec, err: err}
}
func newRepoAccessError(repo Repo, err error) *RepoAccessError {
	// the only reason for this check (and the whole function) is to spare setting up all the reposmocks throughout tests with .On("Spec",...)
	if err == nil {
		return nil
	}
	return NewRepoAccessError(repo.Spec(), err)
}

func (e *RepoAccessError) Error() string {
	return fmt.Sprintf("%v returned: %v", e.spec, e.err)
}

func (e *RepoAccessError) Unwrap() error {
	return e.err
}

func NewUnion(rs ...Repo) *Union {
	return &Union{
		rs: rs,
	}
}

func (u *Union) Fetch(ctx context.Context, id string) (string, []byte, error, []*RepoAccessError) {
	type fetchRes struct {
		id  string
		b   []byte
		err error
	}

	mapper := func(r Repo) mapResult[fetchRes] {
		fid, thing, err := r.Fetch(ctx, id)
		res := fetchRes{id: fid, b: thing, err: err}
		if errors.Is(err, ErrNotFound) {
			return mapResult[fetchRes]{res: res, err: nil}
		}
		return mapResult[fetchRes]{res: res, err: newRepoAccessError(r, err)}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := mapConcurrent(ctx, u.rs, mapper)
	res := fetchRes{err: ErrNotFound}
	res, errs := reduce(results, res, func(r1, r2 fetchRes) fetchRes {
		if r1.err == nil {
			cancel()
			return r1
		}
		if r2.err == nil {
			cancel()
		}
		return r2
	})
	if res.err != nil {
		return "", nil, ErrNotFound, errs
	}

	return res.id, res.b, nil, nil
}

func (u *Union) List(ctx context.Context, search *model.SearchParams) (model.SearchResult, []*RepoAccessError) {
	mapper := func(r Repo) mapResult[*model.SearchResult] {
		idx, err := r.List(ctx, search)
		return mapResult[*model.SearchResult]{res: &idx, err: newRepoAccessError(r, err)}
	}

	reducer := func(t1, t2 *model.SearchResult) *model.SearchResult {
		t1.Merge(t2)
		return t1
	}

	results := mapConcurrent(ctx, u.rs, mapper)
	r, errs := reduce(results, &model.SearchResult{}, reducer)
	return *r, errs
}

// reduce reads results from ch until ch is closed and reduces them to a single result with identity as the starting value
func reduce[T any](ch <-chan mapResult[T], identity T, reducer func(t1, t2 T) T) (T, []*RepoAccessError) {
	accumulator := identity
	var errs []*RepoAccessError
	for res := range ch {
		accumulator = reducer(accumulator, res.res)
		if res.err != nil {
			errs = append(errs, res.err)
		}
	}
	return accumulator, errs
}

// mapConcurrent concurrently maps all repo with the mapper to a mapResult.
// Returns channel with results
func mapConcurrent[T any](ctx context.Context, repos []Repo, mapper func(r Repo) mapResult[T]) (results <-chan mapResult[T]) {
	res := make(chan mapResult[T])
	wg := sync.WaitGroup{}
	wg.Add(len(repos))

	// start goroutines with cancellable mapping functions
	for _, repo := range repos {
		go func(r Repo) {
			defer wg.Done()
			select {
			case <-ctx.Done():
			case res <- mapper(r):
			}
		}(repo)
	}

	// close results channel when all mapping goroutines have finished
	go func() {
		wg.Wait()
		close(res)
	}()

	return res
}

func (u *Union) Versions(ctx context.Context, name string) ([]model.FoundVersion, []*RepoAccessError) {
	mapper := func(r Repo) mapResult[[]model.FoundVersion] {
		vers, err := r.Versions(ctx, name)
		if errors.Is(err, ErrNotFound) {
			return mapResult[[]model.FoundVersion]{res: vers, err: nil}
		}
		return mapResult[[]model.FoundVersion]{res: vers, err: newRepoAccessError(r, err)}
	}
	var ident []model.FoundVersion
	results := mapConcurrent(ctx, u.rs, mapper)
	res, errs := reduce(results, ident, model.MergeFoundVersions)
	return res, errs
}

func (u *Union) ListCompletions(ctx context.Context, kind string, args []string, toComplete string) []string {
	mapper := func(r Repo) mapResult[[]string] {
		rcs, err := r.ListCompletions(ctx, kind, args, toComplete)
		if err != nil {
			rcs = nil
		}
		return mapResult[[]string]{res: rcs, err: nil}
	}
	reducer := func(r1, r2 []string) []string { return append(r1, r2...) }
	var cs []string
	results := mapConcurrent(ctx, u.rs, mapper)
	res, _ := reduce(results, cs, reducer)
	slices.Sort(res)
	return slices.Compact(res)
}
