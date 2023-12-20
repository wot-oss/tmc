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

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

var ErrNotSupported = errors.New("method not supported")

const RelFileUriPlaceholder = "{{ID}}"

type HttpRemote struct {
	root           string
	parsedRoot     *url.URL
	templatedPath  bool
	templatedQuery bool
	name           string
	auth           map[string]any
}

func NewHttpRemote(config map[string]any, name string) (*HttpRemote, error) {
	loc := utils.JsGetString(config, KeyRemoteLoc)
	if loc == nil {
		return nil, fmt.Errorf("invalid http remote config. loc is either not found or not a string")
	}
	u, err := url.Parse(*loc)
	if err != nil {
		return nil, err
	}
	auth := utils.JsGetMap(config, KeyRemoteAuth)
	h := &HttpRemote{root: *loc, parsedRoot: u, name: name, auth: auth}
	cpl := strings.Count(*loc, RelFileUriPlaceholder)
	switch cpl {
	case 0:
	// do nothing
	case 1:
		if strings.Contains(u.RawPath, RelFileUriPlaceholder) || strings.Contains(u.Path, RelFileUriPlaceholder) {
			h.templatedPath = true
		} else if strings.Contains(u.RawQuery, RelFileUriPlaceholder) {
			h.templatedQuery = true
		} else {
			return nil, fmt.Errorf("invalid http remote config. %s placeholder in URL %s is only allowed in path or query", RelFileUriPlaceholder, *loc)
		}
	default:
		return nil, fmt.Errorf("invalid http remote config. At most one instance of %s placeholder is allowed in URL %s", RelFileUriPlaceholder, *loc)
	}

	return h, nil
}

func (h *HttpRemote) Push(_ model.TMID, _ []byte) error {
	return ErrNotSupported
}

func (h *HttpRemote) Fetch(id string) ([]byte, error) {
	reqUrl := h.buildUrl(id)
	resp, err := h.doGet(reqUrl)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(resp.Body)
}

func (h *HttpRemote) buildUrl(fileId string) string {
	if h.templatedPath {
		return strings.Replace(h.root, RelFileUriPlaceholder, url.PathEscape(fileId), 1)
	} else if h.templatedQuery {
		return strings.Replace(h.root, RelFileUriPlaceholder, url.QueryEscape(fileId), 1)
	}
	return h.parsedRoot.JoinPath(fileId).String()
}

func (h *HttpRemote) CreateToC() error {
	return ErrNotSupported
}
func (h *HttpRemote) Name() string {
	return h.name
}

func (h *HttpRemote) List(search *model.SearchParams) (model.SearchResult, error) {
	reqUrl := h.buildUrl(TOCFilename)
	resp, err := h.doGet(reqUrl)
	if err != nil {
		return model.SearchResult{}, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.SearchResult{}, err
	}

	var toc model.TOC
	err = json.Unmarshal(data, &toc)
	toc.Filter(search)
	if err != nil {
		return model.SearchResult{}, err
	}
	return model.NewSearchResultFromTOC(toc, h.Name()), nil
}

func (h *HttpRemote) doGet(reqUrl string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, reqUrl, nil)
	if err != nil {
		return nil, err
	}
	h.addAuth(req)
	resp, err := http.DefaultClient.Do(req)
	return resp, err
}

func (h *HttpRemote) Versions(name string) (model.FoundEntry, error) {
	log := slog.Default()
	if len(name) == 0 {
		log.Error("Please specify a name to show the TM.")
		return model.FoundEntry{}, errors.New("please specify a name to show the TM")
	}
	name = strings.TrimSpace(name)
	toc, err := h.List(&model.SearchParams{Name: name})
	if err != nil {
		return model.FoundEntry{}, err
	}

	if len(toc.Entries) != 1 {
		log.Error(fmt.Sprintf("No thing model found for name: %s", name))
		return model.FoundEntry{}, ErrEntryNotFound
	}

	return toc.Entries[0], nil
}

func (h *HttpRemote) addAuth(req *http.Request) {
	if h.auth != nil {
		bearerToken := utils.JsGetString(h.auth, "bearer")
		if bearerToken != nil {
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *bearerToken))
		}
	}
}

func createHttpRemoteConfig(root string, bytes []byte) (map[string]any, error) {
	if root != "" {
		return map[string]any{
			KeyRemoteType: RemoteTypeHttp,
			KeyRemoteLoc:  root,
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
