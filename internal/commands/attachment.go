package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func AttachmentPush(ctx context.Context, spec model.RepoSpec, ref model.AttachmentContainerRef, attachmentName string, content []byte) error {
	repo, err := repos.Get(spec)
	if err != nil {
		return fmt.Errorf("Could not ìnitialize a repo instance for %s: %w\ncheck config", spec, err)
	}

	sanitizedAttachmentName := strings.ReplaceAll(filepath.ToSlash(filepath.Clean(attachmentName)), "/", "-")
	err = repo.PushAttachment(ctx, ref, sanitizedAttachmentName, content)
	if err != nil {
		return err
	}
	err = repo.Index(ctx, plainIdentifier(ref))
	return err
}

func plainIdentifier(ref model.AttachmentContainerRef) string {
	switch ref.Kind() {
	case model.AttachmentContainerKindTMID:
		return ref.TMID
	case model.AttachmentContainerKindTMName:
		return ref.TMName
	default:
		return ref.String()
	}
}

func AttachmentDelete(ctx context.Context, spec model.RepoSpec, ref model.AttachmentContainerRef, attachmentName string) error {
	repo, err := repos.Get(spec)
	if err != nil {
		return fmt.Errorf("Could not ìnitialize a repo instance for %s: %w\ncheck config", spec, err)
	}

	err = repo.DeleteAttachment(ctx, ref, attachmentName)
	if err != nil {
		return err
	}
	err = repo.Index(ctx, plainIdentifier(ref))
	return err
}
func AttachmentFetch(ctx context.Context, spec model.RepoSpec, ref model.AttachmentContainerRef, attachmentName string) ([]byte, error) {
	repo, err := repos.Get(spec)
	if err != nil {
		return nil, fmt.Errorf("Could not ìnitialize a repo instance for %s: %w\ncheck config", spec, err)
	}

	return repo.FetchAttachment(ctx, ref, attachmentName)
}
