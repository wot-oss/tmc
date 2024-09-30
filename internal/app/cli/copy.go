package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func Copy(ctx context.Context, repo model.RepoSpec, toRepo model.RepoSpec, search *model.SearchParams, opts repos.ImportOptions) error {
	if repo.RepoName() == toRepo.RepoName() && repo.Dir() == toRepo.Dir() {
		Stderrf("Source repo cannot be the same as target")
		return ErrInvalidArgs
	}
	_, err := repos.Get(repo) // ensure that source repo is unambiguous
	if err != nil {
		Stderrf("Could not initialize a source repo instance for %s: %v\ncheck config", repo, err)
		return err
	}
	target, err := repos.Get(toRepo)
	if err != nil {
		Stderrf("Could not initialize a target repo instance for %s: %v\ncheck config", toRepo, err)
		return err
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

	fmt.Printf("Copying %d ThingModels with %d versions and %d attachments...\n", len(searchResult.Entries), vc, ac)

	var totalRes []operationResult
	var copiedIDs []string
	for _, entry := range searchResult.Entries {
		for _, version := range entry.Versions {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			res, cErr := copyThingModel(ctx, version, target, opts)
			tmExisted := false
			var errExists *repos.ErrTMIDConflict
			if errors.As(cErr, &errExists) { // TM exists in target -> add error result and store the total error (unless ought to ignore), but don't skip copying attachments
				tmExisted = true
				totalRes = append(totalRes, operationResult{opResultErr, version.TMID, fmt.Sprintf("already exists as %s", errExists.ExistingId)})
				if err == nil && !opts.IgnoreExisting {
					err = cErr
				}
			} else {
				if cErr != nil {
					cErr = fmt.Errorf("error copying TM %s: %w", version.TMID, cErr)
					totalRes = append(totalRes, operationResult{opResultErr, version.TMID, fmt.Sprintf("%v", cErr)})
					if err == nil {
						err = cErr
					}
					continue
				}
			}

			if !tmExisted {
				copiedIDs = append(copiedIDs, res.TmID)
				iErr := target.Index(ctx, res.TmID) // need to index the TM to be able to push attachments to it
				if iErr != nil {
					totalRes = append(totalRes, operationResult{opResultErr, res.TmID, "could not update index"})
					continue
				}
			}

			switch res.Type {
			case repos.ImportResultWarning:
				warn := res.Message
				var cErr *repos.ErrTMIDConflict
				if errors.As(res.Err, &cErr) {
					warn = fmt.Sprintf("TM's version and timestamp clash with existing one %s", cErr.ExistingId)
				}
				msg := fmt.Sprintf("copied as %s with warning: %s", res.TmID, warn)
				totalRes = append(totalRes, operationResult{opResultWarn, version.TMID, msg})
			case repos.ImportResultOK:
				totalRes = append(totalRes, operationResult{opResultOK, res.TmID, ""})
			}
			spec := model.NewSpecFromFoundSource(entry.FoundIn)
			aRes, aErr := copyAttachments(ctx, spec, target, model.NewTMIDAttachmentContainerRef(version.TMID), version.Attachments, opts.Force, opts.IgnoreExisting)
			if err == nil && aErr != nil {
				err = aErr
			}
			totalRes = append(totalRes, aRes...)
		}
		spec := model.NewSpecFromFoundSource(entry.Versions[0].FoundIn)
		aRes, aErr := copyAttachments(ctx, spec, target, model.NewTMNameAttachmentContainerRef(entry.Name), entry.Attachments, opts.Force, opts.IgnoreExisting)
		if err == nil && aErr != nil {
			err = aErr
		}
		totalRes = append(totalRes, aRes...)
	}

	if err == nil && len(errs) > 0 {
		err = errs[0]
	}

	if len(copiedIDs) > 0 {
		indexErr := target.Index(ctx, copiedIDs...)
		if indexErr != nil {
			Stderrf("Cannot update index: %v", indexErr)
			return indexErr
		}
	}

	for _, res := range totalRes {
		fmt.Println(res)
	}
	printErrs("Errors occurred while listing TMs for export:", errs)

	return err
}

func copyAttachments(ctx context.Context, spec model.RepoSpec, toRepo repos.Repo, ref model.AttachmentContainerRef, attachments []model.Attachment, force, ignoreExisting bool) ([]operationResult, error) {
	relDir, err := model.RelAttachmentsDir(ref)
	if err != nil {
		return nil, err
	}
	var results []operationResult
	for _, att := range attachments {
		var bytes []byte
		var aErr error
		resName := fmt.Sprintf("%s/%s", relDir, att.Name)
		bytes, aErr = commands.AttachmentFetch(ctx, spec, ref, att.Name)
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
		wErr := toRepo.ImportAttachment(ctx, ref, att, bytes, force)
		if wErr != nil {
			results = append(results, operationResult{
				typ:        opResultErr,
				resourceId: resName,
				text:       fmt.Errorf("could not import attachment %s to %v: %w", att.Name, ref, wErr).Error(),
			})
			doIgnore := ignoreExisting && errors.Is(wErr, repos.ErrAttachmentExists)
			if err == nil && !doIgnore {
				err = wErr
			}
			continue
		}
		results = append(results, operationResult{
			typ:        opResultOK,
			resourceId: resName,
		})
	}
	return results, err
}

func copyThingModel(ctx context.Context, version model.FoundVersion, target repos.Repo, opts repos.ImportOptions) (repos.ImportResult, error) {
	spec := model.NewSpecFromFoundSource(version.FoundIn)
	_, thing, err, errs := commands.FetchByTMID(ctx, spec, version.TMID, false)
	if err == nil && len(errs) > 0 { // spec cannot be empty, therefore, there can be at most one RepoAccessError
		err = errs[0]
	}
	if err != nil {
		Stderrf("Error fetching %s: %v", version.TMID, err)
		e := fmt.Errorf("cannot fetch %s from repo %s: %w", version.TMID, version.FoundIn, err)
		return repos.ImportResultFromError(e)
	}

	res, err := commands.NewImportCommand(time.Now).ImportFile(ctx, thing, target, opts)
	return res, err
}
