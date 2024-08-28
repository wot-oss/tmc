package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/commands/validate"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

var errCheckFailed = errors.New("integrity check failed")

func CheckIntegrity(ctx context.Context, spec model.RepoSpec) error {

	repo, err := repos.Get(spec)
	if err != nil {
		Stderrf("could not initialize a repo instance for %v: %v. check config", spec, err)
		return err
	}

	fmt.Printf("Checking integrity of repository (%s) ...\n", spec)

	totalRes, err := checkIndexedResourcesAreValid(ctx, repo)
	results, iErr := repo.CheckIntegrity(ctx)
	if err == nil {
		err = iErr
	}
	totalRes = append(totalRes, results...)
	for _, res := range totalRes {
		if res.Typ != model.CheckOK {
			fmt.Println(res)
		}
		if err == nil && res.Typ == model.CheckErr {
			err = errCheckFailed
		}
	}

	return err
}

func checkIndexedResourcesAreValid(ctx context.Context, repo repos.Repo) ([]model.CheckResult, error) {
	var results []model.CheckResult
	list, err := repo.List(ctx, &model.SearchParams{})
	if err != nil {
		return nil, err
	}
	for _, entry := range list.Entries {
		rs := checkAttachments(ctx, repo, model.NewTMNameAttachmentContainerRef(entry.Name), entry.Attachments)
		results = append(results, rs...)
		for _, version := range entry.Versions {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			tr := checkThingModel(ctx, repo, version.TMID)
			results = append(results, tr)
			rs = checkAttachments(ctx, repo, model.NewTMIDAttachmentContainerRef(version.TMID), version.Attachments)
			results = append(results, rs...)
		}
	}
	return results, nil
}

func checkAttachments(ctx context.Context, repo repos.Repo, ref model.AttachmentContainerRef, attachments []model.Attachment) []model.CheckResult {
	var results []model.CheckResult
	for _, attachment := range attachments {
		_, err := repo.FetchAttachment(ctx, ref, attachment.Name)
		dir, _ := model.RelAttachmentsDir(ref)
		attResourseName := fmt.Sprintf("%s/%s", dir, attachment.Name)
		if err != nil {
			res := model.CheckResult{model.CheckErr, attResourseName, err.Error()}
			results = append(results, res)
		} else {
			res := model.CheckResult{model.CheckOK, attResourseName, "OK"}
			results = append(results, res)
		}
	}
	return results
}

func checkThingModel(ctx context.Context, repo repos.Repo, tmid string) model.CheckResult {
	id, raw, err := repo.Fetch(ctx, tmid)
	if err != nil {
		return model.CheckResult{Typ: model.CheckErr, ResourceName: tmid, Message: err.Error()}
	}
	tm, err := validate.ValidateThingModel(raw)
	if err != nil {
		return model.CheckResult{Typ: model.CheckErr, ResourceName: tmid, Message: fmt.Sprintf("invalid TM content: %s", err.Error())}
	}
	if tm.ID == "" {
		err = errors.New("TM id is missing in the file")
		return model.CheckResult{Typ: model.CheckErr, ResourceName: tmid, Message: err.Error()}
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

	return model.CheckResult{Typ: model.CheckOK, ResourceName: tmid, Message: fmt.Sprintf("OK")}
}
