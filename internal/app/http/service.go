package http

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

//go:generate mockery --name HandlerService --outpkg mocks --output mocks
type HandlerService interface {
	ListInventory(ctx context.Context, repo string, filters *model.Filters) (*model.SearchResult, error)
	SearchInventory(ctx context.Context, repo, query string) (*model.SearchResult, error)
	ListAuthors(ctx context.Context, filters *model.Filters) ([]string, error)
	ListManufacturers(ctx context.Context, filters *model.Filters) ([]string, error)
	ListMpns(ctx context.Context, filters *model.Filters) ([]string, error)
	FindInventoryEntries(ctx context.Context, repo string, name string) ([]model.FoundEntry, error)
	FetchThingModel(ctx context.Context, repo, tmID string, restoreId bool) ([]byte, error)
	FetchLatestThingModel(ctx context.Context, repo, fetchName string, restoreId bool) ([]byte, error)
	ImportThingModel(ctx context.Context, repo string, file []byte, opts repos.ImportOptions) (repos.ImportResult, error)
	DeleteThingModel(ctx context.Context, repo string, tmID string) error
	ExportCatalog(ctx context.Context, repo string) ([]byte, error)
	CheckHealth(ctx context.Context) error
	CheckHealthLive(ctx context.Context) error
	CheckHealthReady(ctx context.Context) error
	CheckHealthStartup(ctx context.Context) error
	GetCompletions(ctx context.Context, kind string, args []string, toComplete string) ([]string, error)
	GetTMMetadata(ctx context.Context, repo string, tmID string) ([]model.FoundVersion, error)
	GetLatestTMMetadata(ctx context.Context, repo string, fetchName string) (model.FoundVersion, error)
	FetchAttachment(ctx context.Context, repo string, ref model.AttachmentContainerRef, attachmentFileName string, concat bool) ([]byte, error)
	ImportAttachment(ctx context.Context, repo string, ref model.AttachmentContainerRef, attachmentFileName string, content []byte, contentType string, force bool) error
	DeleteAttachment(ctx context.Context, repo string, ref model.AttachmentContainerRef, attachmentFileName string) error
	ListRepos(ctx context.Context) ([]model.RepoDescription, error)
}

type defaultHandlerService struct {
	serveRepo model.RepoSpec
}

func NewDefaultHandlerService(servedRepo model.RepoSpec) (*defaultHandlerService, error) {
	dhs := &defaultHandlerService{
		serveRepo: servedRepo,
	}
	return dhs, nil
}

func (dhs *defaultHandlerService) ListInventory(ctx context.Context, repo string, filters *model.Filters) (*model.SearchResult, error) {
	spec, err := dhs.inferTargetRepo(ctx, repo)
	if err != nil {
		return nil, err
	}
	res, err, errs := commands.List(ctx, spec, filters)
	if err != nil {
		return nil, err
	}

	if len(errs) > 0 {
		return nil, errs[0]
	}

	return &res, nil
}

func (dhs *defaultHandlerService) SearchInventory(ctx context.Context, repo, query string) (*model.SearchResult, error) {
	spec, err := dhs.inferTargetRepo(ctx, repo)
	if err != nil {
		return nil, err
	}
	res, err, errs := commands.Search(ctx, spec, query)
	if err != nil {
		return nil, err
	}

	if len(errs) > 0 {
		return nil, errs[0]
	}

	return &res, nil
}

func (dhs *defaultHandlerService) ListAuthors(ctx context.Context, filters *model.Filters) ([]string, error) {
	authors := []string{}

	res, err := dhs.ListInventory(ctx, "", filters) // fixme: replace empty repo
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

func (dhs *defaultHandlerService) ListManufacturers(ctx context.Context, filters *model.Filters) ([]string, error) {
	mans := []string{}

	res, err := dhs.ListInventory(ctx, "", filters) // fixme: replace empty repo
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

func (dhs *defaultHandlerService) ListMpns(ctx context.Context, filters *model.Filters) ([]string, error) {
	mpns := []string{}

	res, err := dhs.ListInventory(ctx, "", filters) // fixme: replace empty repo
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
	slices.SortFunc(ds, func(a, b model.RepoDescription) int {
		return strings.Compare(a.Name, b.Name)
	})
	return ds, err
}

func (dhs *defaultHandlerService) FindInventoryEntries(ctx context.Context, repo string, name string) ([]model.FoundEntry, error) {
	//todo: check if name is valid format
	res, err := dhs.ListInventory(ctx, repo, &model.Filters{Name: name, Options: model.FilterOptions{NameFilterType: model.FullMatch}})
	if err != nil {
		return nil, err
	}
	if len(res.Entries) == 0 {
		return nil, NewNotFoundError(nil, "Inventory item with name %s not found", name)
	}
	return res.Entries, nil
}

func (dhs *defaultHandlerService) FetchThingModel(ctx context.Context, repo string, tmID string, restoreId bool) ([]byte, error) {
	_, err := model.ParseTMID(tmID)
	if err != nil {
		return nil, err
	}
	spec, err := dhs.inferTargetRepo(ctx, repo)
	if err != nil {
		return nil, err
	}

	_, data, err, _ := commands.FetchByTMID(ctx, spec, tmID, restoreId)
	if err != nil {
		return nil, err
	}
	return data, nil
}
func (dhs *defaultHandlerService) FetchLatestThingModel(ctx context.Context, repo string, fetchName string, restoreId bool) ([]byte, error) {
	spec, err := dhs.inferTargetRepo(ctx, repo)
	if err != nil {
		return nil, err
	}
	fn, err := model.ParseFetchName(fetchName)
	if err != nil {
		return nil, err
	}
	id, foundIn, err, _ := commands.ResolveFetchName(ctx, spec, fn)
	if err != nil {
		return nil, err
	}

	_, data, err, _ := commands.FetchByTMID(ctx, foundIn, id, restoreId)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (dhs *defaultHandlerService) ImportThingModel(ctx context.Context, repoName string, file []byte, opts repos.ImportOptions) (repos.ImportResult, error) {
	spec, err := dhs.inferTargetRepo(ctx, repoName)
	if err != nil {
		return repos.ImportResult{}, err
	}

	repo, err := repos.Get(spec)
	if err != nil {
		return repos.ImportResultFromError(err)
	}
	res, err := commands.NewImportCommand(time.Now).ImportFile(ctx, file, repo, opts)
	if err != nil {
		return res, err
	}
	if res.IsSuccessful() {
		err = repo.Index(ctx, res.TmID)
		if err != nil {
			return repos.ImportResultFromError(err)
		}
	}

	return res, nil
}

func (dhs *defaultHandlerService) DeleteThingModel(ctx context.Context, repo string, tmID string) error {
	spec, err := dhs.inferTargetRepo(ctx, repo)
	if err != nil {
		return err
	}
	err = commands.Delete(ctx, spec, tmID)
	return err
}

func (dhs *defaultHandlerService) ExportCatalog(ctx context.Context, repo string) ([]byte, error) {
	zipTarget := commands.NewHttpZipExportTarget()
	defer func() {
		if err := zipTarget.Close(); err != nil {
			fmt.Printf("Warning: error closing zip writer: %v\n", err)
		}
	}()

	searchFilters := &model.Filters{}
	rs := model.NewRepoSpec(repo)

	_, err := commands.ExportThingModels(ctx, rs, searchFilters, zipTarget, true, true)
	if err != nil {
		return nil, fmt.Errorf("failed to export catalog: %w", err)
	}

	if err := zipTarget.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize zip archive: %w", err)
	}

	return zipTarget.Bytes(), nil
}

func (dhs *defaultHandlerService) GetCompletions(ctx context.Context, kind string, args []string, toComplete string) ([]string, error) {
	u, err := repos.GetUnion(dhs.serveRepo)
	if err != nil {
		return nil, err
	}
	return u.ListCompletions(ctx, kind, args, toComplete), nil
}

func (dhs *defaultHandlerService) GetTMMetadata(ctx context.Context, repo string, tmID string) ([]model.FoundVersion, error) {
	tgt, err := dhs.inferTargetRepo(ctx, repo)
	if err != nil {
		return nil, err
	}
	meta, err, errs := commands.GetTMMetadata(ctx, tgt, tmID)
	if err != nil {
		return nil, err
	}
	if len(errs) > 0 {
		return nil, errs[0]
	}

	return meta, nil
}

func (dhs *defaultHandlerService) GetLatestTMMetadata(ctx context.Context, repo string, fetchName string) (model.FoundVersion, error) {
	fn, err := model.ParseFetchName(fetchName)
	if err != nil {
		return model.FoundVersion{}, err
	}
	spec, err := dhs.inferTargetRepo(ctx, repo)
	if err != nil {
		return model.FoundVersion{}, err
	}
	id, foundIn, err, errs := commands.ResolveFetchName(ctx, spec, fn)
	if err != nil {
		return model.FoundVersion{}, err
	}
	if len(errs) > 0 {
		return model.FoundVersion{}, errs[0]
	}
	metas, err, _ := commands.GetTMMetadata(ctx, foundIn, id)
	if err != nil {
		return model.FoundVersion{}, err
	}
	// because the metadata has been requested from exactly one repo and there was no error,
	// metas length must be exactly one, but it does not hurt to check
	if len(metas) != 1 {
		return model.FoundVersion{}, model.ErrTMNotFound
	}
	return metas[0], err
}

func (dhs *defaultHandlerService) FetchAttachment(ctx context.Context, repo string, ref model.AttachmentContainerRef, attachmentFileName string, concat bool) ([]byte, error) {
	spec, err := dhs.inferTargetRepo(ctx, repo)
	if err != nil {
		return nil, err
	}
	content, err := commands.AttachmentFetch(ctx, spec, ref, attachmentFileName, concat)
	return content, err
}
func (dhs *defaultHandlerService) DeleteAttachment(ctx context.Context, repo string, ref model.AttachmentContainerRef, attachmentFileName string) error {
	spec, err := dhs.inferTargetRepo(ctx, repo)
	if err != nil {
		return err
	}
	err = commands.DeleteAttachment(ctx, spec, ref, attachmentFileName)
	return err
}
func (dhs *defaultHandlerService) ImportAttachment(ctx context.Context, repo string, ref model.AttachmentContainerRef, attachmentFileName string, content []byte, contentType string, force bool) error {
	spec, err := dhs.inferTargetRepo(ctx, repo)
	if err != nil {
		return err
	}
	err = commands.ImportAttachment(ctx, spec, ref, model.Attachment{
		Name:      attachmentFileName,
		MediaType: contentType,
	}, content, force)
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

	_, err := repos.GetUnion(dhs.serveRepo)
	if err != nil {
		return errors.New("invalid repo configuration or named repo not found")
	}
	return nil
}

func (dhs *defaultHandlerService) CheckHealthStartup(ctx context.Context) error {
	err := dhs.CheckHealthReady(ctx)
	return err
}

func (dhs *defaultHandlerService) inferTargetRepo(ctx context.Context, repo string) (model.RepoSpec, error) {
	if repo == "" {
		return dhs.serveRepo, nil
	}
	if dhs.serveRepo.Dir() != "" { // never override a local repo
		return model.RepoSpec{}, repos.ErrRepoNotFound
	}
	if servedRepo := dhs.serveRepo.RepoName(); servedRepo != "" { // if single repo is served, `repo` must be the same or empty [covered above]
		if repo != servedRepo {
			return model.RepoSpec{}, repos.ErrRepoNotFound
		}
		return dhs.serveRepo, nil
	}
	// if serveRepo is EmptySpec, ensure that repo is in the list of allowed repos
	rds, err := dhs.ListRepos(ctx)
	if err != nil {
		return model.RepoSpec{}, err
	}
	for _, rd := range rds {
		if rd.Name == repo {
			return model.NewRepoSpec(repo), nil
		}
	}
	return model.RepoSpec{}, repos.ErrRepoNotFound
}
