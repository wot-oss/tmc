package commands

import (
	"context"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func ListAttachments(ctx context.Context, spec model.RepoSpec, identifier string, ref model.AttachmentContainerRef) ([]model.FoundAttachment, error) {
	var atts []model.FoundAttachment
	var err error
	repo, err := repos.Get(spec)
	if err != nil {
		return nil, err
	}
	err = CheckAttachmentRefByType(ctx, repo, ref)
	if err != nil {
		return nil, err
	}

	switch ref.Kind() {
	case model.AttachmentContainerKindTMID:
		var fvs []model.FoundVersion
		fvs, err, _ = GetTMMetadata(ctx, spec, identifier)
		for _, m := range fvs {
			for _, a := range m.Attachments {
				atts = append(atts, model.FoundAttachment{
					Attachment: a,
					FoundIn:    m.FoundIn,
				})
			}
		}
	case model.AttachmentContainerKindAuthor, model.AttachmentContainerKindManufacturer, model.AttachmentContainerKindTMName:
		var res model.SearchResult
		res, err, _ = List(ctx, spec, &model.Filters{Name: identifier})
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
		return nil, err
	}

	return atts, nil
}
