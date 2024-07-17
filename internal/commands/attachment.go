package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func ImportAttachment(ctx context.Context, spec model.RepoSpec, ref model.AttachmentContainerRef, att model.Attachment, content []byte) error {
	repo, err := repos.Get(spec)
	if err != nil {
		return fmt.Errorf("Could not ìnitialize a repo instance for %s: %w\ncheck config", spec, err)
	}

	sanitizedAttachmentName := strings.ReplaceAll(filepath.ToSlash(filepath.Clean(att.Name)), "/", "-")
	sanitizedAtt := model.Attachment{Name: sanitizedAttachmentName, MediaType: att.MediaType}
	err = repo.ImportAttachment(ctx, ref, sanitizedAtt, content)
	return err
}

func DeleteAttachment(ctx context.Context, spec model.RepoSpec, ref model.AttachmentContainerRef, attachmentName string) error {
	repo, err := repos.Get(spec)
	if err != nil {
		return fmt.Errorf("Could not ìnitialize a repo instance for %s: %w\ncheck config", spec, err)
	}

	err = repo.DeleteAttachment(ctx, ref, attachmentName)
	return err
}
func AttachmentFetch(ctx context.Context, spec model.RepoSpec, ref model.AttachmentContainerRef, attachmentName string) ([]byte, error) {
	repo, err := repos.Get(spec)
	if err != nil {
		return nil, fmt.Errorf("Could not ìnitialize a repo instance for %s: %w\ncheck config", spec, err)
	}

	return repo.FetchAttachment(ctx, ref, attachmentName)
}
