package repos

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	subRepo string
}

func (t *TmcRepo) CanonicalRoot() string {
	return t.root
}

func NewTmcRepo(config map[string]any, spec model.RepoSpec) (*TmcRepo, error) {
	base, err := newBaseHttpRepo(config, spec)
	if err != nil {
		return nil, err
	}
	sr, _ := config[keySubRepo].(string)
	r := &TmcRepo{baseHttpRepo: base, subRepo: sr}
	return r, nil
}

func (t *TmcRepo) FetchAttachment(ctx context.Context, container model.AttachmentContainerRef, attachmentName string) ([]byte, error) {
	reqUrl := t.parsedRoot.JoinPath("thing-models", getContainerPath(container), model.AttachmentsDir, attachmentName)
	t.addRepoParam(reqUrl)
	return t.fetchAttachment(ctx, reqUrl.String())
}

func (t *TmcRepo) addRepoParam(u *url.URL) {
	if t.subRepo == "" {
		return
	}
	vals := u.Query()
	vals.Set("repo", t.subRepo)
	u.RawQuery = vals.Encode()
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

func (t *TmcRepo) DeleteAttachment(ctx context.Context, container model.AttachmentContainerRef, attachmentName string) error {
	reqUrl := t.parsedRoot.JoinPath("thing-models", getContainerPath(container), model.AttachmentsDir, attachmentName)
	t.addRepoParam(reqUrl)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqUrl.String(), nil)
	if err != nil {
		return err
	}
	resp, err := t.doHttp(req)
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
		return model.NewErrNotFound(code)
	case http.StatusUnauthorized, http.StatusInternalServerError:
		return newErrorFromResponse(b)
	default:
		return errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}

}

func (t *TmcRepo) GetTMMetadata(ctx context.Context, tmID string) ([]model.FoundVersion, error) {
	reqUrl := t.parsedRoot.JoinPath("inventory", tmID)
	t.addRepoParam(reqUrl)
	resp, err := t.doGet(ctx, reqUrl.String())
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		var r server.InventoryEntryVersionsResponse
		err = json.Unmarshal(b, &r)
		if err != nil {
			return nil, err
		}
		mapper := model.NewInventoryResponseToSearchResultMapper(t.Spec().ToFoundSource(), tmcLinksMapper)
		versions := mapper.ToFoundVersions(r.Data)
		return versions, nil
	case http.StatusNotFound:
		var e server.ErrorResponse
		err := json.Unmarshal(b, &e)
		code := ""
		if err == nil && e.Code != nil {
			code = *e.Code
		}
		return nil, model.NewErrNotFound(code)
	case http.StatusBadRequest:
		return nil, model.ErrInvalidIdOrName
	case http.StatusInternalServerError, http.StatusUnauthorized:
		return nil, newErrorFromResponse(b)
	default:
		return nil, errors.New(fmt.Sprintf("received unexpected HTTP response from remote server: %s", resp.Status))
	}
}

func (t *TmcRepo) ImportAttachment(ctx context.Context, container model.AttachmentContainerRef, attachment model.Attachment, content []byte, force bool) error {
	reqUrl := t.parsedRoot.JoinPath("thing-models", getContainerPath(container), model.AttachmentsDir, attachment.Name)
	vals := reqUrl.Query()
	if force {
		vals["force"] = []string{"true"}
	}
	reqUrl.RawQuery = vals.Encode()
	t.addRepoParam(reqUrl)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqUrl.String(), bytes.NewBuffer(content))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", attachment.MediaType)
	resp, err := t.doHttp(req)
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
		return model.NewErrNotFound(code)
	case http.StatusBadRequest:
		return model.ErrInvalidIdOrName
	case http.StatusConflict:
		return ErrAttachmentExists
	case http.StatusUnauthorized, http.StatusInternalServerError:
		return newErrorFromResponse(b)
	default:
		return errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}
}

func (t *TmcRepo) Import(ctx context.Context, id model.TMID, raw []byte, opts ImportOptions) (ImportResult, error) {
	reqUrl := t.parsedRoot.JoinPath("thing-models")
	t.addRepoParam(reqUrl)
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
		return ImportResultFromError(err)
	}
	req.Header.Add(headerContentType, mimeJSON)
	resp, err := t.doHttp(req)
	if err != nil {
		return ImportResultFromError(err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		err := fmt.Errorf("could not read response body: %w", err)
		utils.GetLogger(ctx, "TmcRepo.Import").Warn(err.Error())
		return ImportResultFromError(err)
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		var res server.ImportThingModelResponse
		err = json.Unmarshal(b, &res)
		if err != nil {
			err := fmt.Errorf("could not unmarshal response from remote tmc: %w", err)
			utils.GetLogger(ctx, "TmcRepo.Import").Warn(err.Error())
			return ImportResultFromError(err)
		}
		msg := ""
		if res.Data.Message != nil {
			msg = *res.Data.Message
		}
		if res.Data.Code != nil && *res.Data.Code != "" {
			cErr, err := ParseErrTMIDConflict(*res.Data.Code)
			if err != nil {
				err := fmt.Errorf("failed to parse returned conflict error code %s: %w", *res.Data.Code, err)
				utils.GetLogger(ctx, "TmcRepo.Import").Warn(err.Error())
				return ImportResultFromError(err)
			}
			return ImportResult{Type: ImportResultWarning, TmID: res.Data.TmID, Message: msg, Err: cErr}, nil
		}
		return ImportResult{Type: ImportResultOK, TmID: res.Data.TmID, Message: msg}, nil
	case http.StatusConflict, http.StatusInternalServerError, http.StatusUnauthorized, http.StatusBadRequest:
		var e server.ErrorResponse
		err = json.Unmarshal(b, &e)
		if err != nil {
			err := fmt.Errorf("could not unmarshal error response from remote tmc: %w", err)
			utils.GetLogger(ctx, "TmcRepo.Import").Warn(err.Error())
			return ImportResultFromError(err)
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
				err := fmt.Errorf("failed to parse returned conflict error code %s: %w", eCode, err)
				utils.GetLogger(ctx, "TmcRepo.Import").Warn(err.Error())
				return ImportResultFromError(err)
			}
			return ImportResultFromError(cErr)
		case http.StatusInternalServerError, http.StatusUnauthorized, http.StatusBadRequest:
			err := fmt.Errorf("received error response from remote tmc server: %v, %s", resp.Status, detail)
			utils.GetLogger(ctx, "TmcRepo.Import").Debug(err.Error())
			return ImportResultFromError(err)
		default:
			panic(fmt.Errorf("unhandled response status code: %v", resp.StatusCode))
		}
	default:
		err := fmt.Errorf("received unexpected HTTP response from remote tmc server: %v", resp.Status)
		utils.GetLogger(ctx, "TmcRepo.Import").Error(err.Error())
		return ImportResultFromError(err)
	}
}
func (t *TmcRepo) Delete(ctx context.Context, id string) error {
	reqUrl := t.parsedRoot.JoinPath("thing-models", id)
	vals := url.Values{
		"force": []string{"true"},
	}
	reqUrl.RawQuery = vals.Encode()
	t.addRepoParam(reqUrl)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqUrl.String(), nil)
	if err != nil {
		return err
	}
	resp, err := t.doHttp(req)
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
		return model.ErrTMNotFound
	case http.StatusInternalServerError, http.StatusUnauthorized:
		err := newErrorFromResponse(b)
		utils.GetLogger(ctx, "TmcRepo.Delete").Error(err.Error())
		return err
	default:
		err := fmt.Errorf("received unexpected HTTP response from remote TM catalog: %s", resp.Status)
		utils.GetLogger(ctx, "TmcRepo.Delete").Error(err.Error())
		return err
	}
}

func (t *TmcRepo) Spec() model.RepoSpec {
	return t.spec
}
func (t *TmcRepo) Fetch(ctx context.Context, id string) (string, []byte, error) {
	reqUrl := t.parsedRoot.JoinPath("thing-models", id)
	t.addRepoParam(reqUrl)
	return t.fetchTM(ctx, reqUrl.String())
}

func (t *TmcRepo) Index(context.Context, ...string) error {
	return nil // ignore request to update index as index updates are presumed to be handled by the underlying repo
}

func (t *TmcRepo) CheckIntegrity(ctx context.Context, filter model.ResourceFilter) (results []model.CheckResult, err error) {
	return nil, nil
}

func (t *TmcRepo) List(ctx context.Context, search *model.Filters) (model.SearchResult, error) {
	reqUrl := t.parsedRoot.JoinPath("inventory")
	t.addRepoParam(reqUrl)

	single := false
	if search != nil && search.Name != "" && search.Options.NameFilterType == model.FullMatch {
		single = true
		reqUrl = reqUrl.JoinPath(tmNamePath, url.PathEscape(search.Name))
	} else {
		addFilters(reqUrl, search)
	}

	resp, err := t.doGet(ctx, reqUrl.String())
	if err != nil {
		return model.SearchResult{}, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.SearchResult{}, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		mapper := model.NewInventoryResponseToSearchResultMapper(t.Spec().ToFoundSource(), tmcLinksMapper) // fixme: should use a different mapper of spec to found source
		if single {
			var tm server.InventoryEntryResponse
			err = json.Unmarshal(data, &tm)
			if err != nil {
				return model.SearchResult{}, err
			}
			return model.SearchResult{
				Entries: mapper.ToFoundEntries(tm.Data),
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

func addFilters(u *url.URL, search *model.Filters) {
	if search == nil {
		return
	}
	if search.Name != "" {
		vals := u.Query()
		vals.Set("filter.name", search.Name)
		u.RawQuery = vals.Encode()
	}
	appendQueryArray(u, "filter.author", search.Author)
	appendQueryArray(u, "filter.manufacturer", search.Manufacturer)
	appendQueryArray(u, "filter.mpn", search.Mpn)
	appendQueryArray(u, "filter.protocol", search.Protocol)
}

func appendQueryArray(u *url.URL, key string, values []string) {
	q := u.Query()
	vals := strings.Join(values, ",")
	if vals != "" {
		q.Set(key, vals)
		u.RawQuery = q.Encode()
	}
}

func (t *TmcRepo) Versions(ctx context.Context, name string) ([]model.FoundVersion, error) {
	log := utils.GetLogger(ctx, "TmcRepo")
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		log.Error("cannot show versions for empty TM name.")
		return nil, errors.New("cannot show versions for empty TM name")
	}
	reqUrl := t.parsedRoot.JoinPath("inventory", tmNamePath, url.PathEscape(name))
	t.addRepoParam(reqUrl)
	resp, err := t.doGet(ctx, reqUrl.String())
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
		var versions []server.InventoryEntryVersion
		for _, e := range vResp.Data {
			versions = append(versions, e.Versions...)
		}
		if len(versions) == 0 {
			log.Error(fmt.Sprintf("No thing models found for TM name: %s", name))
			return nil, model.ErrTMNameNotFound
		}

		mapper := model.NewInventoryResponseToSearchResultMapper(t.Spec().ToFoundSource(), tmcLinksMapper)
		return mapper.ToFoundVersions(versions), nil
	case http.StatusNotFound:
		return nil, model.ErrTMNameNotFound
	case http.StatusInternalServerError, http.StatusUnauthorized, http.StatusBadRequest:
		return nil, newErrorFromResponse(data)
	default:
		return nil, errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}

}

func (t *TmcRepo) ListCompletions(ctx context.Context, kind string, args []string, toComplete string) ([]string, error) {
	u := t.parsedRoot.JoinPath(".completions")
	vals := u.Query()
	vals.Set("kind", kind)
	for _, a := range args {
		vals.Add("args", a)
	}
	vals.Set("toComplete", toComplete)
	u.RawQuery = vals.Encode()

	resp, err := t.doGet(ctx, u.String())
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

func (t *TmcRepo) GetSubRepos(ctx context.Context) ([]model.RepoDescription, error) {
	u := t.parsedRoot.JoinPath("repos")
	resp, err := t.doGet(ctx, u.String())
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var vResp server.ReposResponse
		err = json.Unmarshal(data, &vResp)
		if err != nil {
			return nil, err
		}
		var ds []model.RepoDescription
		for _, d := range vResp.Data {
			descr := ""
			if d.Description != nil {
				descr = *d.Description
			}
			ds = append(ds, model.RepoDescription{
				Name:        d.Name,
				Description: descr,
			})
		}
		return ds, nil
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusUnauthorized:
		return nil, newErrorFromResponse(data)
	default:
		return nil, errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}
}

func createTmcRepoConfig(bytes []byte) (ConfigMap, error) {
	rc, err := AsRepoConfig(bytes)
	if err != nil {
		return nil, err
	}
	if rType, found := utils.JsGetString(rc, KeyRepoType); found {
		if rType != RepoTypeTmc {
			return nil, fmt.Errorf("invalid json config. type must be \"tmc\" or absent")
		}
	}
	rc[KeyRepoType] = RepoTypeTmc
	_, found := utils.JsGetString(rc, KeyRepoLoc)
	if !found {
		return nil, fmt.Errorf("invalid json config. must have string \"loc\"")
	}
	return rc, nil
}
