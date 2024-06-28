package repos

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/wot-oss/tmc/internal/app/http/server"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

const (
	headerContentType = "Content-Type"
	mimeJSON          = "application/json"
	tmNamePath        = ".tmName"
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

func (t TmcRepo) FetchAttachment(ctx context.Context, container model.AttachmentContainerRef, attachmentName string) ([]byte, error) {
	reqUrl := t.parsedRoot.JoinPath("thing-models", getContainerPath(container), model.AttachmentsDir, attachmentName)
	return fetchAttachment(ctx, reqUrl.String(), t.auth)
}

func getContainerPath(ref model.AttachmentContainerRef) string {
	switch ref.Kind() {
	case model.AttachmentContainerKindTMName:
		return fmt.Sprintf("%s/%s", tmNamePath, ref.TMName)
	case model.AttachmentContainerKindTMID:
		return ref.TMID
	default:
		return ref.String()
	}
}

func fetchAttachment(ctx context.Context, reqUrl string, auth map[string]any) ([]byte, error) {
	resp, err := doGet(ctx, reqUrl, auth)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return b, nil
	case http.StatusNotFound:
		var e server.ErrorResponse
		err := json.Unmarshal(b, &e)
		code := ""
		if err == nil && e.Code != nil {
			code = *e.Code
		}
		return nil, NewErrNotFound(code)
	case http.StatusBadRequest:
		return nil, model.ErrInvalidIdOrName
	case http.StatusInternalServerError, http.StatusUnauthorized:
		return nil, newErrorFromResponse(b)
	default:
		return nil, errors.New(fmt.Sprintf("received unexpected HTTP response from remote server: %s", resp.Status))
	}
}

func newErrorFromResponse(b []byte) error {
	var e server.ErrorResponse
	err := json.Unmarshal(b, &e)
	if err != nil {
		return err
	}
	detail := e.Title
	if e.Detail != nil {
		detail = *e.Detail
	}
	return errors.New(detail)
}

func (t TmcRepo) DeleteAttachment(ctx context.Context, container model.AttachmentContainerRef, attachmentName string) error {
	reqUrl := t.parsedRoot.JoinPath("thing-models", getContainerPath(container), model.AttachmentsDir, attachmentName)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqUrl.String(), nil)
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
		return model.ErrInvalidIdOrName
	case http.StatusNotFound:
		var e server.ErrorResponse
		err := json.Unmarshal(b, &e)
		code := ""
		if err == nil && e.Code != nil {
			code = *e.Code
		}
		return NewErrNotFound(code)
	case http.StatusUnauthorized, http.StatusInternalServerError:
		return newErrorFromResponse(b)
	default:
		return errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}

}

func (t TmcRepo) GetTMMetadata(ctx context.Context, tmID string) (*model.FoundVersion, error) {
	reqUrl := t.parsedRoot.JoinPath("inventory", tmID)
	resp, err := doGet(ctx, reqUrl.String(), t.auth)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		var r server.InventoryEntryVersionResponse
		err = json.Unmarshal(b, &r)
		if err != nil {
			return nil, err
		}
		mapper := model.NewInventoryResponseToSearchResultMapper(t.Spec().ToFoundSource(), tmcLinksMapper)
		version := mapper.ToFoundVersion(r.Data)
		return &version, nil
	case http.StatusNotFound:
		var e server.ErrorResponse
		err := json.Unmarshal(b, &e)
		code := ""
		if err == nil && e.Code != nil {
			code = *e.Code
		}
		return nil, NewErrNotFound(code)
	case http.StatusBadRequest:
		return nil, model.ErrInvalidIdOrName
	case http.StatusInternalServerError, http.StatusUnauthorized:
		return nil, newErrorFromResponse(b)
	default:
		return nil, errors.New(fmt.Sprintf("received unexpected HTTP response from remote server: %s", resp.Status))
	}
}

func (t TmcRepo) PushAttachment(ctx context.Context, container model.AttachmentContainerRef, attachmentName string, content []byte) error {
	reqUrl := t.parsedRoot.JoinPath("thing-models", getContainerPath(container), model.AttachmentsDir, attachmentName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqUrl.String(), bytes.NewBuffer(content))
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
	case http.StatusNotFound:
		var e server.ErrorResponse
		err := json.Unmarshal(b, &e)
		code := ""
		if err == nil && e.Code != nil {
			code = *e.Code
		}
		return NewErrNotFound(code)
	case http.StatusBadRequest:
		return model.ErrInvalidIdOrName
	case http.StatusUnauthorized, http.StatusInternalServerError:
		return newErrorFromResponse(b)
	default:
		return errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}
}

func (t TmcRepo) Import(ctx context.Context, id model.TMID, raw []byte, opts ImportOptions) (ImportResult, error) {
	reqUrl := t.parsedRoot.JoinPath("thing-models")
	vals := url.Values{}
	if opts.Force {
		vals["force"] = []string{"true"}
	}
	if opts.OptPath != "" {
		vals["optPath"] = []string{opts.OptPath}
	}
	reqUrl.RawQuery = vals.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqUrl.String(), bytes.NewBuffer(raw))
	if err != nil {
		return ImportResult{}, err
	}
	req.Header.Add(headerContentType, mimeJSON)
	resp, err := doHttp(req, t.auth)
	if err != nil {
		return ImportResult{}, err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return ImportResult{}, err
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		var res server.ImportThingModelResponse
		err = json.Unmarshal(b, &res)
		if err != nil {
			return ImportResult{}, err
		}
		msg := ""
		if res.Data.Message != nil {
			msg = *res.Data.Message
		}
		if res.Data.Code != nil && *res.Data.Code != "" {
			cErr, err := ParseErrTMIDConflict(*res.Data.Code)
			if err != nil {
				return ImportResult{}, err
			}
			return ImportResult{Type: ImportResultWarning, TmID: res.Data.TmID, Message: msg, Err: cErr}, nil
		}
		return ImportResult{Type: ImportResultOK, TmID: res.Data.TmID, Message: msg}, nil
	case http.StatusConflict, http.StatusInternalServerError, http.StatusUnauthorized, http.StatusBadRequest:
		var e server.ErrorResponse
		err = json.Unmarshal(b, &e)
		if err != nil {
			return ImportResult{}, err
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
				return ImportResult{}, err
			}
			return ImportResult{}, cErr
		case http.StatusInternalServerError, http.StatusUnauthorized, http.StatusBadRequest:
			err := errors.New(detail)
			return ImportResult{}, err
		default:
			err := errors.New("unexpected status not handled correctly")
			return ImportResult{}, err
		}
	default:
		err := errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
		return ImportResult{}, err
	}
}
func (t TmcRepo) Delete(ctx context.Context, id string) error {
	reqUrl := t.parsedRoot.JoinPath("thing-models", id)
	vals := url.Values{
		"force": []string{"true"},
	}
	reqUrl.RawQuery = vals.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqUrl.String(), nil)
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
		return ErrTMNotFound
	case http.StatusInternalServerError, http.StatusUnauthorized:
		return newErrorFromResponse(b)
	default:
		return errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}
}

func (t TmcRepo) Spec() model.RepoSpec {
	return t.spec
}
func (t TmcRepo) Fetch(ctx context.Context, id string) (string, []byte, error) {
	reqUrl := t.parsedRoot.JoinPath("thing-models", id)
	return fetchTM(ctx, reqUrl.String(), t.auth)
}

func (t TmcRepo) Index(context.Context, ...string) error {
	return nil // ignore request to update index as index updates are presumed to be handled by the underlying repo
}

func (t TmcRepo) AnalyzeIndex(context.Context) error {
	return ErrNotSupported
}

func (t TmcRepo) RangeResources(context.Context, model.ResourceFilter, func(model.Resource, error) bool) error {
	return ErrNotSupported
}

func (t TmcRepo) List(ctx context.Context, search *model.SearchParams) (model.SearchResult, error) {
	reqUrl := t.parsedRoot.JoinPath("inventory")

	single := false
	if search != nil && search.Name != "" && search.Options.NameFilterType == model.FullMatch {
		single = true
		reqUrl = reqUrl.JoinPath(tmNamePath, url.PathEscape(search.Name))
	} else {
		addSearchParams(reqUrl, search)
	}

	resp, err := doGet(ctx, reqUrl.String(), t.auth)
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
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusInternalServerError:
		return model.SearchResult{}, newErrorFromResponse(data)
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

func (t TmcRepo) Versions(ctx context.Context, name string) ([]model.FoundVersion, error) {
	log := slog.Default()
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		log.Error("Please specify a repoName to show the TM.")
		return nil, errors.New("please specify a repoName to show the TM")
	}
	reqUrl := t.parsedRoot.JoinPath("inventory", tmNamePath, url.PathEscape(name))
	resp, err := doGet(ctx, reqUrl.String(), t.auth)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var vResp server.InventoryEntryResponse
		err = json.Unmarshal(data, &vResp)
		if err != nil {
			return nil, err
		}
		if len(vResp.Data.Versions) != 1 {
			log.Error(fmt.Sprintf("No thing models found for TM name: %s", name))
			return nil, ErrTMNameNotFound
		}

		return model.NewInventoryResponseToSearchResultMapper(t.Spec().ToFoundSource(), tmcLinksMapper).
			ToFoundVersions(vResp.Data.Versions), nil
	case http.StatusNotFound:
		return nil, ErrTMNameNotFound
	case http.StatusInternalServerError, http.StatusUnauthorized, http.StatusBadRequest:
		return nil, newErrorFromResponse(data)
	default:
		return nil, errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}

}

func (t TmcRepo) ListCompletions(ctx context.Context, kind string, args []string, toComplete string) ([]string, error) {
	u := t.parsedRoot.JoinPath(".completions")
	vals := u.Query()
	vals.Set("kind", kind)
	for _, a := range args {
		vals.Add("args", a)
	}
	vals.Set("toComplete", toComplete)
	u.RawQuery = vals.Encode()

	resp, err := doGet(ctx, u.String(), t.auth)
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
	case http.StatusInternalServerError, http.StatusUnauthorized:
		return nil, newErrorFromResponse(data)
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
