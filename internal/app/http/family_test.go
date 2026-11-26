package http

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wot-oss/tmc/internal/app/http/mocks"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
	repoMocks "github.com/wot-oss/tmc/internal/repos/mocks"
	"github.com/wot-oss/tmc/internal/testutils"
	rMocks "github.com/wot-oss/tmc/internal/testutils/reposmocks"
)

func TestGetFamilies(t *testing.T) {
	for _, tt := range []struct {
		name       string
		lines      []string
		err        error
		status     int
		body       string
		namespaces []string
		authors    []string
	}{
		{name: "plain text lines", lines: []string{"Eagle:family-1", "Lightall:family-2"}, status: http.StatusOK, body: "Eagle:family-1\nLightall:family-2\n"},
		{name: "empty", status: http.StatusOK, body: ""},
		{name: "namespace read", lines: []string{"Eagle:family-1"}, status: http.StatusOK, body: "Eagle:family-1\n", namespaces: []string{"a-corp"}, authors: []string{"a-corp"}},
		{name: "wildcard read", lines: []string{"Eagle:family-1"}, status: http.StatusOK, body: "Eagle:family-1\n", namespaces: []string{"*"}},
		{name: "repository error", err: repos.ErrRepoNotFound, status: http.StatusBadRequest},
		{name: "upstream error", err: errors.New("list failed"), status: http.StatusInternalServerError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := mocks.NewHandlerService(t)
			var filters *model.Filters
			if tt.authors != nil {
				filters = &model.Filters{Author: tt.authors}
			}
			service.On("ListFamilies", mock.Anything, "selected", filters).Return(tt.lines, tt.err).Once()
			handler := NewHttpHandler(NewTmcHandler(service, TmcHandlerOptions{JWTValidation: true}), nil)
			request := testutils.NewRequest(http.MethodGet, "/family?repo=selected")
			ctx := context.WithValue(context.Background(), ContextKeyBearerAuthNamespaces, tt.namespaces)
			rec := request.RunOnHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handler.ServeHTTP(w, r.WithContext(ctx))
			}))
			assert.Equal(t, tt.status, rec.Code)
			if tt.err == nil {
				assert.Equal(t, MimeText, rec.Header().Get(HeaderContentType))
				assert.Equal(t, tt.body, rec.Body.String())
			}
		})
	}
}

func TestListFamilies(t *testing.T) {
	for _, tt := range []struct {
		name    string
		entries []model.FoundEntry
		want    []string
		err     error
	}{
		{
			name: "deduplicates IDs and sorts lines without merging equal names",
			entries: []model.FoundEntry{
				{Name: "aut/man/c", FamilyID: "id-c", FamilyName: "Zebra"},
				{Name: "aut/man/a", FamilyID: "id-a", FamilyName: "Eagle"},
				{Name: "aut/man/b", FamilyID: "id-a", FamilyName: "Eagle"},
				{Name: "aut/man/d", FamilyID: "id-b", FamilyName: "Eagle"},
				{Name: "aut/man/no-family"},
				{Name: "aut/man/legacy", FamilyID: "legacy-id"},
			},
			want: []string{":legacy-id", "Eagle:id-a", "Eagle:id-b", "Zebra:id-c"},
		},
		{name: "empty", want: []string{}},
		{
			name: "conflicting metadata",
			entries: []model.FoundEntry{
				{Name: "aut/man/a", FamilyID: "id-a", FamilyName: "Eagle"},
				{Name: "aut/man/b", FamilyID: "id-a", FamilyName: "Other"},
			},
			err: errors.New("inconsistent names"),
		},
		{name: "upstream error", err: errors.New("repository failed")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewDefaultHandlerService(model.EmptySpec)
			require.NoError(t, err)
			repo := repoMocks.NewRepo(t)
			filters := &model.Filters{Author: []string{"aut"}}
			var listErr error
			if tt.name == "upstream error" {
				listErr = tt.err
				repo.On("Spec").Return(model.NewRepoSpec("test"))
			}
			repo.On("List", mock.Anything, filters).Return(model.SearchResult{Entries: tt.entries}, listErr).Once()
			rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, repo))
			lines, err := service.ListFamilies(context.Background(), "", filters)
			if tt.err != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.err.Error())
				assert.Nil(t, lines)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, lines)
			}
		})
	}
}
