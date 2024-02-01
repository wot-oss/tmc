package remotes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

var ErrNotSupported = errors.New("method not supported")

const RelFileUriPlaceholder = "{{ID}}"

type baseHttpRemote struct {
	root       string
	parsedRoot *url.URL
	spec       RepoSpec
	auth       map[string]any
}

// HttpRemote implements a Remote TM repository backed by a http server. It does not allow pushing to the remote
// and is thus a read-only view
type HttpRemote struct {
	baseHttpRemote
	templatedPath  bool
	templatedQuery bool
}

func NewHttpRemote(config map[string]any, spec RepoSpec) (*HttpRemote, error) {
	base, err := newBaseHttpRemote(config, spec)
	if err != nil {
		return nil, err
	}
	h := &HttpRemote{baseHttpRemote: base}
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
			return nil, fmt.Errorf("invalid http remote config. %s placeholder in URL %s is only allowed in path or query", RelFileUriPlaceholder, base.root)
		}
	default:
		return nil, fmt.Errorf("invalid http remote config. At most one instance of %s placeholder is allowed in URL %s", RelFileUriPlaceholder, base.root)
	}

	return h, nil
}

func newBaseHttpRemote(config map[string]any, spec RepoSpec) (baseHttpRemote, error) {
	loc := utils.JsGetString(config, KeyRemoteLoc)
	if loc == nil {
		return baseHttpRemote{}, fmt.Errorf("invalid http remote config. loc is either not found or not a string")
	}
	u, err := url.Parse(*loc)
	if err != nil {
		return baseHttpRemote{}, err
	}
	auth := utils.JsGetMap(config, KeyRemoteAuth)
	base := baseHttpRemote{
		root:       *loc,
		spec:       spec,
		auth:       auth,
		parsedRoot: u,
	}
	return base, nil
}

func (h *HttpRemote) Push(_ model.TMID, _ []byte) error {
	return ErrNotSupported
}

func (h *HttpRemote) Fetch(id string) (string, []byte, error) {
	reqUrl := h.buildUrl(id)
	return fetchTM(reqUrl, h.auth)
}

func fetchTM(tmUrl string, auth map[string]any) (string, []byte, error) {
	resp, err := doGet(tmUrl, auth)
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
		return "", nil, ErrEntryNotFound
	case http.StatusInternalServerError, http.StatusBadRequest:
		return "", nil, errors.New(string(b))
	default:
		return "", nil, errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}

}

func (h *HttpRemote) buildUrl(fileId string) string {
	if h.templatedPath {
		return strings.Replace(h.root, RelFileUriPlaceholder, url.PathEscape(fileId), 1)
	} else if h.templatedQuery {
		return strings.Replace(h.root, RelFileUriPlaceholder, url.QueryEscape(fileId), 1)
	}
	return h.parsedRoot.JoinPath(fileId).String()
}

func (h *HttpRemote) UpdateToc(...string) error {
	return ErrNotSupported
}
func (h *HttpRemote) Spec() RepoSpec {
	return h.spec
}

func (h *HttpRemote) List(search *model.SearchParams) (model.SearchResult, error) {
	reqUrl := h.buildUrl(TOCFilename)
	resp, err := doGet(reqUrl, h.auth)
	if err != nil {
		return model.SearchResult{}, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.SearchResult{}, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var toc model.TOC
		err = json.Unmarshal(data, &toc)
		toc.Filter(search)
		if err != nil {
			return model.SearchResult{}, err
		}
		return model.NewTOCToFoundMapper(h.Spec().ToFoundSource()).ToSearchResult(toc), nil
	default:
		return model.SearchResult{}, errors.New(fmt.Sprintf("received unexpected HTTP response from remote TM catalog: %s", resp.Status))
	}
}

func doGet(reqUrl string, auth map[string]any) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, reqUrl, nil)
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

func (h *HttpRemote) Versions(name string) ([]model.FoundVersion, error) {
	log := slog.Default()
	if len(name) == 0 {
		log.Error("Please specify a remoteName to show the TM.")
		return nil, errors.New("please specify a remoteName to show the TM")
	}
	name = strings.TrimSpace(name)
	toc, err := h.List(&model.SearchParams{Name: name})
	if err != nil {
		return nil, err
	}

	if len(toc.Entries) != 1 {
		log.Error(fmt.Sprintf("No thing model found for remoteName: %s", name))
		return nil, ErrEntryNotFound
	}

	return toc.Entries[0].Versions, nil
}

func createHttpRemoteConfig(loc string, bytes []byte) (map[string]any, error) {
	if loc != "" {
		return map[string]any{
			KeyRemoteType: RemoteTypeHttp,
			KeyRemoteLoc:  loc,
		}, nil
	} else {
		rc, err := AsRemoteConfig(bytes)
		if err != nil {
			return nil, err
		}
		if rType := utils.JsGetString(rc, KeyRemoteType); rType != nil {
			if *rType != RemoteTypeHttp {
				return nil, fmt.Errorf("invalid json config. type must be \"http\" or absent")
			}
		}
		rc[KeyRemoteType] = RemoteTypeHttp
		l := utils.JsGetString(rc, KeyRemoteLoc)
		if l == nil {
			return nil, fmt.Errorf("invalid json config. must have string \"loc\"")
		}
		rc[KeyRemoteLoc] = *l
		return rc, nil
	}
}
