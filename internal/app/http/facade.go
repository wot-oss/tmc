package http

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"sort"
)

func listToc(filter *FilterParams, search *SearchParams) (*model.Toc, error) {
	remote, err := remotes.Get("")
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

func listTocAuthors(toc *model.Toc) []string {
	authors := []string{}
	check := map[string]bool{}
	for _, v := range toc.Contents {
		if _, ok := check[v.Author.Name]; !ok {
			check[v.Author.Name] = true
			authors = append(authors, v.Author.Name)
		}
	}
	sort.Strings(authors)
	return authors
}

func listTocManufacturers(toc *model.Toc) []string {
	mans := []string{}
	check := map[string]bool{}
	for _, v := range toc.Contents {
		if _, ok := check[v.Manufacturer.Name]; !ok {
			check[v.Manufacturer.Name] = true
			mans = append(mans, v.Manufacturer.Name)
		}
	}
	sort.Strings(mans)
	return mans
}

func listTocMpns(toc *model.Toc) []string {
	mpns := []string{}
	check := map[string]bool{}
	for _, v := range toc.Contents {
		if _, ok := check[v.Mpn]; !ok {
			check[v.Mpn] = true
			mpns = append(mpns, v.Mpn)
		}
	}
	sort.Strings(mpns)
	return mpns
}

func findTocEntry(id string) (*model.TocThing, error) {
	//todo: check if iventoryId is valid format
	toc, err := listToc(nil, nil)
	if err != nil {
		return nil, err
	}

	tocEntry, ok := toc.Contents[id]
	if !ok {
		return nil, NewNotFoundError(nil, "Inventory with Id %s not found", id)
	}
	return &tocEntry, nil
}

func fetchThingModel(tmId string) ([]byte, error) {
	remote, err := remotes.Get("")
	if err != nil {
		return nil, err
	}

	mTmId, err := model.ParseTMID(tmId, false)
	if err == model.ErrInvalidId {
		return nil, NewBadRequestError(err, "Invalid parameter: %s", tmId)
	} else if err != nil {
		return nil, err
	}

	data, err := remote.Fetch(mTmId)
	if err != nil && err.Error() == "file does not exist" {
		return nil, NewNotFoundError(err, "File does not exists")
	} else if err != nil {
		return nil, err
	}
	return data, nil
}

func pushThingModel(file []byte) (tmid *model.TMID, err error) {
	remote, err := remotes.Get("")
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
