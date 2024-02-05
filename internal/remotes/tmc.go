package remotes

import (
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

// TmcRemote implements a Remote TM repository backed by an instance of TM catalog REST API server
type TmcRemote struct {
	baseHttpRemote
}

func NewTmcRemote(config map[string]any, spec RepoSpec) (*TmcRemote, error) {
	base, err := newBaseHttpRemote(config, spec)
	if err != nil {
		return nil, err
	}
	r := &TmcRemote{baseHttpRemote: base}
	return r, nil
}

func (t TmcRemote) Push(id model.TMID, raw []byte) error {
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
			err := &ErrTMExists{}
			err.FromString(detail)
			return err
		case http.StatusInternalServerError, http.StatusBadRequest:
			return errors.New(detail)
		default:
			return errors.New("unexpected status not handled correctly")
		}
	default:
		return errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}
}

func (t TmcRemote) Spec() RepoSpec {
	return t.spec
}
func (t TmcRemote) Fetch(id string) (string, []byte, error) {
	reqUrl := t.parsedRoot.JoinPath("thing-models", id)
	return fetchTM(reqUrl.String(), t.auth)
}

func (t TmcRemote) UpdateToc(...string) error {
	return nil // ignore request to update toc as toc updates are presumed to be handled by the underlying remote
}

func (t TmcRemote) List(search *model.SearchParams) (model.SearchResult, error) {
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
	appendQueryArray(u, "filter.externalID", search.ExternalID)
}

func appendQueryArray(u *url.URL, key string, values []string) {
	q := u.Query()
	vals := strings.Join(values, ",")
	if vals != "" {
		q.Set(key, vals)
		u.RawQuery = q.Encode()
	}
}

func (t TmcRemote) Versions(name string) ([]model.FoundVersion, error) {
	log := slog.Default()
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		log.Error("Please specify a remoteName to show the TM.")
		return nil, errors.New("please specify a remoteName to show the TM")
	}
	reqUrl := t.parsedRoot.JoinPath("inventory", url.PathEscape(name), "versions")
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
			log.Error(fmt.Sprintf("No thing model found for remoteName: %s", name))
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

func toFoundVersions(data []server.InventoryEntryVersion, spec RepoSpec) []model.FoundVersion {
	var res []model.FoundVersion
	for _, v := range data {
		fv := model.FoundVersion{
			TOCVersion: model.TOCVersion{
				Description: v.Description,
				Version: model.Version{
					Model: v.Version.Model,
				},
				Links: map[string]string{
					"content": v.TmID,
				},
				TMID:       v.TmID,
				Digest:     v.Digest,
				TimeStamp:  v.Timestamp,
				ExternalID: v.ExternalID,
			},
			FoundIn: spec.ToFoundSource(),
		}
		res = append(res, fv)
	}
	return res
}

func createTmcRemoteConfig(loc string, bytes []byte) (map[string]any, error) {
	if loc != "" {
		return map[string]any{
			KeyRemoteType: RemoteTypeTmc,
			KeyRemoteLoc:  loc,
		}, nil
	} else {
		rc, err := AsRemoteConfig(bytes)
		if err != nil {
			return nil, err
		}
		if rType := utils.JsGetString(rc, KeyRemoteType); rType != nil {
			if *rType != RemoteTypeTmc {
				return nil, fmt.Errorf("invalid json config. type must be \"tmc\" or absent")
			}
		}
		rc[KeyRemoteType] = RemoteTypeTmc
		l := utils.JsGetString(rc, KeyRemoteLoc)
		if l == nil {
			return nil, fmt.Errorf("invalid json config. must have string \"loc\"")
		}
		rc[KeyRemoteLoc] = *l
		return rc, nil
	}
}
