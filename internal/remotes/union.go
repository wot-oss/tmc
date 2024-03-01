package remotes

import (
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

type joinedResult[T any] struct {
	res  T
	errs []*RepoAccessError
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
	// paranoia calling: flatten the list to disallow union of union until it's necessary
	var ers []Remote
	for _, r := range rs {
		ers = append(ers, r)
	}

	return &Union{
		rs: ers,
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
	res, errs := mapFirst[fetchRes](u.rs, mapper, func(r fetchRes) bool { return r.err == nil }, fetchRes{err: ErrTmNotFound})

	if res.err != nil {
		msg := fmt.Sprintf("No thing model found for %v", id)
		slog.Default().Error(msg)
		return "", nil, ErrTmNotFound, errs
	}

	return res.id, res.b, nil, errs
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

	res, errs := mapReduce[*model.SearchResult](u.rs, mapper, &model.SearchResult{}, reducer)
	return *res, errs
}

// mapReduce performs a concurrent map of remotes to mapResult[T], then reduces the results to a single joinedResult[T]
func mapReduce[T any](remotes []Remote, mapper func(r Remote) mapResult[T], identity T, reducer func(t1, t2 T) T) (T, []*RepoAccessError) {
	results, _ := mapConcurrent(remotes, mapper)
	r := reduce(results, identity, reducer)
	return r.res, r.errs
}

// reduce reads results from ch until ch is closed and reduces them to a single joinedResult with identity as the starting value
func reduce[T any](ch <-chan mapResult[T], identity T, reducer func(t1, t2 T) T) joinedResult[T] {
	accumulator := identity
	var errs []*RepoAccessError
	for res := range ch {
		accumulator = reducer(accumulator, res.res)
		if res.err != nil {
			errs = append(errs, res.err)
		}
	}
	return joinedResult[T]{
		res:  accumulator,
		errs: errs,
	}
}

// mapConcurrent concurrently maps all remotes with the mapper to a mapResult.
// Returns channels with results and a done channel, which can be close to abort processing (e.g. if enough results have been received)
func mapConcurrent[T any](remotes []Remote, mapper func(r Remote) mapResult[T]) (results <-chan mapResult[T], done chan struct{}) {
	res := make(chan mapResult[T])
	done = make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(len(remotes))

	// start goroutines with cancellable mapping functions
	for _, remote := range remotes {
		go func(r Remote) {
			select {
			case <-done:
			case res <- mapper(r):
			}
			wg.Done()
		}(remote)
	}

	// stop processing results when all mapping goroutines are done
	go func() {
		wg.Wait()
		close(res)
	}()

	return res, done
}

// mapFirst maps all remotes concurrently and returns the first successful result or none if none of the results were successful
func mapFirst[T any](remotes []Remote, mapper func(r Remote) mapResult[T], isSuccess func(t T) bool, none T) (T, []*RepoAccessError) {
	results, done := mapConcurrent(remotes, mapper)
	r := selectFirstSuccessful(results, done, isSuccess, none)
	return r.res, r.errs
}

// selectFirstSuccessful reads results from ch until it finds the first successful with isSuccess or until ch is closed
func selectFirstSuccessful[T any](ch <-chan mapResult[T], done chan struct{}, isSuccess func(res T) bool, none T) joinedResult[T] {
	var errs []*RepoAccessError
	for res := range ch {
		if isSuccess(res.res) {
			close(done)
			return joinedResult[T]{
				res:  res.res,
				errs: nil,
			}
		}
		if res.err != nil {
			errs = append(errs, res.err)
		}
	}
	return joinedResult[T]{
		res:  none,
		errs: errs,
	}
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
	res, errs := mapReduce[[]model.FoundVersion](u.rs, mapper, ident, model.MergeFoundVersions)
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
	cs, _ = mapReduce(u.rs, mapper, cs, reducer)
	slices.Sort(cs)
	return slices.Compact(cs)
}
