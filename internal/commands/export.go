package commands

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

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

type FileSystemExportTarget struct {
	basePath string
}

type HttpZipExportTarget struct {
	buffer    *bytes.Buffer
	zipWriter *zip.Writer
	mu        sync.Mutex
}

type ZipEntryWriter struct {
	io.Writer
	closer func() error // Function to call when Close is invoked
}

func NewFileSystemExportTarget(basePath string) (*FileSystemExportTarget, error) {
	f, err := os.Stat(basePath)
	if f != nil && !f.IsDir() {
		return nil, fmt.Errorf("output target folder %s is not a folder", basePath)
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error checking output path %s: %w", basePath, err)
	}
	return &FileSystemExportTarget{basePath: basePath}, nil
}

func (fset *FileSystemExportTarget) CreateWriter(ctx context.Context, logicalPath string) (io.WriteCloser, error) {
	finalPath := filepath.Join(fset.basePath, logicalPath)

	dir := filepath.Dir(finalPath)
	if err := os.MkdirAll(dir, 0770); err != nil {
		return nil, fmt.Errorf("could not create output directory %s: %w", dir, err)
	}

	file, err := os.OpenFile(finalPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s for writing: %w", finalPath, err)
	}
	return file, nil
}

func (w *ZipEntryWriter) Close() error {
	if w.closer != nil {
		return w.closer()
	}
	return nil
}

func (zt *HttpZipExportTarget) CreateWriter(ctx context.Context, logicalPath string) (io.WriteCloser, error) {
	zt.mu.Lock()
	defer zt.mu.Unlock()

	header := &zip.FileHeader{
		Name:     logicalPath,
		Method:   zip.Deflate,
		Modified: time.Now(),
	}
	header.Modified = time.Now()

	entryWriter, err := zt.zipWriter.CreateHeader(header)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip entry for %s: %w", logicalPath, err)
	}

	return &ZipEntryWriter{Writer: entryWriter}, nil
}

func (zt *HttpZipExportTarget) Close() error {
	zt.mu.Lock()
	defer zt.mu.Unlock()
	return zt.zipWriter.Close()
}

func (zt *HttpZipExportTarget) Bytes() []byte {
	return zt.buffer.Bytes()
}

func NewHttpZipExportTarget() *HttpZipExportTarget {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	return &HttpZipExportTarget{
		buffer:    buf,
		zipWriter: zw,
	}
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
			attResults, attErr := exportAttachments(ctx, repo, target, model.NewTMNameAttachmentContainerRef(entry.Name), entry.Attachments)
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

			tmResult, tmErr := exportThingModel(ctx, repo, target, version, restoreId)
			results = append(results, tmResult)
			if tmErr != nil && overallErr == nil {
				overallErr = tmErr
			}

			if withAttachments {
				attResults, attErr := exportAttachments(ctx, repo, target, model.NewTMIDAttachmentContainerRef(version.TMID), version.Attachments)
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

func exportThingModel(ctx context.Context, repo model.RepoSpec, target ExportTarget, version model.FoundVersion, restoreId bool) (ExportResult, error) {
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

	return ExportResult{ResourceId: version.TMID, Path: logicalPath, Error: nil}, nil
}

func exportAttachments(ctx context.Context, repo model.RepoSpec, target ExportTarget, ref model.AttachmentContainerRef, attachments []model.Attachment) ([]ExportResult, error) {
	var results []ExportResult
	var currentErr error

	for _, att := range attachments {
		relDir, err := model.RelAttachmentsDir(ref)
		logicalPath := fmt.Sprintf("%s/%s", relDir, att.Name)
		if err != nil {
			results = append(results, ExportResult{ResourceId: logicalPath, Error: fmt.Errorf("failed to get relative directory for attachment %s: %w", att.Name, err)})
			if currentErr == nil {
				currentErr = err
			}
			continue
		}

		bytes, err := AttachmentFetch(ctx, repo, ref, att.Name, false)
		if err != nil {
			results = append(results, ExportResult{ResourceId: logicalPath, Error: fmt.Errorf("failed to fetch attachment %s: %w", att.Name, err)})
			if currentErr == nil {
				currentErr = err
			}
			continue
		}

		writer, err := target.CreateWriter(ctx, logicalPath)
		if err != nil {
			results = append(results, ExportResult{ResourceId: logicalPath, Error: fmt.Errorf("failed to create writer for attachment %s at %s: %w", att.Name, logicalPath, err)})
			if currentErr == nil {
				currentErr = err
			}
			continue
		}
		defer writer.Close()

		_, err = writer.Write(bytes)
		if err != nil {
			results = append(results, ExportResult{ResourceId: logicalPath, Error: fmt.Errorf("failed to write attachment %s to %s: %w", att.Name, logicalPath, err)})
			if currentErr == nil {
				currentErr = err
			}
			continue
		}
		results = append(results, ExportResult{ResourceId: logicalPath, Path: logicalPath})
	}
	return results, currentErr
}
