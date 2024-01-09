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

func listToc(ctx context.Context, search *model.SearchParams) (*model.SearchResult, error) {
	rm, err := getRemoteManager(ctx)
	if err != nil {
		return nil, err
	}

	c := commands.NewListCommand(rm)
	toc, err := c.List("", search)
	if err != nil {
		return nil, err
	}

	return &toc, nil
}

func listTocAuthors(toc *model.SearchResult) []string {
	authors := []string{}
	check := map[string]bool{}
	for _, v := range toc.Entries {
		if _, ok := check[v.Author.Name]; !ok {
			check[v.Author.Name] = true
			authors = append(authors, v.Author.Name)
		}
	}
	sort.Strings(authors)
	return authors
}

func listTocManufacturers(toc *model.SearchResult) []string {
	mans := []string{}
	check := map[string]bool{}
	for _, v := range toc.Entries {
		if _, ok := check[v.Manufacturer.Name]; !ok {
			check[v.Manufacturer.Name] = true
			mans = append(mans, v.Manufacturer.Name)
		}
	}
	sort.Strings(mans)
	return mans
}

func listTocMpns(toc *model.SearchResult) []string {
	mpns := []string{}
	check := map[string]bool{}
	for _, v := range toc.Entries {
		if _, ok := check[v.Mpn]; !ok {
			check[v.Mpn] = true
			mpns = append(mpns, v.Mpn)
		}
	}
	sort.Strings(mpns)
	return mpns
}

func findTocEntry(ctx context.Context, name string) (*model.FoundEntry, error) {
	//todo: check if name is valid format
	toc, err := listToc(ctx, &model.SearchParams{Name: name})
	if err != nil {
		return nil, err
	}
	if len(toc.Entries) != 1 {
		return nil, NewNotFoundError(nil, "Inventory with name %s not found", name)
	}
	return &toc.Entries[0], nil
}

func fetchThingModel(ctx context.Context, tmID string) ([]byte, error) {
	_, err := model.ParseTMID(tmID, true)
	if errors.Is(err, model.ErrInvalidId) {
		return nil, NewBadRequestError(err, "Invalid parameter: %s", tmID)
	} else if err != nil {
		return nil, err
	}

	rm, err := getRemoteManager(ctx)
	if err != nil {
		return nil, err
	}

	_, data, err := commands.NewFetchCommand(rm).FetchByTMID("", tmID)
	if errors.Is(err, commands.ErrTmNotFound) {
		return nil, NewNotFoundError(err, "File does not exist")
	} else if err != nil {
		return nil, err
	}
	return data, nil
}

func pushThingModel(ctx context.Context, file []byte) (string, error) {
	rm, err := getRemoteManager(ctx)
	if err != nil {
		return "", err
	}

	remoteName := ctx.Value(ctxPushRemote).(string)

	remote, err := rm.Get(remoteName)
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

func checkHealth(ctx context.Context) error {
	err := checkHealthLive(ctx)
	if err != nil {
		return err
	}

	err = checkHealthReady(ctx)
	return err
}

func checkHealthLive(ctx context.Context) error {
	return nil
}

func checkHealthReady(ctx context.Context) error {
	rm, err := getRemoteManager(ctx)
	if err != nil {
		return err
	}
	remoteName := ctx.Value(ctxPushRemote).(string)

	_, err = rm.Get(remoteName)
	if err != nil {
		return errors.New("invalid remotes configuration or no default remote found")
	}
	return nil
}

func checkHealthStartup(ctx context.Context) error {
	err := checkHealthReady(ctx)
	return err
}

func getRemoteManager(ctx context.Context) (remotes.RemoteManager, error) {
	rm := ctx.Value(ctxRemoteManager)

	if rm == nil {
		return nil, errors.New("remote manager is unset")
	}

	return rm.(remotes.RemoteManager), nil
}
