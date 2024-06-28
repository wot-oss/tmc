package http

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

//go:generate mockery --name HandlerService --outpkg mocks --output mocks
type HandlerService interface {
	ListInventory(ctx context.Context, search *model.SearchParams) (*model.SearchResult, error)
	ListAuthors(ctx context.Context, search *model.SearchParams) ([]string, error)
	ListManufacturers(ctx context.Context, search *model.SearchParams) ([]string, error)
	ListMpns(ctx context.Context, search *model.SearchParams) ([]string, error)
	FindInventoryEntry(ctx context.Context, name string) (*model.FoundEntry, error)
	FetchThingModel(ctx context.Context, tmID string, restoreId bool) ([]byte, error)
	FetchLatestThingModel(ctx context.Context, fetchName string, restoreId bool) ([]byte, error)
	ImportThingModel(ctx context.Context, file []byte, opts repos.ImportOptions) (repos.ImportResult, error)
	DeleteThingModel(ctx context.Context, tmID string) error
	CheckHealth(ctx context.Context) error
	CheckHealthLive(ctx context.Context) error
	CheckHealthReady(ctx context.Context) error
	CheckHealthStartup(ctx context.Context) error
	GetCompletions(ctx context.Context, kind string, args []string, toComplete string) ([]string, error)
	GetTMMetadata(ctx context.Context, tmID string) (*model.FoundVersion, error)
	GetLatestTMMetadata(ctx context.Context, fetchName string) (*model.FoundVersion, error)
	FetchAttachment(ctx context.Context, ref model.AttachmentContainerRef, attachmentFileName string) ([]byte, error)
	PushAttachment(ctx context.Context, ref model.AttachmentContainerRef, attachmentFileName string, content []byte) error
	DeleteAttachment(ctx context.Context, ref model.AttachmentContainerRef, attachmentFileName string) error
	ListRepos(ctx context.Context) ([]model.RepoDescription, error)
}

type defaultHandlerService struct {
	serveRepo  model.RepoSpec
	importRepo model.RepoSpec
}

func NewDefaultHandlerService(servedRepo model.RepoSpec, importRepo model.RepoSpec) (*defaultHandlerService, error) {
	dhs := &defaultHandlerService{
		serveRepo:  servedRepo,
		importRepo: importRepo,
	}
	return dhs, nil
}

func (dhs *defaultHandlerService) ListInventory(ctx context.Context, search *model.SearchParams) (*model.SearchResult, error) {
	res, err, errs := commands.List(ctx, dhs.serveRepo, search)
	if err != nil {
		return nil, err
	}

	if len(errs) > 0 {
		return nil, errs[0]
	}

	return &res, nil
}

func (dhs *defaultHandlerService) ListAuthors(ctx context.Context, search *model.SearchParams) ([]string, error) {
	authors := []string{}

	res, err := dhs.ListInventory(ctx, search)
	if err != nil {
		return authors, err
	}

	check := map[string]bool{}
	for _, v := range res.Entries {
		if _, ok := check[v.Author.Name]; !ok {
			check[v.Author.Name] = true
			authors = append(authors, v.Author.Name)
		}
	}
	sort.Strings(authors)
	return authors, nil
}

func (dhs *defaultHandlerService) ListManufacturers(ctx context.Context, search *model.SearchParams) ([]string, error) {
	mans := []string{}

	res, err := dhs.ListInventory(ctx, search)
	if err != nil {
		return mans, err
	}

	check := map[string]bool{}
	for _, v := range res.Entries {
		if _, ok := check[v.Manufacturer.Name]; !ok {
			check[v.Manufacturer.Name] = true
			mans = append(mans, v.Manufacturer.Name)
		}
	}
	sort.Strings(mans)
	return mans, nil
}

func (dhs *defaultHandlerService) ListMpns(ctx context.Context, search *model.SearchParams) ([]string, error) {
	mpns := []string{}

	res, err := dhs.ListInventory(ctx, search)
	if err != nil {
		return mpns, err
	}

	check := map[string]bool{}
	for _, v := range res.Entries {
		if _, ok := check[v.Mpn]; !ok {
			check[v.Mpn] = true
			mpns = append(mpns, v.Mpn)
		}
	}
	sort.Strings(mpns)
	return mpns, nil
}

func (dhs *defaultHandlerService) ListRepos(ctx context.Context) ([]model.RepoDescription, error) {
	ds, err := repos.GetDescriptions(ctx, dhs.serveRepo)
	if err != nil {
		return nil, err
	}
	if len(ds) < 2 { // no need to return a single description as it is unambiguous
		return nil, nil
	}
	return ds, err
}

func (dhs *defaultHandlerService) FindInventoryEntry(ctx context.Context, name string) (*model.FoundEntry, error) {
	//todo: check if name is valid format
	res, err := dhs.ListInventory(ctx, &model.SearchParams{Name: name, Options: model.SearchOptions{NameFilterType: model.FullMatch}})
	if err != nil {
		return nil, err
	}
	if len(res.Entries) != 1 {
		return nil, NewNotFoundError(nil, "Inventory item with name %s not found", name)
	}
	return &res.Entries[0], nil
}

func (dhs *defaultHandlerService) FetchThingModel(ctx context.Context, tmID string, restoreId bool) ([]byte, error) {
	_, err := model.ParseTMID(tmID)
	if err != nil {
		return nil, err
	}

	_, data, err, _ := commands.FetchByTMID(ctx, dhs.serveRepo, tmID, restoreId)
	if err != nil {
		return nil, err
	}
	return data, nil
}
func (dhs *defaultHandlerService) FetchLatestThingModel(ctx context.Context, fetchName string, restoreId bool) ([]byte, error) {
	fn, err := model.ParseFetchName(fetchName)
	if err != nil {
		return nil, err
	}
	id, foundIn, err, _ := commands.ResolveFetchName(ctx, dhs.serveRepo, fn)
	if err != nil {
		return nil, err
	}

	_, data, err, _ := commands.FetchByTMID(ctx, foundIn, id, restoreId)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (dhs *defaultHandlerService) ImportThingModel(ctx context.Context, file []byte, opts repos.ImportOptions) (repos.ImportResult, error) {
	importRepo := dhs.importRepo

	repo, err := repos.Get(importRepo)
	if err != nil {
		return repos.ImportResult{}, err
	}
	res, err := commands.NewImportCommand(time.Now).ImportFile(ctx, file, repo, opts)
	if err != nil {
		return res, err
	}
	if res.IsSuccessful() {
		err = repo.Index(ctx, res.TmID)
		if err != nil {
			return repos.ImportResult{}, err
		}
	}

	return res, nil
}

func (dhs *defaultHandlerService) DeleteThingModel(ctx context.Context, tmID string) error {
	importRepo := dhs.importRepo

	err := commands.Delete(ctx, importRepo, tmID)
	return err
}

func (dhs *defaultHandlerService) GetCompletions(ctx context.Context, kind string, args []string, toComplete string) ([]string, error) {
	rs, err := repos.GetSpecdOrAll(dhs.serveRepo)
	if err != nil {
		return nil, err
	}
	return rs.ListCompletions(ctx, kind, args, toComplete), nil
}

func (dhs *defaultHandlerService) GetTMMetadata(ctx context.Context, tmID string) (*model.FoundVersion, error) {
	// fixme: should it be importRepo, or serveRepo? interesting implications
	meta, err := commands.GetTMMetadata(ctx, dhs.importRepo, tmID)
	return meta, err
}

func (dhs *defaultHandlerService) GetLatestTMMetadata(ctx context.Context, fetchName string) (*model.FoundVersion, error) {
	fn, err := model.ParseFetchName(fetchName)
	if err != nil {
		return nil, err
	}
	id, foundIn, err, _ := commands.ResolveFetchName(ctx, dhs.serveRepo, fn)
	if err != nil {
		return nil, err
	}
	meta, err := commands.GetTMMetadata(ctx, foundIn, id)
	return meta, err
}

func (dhs *defaultHandlerService) FetchAttachment(ctx context.Context, ref model.AttachmentContainerRef, attachmentFileName string) ([]byte, error) {
	content, err := commands.AttachmentFetch(ctx, dhs.importRepo, ref, attachmentFileName)
	return content, err
}
func (dhs *defaultHandlerService) DeleteAttachment(ctx context.Context, ref model.AttachmentContainerRef, attachmentFileName string) error {
	err := commands.AttachmentDelete(ctx, dhs.importRepo, ref, attachmentFileName)
	return err
}
func (dhs *defaultHandlerService) PushAttachment(ctx context.Context, ref model.AttachmentContainerRef, attachmentFileName string, content []byte) error {
	err := commands.AttachmentPush(ctx, dhs.importRepo, ref, attachmentFileName, content)
	return err
}

func (dhs *defaultHandlerService) CheckHealth(ctx context.Context) error {
	err := dhs.CheckHealthLive(ctx)
	if err != nil {
		return err
	}

	err = dhs.CheckHealthReady(ctx)
	return err
}

func (dhs *defaultHandlerService) CheckHealthLive(ctx context.Context) error {
	return nil
}

func (dhs *defaultHandlerService) CheckHealthReady(ctx context.Context) error {

	importRepo := dhs.importRepo

	_, err := repos.Get(importRepo)
	if err != nil {
		return errors.New("invalid repo configuration or import repo not found")
	}
	return nil
}

func (dhs *defaultHandlerService) CheckHealthStartup(ctx context.Context) error {
	err := dhs.CheckHealthReady(ctx)
	return err
}
