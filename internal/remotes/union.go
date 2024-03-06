package remotes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

type Union struct {
	rs []Remote
}

type mapResult[T any] struct {
	res T
	err *RepoAccessError
}

type RepoAccessError struct {
	spec RepoSpec
	err  error
}

func NewRepoAccessError(spec RepoSpec, err error) *RepoAccessError {
	return &RepoAccessError{
		spec: spec,
		err:  err,
	}
}

func (e *RepoAccessError) Error() string {
	return fmt.Sprintf("%v returned: %v", e.spec, e.err)
}

func (e *RepoAccessError) Unwrap() error {
	return e.err
}
func (e *RepoAccessError) Spec() RepoSpec {
	return e.spec
}

func NewUnion(rs ...Remote) *Union {
	return &Union{
		rs: rs,
	}
}

func (u *Union) Fetch(id string) (string, []byte, error, []*RepoAccessError) {
	type fetchRes struct {
		id  string
		b   []byte
		err error
	}

	mapper := func(r Remote) mapResult[fetchRes] {
		fid, thing, err := r.Fetch(id)
		res := fetchRes{id: fid, b: thing, err: err}
		if err != nil && !errors.Is(err, ErrTmNotFound) {
			return mapResult[fetchRes]{res: res, err: NewRepoAccessError(r.Spec(), err)}
		}
		return mapResult[fetchRes]{res: res, err: nil}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	results := mapConcurrent(ctx, u.rs, mapper)
	res := fetchRes{err: ErrTmNotFound}
	res, errs := reduce(results, res, func(r1, r2 fetchRes) fetchRes {
		// the starting value cannot have err == nil, as cannot any intermediate accumulated result (r1)
		// hence, only the r2 could possibly be a successful fetch result
		if r2.err == nil {
			cancel()
		}
		return r2
	})
	if res.err != nil {
		msg := fmt.Sprintf("No thing model found for %v", id)
		slog.Default().Error(msg)
		return "", nil, ErrTmNotFound, errs
	}

	return res.id, res.b, nil, nil
}

func (u *Union) List(search *model.SearchParams) (model.SearchResult, []*RepoAccessError) {
	mapper := func(r Remote) mapResult[*model.SearchResult] {
		var raErr *RepoAccessError
		toc, err := r.List(search)
		if err != nil {
			raErr = NewRepoAccessError(r.Spec(), err)
		}
		return mapResult[*model.SearchResult]{
			res: &toc,
			err: raErr,
		}
	}

	reducer := func(t1, t2 *model.SearchResult) *model.SearchResult {
		t1.Merge(t2)
		return t1
	}

	results := mapConcurrent(context.Background(), u.rs, mapper)
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

// mapConcurrent concurrently maps all remotes with the mapper to a mapResult.
// Returns channel with results and a cancel channel, which can be closed to abort processing (e.g. if enough results have been received)
func mapConcurrent[T any](ctx context.Context, remotes []Remote, mapper func(r Remote) mapResult[T]) (results <-chan mapResult[T]) {
	res := make(chan mapResult[T])
	wg := sync.WaitGroup{}
	wg.Add(len(remotes))

	// start goroutines with cancellable mapping functions
	for _, remote := range remotes {
		go func(r Remote) {
			defer wg.Done()
			select {
			case <-ctx.Done():
			case res <- mapper(r):
			}
		}(remote)
	}

	// close results channel when all mapping goroutines have finished
	go func() {
		wg.Wait()
		close(res)
	}()

	return res
}

func (u *Union) Versions(name string) ([]model.FoundVersion, []*RepoAccessError) {
	mapper := func(r Remote) mapResult[[]model.FoundVersion] {
		var raErr *RepoAccessError
		vers, err := r.Versions(name)
		if err != nil {
			if !errors.Is(err, ErrTmNotFound) {
				raErr = NewRepoAccessError(r.Spec(), err)
			}
		}
		return mapResult[[]model.FoundVersion]{
			res: vers,
			err: raErr,
		}
	}
	var ident []model.FoundVersion
	results := mapConcurrent(context.Background(), u.rs, mapper)
	res, errs := reduce(results, ident, model.MergeFoundVersions)
	return res, errs
}

func (u *Union) ListCompletions(kind string, toComplete string) []string {
	mapper := func(r Remote) mapResult[[]string] {
		rcs, err := r.ListCompletions(kind, toComplete)
		if err != nil {
			rcs = nil
		}
		return mapResult[[]string]{res: rcs, err: nil}
	}
	reducer := func(r1, r2 []string) []string { return append(r1, r2...) }
	var cs []string
	results := mapConcurrent(context.Background(), u.rs, mapper)
	res, _ := reduce(results, cs, reducer)
	slices.Sort(res)
	return slices.Compact(res)
}
