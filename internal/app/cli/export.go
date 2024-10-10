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

func Export(ctx context.Context, repo model.RepoSpec, search *model.SearchParams, outputPath string, restoreId bool, withAttachments bool) error {
	if len(outputPath) == 0 {
		Stderrf("requires output target folder --output")
		return errors.New("--output not provided")
	}

	f, err := os.Stat(outputPath)
	if f != nil && !f.IsDir() {
		Stderrf("output target folder --output is not a folder")
		return errors.New("output target folder --output is not a folder")
	}

	searchResult, err, errs := commands.List(ctx, repo, search)
	if err != nil {
		Stderrf("Error listing: %v", err)
		return err
	}

	vc := 0
	ac := 0
	for _, m := range searchResult.Entries {
		vc += len(m.Versions)
		ac += len(m.Attachments)
		for _, v := range m.Versions {
			ac += len(v.Attachments)
		}
	}

	if withAttachments {
		fmt.Printf("Exporting %d ThingModels with %d versions and %d attachments...\n", len(searchResult.Entries), vc, ac)
	} else {
		fmt.Printf("Exporting %d ThingModels with %d versions...\n", len(searchResult.Entries), vc)
	}

	var totalRes []operationResult
	for _, entry := range searchResult.Entries {
		if withAttachments {
			spec := model.NewSpecFromFoundSource(entry.FoundIn)
			aRes, aErr := exportAttachments(ctx, spec, outputPath, model.NewTMNameAttachmentContainerRef(entry.Name), entry.Attachments)
			if err == nil && aErr != nil {
				err = aErr
			}
			totalRes = append(totalRes, aRes...)
		}
		for _, version := range entry.Versions {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			res, eErr := exportThingModel(ctx, outputPath, version, restoreId)
			if err == nil && eErr != nil {
				err = eErr
			}
			totalRes = append(totalRes, res)
			if withAttachments {
				spec := model.NewSpecFromFoundSource(entry.Versions[0].FoundIn)
				aRes, aErr := exportAttachments(ctx, spec, outputPath, model.NewTMIDAttachmentContainerRef(version.TMID), version.Attachments)
				if err == nil && aErr != nil {
					err = aErr
				}
				totalRes = append(totalRes, aRes...)
			}

		}
	}

	if err == nil && len(errs) > 0 {
		err = errs[0]
	}

	for _, res := range totalRes {
		fmt.Println(res)
	}
	printErrs("Errors occurred while listing TMs for export:", errs)

	return err
}

func exportAttachments(ctx context.Context, spec model.RepoSpec, outputPath string, ref model.AttachmentContainerRef, attachments []model.Attachment) ([]operationResult, error) {
	relDir, err := model.RelAttachmentsDir(ref)
	if err != nil {
		return nil, err
	}
	attDir := filepath.Join(outputPath, relDir)
	err = os.MkdirAll(attDir, 0770)
	if err != nil {
		Stderrf("could not create output directory %s: %v", attDir, err)
		return nil, err
	}
	var results []operationResult
	for _, att := range attachments {
		var bytes []byte
		var aErr error
		resName := fmt.Sprintf("%s/%s", relDir, att.Name)
		finalOutput := filepath.Join(attDir, att.Name)
		bytes, aErr = commands.AttachmentFetch(ctx, spec, ref, att.Name, false)
		if aErr != nil {
			if err == nil {
				err = aErr
			}
			results = append(results, operationResult{
				typ:        opResultErr,
				resourceId: resName,
				text:       fmt.Errorf("could not fetch attachment %s to %v: %w", att.Name, ref, err).Error(),
			})
			continue
		}
		wErr := os.WriteFile(finalOutput, bytes, 0660)
		if wErr != nil {
			if err == nil {
				err = wErr
			}
			results = append(results, operationResult{
				typ:        opResultErr,
				resourceId: resName,
				text:       fmt.Errorf("could not write attachment %s to %v: %w", att.Name, ref, err).Error(),
			})
			continue
		}
		results = append(results, operationResult{
			typ:        opResultOK,
			resourceId: resName,
		})
	}
	return results, err
}

func exportThingModel(ctx context.Context, outputPath string, version model.FoundVersion, restoreId bool) (operationResult, error) {
	spec := model.NewSpecFromFoundSource(version.FoundIn)
	id, thing, err, errs := commands.FetchByTMID(ctx, spec, version.TMID, restoreId)
	if err == nil && len(errs) > 0 { // spec cannot be empty, therefore, there can be at most one RepoAccessError
		err = errs[0]
	}
	if err != nil {
		Stderrf("Error fetch %s: %v", version.TMID, err)
		return operationResult{opResultErr, version.TMID, fmt.Sprintf("(cannot fetch from repo %s)", version.FoundIn)}, err
	}
	thing = utils.ConvertToNativeLineEndings(thing)

	finalOutput := filepath.Join(outputPath, id)

	err = os.MkdirAll(filepath.Dir(finalOutput), 0770)
	if err != nil {
		Stderrf("Could not write ThingModel to file %s: %v", finalOutput, err)
		return operationResult{opResultErr, version.TMID, fmt.Sprintf("(cannot write to ouput directory %s)", outputPath)}, err
	}

	err = os.WriteFile(finalOutput, thing, 0660)
	if err != nil {
		Stderrf("Could not write ThingModel to file %s: %v", finalOutput, err)
		return operationResult{opResultErr, version.TMID, fmt.Sprintf("(cannot write to ouput directory %s)", outputPath)}, err
	}

	return operationResult{opResultOK, version.TMID, ""}, err
}
