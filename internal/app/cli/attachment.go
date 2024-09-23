package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/utils"
)

func AttachmentList(ctx context.Context, spec model.RepoSpec, tmNameOrId string) error {
	ref := toAttachmentContainerRef(tmNameOrId)

	var atts []model.FoundAttachment
	var err error
	switch ref.Kind() {
	case model.AttachmentContainerKindTMID:
		var fvs []model.FoundVersion
		var errs []*repos.RepoAccessError
		fvs, err, errs = commands.GetTMMetadata(ctx, spec, tmNameOrId)
		defer printErrs("Errors occurred while getting TM metadata:", errs)
		for _, m := range fvs {
			for _, a := range m.Attachments {
				atts = append(atts, model.FoundAttachment{
					Attachment: a,
					FoundIn:    m.FoundIn,
				})
			}
		}
	case model.AttachmentContainerKindTMName:
		var res model.SearchResult
		var errs []*repos.RepoAccessError
		res, err, errs = commands.List(ctx, spec, &model.SearchParams{Name: tmNameOrId})
		defer printErrs("Errors occurred while listing:", errs)
		for _, m := range res.Entries {
			for _, a := range m.Attachments {
				atts = append(atts, model.FoundAttachment{
					Attachment: a,
					FoundIn:    m.FoundIn,
				})
			}
		}
	}
	if err != nil {
		Stderrf("Could not list attachments for %s: %v", tmNameOrId, err)
		return err
	}

	printAttachments(atts)
	return nil
}

func printAttachments(atts []model.FoundAttachment) {
	colWidth := columnWidth()
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintf(table, "NAME\tMEDIATYPE\tREPO\n")
	for _, value := range atts {
		name := value.Name
		ct := elideString(fmt.Sprintf("%v", value.MediaType), colWidth)
		repo := elideString(fmt.Sprintf("%v", value.FoundIn), colWidth)
		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\n", name, ct, repo)
	}
	_ = table.Flush()

}

func AttachmentImport(ctx context.Context, spec model.RepoSpec, tmNameOrId, filename, attachmentName, mediaType string, force bool) error {
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
	if attachmentName == "" {
		attachmentName = filepath.Base(filename)
	}
	err = commands.ImportAttachment(ctx, spec, toAttachmentContainerRef(tmNameOrId), model.Attachment{
		Name:      attachmentName,
		MediaType: mediaType,
	}, raw, force)
	if err != nil {
		Stderrf("Failed to put attachment %s to %s: %v", filename, tmNameOrId, err)
	}

	return err
}
func AttachmentDelete(ctx context.Context, spec model.RepoSpec, tmNameOrId, attachmentName string) error {
	err := commands.DeleteAttachment(ctx, spec, toAttachmentContainerRef(tmNameOrId), attachmentName)
	if err != nil {
		Stderrf("Failed to delete attachment %s to %s: %v", attachmentName, tmNameOrId, err)
	}

	return err
}
func AttachmentFetch(ctx context.Context, spec model.RepoSpec, tmNameOrId, attachmentName string, concat bool) error {
	content, err := commands.AttachmentFetch(ctx, spec, toAttachmentContainerRef(tmNameOrId), attachmentName, concat)
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
