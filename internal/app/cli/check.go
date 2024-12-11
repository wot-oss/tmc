package cli

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/commands/validate"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

var (
	errCheckFailed = errors.New("integrity check failed")
	errNotARepo    = errors.New("not a TMC repository")
)

func CheckIntegrity(ctx context.Context, spec model.RepoSpec, args []string, format string) error {
	if !IsValidOutputFormat(format) {
		Stderrf("%v", ErrInvalidOutputFormat)
		return ErrInvalidOutputFormat
	}

	repo, err := repos.Get(spec)
	if err != nil {
		Stderrf("could not initialize a repo instance for %v: %v. check config", spec, err)
		return err
	}

	resFilter := resourceFilterFromArgs(args)
	totalRes, err := checkIndexedResourcesAreValid(ctx, repo, resFilter)
	if errors.Is(err, errNotARepo) {
		Stderrf("(%s) is not a TMC repository\n", spec)
		return nil
	}
	results, iErr := repo.CheckIntegrity(ctx, resFilter)
	if err == nil {
		err = iErr
	}
	totalRes = append(totalRes, results...)
	var errRes []model.CheckResult
	for _, res := range totalRes {
		if res.Typ != model.CheckOK {
			errRes = append(errRes, res)
		}
		if err == nil && res.Typ == model.CheckErr {
			err = errCheckFailed
		}
	}

	switch format {
	case OutputFormatJSON:
		printJSON(totalRes)
	case OutputFormatPlain:
		for _, res := range errRes {
			fmt.Println(res.String())
		}
	}

	if err != nil && !errors.Is(err, errCheckFailed) {
		Stderrf("%v", err)
	}
	return err
}

func resourceFilterFromArgs(args []string) model.ResourceFilter {
	if len(args) == 0 {
		return func(s string) bool {
			return true
		}
	}
	slices.Sort(args)
	return func(s string) bool {
		_, found := slices.BinarySearch(args, s)
		return found
	}

}

func checkIndexedResourcesAreValid(ctx context.Context, repo repos.Repo, filter model.ResourceFilter) ([]model.CheckResult, error) {
	var results []model.CheckResult
	list, err := repo.List(ctx, nil)
	if err != nil {
		if errors.Is(err, repos.ErrNoIndex) {
			return nil, errNotARepo
		}
		return nil, err
	}
	for _, entry := range list.Entries {
		rs := checkAttachments(ctx, repo, model.NewTMNameAttachmentContainerRef(entry.Name), entry.Attachments, filter)
		results = append(results, rs...)
		for _, version := range entry.Versions {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			if filter(version.TMID) {
				tr := checkThingModel(ctx, repo, version.TMID)
				results = append(results, tr)
			}
			rs = checkAttachments(ctx, repo, model.NewTMIDAttachmentContainerRef(version.TMID), version.Attachments, filter)
			results = append(results, rs...)
		}
	}
	return results, nil
}

func checkAttachments(ctx context.Context, repo repos.Repo, ref model.AttachmentContainerRef, attachments []model.Attachment, filter model.ResourceFilter) []model.CheckResult {
	var results []model.CheckResult
	relDir, _ := model.RelAttachmentsDir(ref) // no error expected here, because ref comes from index
	for _, attachment := range attachments {
		resourceName := path.Join(relDir, attachment.Name)
		if !filter(resourceName) {
			continue
		}
		_, err := repo.FetchAttachment(ctx, ref, attachment.Name)
		dir, _ := model.RelAttachmentsDir(ref)
		attResourceName := fmt.Sprintf("%s/%s", dir, attachment.Name)
		if err != nil {
			res := model.CheckResult{model.CheckErr, attResourceName, err.Error()}
			results = append(results, res)
		} else {
			res := model.CheckResult{model.CheckOK, attResourceName, ""}
			results = append(results, res)
		}
	}
	return results
}

func checkThingModel(ctx context.Context, repo repos.Repo, tmid string) model.CheckResult {
	id, raw, err := repo.Fetch(ctx, tmid)
	if err != nil {
		return model.CheckResult{Typ: model.CheckErr, ResourceName: tmid, Message: fmt.Sprintf("could not fetch the TM file to verify integrity: %s", err.Error())}
	}
	tm, err := validate.ValidateThingModel(raw)
	if err != nil {
		return model.CheckResult{Typ: model.CheckErr, ResourceName: tmid, Message: fmt.Sprintf("invalid TM content: %s", err.Error())}
	}
	if tm.ID == "" {
		return model.CheckResult{Typ: model.CheckErr, ResourceName: tmid, Message: "TM id is missing in the file"}
	}
	idInFile, err := model.ParseTMID(tm.ID)
	if err != nil {
		return model.CheckResult{Typ: model.CheckErr, ResourceName: tmid, Message: "TM id in the file is invalid"}
	}
	if tm.ID != tmid || id != tmid {
		err = errors.New("TM id does not match the file location")
		return model.CheckResult{Typ: model.CheckErr, ResourceName: tmid, Message: err.Error()}
	}
	hashStr, _, _ := commands.CalculateFileDigest(raw) // ignore the error, because the file has been validated already

	if idInFile.Version.Hash != hashStr {
		return model.CheckResult{Typ: model.CheckErr, ResourceName: tmid, Message: "file content does not match the digest in ID"}
	}

	return model.CheckResult{Typ: model.CheckOK, ResourceName: tmid, Message: fmt.Sprintf("")}
}
