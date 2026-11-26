package commands

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

type variantCall struct {
	tmName     string
	familyID   string
	familyName string
}

type stubVariantRepo struct {
	fetchRaw     []byte
	fetchErr     error
	importErr    error
	importResult repos.ImportResult
	indexErr     error
	setFamilyErr error
	listResult   model.SearchResult
	versions     []model.FoundVersion

	lastFetchID       string
	fetchIDs          []string
	importedTMIDs     []string
	importedRaws      [][]byte
	indexIDs          []string
	familyCalls       []variantCall
	attachmentTargets []model.AttachmentContainerRef
}

func (s *stubVariantRepo) SetFamily(_ context.Context, tmName string, familyID string, familyName string) error {
	s.familyCalls = append(s.familyCalls, variantCall{tmName: tmName, familyID: familyID, familyName: familyName})
	if s.setFamilyErr != nil {
		return s.setFamilyErr
	}
	for i := range s.listResult.Entries {
		if s.listResult.Entries[i].Name == tmName {
			s.listResult.Entries[i].FamilyID = familyID
			s.listResult.Entries[i].FamilyName = familyName
		}
	}
	return nil
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
	s.fetchIDs = append(s.fetchIDs, id)
	return id, s.fetchRaw, s.fetchErr
}

func (s *stubVariantRepo) Index(_ context.Context, updatedIDs ...string) ([]string, []string, []string, []string, error) {
	s.indexIDs = append(s.indexIDs, updatedIDs...)
	return nil, nil, nil, nil, s.indexErr
}

func (s *stubVariantRepo) CheckIntegrity(_ context.Context, _ model.ResourceFilter) ([]model.CheckResult, error) {
	return nil, nil
}

func (s *stubVariantRepo) List(_ context.Context, _ *model.Filters) (model.SearchResult, error) {
	return s.listResult, nil
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
	return s.versions, nil
}

func (s *stubVariantRepo) ImportAttachment(_ context.Context, target model.AttachmentContainerRef, _ model.Attachment, _ []byte, _ bool) error {
	s.attachmentTargets = append(s.attachmentTargets, target)
	return nil
}

func (s *stubVariantRepo) FetchAttachment(_ context.Context, _ model.AttachmentContainerRef, _ string) ([]byte, error) {
	return nil, nil
}

func (s *stubVariantRepo) DeleteAttachment(_ context.Context, _ model.AttachmentContainerRef, _ string) error {
	return nil
}

func searchResultWithTM(tmID, familyID string) model.SearchResult {
	parsed, _ := model.ParseTMID(tmID)
	familyName := ""
	if familyID != "" {
		familyName = "Test family"
	}
	return model.SearchResult{Entries: []model.FoundEntry{{
		Name:       parsed.Name,
		FamilyID:   familyID,
		FamilyName: familyName,
		Versions:   []model.FoundVersion{{IndexVersion: &model.IndexVersion{TMID: tmID}}},
	}}}
}

func rawThingModel(tmID, mpn string) []byte {
	return []byte(`{
		"id":"` + tmID + `",
		"description":"source",
		"schema:manufacturer":{"schema:name":"man"},
		"schema:mpn":"` + mpn + `",
		"schema:author":{"schema:name":"aut"},
		"version":{"model":"v1.0.0"}
	}`)
}

func TestAddVariantToRepo_WithExistingVariantID(t *testing.T) {
	parentTMID := "aut/man/mpn/v1.0.0-20260326150433-965fd7c3238b.tm.json"
	variantTMID := "aut/man/mpn-variant/v1.0.0-20260326150433-965fd7c3238c.tm.json"
	repo := &stubVariantRepo{listResult: searchResultWithTM(parentTMID, "family-1")}

	id, err := addVariantToRepo(context.Background(), repo, parentTMID, AddVariantOptions{FamilyName: "Test family", VariantID: variantTMID})
	assert.NoError(t, err)
	assert.Equal(t, variantTMID, id)
	if assert.Len(t, repo.familyCalls, 1) {
		assert.Equal(t, "aut/man/mpn-variant", repo.familyCalls[0].tmName)
		assert.Equal(t, "family-1", repo.familyCalls[0].familyID)
	}
	assert.Empty(t, repo.importedTMIDs)
	assert.Empty(t, repo.indexIDs)
}

func TestAddVariantToRepo_WithExistingVariantID_MismatchedManufacturer(t *testing.T) {
	parentTMID := "aut/man-a/mpn/v1.0.0-20260326150433-965fd7c3238b.tm.json"
	variantTMID := "aut/man-b/mpn-variant/v1.0.0-20260326150433-965fd7c3238c.tm.json"
	repo := &stubVariantRepo{listResult: searchResultWithTM(parentTMID, "family-1")}

	_, err := addVariantToRepo(context.Background(), repo, parentTMID, AddVariantOptions{FamilyName: "Test family", VariantID: variantTMID})
	assert.ErrorContains(t, err, "variant-id must match parent tm-id author and manufacturer")
	assert.Empty(t, repo.familyCalls)
	assert.Empty(t, repo.importedTMIDs)
	assert.Empty(t, repo.indexIDs)
}

func TestAddVariantToRepo_CreateFromParent(t *testing.T) {
	parentTMID := "mycompany/bartech/5mn512me/special/v0.0.1-20260326150433-965fd7c3238b.tm.json"
	repo := &stubVariantRepo{fetchRaw: []byte(`{
		"id":"mycompany/bartech/5mn512me/special/v0.0.1-20260326150433-965fd7c3238b.tm.json",
		"description":"parent description",
		"schema:manufacturer":{"schema:name":"bartech"},
		"schema:mpn":"5mn512me",
		"schema:author":{"schema:name":"mycompany"},
		"version":{"model":"v0.0.1"}
	}`), listResult: searchResultWithTM(parentTMID, "")}

	_, err := addVariantToRepo(context.Background(), repo, parentTMID, AddVariantOptions{
		FamilyName:  "Test family",
		Mpn:         "5mn512mb",
		Description: "variant description",
		Title:       "mytitle",
	})
	assert.NoError(t, err)
	assert.Equal(t, parentTMID, repo.lastFetchID)

	if assert.Len(t, repo.importedTMIDs, 1) {
		importedID := repo.importedTMIDs[0]
		parsedID, parseErr := model.ParseTMID(importedID)
		assert.NoError(t, parseErr)
		assert.Equal(t, "mycompany/bartech/5mn512mb/special", parsedID.Name)
		if assert.NotNil(t, parsedID.Version.Base) {
			assert.Equal(t, "v0.0.1", parsedID.Version.Base.Original())
		}

		if assert.Len(t, repo.importedRaws, 1) {
			parsedTM, parseTMErr := model.ParseThingModel(repo.importedRaws[0])
			assert.NoError(t, parseTMErr)
			assert.Equal(t, "5mn512mb", parsedTM.Mpn)
			assert.Equal(t, "variant description", parsedTM.Description)
			assert.Equal(t, "mytitle", parsedTM.Title)

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
		if assert.Len(t, repo.familyCalls, 2) {
			assert.Equal(t, "mycompany/bartech/5mn512me/special", repo.familyCalls[0].tmName)
			assert.Equal(t, "mycompany/bartech/5mn512mb/special", repo.familyCalls[1].tmName)
			assert.NotEmpty(t, repo.familyCalls[0].familyID)
			assert.Equal(t, repo.familyCalls[0].familyID, repo.familyCalls[1].familyID)
			assert.Equal(t, "Test family", repo.familyCalls[0].familyName)
			assert.Equal(t, "Test family", repo.familyCalls[1].familyName)
		}
	}
}

func TestAddVariantToRepo_InvalidOptionCombination(t *testing.T) {
	repo := &stubVariantRepo{}
	parentTMID := "aut/man/mpn/v1.0.0-20260326150433-965fd7c3238b.tm.json"

	_, err := addVariantToRepo(context.Background(), repo, parentTMID, AddVariantOptions{
		VariantID: "aut/man/mpn-variant/v1.0.0-20260326150433-965fd7c3238c.tm.json",
		Mpn:       "mpn2",
	})
	assert.Error(t, err)
}

func TestCreateVariantFromParent_MpnOnlyChangeGeneratesDifferentTMID(t *testing.T) {
	parentTMID := "aut/man/source/v1.0.0-20260326150433-965fd7c3238b.tm.json"
	parentRaw := rawThingModel(parentTMID, "source")
	parentHash, _, err := CalculateFileDigest(parentRaw)
	assert.NoError(t, err)
	repo := &stubVariantRepo{fetchRaw: parentRaw}

	variantID, err := createVariantFromParent(context.Background(), repo, parentTMID, AddVariantOptions{Mpn: "variant-a"})
	assert.NoError(t, err)

	parentID, err := model.ParseTMID(parentTMID)
	assert.NoError(t, err)
	variantTMID, err := model.ParseTMID(variantID)
	assert.NoError(t, err)
	assert.NotEqual(t, parentTMID, variantID)
	assert.NotEqual(t, parentID.Name, variantTMID.Name)
	assert.Equal(t, "aut/man/variant-a", variantTMID.Name)
	assert.NotEqual(t, parentHash, variantTMID.Version.Hash)
}

func TestAddVariantsBatchUsesOneParentAndFamily(t *testing.T) {
	parentTMID := "aut/man/mpn/v1.0.0-20260326150433-965fd7c3238b.tm.json"
	for _, familyID := range []string{"", "family-1"} {
		t.Run("family="+familyID, func(t *testing.T) {
			repo := &stubVariantRepo{
				fetchRaw:   rawThingModel(parentTMID, "mpn"),
				listResult: searchResultWithTM(parentTMID, familyID),
			}
			originalGet := repos.Get
			t.Cleanup(func() { repos.Get = originalGet })
			getCalls := 0
			repos.Get = func(model.RepoSpec) (repos.Repo, error) {
				getCalls++
				return repo, nil
			}

			results, err := AddVariantsBatch(context.Background(), model.EmptySpec, parentTMID, AddVariantsBatchRequest{
				FamilyName: "Test family",
				Variants: []AddVariantBatchRequest{
					{Mpn: "variant-a", Title: "Title A", Description: "Description A"},
					{Mpn: "variant-b"},
				}})
			require.NoError(t, err)

			assert.Equal(t, 1, getCalls)
			assert.Equal(t, []string{parentTMID, parentTMID}, repo.fetchIDs)
			assert.Equal(t, repo.importedTMIDs, repo.indexIDs)
			if assert.Len(t, results, 2) {
				for i, result := range results {
					assert.Empty(t, result.Error)
					assert.Equal(t, parentTMID, result.TmID)
					assert.Equal(t, repo.importedTMIDs[i], result.VariantID)
				}
			}
			if assert.Len(t, repo.importedRaws, 2) {
				tm, err := model.ParseThingModel(repo.importedRaws[0])
				assert.NoError(t, err)
				assert.Equal(t, "Title A", tm.Title)
				assert.Equal(t, "Description A", tm.Description)
			}
			expectedNames := []string{"aut/man/variant-a", "aut/man/variant-b"}
			if familyID == "" {
				expectedNames = append([]string{"aut/man/mpn"}, expectedNames...)
			}
			if assert.Len(t, repo.familyCalls, len(expectedNames)) {
				actualFamilyID := repo.familyCalls[0].familyID
				assert.NotEmpty(t, actualFamilyID)
				if familyID != "" {
					assert.Equal(t, familyID, actualFamilyID)
				}
				for i, call := range repo.familyCalls {
					assert.Equal(t, expectedNames[i], call.tmName)
					assert.Equal(t, actualFamilyID, call.familyID)
					assert.Equal(t, "Test family", call.familyName)
				}
			}
		})
	}
}

func TestExistingFamilyNameIsReused(t *testing.T) {
	parentTMID := "aut/man/source/v1.0.0-20260326150433-965fd7c3238b.tm.json"
	variantTMID := "aut/man/existing/v1.0.0-20260326150433-965fd7c3238c.tm.json"
	for _, kind := range []string{"single", "link", "batch"} {
		t.Run(kind, func(t *testing.T) {
			repo := &stubVariantRepo{
				fetchRaw:   rawThingModel(parentTMID, "source"),
				listResult: searchResultWithTM(parentTMID, "family-1"),
			}
			originalGet := repos.Get
			t.Cleanup(func() { repos.Get = originalGet })
			repos.Get = func(model.RepoSpec) (repos.Repo, error) { return repo, nil }
			switch kind {
			case "batch":
				results, err := AddVariantsBatch(context.Background(), model.EmptySpec, parentTMID, AddVariantsBatchRequest{
					Variants: []AddVariantBatchRequest{{Mpn: "variant-a"}, {VariantID: variantTMID}},
				})
				require.NoError(t, err)
				require.Len(t, results, 2)
				for _, result := range results {
					assert.Empty(t, result.Error)
					assert.NotEmpty(t, result.VariantID)
				}
			case "link":
				err := AddVariant(context.Background(), model.EmptySpec, parentTMID, AddVariantOptions{VariantID: variantTMID})
				require.NoError(t, err)
			default:
				err := AddVariant(context.Background(), model.EmptySpec, parentTMID, AddVariantOptions{Mpn: "variant-a"})
				require.NoError(t, err)
			}
			require.NotEmpty(t, repo.familyCalls)
			for _, call := range repo.familyCalls {
				assert.Equal(t, "family-1", call.familyID)
				assert.Equal(t, "Test family", call.familyName)
			}
		})
	}
}

func TestAddVariantsBatchLinksExistingAndContinuesAfterInvalidItems(t *testing.T) {
	parentTMID := "aut/man/source/v1.0.0-20260326150433-965fd7c3238b.tm.json"
	variantTMID := "aut/man/existing/v1.0.0-20260326150433-965fd7c3238c.tm.json"
	repo := &stubVariantRepo{
		fetchRaw:   rawThingModel(parentTMID, "source"),
		listResult: searchResultWithTM(parentTMID, "family-1"),
	}
	originalGet := repos.Get
	t.Cleanup(func() { repos.Get = originalGet })
	repos.Get = func(model.RepoSpec) (repos.Repo, error) { return repo, nil }

	results, err := AddVariantsBatch(context.Background(), model.EmptySpec, parentTMID, AddVariantsBatchRequest{
		FamilyName: "Test family",
		Variants: []AddVariantBatchRequest{
			{VariantID: variantTMID},
			{},
			{VariantID: variantTMID, Mpn: "invalid"},
			{VariantID: "aut/other/existing/v1.0.0-20260326150433-965fd7c3238c.tm.json"},
			{Mpn: "variant-c"},
		}})
	require.NoError(t, err)
	if assert.Len(t, results, 5) {
		assert.Empty(t, results[0].Error)
		assert.Equal(t, variantTMID, results[0].VariantID)
		assert.Contains(t, results[1].Error, "either variantID or mpn")
		assert.Contains(t, results[2].Error, "mutually exclusive")
		assert.Contains(t, results[3].Error, "author and manufacturer")
		assert.Empty(t, results[4].Error)
		assert.NotEmpty(t, results[4].VariantID)
		for _, result := range results {
			assert.Equal(t, parentTMID, result.TmID)
			if result.Error != "" {
				assert.Empty(t, result.VariantID)
			}
		}
	}
	assert.Len(t, repo.importedTMIDs, 1)
	if assert.Len(t, repo.familyCalls, 2) {
		assert.Equal(t, variantCall{"aut/man/existing", "family-1", "Test family"}, repo.familyCalls[0])
		assert.Equal(t, variantCall{"aut/man/variant-c", "family-1", "Test family"}, repo.familyCalls[1])
	}
}

func TestAddVariantsBatchCopiesAttachments(t *testing.T) {
	parentTMID := "aut/man/source/v1.0.0-20260326150433-965fd7c3238b.tm.json"
	repo := &stubVariantRepo{
		fetchRaw:   rawThingModel(parentTMID, "source"),
		listResult: searchResultWithTM(parentTMID, "family-1"),
		versions: []model.FoundVersion{{
			IndexVersion: &model.IndexVersion{
				TMID: parentTMID,
				AttachmentContainer: model.AttachmentContainer{
					Attachments: []model.Attachment{{Name: "readme.txt"}},
				},
			},
		}},
	}
	originalGet := repos.Get
	t.Cleanup(func() { repos.Get = originalGet })
	repos.Get = func(model.RepoSpec) (repos.Repo, error) { return repo, nil }
	results, err := AddVariantsBatch(context.Background(), model.EmptySpec, parentTMID, AddVariantsBatchRequest{
		FamilyName: "Test family",
		Variants: []AddVariantBatchRequest{
			{Mpn: "with-attachments", WithAttachments: true},
			{Mpn: "without-attachments"},
		}})
	require.NoError(t, err)
	if assert.Len(t, results, 2) {
		assert.Empty(t, results[0].Error)
		assert.Empty(t, results[1].Error)
		assert.Equal(t, []model.AttachmentContainerRef{model.NewTMIDAttachmentContainerRef(results[0].VariantID)}, repo.attachmentTargets)
	}
}

func TestAddVariantsBatchReportsRepositoryError(t *testing.T) {
	originalGet := repos.Get
	t.Cleanup(func() { repos.Get = originalGet })
	repos.Get = func(model.RepoSpec) (repos.Repo, error) { return nil, errors.New("repo unavailable") }
	results, err := AddVariantsBatch(context.Background(), model.EmptySpec, "parent", AddVariantsBatchRequest{
		FamilyName: "Test family",
		Variants: []AddVariantBatchRequest{
			{Mpn: "variant-a"}, {Mpn: "variant-b"},
		}})
	assert.ErrorContains(t, err, "failed to initialize repo: repo unavailable")
	assert.Nil(t, results)
}

func TestAddVariantsBatchRejectsInvalidParent(t *testing.T) {
	for _, tmID := range []string{"", "invalid"} {
		t.Run(tmID, func(t *testing.T) {
			repo := &stubVariantRepo{}
			originalGet := repos.Get
			t.Cleanup(func() { repos.Get = originalGet })
			repos.Get = func(model.RepoSpec) (repos.Repo, error) { return repo, nil }
			results, err := AddVariantsBatch(context.Background(), model.EmptySpec, tmID, AddVariantsBatchRequest{
				FamilyName: "Test family",
				Variants: []AddVariantBatchRequest{
					{Mpn: "variant-a"}, {Mpn: "variant-b"},
				}})
			assert.Error(t, err)
			assert.Nil(t, results)
			assert.Empty(t, repo.fetchIDs)
			assert.Empty(t, repo.importedTMIDs)
			assert.Empty(t, repo.familyCalls)
		})
	}
}

func TestVariantFamilyValidationRejectsBeforeAnyWrites(t *testing.T) {
	parentTMID := "aut/man/source/v1.0.0-20260326150433-965fd7c3238b.tm.json"
	for _, tt := range []struct {
		name        string
		familyID    string
		storedName  string
		requestName string
		memberName  string
		wantErr     error
	}{
		{"missing name", "", "", "", "", model.ErrFamilyNameRequired},
		{"blank name", "", "", " \t ", "", model.ErrFamilyNameRequired},
		{"blank name for existing family", "family-1", "Original", " \t ", "", model.ErrFamilyNameRequired},
		{"different name", "family-1", "Original", "Different", "", model.ErrFamilyNameConflict},
		{"case differs", "family-1", "Original", "original", "", model.ErrFamilyNameConflict},
		{"unnamed family", "family-1", "", "Original", "", model.ErrFamilyUnnamed},
		{"unnamed family without supplied name", "family-1", "", "", "", model.ErrFamilyUnnamed},
		{"inconsistent member", "family-1", "Original", "Original", "Different", model.ErrFamilyNameConflict},
		{"inconsistent member without supplied name", "family-1", "Original", "", "Different", model.ErrFamilyNameConflict},
	} {
		for _, batch := range []bool{false, true} {
			kind := "single"
			if batch {
				kind = "batch"
			}
			t.Run(tt.name+"/"+kind, func(t *testing.T) {
				result := searchResultWithTM(parentTMID, tt.familyID)
				result.Entries[0].FamilyName = tt.storedName
				if tt.memberName != "" {
					result.Entries = append(result.Entries, model.FoundEntry{
						Name: "aut/man/other", FamilyID: tt.familyID, FamilyName: tt.memberName,
					})
				}
				repo := &stubVariantRepo{listResult: result}
				originalGet := repos.Get
				t.Cleanup(func() { repos.Get = originalGet })
				repos.Get = func(model.RepoSpec) (repos.Repo, error) { return repo, nil }
				if batch {
					results, err := AddVariantsBatch(context.Background(), model.EmptySpec, parentTMID, AddVariantsBatchRequest{
						FamilyName: tt.requestName,
						Variants:   []AddVariantBatchRequest{{Mpn: "variant-a"}, {Mpn: "variant-b"}},
					})
					assert.ErrorIs(t, err, tt.wantErr)
					assert.Nil(t, results)
				} else {
					err := AddVariant(context.Background(), model.EmptySpec, parentTMID, AddVariantOptions{
						FamilyName: tt.requestName, Mpn: "variant-a",
					})
					assert.ErrorIs(t, err, tt.wantErr)
				}
				assert.Empty(t, repo.fetchIDs)
				assert.Empty(t, repo.importedTMIDs)
				assert.Empty(t, repo.indexIDs)
				assert.Empty(t, repo.familyCalls)
				assert.Empty(t, repo.attachmentTargets)
				assert.Equal(t, tt.storedName, repo.listResult.Entries[0].FamilyName)
			})
		}
	}
}

func TestAddVariantsBatchPersistsParentFamily(t *testing.T) {
	ctx := context.Background()
	parentTMID := "aut/man/source/v1.0.0-20260326150433-965fd7c3238b.tm.json"
	config := repos.ConfigMap{"loc": t.TempDir()}
	repo, err := repos.NewFileRepo(config, model.EmptySpec)
	require.NoError(t, err)
	parentID, err := model.ParseTMID(parentTMID)
	require.NoError(t, err)
	_, err = repo.Import(ctx, parentID, rawThingModel(parentTMID, "source"), repos.ImportOptions{})
	require.NoError(t, err)
	_, _, _, _, err = repo.Index(ctx, parentTMID)
	require.NoError(t, err)

	originalGet := repos.Get
	t.Cleanup(func() { repos.Get = originalGet })
	repos.Get = func(model.RepoSpec) (repos.Repo, error) { return repo, nil }
	results, err := AddVariantsBatch(ctx, model.EmptySpec, parentTMID, AddVariantsBatchRequest{
		FamilyName: "Test family",
		Variants: []AddVariantBatchRequest{
			{Mpn: "variant-a"}, {Mpn: "variant-b"},
		}})
	require.NoError(t, err)
	require.Len(t, results, 2)
	for _, result := range results {
		require.Empty(t, result.Error)
	}

	_, _, _, _, err = repo.Index(ctx)
	require.NoError(t, err)
	reopened, err := repos.NewFileRepo(config, model.EmptySpec)
	require.NoError(t, err)
	familyID, familyName, err := findFamilyByTMID(ctx, reopened, parentTMID)
	require.NoError(t, err)
	assert.NotEmpty(t, familyID)
	assert.Equal(t, "Test family", familyName)
	for _, result := range results {
		variantFamilyID, variantFamilyName, err := findFamilyByTMID(ctx, reopened, result.VariantID)
		require.NoError(t, err)
		assert.Equal(t, familyID, variantFamilyID)
		assert.Equal(t, familyName, variantFamilyName)
	}
}

func TestApplyVariantOverrides_UpdatesExpectedFields(t *testing.T) {
	input := []byte(`{
		"description":"parent",
		"schema:mpn":"old",
		"title":"old-title"
	}`)

	output, err := applyVariantOverrides(input, AddVariantOptions{
		Mpn:         "new-mpn",
		Description: "new-desc",
		Title:       "new-title",
	})
	assert.NoError(t, err)

	var doc map[string]any
	assert.NoError(t, json.Unmarshal(output, &doc))
	assert.Equal(t, "new-desc", doc["description"])
	assert.Equal(t, "new-mpn", doc["schema:mpn"])
	assert.Equal(t, "new-title", doc["title"])
}
