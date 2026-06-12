package commands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

type variantCall struct {
	tmID      string
	variantID string
}

type stubVariantRepo struct {
	fetchRaw      []byte
	fetchErr      error
	importErr     error
	importResult  repos.ImportResult
	indexErr      error
	addVariantErr error

	lastFetchID   string
	importedTMIDs []string
	importedRaws  [][]byte
	indexIDs      []string
	variantCalls  []variantCall
}

func (s *stubVariantRepo) AddVariant(_ context.Context, tmID string, variantID string) error {
	s.variantCalls = append(s.variantCalls, variantCall{tmID: tmID, variantID: variantID})
	return s.addVariantErr
}

func (s *stubVariantRepo) Import(_ context.Context, id model.TMID, raw []byte, _ repos.ImportOptions) (repos.ImportResult, error) {
	s.importedTMIDs = append(s.importedTMIDs, id.String())
	s.importedRaws = append(s.importedRaws, raw)
	if s.importErr != nil {
		return repos.ImportResult{}, s.importErr
	}
	res := s.importResult
	if res.Type == 0 {
		res.Type = repos.ImportResultOK
	}
	if res.TmID == "" {
		res.TmID = id.String()
	}
	return res, nil
}

func (s *stubVariantRepo) Fetch(_ context.Context, id string) (string, []byte, error) {
	s.lastFetchID = id
	return id, s.fetchRaw, s.fetchErr
}

func (s *stubVariantRepo) Index(_ context.Context, updatedIDs ...string) ([]string, []string, []string, error) {
	s.indexIDs = append(s.indexIDs, updatedIDs...)
	return nil, nil, nil, s.indexErr
}

func (s *stubVariantRepo) CheckIntegrity(_ context.Context, _ model.ResourceFilter) ([]model.CheckResult, error) {
	return nil, nil
}

func (s *stubVariantRepo) List(_ context.Context, _ *model.Filters) (model.SearchResult, error) {
	return model.SearchResult{}, nil
}

func (s *stubVariantRepo) Versions(_ context.Context, _ string) ([]model.FoundVersion, error) {
	return nil, nil
}

func (s *stubVariantRepo) Spec() model.RepoSpec {
	return model.EmptySpec
}

func (s *stubVariantRepo) CanonicalRoot() string {
	return ""
}

func (s *stubVariantRepo) Delete(_ context.Context, _ string) error {
	return nil
}

func (s *stubVariantRepo) ListCompletions(_ context.Context, _ string, _ []string, _ string) ([]string, error) {
	return nil, nil
}

func (s *stubVariantRepo) GetTMMetadata(_ context.Context, _ string) ([]model.FoundVersion, error) {
	return nil, nil
}

func (s *stubVariantRepo) ImportAttachment(_ context.Context, _ model.AttachmentContainerRef, _ model.Attachment, _ []byte, _ bool) error {
	return nil
}

func (s *stubVariantRepo) FetchAttachment(_ context.Context, _ model.AttachmentContainerRef, _ string) ([]byte, error) {
	return nil, nil
}

func (s *stubVariantRepo) DeleteAttachment(_ context.Context, _ model.AttachmentContainerRef, _ string) error {
	return nil
}

func TestAddVariantToRepo_WithExistingVariantID(t *testing.T) {
	repo := &stubVariantRepo{}
	parentTMID := "aut/man/mpn/v1.0.0-20260326150433-965fd7c3238b.tm.json"
	variantTMID := "aut/man/mpn-variant/v1.0.0-20260326150433-965fd7c3238c.tm.json"

	err := addVariantToRepo(context.Background(), repo, parentTMID, AddVariantOptions{VariantID: variantTMID})
	assert.NoError(t, err)
	if assert.Len(t, repo.variantCalls, 1) {
		assert.Equal(t, parentTMID, repo.variantCalls[0].tmID)
		assert.Equal(t, variantTMID, repo.variantCalls[0].variantID)
	}
	assert.Empty(t, repo.importedTMIDs)
	assert.Empty(t, repo.indexIDs)
}

func TestAddVariantToRepo_CreateFromParent(t *testing.T) {
	repo := &stubVariantRepo{fetchRaw: []byte(`{
		"id":"mycompany/bartech/5mn512me/special/v0.0.1-20260326150433-965fd7c3238b.tm.json",
		"description":"parent description",
		"schema:manufacturer":{"schema:name":"bartech"},
		"schema:mpn":"5mn512me",
		"schema:author":{"schema:name":"mycompany"},
		"version":{"model":"v0.0.1"}
	}`)}
	parentTMID := "mycompany/bartech/5mn512me/special/v0.0.1-20260326150433-965fd7c3238b.tm.json"

	err := addVariantToRepo(context.Background(), repo, parentTMID, AddVariantOptions{
		Mpn:         "5mn512mb",
		Description: "variant description",
		Author:      "mynewco",
	})
	assert.NoError(t, err)
	assert.Equal(t, parentTMID, repo.lastFetchID)

	if assert.Len(t, repo.importedTMIDs, 1) {
		importedID := repo.importedTMIDs[0]
		parsedID, parseErr := model.ParseTMID(importedID)
		assert.NoError(t, parseErr)
		assert.Equal(t, "mynewco/bartech/5mn512mb/special", parsedID.Name)
		if assert.NotNil(t, parsedID.Version.Base) {
			assert.Equal(t, "v0.0.1", parsedID.Version.Base.Original())
		}

		if assert.Len(t, repo.importedRaws, 1) {
			parsedTM, parseTMErr := model.ParseThingModel(repo.importedRaws[0])
			assert.NoError(t, parseTMErr)
			assert.Equal(t, "5mn512mb", parsedTM.Mpn)
			assert.Equal(t, "variant description", parsedTM.Description)
			assert.Equal(t, "mynewco", parsedTM.Author.Name)

			var doc map[string]any
			assert.NoError(t, json.Unmarshal(repo.importedRaws[0], &doc))
			assert.Equal(t, importedID, doc["id"])

			// Verify pretty-printing: should contain newlines and indentation
			assert.Contains(t, string(repo.importedRaws[0]), "\n")
			assert.Contains(t, string(repo.importedRaws[0]), "  ")
		}

		if assert.Len(t, repo.indexIDs, 1) {
			assert.Equal(t, importedID, repo.indexIDs[0])
		}
		if assert.Len(t, repo.variantCalls, 1) {
			assert.Equal(t, parentTMID, repo.variantCalls[0].tmID)
			assert.Equal(t, importedID, repo.variantCalls[0].variantID)
		}
	}
}

func TestAddVariantToRepo_InvalidOptionCombination(t *testing.T) {
	repo := &stubVariantRepo{}
	parentTMID := "aut/man/mpn/v1.0.0-20260326150433-965fd7c3238b.tm.json"

	err := addVariantToRepo(context.Background(), repo, parentTMID, AddVariantOptions{
		VariantID: "aut/man/mpn-variant/v1.0.0-20260326150433-965fd7c3238c.tm.json",
		Mpn:       "mpn2",
	})
	assert.Error(t, err)
}

func TestApplyVariantOverrides_UpdatesExpectedFields(t *testing.T) {
	input := []byte(`{
		"description":"parent",
		"schema:mpn":"old",
		"schema:author":{"schema:name":"old-author"}
	}`)

	output, err := applyVariantOverrides(input, AddVariantOptions{
		Mpn:         "new-mpn",
		Description: "new-desc",
		Author:      "new-author",
	})
	assert.NoError(t, err)

	var doc map[string]any
	assert.NoError(t, json.Unmarshal(output, &doc))
	assert.Equal(t, "new-desc", doc["description"])
	assert.Equal(t, "new-mpn", doc["schema:mpn"])
	authorObj, ok := doc["schema:author"].(map[string]any)
	if assert.True(t, ok) {
		assert.Equal(t, "new-author", authorObj["schema:name"])
	}
}
