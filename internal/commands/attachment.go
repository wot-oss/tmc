package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func AttachmentList(ctx context.Context, spec model.RepoSpec, tmNameOrId string) ([]string, error) {
	repo, err := repos.Get(spec)
	if err != nil {
		return nil, fmt.Errorf("Could not ìnitialize a repo instance for %s: %w\ncheck config", spec, err)
	}

	sr, err := repo.ListAttachments(ctx, tmNameOrId)
	return sr, err
}

func AttachmentPush(ctx context.Context, spec model.RepoSpec, tmNameOrId string, attachmentName string, content []byte) error {
	repo, err := repos.Get(spec)
	if err != nil {
		return fmt.Errorf("Could not ìnitialize a repo instance for %s: %w\ncheck config", spec, err)
	}

	sanitizedAttachmentName := strings.ReplaceAll(filepath.ToSlash(filepath.Clean(attachmentName)), "/", "-")
	return repo.PushAttachment(ctx, tmNameOrId, sanitizedAttachmentName, content)
}
func AttachmentDelete(ctx context.Context, spec model.RepoSpec, tmNameOrId string, attachmentName string) error {
	repo, err := repos.Get(spec)
	if err != nil {
		return fmt.Errorf("Could not ìnitialize a repo instance for %s: %w\ncheck config", spec, err)
	}

	return repo.DeleteAttachment(ctx, tmNameOrId, attachmentName)
}
func AttachmentFetch(ctx context.Context, spec model.RepoSpec, tmNameOrId string, attachmentName string) ([]byte, error) {
	repo, err := repos.Get(spec)
	if err != nil {
		return nil, fmt.Errorf("Could not ìnitialize a repo instance for %s: %w\ncheck config", spec, err)
	}

	return repo.FetchAttachment(ctx, tmNameOrId, attachmentName)
}
