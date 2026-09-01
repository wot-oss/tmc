package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/google/uuid"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

type AddVariantOptions struct {
	VariantID       string
	Mpn             string
	Description     string
	Title           string
	WithAttachments bool
}

type AddVariantBatchRequest struct {
	TmID            string `json:"tm-id"`
	Mpn             string `json:"mpn"`
	Description     string `json:"description,omitempty"`
	Title           string `json:"title,omitempty"`
	WithAttachments bool   `json:"with-attachments,omitempty"`
}

type AddVariantBatchResult struct {
	TmID      string `json:"tm-id"`
	VariantID string `json:"variant-id"`
	Error     string `json:"error,omitempty"`
}

func (o AddVariantOptions) checkOptions() error {
	if o.VariantID == "" && o.Mpn == "" {
		return errors.New("either variantID or mpn must be provided")
	}
	if o.VariantID != "" && o.Mpn != "" {
		return errors.New("variantID and mpn are mutually exclusive")
	}
	if o.VariantID != "" && (o.Description != "" || o.Title != "") {
		return errors.New("description and title can only be used when creating a variant via mpn")
	}
	if o.VariantID != "" {
		if _, err := model.ParseTMID(o.VariantID); err != nil {
			return err
		}
	}
	if o.Mpn != "" && strings.TrimSpace(o.Mpn) == "" {
		return errors.New("mpn must not be empty")
	}
	return nil
}

func AddVariant(ctx context.Context, spec model.RepoSpec, tmID string, opts AddVariantOptions) error {
	repo, err := repos.Get(spec)
	if err != nil {
		return err
	}
	return addVariantToRepo(ctx, repo, tmID, opts)
}

func AddVariantsBatch(ctx context.Context, spec model.RepoSpec, familyTMID string, requests []AddVariantBatchRequest) []AddVariantBatchResult {
	var results []AddVariantBatchResult
	repo, err := repos.Get(spec)
	if err != nil {
		return []AddVariantBatchResult{{Error: fmt.Sprintf("failed to initialize repo: %v", err)}}
	}
	familyID := uuid.NewString()
	if familyTMID != "" {
		familyID, err = findFamilyByTMID(ctx, repo, familyTMID)
		if err != nil {
			return []AddVariantBatchResult{{Error: fmt.Sprintf("failed to resolve family: %v", err)}}
		}
	}

	for _, req := range requests {
		tmID := strings.TrimSpace(req.TmID)
		if tmID == "" {
			results = append(results, AddVariantBatchResult{
				TmID:  req.TmID,
				Error: "tm-id is required",
			})
			continue
		}

		if _, err := model.ParseTMID(tmID); err != nil {
			results = append(results, AddVariantBatchResult{
				TmID:  req.TmID,
				Error: fmt.Sprintf("invalid tm-id: %v", err),
			})
			continue
		}

		opts := AddVariantOptions{
			Mpn:         strings.TrimSpace(req.Mpn),
			Description: strings.TrimSpace(req.Description),
			Title:       strings.TrimSpace(req.Title),
		}

		if opts.Mpn == "" {
			results = append(results, AddVariantBatchResult{
				TmID:  req.TmID,
				Error: "mpn is required for variant creation",
			})
			continue
		}

		variantID, err := createVariantFromParent(ctx, repo, tmID, opts)
		if err != nil {
			results = append(results, AddVariantBatchResult{
				TmID:  req.TmID,
				Error: fmt.Sprintf("failed to create variant: %v", err),
			})
			continue
		}

		_, _, _, err = repo.Index(ctx, variantID)
		if err != nil {
			results = append(results, AddVariantBatchResult{
				TmID:  req.TmID,
				Error: fmt.Sprintf("failed to index variant: %v", err),
			})
			continue
		}
		if opts.WithAttachments {
			err = copyVariantAttachments(ctx, repo, tmID, variantID)
			if err != nil {
				results = append(results, AddVariantBatchResult{
					TmID:  req.TmID,
					Error: fmt.Sprintf("failed to copy variant attachments: %v", err),
				})
				continue
			}
		}

		variantTMID, err := model.ParseTMID(variantID)
		if err == nil {
			err = repo.SetFamily(ctx, variantTMID.Name, familyID)
		}
		if err != nil {
			results = append(results, AddVariantBatchResult{
				TmID:  req.TmID,
				Error: fmt.Sprintf("failed to assign variant family: %v", err),
			})
			continue
		}

		results = append(results, AddVariantBatchResult{
			TmID:      req.TmID,
			VariantID: variantID,
		})
	}

	return results
}

var errTMNotInFamily = errors.New("Thing Model is not part of a variant family")

func findFamilyByTMID(ctx context.Context, repo repos.Repo, tmID string) (string, error) {
	parsed, err := model.ParseTMID(tmID)
	if err != nil {
		return "", err
	}
	result, err := repo.List(ctx, &model.Filters{
		Name:    parsed.Name,
		Options: model.FilterOptions{NameFilterType: model.FullMatch},
	})
	if err != nil {
		return "", err
	}
	for _, entry := range result.Entries {
		for _, version := range entry.Versions {
			if version.TMID == tmID {
				if entry.FamilyID == "" {
					return "", errTMNotInFamily
				}
				return entry.FamilyID, nil
			}
		}
	}
	return "", model.ErrTMNotFound
}

func addVariantToRepo(ctx context.Context, repo repos.Repo, tmID string, opts AddVariantOptions) error {
	parentTMID, err := model.ParseTMID(tmID)
	if err != nil {
		return err
	}
	if err := opts.checkOptions(); err != nil {
		return err
	}
	if opts.VariantID != "" {
		variantTMID, err := model.ParseTMID(opts.VariantID)
		if err != nil {
			return err
		}
		parentParts := strings.Split(parentTMID.Name, "/")
		variantParts := strings.Split(variantTMID.Name, "/")
		if !(parentParts[0] == variantParts[0] && parentParts[1] == variantParts[1]) {
			return errors.New("variant-id must match parent tm-id author and manufacturer")
		}
	}

	variantID := opts.VariantID
	if variantID == "" {
		variantID, err = createVariantFromParent(ctx, repo, tmID, opts)
		if err != nil {
			return err
		}
		_, _, _, err = repo.Index(ctx, variantID)
		if err != nil {
			return err
		}
		if opts.WithAttachments {
			err = copyVariantAttachments(ctx, repo, tmID, variantID)
			if err != nil {
				return err
			}
		}
	}

	familyID, err := findFamilyByTMID(ctx, repo, tmID)
	if errors.Is(err, errTMNotInFamily) {
		familyID = uuid.NewString()
		err = repo.SetFamily(ctx, parentTMID.Name, familyID)
	}
	if err != nil {
		return err
	}
	variantTMID, err := model.ParseTMID(variantID)
	if err != nil {
		return err
	}
	return repo.SetFamily(ctx, variantTMID.Name, familyID)
}

func copyVariantAttachments(ctx context.Context, repo repos.Repo, parentTMID string, variantID string) error {
	parsedParentTMID, err := model.ParseTMID(parentTMID)
	if err != nil {
		return err
	}

	versions, err := repo.GetTMMetadata(ctx, parentTMID)
	if err != nil {
		return err
	}

	parentTMIDRef := model.NewTMIDAttachmentContainerRef(parentTMID)
	parentTMNameRef := model.NewTMNameAttachmentContainerRef(parsedParentTMID.Name)
	variantRef := model.NewTMIDAttachmentContainerRef(variantID)
	copied := map[string]struct{}{}

	searchResult, err := repo.List(ctx, &model.Filters{Name: parsedParentTMID.Name, Options: model.FilterOptions{NameFilterType: model.FullMatch}})
	if err != nil {
		return err
	}
	for _, entry := range searchResult.Entries {
		if entry.Name != parsedParentTMID.Name {
			continue
		}
		for _, attachment := range entry.Attachments {
			if _, exists := copied[attachment.Name]; exists {
				continue
			}
			content, err := repo.FetchAttachment(ctx, parentTMNameRef, attachment.Name)
			if err != nil {
				return err
			}
			err = repo.ImportAttachment(ctx, variantRef, attachment, content, false)
			if err != nil && !errors.Is(err, repos.ErrAttachmentExists) {
				return err
			}
			copied[attachment.Name] = struct{}{}
		}
	}

	for _, version := range versions {
		if version.TMID != parentTMID {
			continue
		}
		for _, attachment := range version.Attachments {
			if _, exists := copied[attachment.Name]; exists {
				continue
			}
			content, err := repo.FetchAttachment(ctx, parentTMIDRef, attachment.Name)
			if err != nil {
				return err
			}
			err = repo.ImportAttachment(ctx, variantRef, attachment, content, false)
			if err != nil && !errors.Is(err, repos.ErrAttachmentExists) {
				return err
			}
			copied[attachment.Name] = struct{}{}
		}
		return nil
	}

	return model.ErrTMNotFound
}

func createVariantFromParent(ctx context.Context, repo repos.Repo, tmID string, opts AddVariantOptions) (string, error) {
	parentTMID, err := model.ParseTMID(tmID)
	if err != nil {
		return "", err
	}

	_, parentRaw, err := repo.Fetch(ctx, tmID)
	if err != nil {
		return "", err
	}

	variantRaw, err := applyVariantOverrides(parentRaw, AddVariantOptions{
		Mpn:             strings.TrimSpace(opts.Mpn),
		Description:     opts.Description,
		Title:           opts.Title,
		WithAttachments: opts.WithAttachments,
	})
	if err != nil {
		return "", err
	}

	variantTM, err := model.ParseThingModel(variantRaw)
	if err != nil {
		return "", err
	}

	hashStr, normalized, err := CalculateFileDigest(variantRaw)
	if err != nil {
		return "", err
	}
	ver := model.TMVersionFromOriginal(variantTM.Version.Model)
	ver.Hash = hashStr
	ver.Timestamp = time.Now().UTC().Format(model.PseudoVersionTimestampFormat)

	optPath := variantOptPathFromParent(parentTMID.Name)
	variantID := model.NewTMID(variantTM.Author.Name, variantTM.Manufacturer.Name, variantTM.Mpn, optPath, ver)
	finalRaw, err := setField(normalized, "id", variantID.String())
	if err != nil {
		return "", err
	}

	res, err := repo.Import(ctx, variantID, finalRaw, repos.ImportOptions{})
	if err != nil {
		return "", err
	}
	if !res.IsSuccessful() {
		return "", fmt.Errorf("importing generated variant failed: %s", res.Message)
	}
	return variantID.String(), nil
}

func applyVariantOverrides(parentRaw []byte, opts AddVariantOptions) ([]byte, error) {
	mpn, err := json.Marshal(opts.Mpn)
	if err != nil {
		return nil, err
	}
	modified, err := jsonparser.Set(parentRaw, mpn, "schema:mpn")
	if err != nil {
		return nil, err
	}

	if opts.Description != "" {
		description, err := json.Marshal(opts.Description)
		if err != nil {
			return nil, err
		}
		modified, err = jsonparser.Set(modified, description, "description")
		if err != nil {
			return nil, err
		}
	}

	if opts.Title != "" {
		title, err := json.Marshal(opts.Title)
		if err != nil {
			return nil, err
		}
		modified, err = jsonparser.Set(modified, title, "title")
		if err != nil {
			return nil, err
		}
	}

	return modified, nil
}

func setField(raw []byte, fieldName string, value any) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	doc[fieldName] = value
	return json.MarshalIndent(doc, "", "  ")
}

func variantOptPathFromParent(parentName string) string {
	parts := strings.Split(parentName, "/")
	if len(parts) <= 3 {
		return ""
	}
	return path.Join(parts[3:]...)
}
