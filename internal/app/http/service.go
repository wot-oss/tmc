package http

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

//go:generate mockery --name HandlerService --inpackage
type HandlerService interface {
	ListInventory(ctx context.Context, search *model.SearchParams) (*model.SearchResult, error)
	ListAuthors(ctx context.Context, search *model.SearchParams) ([]string, error)
	ListManufacturers(ctx context.Context, search *model.SearchParams) ([]string, error)
	ListMpns(ctx context.Context, search *model.SearchParams) ([]string, error)
	FindInventoryEntry(ctx context.Context, name string) (*model.FoundEntry, error)
	FetchThingModel(ctx context.Context, tmID string) ([]byte, error)
	PushThingModel(ctx context.Context, file []byte) (string, error)
	CheckHealth(ctx context.Context) error
	CheckHealthLive(ctx context.Context) error
	CheckHealthReady(ctx context.Context) error
	CheckHealthStartup(ctx context.Context) error
}

type defaultHandlerService struct {
	remoteManager remotes.RemoteManager
	pushRemote    remotes.RepoSpec
}

func NewDefaultHandlerService(rm remotes.RemoteManager, pushRemote remotes.RepoSpec) *defaultHandlerService {
	return &defaultHandlerService{
		remoteManager: rm,
		pushRemote:    pushRemote,
	}
}

func (dhs *defaultHandlerService) ListInventory(ctx context.Context, search *model.SearchParams) (*model.SearchResult, error) {
	rm, err := dhs.getRemoteManager()
	if err != nil {
		return nil, err
	}

	c := commands.NewListCommand(rm)
	toc, err := c.List(remotes.EmptySpec, search)
	if err != nil {
		return nil, err
	}

	return &toc, nil
}

func (dhs *defaultHandlerService) ListAuthors(ctx context.Context, search *model.SearchParams) ([]string, error) {
	authors := []string{}

	toc, err := dhs.ListInventory(ctx, search)
	if err != nil {
		return authors, err
	}

	check := map[string]bool{}
	for _, v := range toc.Entries {
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

	toc, err := dhs.ListInventory(ctx, search)
	if err != nil {
		return mans, err
	}

	check := map[string]bool{}
	for _, v := range toc.Entries {
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

	toc, err := dhs.ListInventory(ctx, search)
	if err != nil {
		return mpns, err
	}

	check := map[string]bool{}
	for _, v := range toc.Entries {
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
	toc, err := dhs.ListInventory(ctx, &model.SearchParams{Name: name})
	if err != nil {
		return nil, err
	}
	if len(toc.Entries) != 1 {
		return nil, NewNotFoundError(nil, "Inventory with name %s not found", name)
	}
	return &toc.Entries[0], nil
}

func (dhs *defaultHandlerService) FetchThingModel(ctx context.Context, tmID string) ([]byte, error) {
	_, err := model.ParseTMID(tmID, true)
	if errors.Is(err, model.ErrInvalidId) {
		return nil, NewBadRequestError(err, "Invalid parameter: %s", tmID)
	} else if err != nil {
		return nil, err
	}

	rm, err := dhs.getRemoteManager()
	if err != nil {
		return nil, err
	}

	_, data, err := commands.NewFetchCommand(rm).FetchByTMID(remotes.EmptySpec, tmID)
	if errors.Is(err, commands.ErrTmNotFound) {
		return nil, NewNotFoundError(err, "File does not exist")
	} else if err != nil {
		return nil, err
	}
	return data, nil
}

func (dhs *defaultHandlerService) PushThingModel(ctx context.Context, file []byte) (string, error) {
	rm, err := dhs.getRemoteManager()
	if err != nil {
		return "", err
	}

	remoteSpec := dhs.pushRemote
	if remoteSpec.IsEmpty() {
		return "", errors.New("push remote spec is unset or empty")
	}

	remote, err := rm.Get(remoteSpec)
	if err != nil {
		return "", err
	}
	tmID, err := commands.NewPushCommand(time.Now).PushFile(file, remote, "")
	if err != nil {
		return "", err
	}
	err = remote.CreateToC()
	if err != nil {
		return "", err
	}

	return tmID, nil
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
	rm, err := dhs.getRemoteManager()
	if err != nil {
		return err
	}
	remoteName := dhs.pushRemote

	_, err = rm.Get(remoteName)
	if err != nil {
		return errors.New("invalid remotes configuration or no default remote found")
	}
	return nil
}

func (dhs *defaultHandlerService) CheckHealthStartup(ctx context.Context) error {
	err := dhs.CheckHealthReady(ctx)
	return err
}

func (dhs *defaultHandlerService) getRemoteManager() (remotes.RemoteManager, error) {
	if dhs.remoteManager == nil {
		return nil, errors.New("remote manager is unset")
	}

	return dhs.remoteManager, nil
}
