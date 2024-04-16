package repos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

var ErrNotSupported = errors.New("method not supported")

const RelFileUriPlaceholder = "{{ID}}"

type baseHttpRepo struct {
	root       string
	parsedRoot *url.URL
	spec       model.RepoSpec
	auth       map[string]any
}

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

func newBaseHttpRepo(config map[string]any, spec model.RepoSpec) (baseHttpRepo, error) {
	loc := utils.JsGetString(config, KeyRepoLoc)
	if loc == nil {
		return baseHttpRepo{}, fmt.Errorf("invalid http repo config. loc is either not found or not a string")
	}
	u, err := url.Parse(*loc)
	if err != nil {
		return baseHttpRepo{}, err
	}
	auth := utils.JsGetMap(config, KeyRepoAuth)
	base := baseHttpRepo{
		root:       *loc,
		spec:       spec,
		auth:       auth,
		parsedRoot: u,
	}
	return base, nil
}

func (h *HttpRepo) Push(ctx context.Context, id model.TMID, raw []byte) error {
	return ErrNotSupported
}
func (h *HttpRepo) Delete(ctx context.Context, id string) error {
	return ErrNotSupported
}

func (h *HttpRepo) Fetch(ctx context.Context, id string) (string, []byte, error) {
	reqUrl := h.buildUrl(id)
	return fetchTM(ctx, reqUrl, h.auth)
}

func fetchTM(ctx context.Context, tmUrl string, auth map[string]any) (string, []byte, error) {
	resp, err := doGet(ctx, tmUrl, auth)
	if err != nil {
		return "", nil, err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		value, dataType, _, err := jsonparser.Get(b, "id")
		if err != nil && dataType != jsonparser.NotExist {
			return "", b, err
		}
		switch dataType {
		case jsonparser.String:
			return string(value), b, nil
		default:
			return fmt.Sprintf("%v", value), b, fmt.Errorf("unexpected type of 'id': %v", value)
		}
	case http.StatusNotFound:
		return "", nil, ErrTmNotFound
	case http.StatusInternalServerError, http.StatusBadRequest:
		return "", nil, errors.New(string(b))
	default:
		return "", nil, errors.New(fmt.Sprintf("received unexpected HTTP response from remote server: %s", resp.Status))
	}

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
func (h *HttpRepo) Spec() model.RepoSpec {
	return h.spec
}

func (h *HttpRepo) List(ctx context.Context, search *model.SearchParams) (model.SearchResult, error) {
	reqUrl := h.buildUrl(fmt.Sprintf("%s/%s", RepoConfDir, IndexFilename))
	resp, err := doGet(ctx, reqUrl, h.auth)
	if err != nil {
		return model.SearchResult{}, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.SearchResult{}, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var idx model.Index
		err = json.Unmarshal(data, &idx)
		idx.Filter(search)
		if err != nil {
			return model.SearchResult{}, err
		}
		return model.NewIndexToFoundMapper(h.Spec().ToFoundSource()).ToSearchResult(idx), nil
	default:
		return model.SearchResult{}, errors.New(fmt.Sprintf("received unexpected HTTP response from remote server: %s", resp.Status))
	}
}

func doGet(ctx context.Context, reqUrl string, auth map[string]any) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
	if err != nil {
		return nil, err
	}
	return doHttp(req, auth)
}

func doHttp(req *http.Request, auth map[string]any) (*http.Response, error) {
	if auth != nil {
		bearerToken := utils.JsGetString(auth, "bearer")
		if bearerToken != nil {
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *bearerToken))
		}
	}
	resp, err := http.DefaultClient.Do(req)
	return resp, err
}

func (h *HttpRepo) Versions(ctx context.Context, name string) ([]model.FoundVersion, error) {
	log := slog.Default()
	if len(name) == 0 {
		log.Error("Please specify a repoName to show the TM.")
		return nil, errors.New("please specify a repoName to show the TM")
	}
	name = strings.TrimSpace(name)
	idx, err := h.List(ctx, &model.SearchParams{Name: name})
	if err != nil {
		return nil, err
	}

	if len(idx.Entries) != 1 {
		log.Error(fmt.Sprintf("No thing model found for repoName: %s", name))
		return nil, ErrTmNotFound
	}

	return idx.Entries[0].Versions, nil
}

func (h *HttpRepo) ListCompletions(ctx context.Context, kind string, toComplete string) ([]string, error) {
	switch kind {
	case CompletionKindNames:
		namePrefix, seg := longestPath(toComplete)
		sr, err := h.List(ctx, model.ToSearchParams(nil, nil, nil, &namePrefix, nil,
			&model.SearchOptions{NameFilterType: model.PrefixMatch}))
		if err != nil {
			return nil, err
		}
		var ns []string
		for _, e := range sr.Entries {
			ns = append(ns, e.Name)
		}
		names := namesToCompletions(ns, toComplete, seg+1)
		return names, nil
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

func createHttpRepoConfig(loc string, bytes []byte) (map[string]any, error) {
	if loc != "" {
		return map[string]any{
			KeyRepoType: RepoTypeHttp,
			KeyRepoLoc:  loc,
		}, nil
	} else {
		rc, err := AsRepoConfig(bytes)
		if err != nil {
			return nil, err
		}
		if rType := utils.JsGetString(rc, KeyRepoType); rType != nil {
			if *rType != RepoTypeHttp {
				return nil, fmt.Errorf("invalid json config. type must be \"http\" or absent")
			}
		}
		rc[KeyRepoType] = RepoTypeHttp
		l := utils.JsGetString(rc, KeyRepoLoc)
		if l == nil {
			return nil, fmt.Errorf("invalid json config. must have string \"loc\"")
		}
		rc[KeyRepoLoc] = *l
		return rc, nil
	}
}
