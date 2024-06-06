package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/utils"
)

func AttachmentList(ctx context.Context, spec model.RepoSpec, tmNameOrId string) error {
	ref := toAttachmentContainerRef(tmNameOrId)

	var atts []model.Attachment
	var err error
	switch ref.Kind() {
	case model.AttachmentContainerKindTMID:
		var meta *model.FoundVersion
		meta, err = commands.GetTMMetadata(ctx, spec, tmNameOrId)
		atts = meta.Attachments
	case model.AttachmentContainerKindTMName:
		var res model.SearchResult
		var errs []*repos.RepoAccessError
		res, err, errs = commands.List(ctx, spec, &model.SearchParams{Name: tmNameOrId})
		defer printErrs("Errors occurred while listing:", errs)
		if len(res.Entries) != 0 {
			atts = res.Entries[0].Attachments
		}
	}
	if err != nil {
		Stderrf("Could not list attachments for %s: %v", tmNameOrId, err)
		return err
	}
	for _, v := range atts {
		fmt.Println(v.Name)
	}
	return nil
}

func AttachmentPush(ctx context.Context, spec model.RepoSpec, tmNameOrId, filename string) error {
	abs, err := filepath.Abs(filename)
	if err != nil {
		Stderrf("Error expanding file name %s: %v", filename, err)
		return err
	}

	stat, err := os.Stat(abs)
	if err != nil || stat.IsDir() {
		Stderrf("Cannot read file %s: %v", filename, err)
		return err
	}
	_, raw, err := utils.ReadRequiredFile(filename)
	if err != nil {
		Stderrf("Couldn't read file %s: %v", filename, err)
	}
	err = commands.AttachmentPush(ctx, spec, toAttachmentContainerRef(tmNameOrId), filepath.Base(filename), raw)
	if err != nil {
		Stderrf("Failed to put attachment %s to %s: %v", filename, tmNameOrId, err)
	}

	return err
}
func AttachmentDelete(ctx context.Context, spec model.RepoSpec, tmNameOrId, attachmentName string) error {
	err := commands.AttachmentDelete(ctx, spec, toAttachmentContainerRef(tmNameOrId), attachmentName)
	if err != nil {
		Stderrf("Failed to delete attachment %s to %s: %v", attachmentName, tmNameOrId, err)
	}

	return err
}
func AttachmentFetch(ctx context.Context, spec model.RepoSpec, tmNameOrId, attachmentName string) error {
	content, err := commands.AttachmentFetch(ctx, spec, toAttachmentContainerRef(tmNameOrId), attachmentName)
	if err != nil {
		Stderrf("Failed to fetch attachment %s to %s: %v", attachmentName, tmNameOrId, err)
	}

	fmt.Print(string(content))
	return nil
}

func toAttachmentContainerRef(tmNameOrId string) model.AttachmentContainerRef {
	_, err := model.ParseTMID(tmNameOrId)
	if err != nil {
		return model.NewTMNameAttachmentContainerRef(tmNameOrId)
	}
	return model.NewTMIDAttachmentContainerRef(tmNameOrId)
}
