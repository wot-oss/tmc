package commands

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func ImportAttachment(ctx context.Context, spec model.RepoSpec, ref model.AttachmentContainerRef, att model.Attachment, content []byte, force bool) error {
	repo, err := repos.Get(spec)
	if err != nil {
		return err
	}

	sanitizedAttachmentName := strings.ReplaceAll(filepath.ToSlash(filepath.Clean(att.Name)), "/", "-")
	sanitizedAtt := model.Attachment{Name: sanitizedAttachmentName, MediaType: att.MediaType}
	err = repo.ImportAttachment(ctx, ref, sanitizedAtt, content, force)
	return err
}

func DeleteAttachment(ctx context.Context, spec model.RepoSpec, ref model.AttachmentContainerRef, attachmentName string) error {
	repo, err := repos.Get(spec)
	if err != nil {
		return err
	}

	err = repo.DeleteAttachment(ctx, ref, attachmentName)
	return err
}
func AttachmentFetch(ctx context.Context, spec model.RepoSpec, ref model.AttachmentContainerRef, attachmentName string, concat bool) ([]byte, error) {
	repo, err := repos.Get(spec)
	if err != nil {
		return nil, err
	}

	attFound := false
	att, err := repo.FetchAttachment(ctx, ref, attachmentName)
	if err != nil {
		if concat && errors.Is(err, model.ErrAttachmentNotFound) {
			att = nil
		} else {
			return att, err
		}
	} else {
		attFound = true
	}
	if !concat || ref.Kind() != model.AttachmentContainerKindTMName {
		return att, err
	}

	searchResult, err := repo.List(ctx, &model.Filters{Name: ref.TMName})
	if err != nil {
		return att, err
	}
	for _, e := range searchResult.Entries { // there's supposed to be exactly one entry, actually
		for _, v := range e.Versions {
			_, found := v.FindAttachment(attachmentName)
			if !found {
				continue
			}
			attFound = true
			vAtt, err := repo.FetchAttachment(ctx, model.NewTMIDAttachmentContainerRef(v.TMID), attachmentName)
			if err != nil {
				return att, err
			}
			att = append(att, vAtt...)
		}
	}
	if !attFound {
		return nil, model.ErrAttachmentNotFound
	}
	return att, nil
}
