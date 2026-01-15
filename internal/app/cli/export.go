package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
)

type FileSystemExportTarget struct {
	basePath string
}

func Export(ctx context.Context, repo model.RepoSpec, search *model.Filters, outputPath string, restoreId bool, withAttachments bool, format string) error {
	if !IsValidOutputFormat(format) {
		Stderrf("%v", ErrInvalidOutputFormat)
		return ErrInvalidOutputFormat
	}
	if len(outputPath) == 0 {
		Stderrf("requires output target folder --output")
		return errors.New("--output not provided")
	}

	fsTarget, err := NewFileSystemExportTarget(outputPath)
	if err != nil {
		Stderrf("%v", err)
		return err
	}

	cmdResults, cmdErr := commands.ExportThingModels(ctx, repo, search, fsTarget, restoreId, withAttachments)

	var totalRes []OperationResult
	for _, cr := range cmdResults {
		opType := opResultOK
		text := ""
		if cr.Error != nil {
			opType = opResultErr
			text = cr.Error.Error()
		}
		totalRes = append(totalRes, OperationResult{
			Type:       opType,
			ResourceId: cr.ResourceId,
			Text:       text,
		})
	}

	if cmdErr != nil {
		Stderrf("Error during export: %v", cmdErr)
		return cmdErr
	}

	switch format {
	case OutputFormatJSON:
		printJSON(totalRes)
	case OutputFormatPlain:
		for _, res := range totalRes {
			fmt.Println(res)
		}
	}

	return nil
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
