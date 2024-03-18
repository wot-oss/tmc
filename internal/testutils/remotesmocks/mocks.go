package remotesmocks

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

func FailTest(t interface {
	Fail()
	Error(args ...any)
}, args ...any) {
	t.Error(args...)
	t.Fail()
}

// MockRemotesAll temporarily replaces the All() function with the provided mock for testing purposes.
// If you use own implementation of t, as opposed to *testing.T, you must make sure that registered cleanup function is called
// to restore the original All()
func MockRemotesAll(t interface {
	Cleanup(func())
}, mock func() ([]remotes.Remote, error)) {
	org := remotes.All
	remotes.All = mock
	t.Cleanup(func() { remotes.All = org })
}

func CreateMockAllFunction(err error, rs ...remotes.Remote) func() ([]remotes.Remote, error) {
	return func() ([]remotes.Remote, error) {
		return rs, err
	}
}

// MockRemotesGet temporarily replaces the Get() function with the provided mock for testing purposes.
// If you use own implementation of t, as opposed to *testing.T, you must make sure that registered cleanup function is called
// to restore the original Get()
func MockRemotesGet(t interface {
	Cleanup(func())
}, mock func(spec model.RepoSpec) (remotes.Remote, error)) {
	org := remotes.Get
	remotes.Get = mock
	t.Cleanup(func() { remotes.Get = org })
}

func CreateMockGetFunction(t *testing.T, spec model.RepoSpec, r remotes.Remote, err error) func(s model.RepoSpec) (remotes.Remote, error) {
	return func(s model.RepoSpec) (remotes.Remote, error) {
		if assert.Equal(t, spec, s, "unexpected spec in mock") {
			return r, err
		}
		err := fmt.Errorf("unexpected spec in mock. want: %v, got: %v", spec, s)
		FailTest(t, err)
		return nil, err
	}
}
