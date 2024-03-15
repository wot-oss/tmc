package repos

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/server"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

const (
	headerContentType = "Content-Type"
	mimeJSON          = "application/json"
)

// TmcRepo implements a Repo TM repository backed by an instance of TM catalog REST API server
type TmcRepo struct {
	baseHttpRepo
}

func NewTmcRepo(config map[string]any, spec model.RepoSpec) (*TmcRepo, error) {
	base, err := newBaseHttpRepo(config, spec)
	if err != nil {
		return nil, err
	}
	r := &TmcRepo{baseHttpRepo: base}
	return r, nil
}

func (t TmcRepo) Push(id model.TMID, raw []byte) error {
	reqUrl := t.parsedRoot.JoinPath("thing-models")
	req, err := http.NewRequest(http.MethodPost, reqUrl.String(), bytes.NewBuffer(raw))
	if err != nil {
		return err
	}
	req.Header.Add(headerContentType, mimeJSON)
	resp, err := doHttp(req, t.auth)
	if err != nil {
		return err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusConflict, http.StatusInternalServerError, http.StatusBadRequest:
		var e server.ErrorResponse
		err = json.Unmarshal(b, &e)
		if err != nil {
			return err
		}
		detail := e.Title
		if e.Detail != nil {
			detail = *e.Detail
		}
		switch resp.StatusCode {
		case http.StatusConflict:
			eCode := ""
			if e.Code != nil {
				eCode = *e.Code
			}
			cErr, err := ParseErrTMIDConflict(eCode)
			if err != nil {
				return err
			}
			return cErr
		case http.StatusInternalServerError, http.StatusBadRequest:
			return errors.New(detail)
		default:
			return errors.New("unexpected status not handled correctly")
		}
	default:
		return errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}
}
func (t TmcRepo) Delete(id string) error {
	reqUrl := t.parsedRoot.JoinPath("thing-models", id)
	vals := url.Values{
		"force": []string{"true"},
	}
	reqUrl.RawQuery = vals.Encode()

	req, err := http.NewRequest(http.MethodDelete, reqUrl.String(), nil)
	if err != nil {
		return err
	}
	resp, err := doHttp(req, t.auth)
	if err != nil {
		return err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusBadRequest:
		// there are two reasons why we could receive a 400 response: invalid 'force' flag or invalid id
		// we're sure that we've passed a valid 'force' flag, so it must be the id
		return model.ErrInvalidId
	case http.StatusNotFound:
		return ErrTmNotFound
	case http.StatusInternalServerError:
		var e server.ErrorResponse
		err = json.Unmarshal(b, &e)
		if err != nil {
			return err
		}
		detail := e.Title
		if e.Detail != nil {
			detail = *e.Detail
		}
		return errors.New(detail)
	default:
		return errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}
}

func (t TmcRepo) Spec() model.RepoSpec {
	return t.spec
}
func (t TmcRepo) Fetch(id string) (string, []byte, error) {
	reqUrl := t.parsedRoot.JoinPath("thing-models", id)
	return fetchTM(reqUrl.String(), t.auth)
}

func (t TmcRepo) Index(...string) error {
	return nil // ignore request to update index as index updates are presumed to be handled by the underlying repo
}

func (t TmcRepo) List(search *model.SearchParams) (model.SearchResult, error) {
	reqUrl := t.parsedRoot.JoinPath("inventory")

	single := false
	if search != nil && search.Name != "" && search.Options.NameFilterType == model.FullMatch {
		single = true
		reqUrl = reqUrl.JoinPath(url.PathEscape(search.Name))
	} else {
		addSearchParams(reqUrl, search)
	}

	resp, err := doGet(reqUrl.String(), t.auth)
	if err != nil {
		return model.SearchResult{}, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.SearchResult{}, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		mapper := model.NewInventoryResponseToSearchResultMapper(t.Spec().ToFoundSource(), tmcLinksMapper)
		if single {
			var tm server.InventoryEntryResponse
			err = json.Unmarshal(data, &tm)
			if err != nil {
				return model.SearchResult{}, err
			}
			return model.SearchResult{
				Entries: []model.FoundEntry{
					mapper.ToFoundEntry(tm.Data),
				},
			}, nil
		} else {
			var inv server.InventoryResponse
			err = json.Unmarshal(data, &inv)
			if err != nil {
				return model.SearchResult{}, err
			}
			return mapper.ToSearchResult(inv), nil
		}
	case http.StatusNotFound:
		return model.SearchResult{}, nil
	case http.StatusBadRequest, http.StatusInternalServerError:
		return model.SearchResult{}, errors.New(string(data))
	default:
		return model.SearchResult{}, errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}
}

func tmcLinksMapper(links server.InventoryEntryVersion) map[string]string {
	c := ""
	if links.Links != nil {
		c = links.Links.Content
	}
	b, a, f := strings.Cut(c, "thing-models/")
	l := a
	if !f {
		l = b
	}
	return map[string]string{
		"content": l,
	}
}

func addSearchParams(u *url.URL, search *model.SearchParams) {
	if search == nil {
		return
	}
	if search.Query != "" {
		vals := u.Query()
		vals.Set("search", search.Query)
		u.RawQuery = vals.Encode()
	}
	if search.Name != "" {
		vals := u.Query()
		vals.Set("filter.name", search.Name)
		u.RawQuery = vals.Encode()
	}
	appendQueryArray(u, "filter.author", search.Author)
	appendQueryArray(u, "filter.manufacturer", search.Manufacturer)
	appendQueryArray(u, "filter.mpn", search.Mpn)
}

func appendQueryArray(u *url.URL, key string, values []string) {
	q := u.Query()
	vals := strings.Join(values, ",")
	if vals != "" {
		q.Set(key, vals)
		u.RawQuery = q.Encode()
	}
}

func (t TmcRepo) Versions(name string) ([]model.FoundVersion, error) {
	log := slog.Default()
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		log.Error("Please specify a repoName to show the TM.")
		return nil, errors.New("please specify a repoName to show the TM")
	}
	reqUrl := t.parsedRoot.JoinPath("inventory", url.PathEscape(name), ".versions")
	resp, err := doGet(reqUrl.String(), t.auth)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var vResp server.InventoryEntryVersionsResponse
		err = json.Unmarshal(data, &vResp)
		if err != nil {
			return nil, err
		}
		if len(vResp.Data) != 1 {
			log.Error(fmt.Sprintf("No thing model found for repoName: %s", name))
			return nil, ErrTmNotFound
		}

		return model.NewInventoryResponseToSearchResultMapper(t.Spec().ToFoundSource(), tmcLinksMapper).
			ToFoundVersions(vResp.Data), nil
	case http.StatusNotFound:
		return nil, ErrTmNotFound
	case http.StatusInternalServerError, http.StatusBadRequest:
		return nil, errors.New(string(data))
	default:
		return nil, errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}

}

func (t TmcRepo) ListCompletions(kind, toComplete string) ([]string, error) {
	u := t.parsedRoot.JoinPath(".completions")
	vals := u.Query()
	vals.Set("kind", kind)
	vals.Set("toComplete", toComplete)
	u.RawQuery = vals.Encode()

	resp, err := doGet(u.String(), t.auth)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var lines []string
		scanner := bufio.NewScanner(bytes.NewBuffer(data))
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		return lines, scanner.Err()
	case http.StatusBadRequest:
		return nil, ErrInvalidCompletionParams
	case http.StatusInternalServerError:
		return nil, errors.New(string(data))
	default:
		return nil, errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}
}

func createTmcRepoConfig(loc string, bytes []byte) (map[string]any, error) {
	if loc != "" {
		return map[string]any{
			KeyRepoType: RepoTypeTmc,
			KeyRepoLoc:  loc,
		}, nil
	} else {
		rc, err := AsRepoConfig(bytes)
		if err != nil {
			return nil, err
		}
		if rType := utils.JsGetString(rc, KeyRepoType); rType != nil {
			if *rType != RepoTypeTmc {
				return nil, fmt.Errorf("invalid json config. type must be \"tmc\" or absent")
			}
		}
		rc[KeyRepoType] = RepoTypeTmc
		l := utils.JsGetString(rc, KeyRepoLoc)
		if l == nil {
			return nil, fmt.Errorf("invalid json config. must have string \"loc\"")
		}
		rc[KeyRepoLoc] = *l
		return rc, nil
	}
}
