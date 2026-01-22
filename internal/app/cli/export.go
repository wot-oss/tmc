package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

func Export(ctx context.Context, repo model.RepoSpec, search *model.Filters, outputPath string, restoreId bool, withAttachments bool, format string) error {
	if !IsValidOutputFormat(format) {
		Stderrf("%v", ErrInvalidOutputFormat)
		return ErrInvalidOutputFormat
	}
	if len(outputPath) == 0 {
		Stderrf("requires output target folder --output")
		return errors.New("--output not provided")
	}

	fsTarget, err := commands.NewFileSystemExportTarget(outputPath)
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

	switch format {
	case OutputFormatJSON:
		printJSON(totalRes)
	case OutputFormatPlain:
		for _, res := range totalRes {
			fmt.Println(res)
		}
	}

	if cmdErr != nil {
		Stderrf("Error during export: %v", cmdErr)
		return cmdErr
	}

	return nil
}

func exportThingModel(ctx context.Context, outputPath string, version model.FoundVersion, restoreId bool) (OperationResult, error) {
	spec := model.NewSpecFromFoundSource(version.FoundIn)
	id, thing, err, errs := commands.FetchByTMID(ctx, spec, version.TMID, restoreId)
	if err == nil && len(errs) > 0 { // spec cannot be empty, therefore, there can be at most one RepoAccessError
		err = errs[0]
	}
	if err != nil {
		Stderrf("Error fetch %s: %v", version.TMID, err)
		return OperationResult{opResultErr, version.TMID, fmt.Sprintf("(cannot fetch from repo %s)", version.FoundIn)}, err
	}
	thing = utils.ConvertToNativeLineEndings(thing)
	finalOutput := filepath.Join(outputPath, id)
	err = os.MkdirAll(filepath.Dir(finalOutput), 0770)
	if err != nil {
		Stderrf("Could not write ThingModel to file %s: %v", finalOutput, err)
		return OperationResult{opResultErr, version.TMID, fmt.Sprintf("(cannot write to ouput directory %s)", outputPath)}, err
	}
	err = os.WriteFile(finalOutput, thing, 0660)
	if err != nil {
		Stderrf("Could not write ThingModel to file %s: %v", finalOutput, err)
		return OperationResult{opResultErr, version.TMID, fmt.Sprintf("(cannot write to ouput directory %s)", outputPath)}, err
	}
	return OperationResult{opResultOK, version.TMID, ""}, err
}
