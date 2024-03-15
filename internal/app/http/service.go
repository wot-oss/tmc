package http

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/repos"
)

//go:generate mockery --name HandlerService --outpkg mocks --output mocks
type HandlerService interface {
	ListInventory(ctx context.Context, search *model.SearchParams) (*model.SearchResult, error)
	ListAuthors(ctx context.Context, search *model.SearchParams) ([]string, error)
	ListManufacturers(ctx context.Context, search *model.SearchParams) ([]string, error)
	ListMpns(ctx context.Context, search *model.SearchParams) ([]string, error)
	FindInventoryEntry(ctx context.Context, name string) (*model.FoundEntry, error)
	FetchThingModel(ctx context.Context, tmID string, restoreId bool) ([]byte, error)
	PushThingModel(ctx context.Context, file []byte) (string, error)
	DeleteThingModel(ctx context.Context, tmID string) error
	CheckHealth(ctx context.Context) error
	CheckHealthLive(ctx context.Context) error
	CheckHealthReady(ctx context.Context) error
	CheckHealthStartup(ctx context.Context) error
	GetCompletions(ctx context.Context, kind, toComplete string) ([]string, error)
}

type defaultHandlerService struct {
	serveRepo model.RepoSpec
	pushRepo  model.RepoSpec
}

func NewDefaultHandlerService(servedRepo model.RepoSpec, pushRepo model.RepoSpec) (*defaultHandlerService, error) {
	dhs := &defaultHandlerService{
		serveRepo: servedRepo,
		pushRepo:  pushRepo,
	}
	return dhs, nil
}

func (dhs *defaultHandlerService) ListInventory(ctx context.Context, search *model.SearchParams) (*model.SearchResult, error) {
	res, err, errs := commands.List(dhs.serveRepo, search)
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
	_, _, err := commands.ParseAsTMIDOrFetchName(tmID)
	if err != nil {
		return nil, err
	}

	_, data, err, _ := commands.NewFetchCommand().FetchByTMIDOrName(dhs.serveRepo, tmID, restoreId)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (dhs *defaultHandlerService) PushThingModel(ctx context.Context, file []byte) (string, error) {
	pushRepo := dhs.pushRepo

	repo, err := repos.Get(pushRepo)
	if err != nil {
		return "", err
	}
	tmID, err := commands.NewPushCommand(time.Now).PushFile(file, repo, "")
	if err != nil {
		return "", err
	}
	err = repo.Index(tmID)
	if err != nil {
		return "", err
	}

	return tmID, nil
}

func (dhs *defaultHandlerService) DeleteThingModel(ctx context.Context, tmID string) error {
	pushRepo := dhs.pushRepo

	err := commands.NewDeleteCommand().Delete(pushRepo, tmID)
	return err
}

func (dhs *defaultHandlerService) GetCompletions(ctx context.Context, kind, toComplete string) ([]string, error) {
	rs, err := repos.GetSpecdOrAll(dhs.serveRepo)
	if err != nil {
		return nil, err
	}
	return rs.ListCompletions(kind, toComplete), nil
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

	pushRepo := dhs.pushRepo

	_, err := repos.Get(pushRepo)
	if err != nil {
		return errors.New("invalid repo configuration or push repo not found")
	}
	return nil
}

func (dhs *defaultHandlerService) CheckHealthStartup(ctx context.Context) error {
	err := dhs.CheckHealthReady(ctx)
	return err
}
