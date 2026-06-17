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
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

type variantAdder interface {
	AddVariant(ctx context.Context, tmID string, variantID string) error
}

type AddVariantOptions struct {
	VariantID   string
	Mpn         string
	Description string
	Title       string
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

func addVariantToRepo(ctx context.Context, repo repos.Repo, tmID string, opts AddVariantOptions) error {
	if _, err := model.ParseTMID(tmID); err != nil {
		return err
	}
	if err := opts.checkOptions(); err != nil {
		return err
	}

	r, ok := repo.(variantAdder)
	if !ok {
		return repos.ErrNotSupported
	}

	variantID := opts.VariantID
	var err error
	if variantID == "" {
		variantID, err = createVariantFromParent(ctx, repo, tmID, opts)
		if err != nil {
			return err
		}
		_, _, _, err = repo.Index(ctx, variantID)
		if err != nil {
			return err
		}
	}

	return r.AddVariant(ctx, tmID, variantID)
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
		Mpn:         strings.TrimSpace(opts.Mpn),
		Description: opts.Description,
		Title:       opts.Title,
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
	finalRaw, err := setIDField(normalized, variantID.String())
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

func setIDField(raw []byte, id string) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	doc["id"] = id
	return json.MarshalIndent(doc, "", "  ")
}

func variantOptPathFromParent(parentName string) string {
	parts := strings.Split(parentName, "/")
	if len(parts) <= 3 {
		return ""
	}
	return path.Join(parts[3:]...)
}
