package commands

import (
	"context"
	"fmt"
	"io"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

type ExportTarget interface {
	CreateWriter(ctx context.Context, logicalPath string) (io.WriteCloser, error)
}

type ExportResult struct {
	ResourceId string
	Path       string
	Error      error
}

func ExportThingModels(ctx context.Context, repo model.RepoSpec, search *model.Filters, target ExportTarget, restoreId bool, withAttachments bool) ([]ExportResult, error) {
	searchResult, err, errs := List(ctx, repo, search)
	if err != nil {
		return nil, err
	}

	var results []ExportResult
	var overallErr error

	for _, entry := range searchResult.Entries {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		if withAttachments {
			attResults, attErr := exportAttachmentsToTarget(ctx, repo, target, model.NewTMNameAttachmentContainerRef(entry.Name), entry.Attachments)
			results = append(results, attResults...)
			if attErr != nil && overallErr == nil {
				overallErr = attErr
			}
		}

		for _, version := range entry.Versions {
			select {
			case <-ctx.Done():
				return results, ctx.Err()
			default:
			}

			tmResult, tmErr := exportSingleThingModelToTarget(ctx, repo, target, version, restoreId)
			results = append(results, tmResult)
			if tmErr != nil && overallErr == nil {
				overallErr = tmErr
			}

			if withAttachments {
				attResults, attErr := exportAttachmentsToTarget(ctx, repo, target, model.NewTMIDAttachmentContainerRef(version.TMID), version.Attachments)
				results = append(results, attResults...)
				if attErr != nil && overallErr == nil {
					overallErr = attErr
				}
			}
		}
	}

	if overallErr == nil && len(errs) > 0 {
		overallErr = fmt.Errorf("errors during listing: %v", errs)
	}

	return results, overallErr
}

func exportSingleThingModelToTarget(ctx context.Context, repo model.RepoSpec, target ExportTarget, version model.FoundVersion, restoreId bool) (ExportResult, error) {
	spec := model.NewSpecFromFoundSource(version.FoundIn)
	id, thingBytes, err, _ := FetchByTMID(ctx, spec, version.TMID, restoreId)
	if err != nil {
		return ExportResult{ResourceId: version.TMID, Error: fmt.Errorf("failed to fetch TM %s: %w", version.TMID, err)}, err
	}
	thingBytes = utils.ConvertToNativeLineEndings(thingBytes)

	logicalPath := id

	writer, err := target.CreateWriter(ctx, logicalPath)
	if err != nil {
		return ExportResult{ResourceId: version.TMID, Error: fmt.Errorf("failed to create writer for TM %s at %s: %w", version.TMID, logicalPath, err)}, err
	}
	defer writer.Close()

	_, err = writer.Write(thingBytes)
	if err != nil {
		return ExportResult{ResourceId: version.TMID, Error: fmt.Errorf("failed to write TM %s to %s: %w", version.TMID, logicalPath, err)}, err
	}

	return ExportResult{ResourceId: version.TMID, Path: logicalPath}, nil
}

func exportAttachmentsToTarget(ctx context.Context, repo model.RepoSpec, target ExportTarget, ref model.AttachmentContainerRef, attachments []model.Attachment) ([]ExportResult, error) {
	var results []ExportResult
	var currentErr error

	for _, att := range attachments {
		relDir, err := model.RelAttachmentsDir(ref)
		if err != nil {
			results = append(results, ExportResult{ResourceId: att.Name, Error: fmt.Errorf("failed to get relative directory for attachment %s: %w", att.Name, err)})
			if currentErr == nil {
				currentErr = err
			}
			continue
		}
		logicalPath := fmt.Sprintf("%s/%s", relDir, att.Name)

		bytes, err := AttachmentFetch(ctx, repo, ref, att.Name, false)
		if err != nil {
			results = append(results, ExportResult{ResourceId: att.Name, Error: fmt.Errorf("failed to fetch attachment %s: %w", att.Name, err)})
			if currentErr == nil {
				currentErr = err
			}
			continue
		}

		writer, err := target.CreateWriter(ctx, logicalPath)
		if err != nil {
			results = append(results, ExportResult{ResourceId: att.Name, Error: fmt.Errorf("failed to create writer for attachment %s at %s: %w", att.Name, logicalPath, err)})
			if currentErr == nil {
				currentErr = err
			}
			continue
		}
		defer writer.Close()

		_, err = writer.Write(bytes)
		if err != nil {
			results = append(results, ExportResult{ResourceId: att.Name, Error: fmt.Errorf("failed to write attachment %s to %s: %w", att.Name, logicalPath, err)})
			if currentErr == nil {
				currentErr = err
			}
			continue
		}
		results = append(results, ExportResult{ResourceId: att.Name, Path: logicalPath})
	}
	return results, currentErr
}
