package commands

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func ImportAttachment(ctx context.Context, spec model.RepoSpec, ref model.AttachmentContainerRef, att model.Attachment, content []byte, force bool) error {
	repo, err := repos.Get(spec)

	if err != nil {
		return err
	}

	err = CheckAttachmentRefByType(ctx, repo, ref)
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

	err = CheckAttachmentRefByType(ctx, repo, ref)
	if err != nil {
		return err
	}

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

	err = CheckAttachmentRefByType(ctx, repo, ref)
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

func CheckAttachmentRefByType(ctx context.Context, repo repos.Repo, ref model.AttachmentContainerRef) error {
	authors, manufacturers, _, err := repo.Index(ctx)
	if err != nil {
		return err
	}
	switch ref.Kind() {
	case model.AttachmentContainerKindAuthor:
		if !slices.Contains(authors, ref.Author) {
			return errors.New("author not found in repo")
		}
	case model.AttachmentContainerKindManufacturer:
		if !slices.Contains(authors, strings.Split(ref.Manufacturer, "/")[0]) || !slices.Contains(manufacturers, strings.Split(ref.Manufacturer, "/")[1]) {
			return errors.New("manufacturer not found in repo")
		}
	case model.AttachmentContainerKindTMName:
		// no need
	case model.AttachmentContainerKindTMID:
		// no need to check existence of TM ID as it will be checked when trying to import the attachment
	default:
		return errors.New("invalid attachment container type")
	}
	if err != nil {
		return err
	}
	return nil
}
