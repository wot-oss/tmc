package http

import (
	"sort"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

func listToc(filter *FilterParams, search *SearchParams) (*model.TOC, error) {
	remote, err := remotes.DefaultManager().Get("")
	if err != nil {
		return nil, err
	}

	toc, err := remote.List("")
	if err != nil {
		return nil, err
	}

	if filter != nil {
		Filter(&toc, filter)
	}
	if search != nil {
		Search(&toc, search)
	}

	return &toc, nil
}

func listTocAuthors(toc *model.TOC) []string {
	authors := []string{}
	check := map[string]bool{}
	for _, v := range toc.Data {
		if _, ok := check[v.Author.Name]; !ok {
			check[v.Author.Name] = true
			authors = append(authors, v.Author.Name)
		}
	}
	sort.Strings(authors)
	return authors
}

func listTocManufacturers(toc *model.TOC) []string {
	mans := []string{}
	check := map[string]bool{}
	for _, v := range toc.Data {
		if _, ok := check[v.Manufacturer.Name]; !ok {
			check[v.Manufacturer.Name] = true
			mans = append(mans, v.Manufacturer.Name)
		}
	}
	sort.Strings(mans)
	return mans
}

func listTocMpns(toc *model.TOC) []string {
	mpns := []string{}
	check := map[string]bool{}
	for _, v := range toc.Data {
		if _, ok := check[v.Mpn]; !ok {
			check[v.Mpn] = true
			mpns = append(mpns, v.Mpn)
		}
	}
	sort.Strings(mpns)
	return mpns
}

func findTocEntry(name string) (*model.TOCEntry, error) {
	//todo: check if name is valid format
	toc, err := listToc(nil, nil)
	if err != nil {
		return nil, err
	}

	tocEntry := toc.FindByName(name)
	if tocEntry == nil {
		return nil, NewNotFoundError(nil, "Inventory with name %s not found", name)
	}
	return tocEntry, nil
}

func fetchThingModel(tmID string) ([]byte, error) {
	remote, err := remotes.DefaultManager().Get("")
	if err != nil {
		return nil, err
	}

	mTmID, err := model.ParseTMID(tmID, false)
	if err == model.ErrInvalidId {
		return nil, NewBadRequestError(err, "Invalid parameter: %s", tmID)
	} else if err != nil {
		return nil, err
	}

	data, err := remote.Fetch(mTmID)
	if err != nil && err.Error() == "file does not exist" {
		return nil, NewNotFoundError(err, "File does not exists")
	} else if err != nil {
		return nil, err
	}
	return data, nil
}

func pushThingModel(file []byte) (*model.TMID, error) {
	remote, err := remotes.DefaultManager().Get("")
	if err != nil {
		return nil, err
	}
	tmID, err := commands.PushFile(file, remote, "")
	if err != nil {
		return nil, err
	}
	err = remote.CreateToC()
	if err != nil {
		return nil, err
	}

	return &tmID, nil
}
