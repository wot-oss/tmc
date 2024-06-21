package reposmocks

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func FailTest(t interface {
	Fail()
	Error(args ...any)
}, args ...any) {
	t.Error(args...)
	t.Fail()
}

// MockReposAll temporarily replaces the All() function with the provided mock for testing purposes.
// If you use own implementation of t, as opposed to *testing.T, you must make sure that registered cleanup function is called
// to restore the original All()
func MockReposAll(t interface {
	Cleanup(func())
}, mock func() ([]repos.Repo, error)) {
	org := repos.All
	repos.All = mock
	t.Cleanup(func() { repos.All = org })
}

func CreateMockAllFunction(err error, rs ...repos.Repo) func() ([]repos.Repo, error) {
	return func() ([]repos.Repo, error) {
		return rs, err
	}
}

// MockReposGet temporarily replaces the Get() function with the provided mock for testing purposes.
// If you use own implementation of t, as opposed to *testing.T, you must make sure that registered cleanup function is called
// to restore the original Get()
func MockReposGet(t interface {
	Cleanup(func())
}, mock func(spec model.RepoSpec) (repos.Repo, error)) {
	orig := repos.Get
	repos.Get = mock
	t.Cleanup(func() { repos.Get = orig })
}

func CreateMockGetFunction(t *testing.T, spec model.RepoSpec, r repos.Repo, err error) func(s model.RepoSpec) (repos.Repo, error) {
	return CreateMockGetFunctionFromList(t, []model.RepoSpec{spec}, []repos.Repo{r}, []error{err})
}

func CreateMockGetFunctionFromList(t *testing.T, specs []model.RepoSpec, r []repos.Repo, errs []error) func(s model.RepoSpec) (repos.Repo, error) {
	return func(s model.RepoSpec) (repos.Repo, error) {
		if assert.Contains(t, specs, s, "unexpected spec in mock") {
			for i, spec := range specs {
				if reflect.DeepEqual(spec, s) {
					return r[i], errs[i]
				}
			}
		}
		err := fmt.Errorf("unexpected spec in mock. want one of: %v, got: %v", specs, s)
		FailTest(t, err)
		return nil, err
	}
}
