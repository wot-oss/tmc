package repos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

const RelFileUriPlaceholder = "{{ID}}"

// HttpRepo implements a Repo backed by a http server. It does not allow writing to the repository
// and is thus a read-only view
type HttpRepo struct {
	baseHttpRepo
	templatedPath  bool
	templatedQuery bool
}

func NewHttpRepo(config map[string]any, spec model.RepoSpec) (*HttpRepo, error) {
	base, err := newBaseHttpRepo(config, spec)
	if err != nil {
		return nil, err
	}
	h := &HttpRepo{baseHttpRepo: base}
	cpl := strings.Count(base.root, RelFileUriPlaceholder)
	switch cpl {
	case 0:
	// do nothing
	case 1:
		if strings.Contains(base.parsedRoot.RawPath, RelFileUriPlaceholder) || strings.Contains(base.parsedRoot.Path, RelFileUriPlaceholder) {
			h.templatedPath = true
		} else if strings.Contains(base.parsedRoot.RawQuery, RelFileUriPlaceholder) {
			h.templatedQuery = true
		} else {
			return nil, fmt.Errorf("invalid http repo config. %s placeholder in URL %s is only allowed in path or query", RelFileUriPlaceholder, base.root)
		}
	default:
		return nil, fmt.Errorf("invalid http repo config. At most one instance of %s placeholder is allowed in URL %s", RelFileUriPlaceholder, base.root)
	}

	return h, nil
}

func (h *HttpRepo) Import(ctx context.Context, id model.TMID, raw []byte, opts ImportOptions) (ImportResult, error) {
	return ImportResultFromError(ErrNotSupported)
}
func (h *HttpRepo) Delete(ctx context.Context, id string) error {
	return ErrNotSupported
}

func (h *HttpRepo) Fetch(ctx context.Context, id string) (string, []byte, error) {
	reqUrl := h.buildUrl(id)
	return h.fetchTM(ctx, reqUrl)
}

func (h *HttpRepo) buildUrl(fileId string) string {
	if h.templatedPath {
		return strings.Replace(h.root, RelFileUriPlaceholder, url.PathEscape(fileId), 1)
	} else if h.templatedQuery {
		return strings.Replace(h.root, RelFileUriPlaceholder, url.QueryEscape(fileId), 1)
	}
	return h.parsedRoot.JoinPath(fileId).String()
}

func (h *HttpRepo) Index(context.Context, ...string) error {
	return ErrNotSupported
}

func (h *HttpRepo) CheckIntegrity(ctx context.Context, filter model.ResourceFilter) (results []model.CheckResult, err error) {
	return nil, nil
}

func (h *HttpRepo) Spec() model.RepoSpec {
	return h.spec
}

func (h *HttpRepo) List(ctx context.Context, search *model.SearchParams) (model.SearchResult, error) {
	idx, err := h.getIndex(ctx)
	if err != nil {
		return model.SearchResult{}, err
	}
	sr := model.NewIndexToFoundMapper(h.Spec().ToFoundSource()).ToSearchResult(*idx)
	filtered := &sr
	err = filtered.Filter(search)
	if err != nil {
		return model.SearchResult{}, err
	}
	return *filtered, err
}

func (h *HttpRepo) getIndex(ctx context.Context) (*model.Index, error) {
	reqUrl := h.buildUrl(fmt.Sprintf("%s/%s", RepoConfDir, IndexFilename))
	resp, err := h.doGet(ctx, reqUrl)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var idx model.Index
		err = json.Unmarshal(data, &idx)
		return &idx, err
	default:
		return nil, errors.New(fmt.Sprintf("received unexpected HTTP response from remote server: %s", resp.Status))
	}
}

func (h *HttpRepo) Versions(ctx context.Context, name string) ([]model.FoundVersion, error) {
	if len(name) == 0 {
		return nil, errors.New("cannot list versions for empty TM name")
	}
	name = strings.TrimSpace(name)
	idx, err := h.List(ctx, &model.SearchParams{Name: name})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, model.ErrTMNameNotFound)
	}

	if len(idx.Entries) != 1 {
		return nil, fmt.Errorf("%s: %w", name, model.ErrTMNameNotFound)
	}

	return idx.Entries[0].Versions, nil
}

func (h *HttpRepo) GetTMMetadata(ctx context.Context, tmID string) ([]model.FoundVersion, error) {
	idx, err := h.getIndex(ctx)
	if err != nil {
		return nil, err
	}
	_, err = model.ParseTMID(tmID)
	if err != nil {
		return nil, err
	}
	v := idx.FindByTMID(tmID)
	if v == nil {
		return nil, model.ErrTMNotFound
	}
	mapper := model.NewIndexToFoundMapper(h.Spec().ToFoundSource())
	fv := mapper.ToFoundVersion(v)
	return []model.FoundVersion{fv}, nil
}

func (h *HttpRepo) ImportAttachment(ctx context.Context, container model.AttachmentContainerRef, attachment model.Attachment, content []byte, force bool) error {
	return ErrNotSupported
}

func (h *HttpRepo) DeleteAttachment(ctx context.Context, container model.AttachmentContainerRef, attachmentName string) error {
	return ErrNotSupported
}

func (h *HttpRepo) FetchAttachment(ctx context.Context, container model.AttachmentContainerRef, attachmentName string) ([]byte, error) {
	attDir, err := model.RelAttachmentsDir(container)
	if err != nil {
		return nil, err
	}
	reqUrl := h.buildUrl(fmt.Sprintf("%s/%s", attDir, attachmentName))
	return h.fetchAttachment(ctx, reqUrl)
}

func (h *HttpRepo) ListCompletions(ctx context.Context, kind string, args []string, toComplete string) ([]string, error) {
	switch kind {
	case CompletionKindNames:
		namePrefix, seg := longestPath(toComplete)
		sr, err := h.List(ctx, model.ToSearchParams(nil, nil, nil, &namePrefix, nil,
			&model.SearchOptions{NameFilterType: model.PrefixMatch}))
		if err != nil {
			return nil, err
		}
		var names []string
		for _, e := range sr.Entries {
			names = append(names, e.Name)
		}
		comps := namesToCompletions(names, toComplete, seg+1)
		return comps, nil
	case CompletionKindFetchNames:
		if strings.Contains(toComplete, "..") {
			return nil, fmt.Errorf("%w :no completions for name containing '..'", ErrInvalidCompletionParams)
		}

		name, _, _ := strings.Cut(toComplete, ":")
		versions, err := h.Versions(ctx, name)
		if err != nil {
			return nil, err
		}
		var vs []string
		for _, fv := range versions {
			vs = append(vs, fmt.Sprintf("%s:%s", name, fv.Version.Model))
		}
		return vs, nil
	case CompletionKindNamesOrIds:
		namePrefix, seg := longestPath(toComplete)
		sr, err := h.List(ctx, model.ToSearchParams(nil, nil, nil, &namePrefix, nil,
			&model.SearchOptions{NameFilterType: model.PrefixMatch}))
		if err != nil {
			return nil, err
		}
		var names, comps []string
		for _, e := range sr.Entries {
			names = append(names, e.Name)
			if namePrefix == e.Name {
				for _, v := range e.Versions {
					comps = append(comps, v.TMID)
				}
			}
		}
		comps = append(comps, namesToCompletions(names, toComplete, seg+1)...)
		return comps, nil
	case CompletionKindAttachments:
		return getAttachmentCompletions(ctx, args, h)
	default:
		return nil, ErrInvalidCompletionParams
	}
}

func namesToCompletions(names []string, toComplete string, segments int) []string {
	var res []string
	for _, n := range names {
		if strings.HasPrefix(n, toComplete) {
			res = append(res, cutToNSegments(n, segments))
		}
	}
	slices.Sort(res)
	res = slices.Compact(res)
	return res
}

// longestPath returns the longest substring of s consisting of full path segments and the number of path segments
func longestPath(s string) (string, int) {
	lastSlash := strings.LastIndex(s, "/")
	if lastSlash == -1 {
		return "", 0
	}
	return s[0:lastSlash], strings.Count(s, "/")
}

func cutToNSegments(s string, n int) string {
	segments := strings.FieldsFunc(s, func(r rune) bool { return r == '/' })
	if len(segments) > n {
		return strings.Join(segments[0:n], "/") + "/"
	}
	return strings.Join(segments[0:n], "/")
}

func createHttpRepoConfig(bytes []byte) (ConfigMap, error) {
	rc, err := AsRepoConfig(bytes)
	if err != nil {
		return nil, err
	}
	if rType, found := utils.JsGetString(rc, KeyRepoType); found {
		if rType != RepoTypeHttp {
			return nil, fmt.Errorf("invalid json config. type must be \"http\" or absent")
		}
	}
	rc[KeyRepoType] = RepoTypeHttp
	_, found := utils.JsGetString(rc, KeyRepoLoc)
	if !found {
		return nil, fmt.Errorf("invalid json config. must have string \"loc\"")
	}
	return rc, nil
}
