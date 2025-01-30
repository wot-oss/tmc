package repos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/buger/jsonparser"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/wot-oss/tmc/internal/app/http/server"
	"github.com/wot-oss/tmc/internal/config"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

var httpTransport http.RoundTripper
var once sync.Once

func getCachingTransport() http.RoundTripper {
	once.Do(func() {
		if config.ConfigDir == "" { // this is probably a test run, but even if it isn't, we don't want to write the cache in the working directory
			httpTransport = http.DefaultTransport
			return
		}
		cacheDir := filepath.Join(config.ConfigDir, ".http-cache")
		err := os.MkdirAll(cacheDir, 0770)
		if err != nil {
			panic(err)
		}
		cache := diskcache.New(cacheDir)
		httpTransport = httpcache.NewTransport(cache)
	})
	return httpTransport
}

type baseHttpRepo struct {
	root       string
	parsedRoot *url.URL
	spec       model.RepoSpec
	auth       ConfigMap
	headers    ConfigMap
	client     *http.Client
}

func newBaseHttpRepo(config ConfigMap, spec model.RepoSpec) (baseHttpRepo, error) {
	loc, found := config.GetString(KeyRepoLoc)
	if !found {
		return baseHttpRepo{}, fmt.Errorf("invalid http repo config. loc is either not found or not a string")
	}
	u, err := url.Parse(loc)
	if err != nil {
		return baseHttpRepo{}, err
	}
	auth, _ := utils.JsGetMap(config, KeyRepoAuth)
	client, err := getHttpClient(auth)
	if err != nil {
		return baseHttpRepo{}, fmt.Errorf("invalid http repo config: %v", err)
	}
	headers, _ := utils.JsGetMap(config, KeyRepoHeaders)
	base := baseHttpRepo{
		root:       loc,
		spec:       spec,
		auth:       auth,
		headers:    headers,
		client:     client,
		parsedRoot: u,
	}
	return base, nil
}

func getHttpClient(auth map[string]any) (*http.Client, error) {
	client := &http.Client{Transport: getCachingTransport()}
	//if auth != nil {
	//	credConf := utils.JsGetMap(auth, AuthMethodOauthClientCredentials)
	//	if credConf != nil {
	//		scopes := utils.JsGetString(credConf, "scopes")
	//		var scps []string
	//		if scopes != nil {
	//			scps = strings.Split(*scopes, ",")
	//		}
	//		conf := clientcredentials.Config{
	//			ClientID:     utils.JsGetStringOrEmpty(credConf, "client-id"),
	//			ClientSecret: utils.JsGetStringOrEmpty(credConf, "client-secret"),
	//			TokenURL:     utils.JsGetStringOrEmpty(credConf, "token-url"),
	//			Scopes:       scps,
	//			AuthStyle:    oauth2.AuthStyleAutoDetect,
	//		}
	//		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
	//		return conf.Client(ctx), nil
	//	}
	//}
	return client, nil
}

func (b *baseHttpRepo) doGet(ctx context.Context, reqUrl string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
	if err != nil {
		return nil, err
	}
	return b.doHttp(req)
}

func (b *baseHttpRepo) doHttp(req *http.Request) (*http.Response, error) {
	if b.auth != nil {
		basicAuth, found := utils.JsGetMap(b.auth, AuthMethodBasic)
		if found {
			ba := ConfigMap(basicAuth)
			username, _ := ba.GetString("username")
			password, _ := ba.GetString("password")
			req.SetBasicAuth(username, password)
		} else {

			bearerToken, tf := b.auth.GetString(AuthMethodBearerToken)
			if tf {
				req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
			}
		}
	}
	for h, v := range b.headers {
		if vs, ok := v.(string); ok {
			req.Header.Add(expandVar(h), expandVar(vs))
		} else if varr, ok := v.([]any); ok {
			for _, vv := range varr {
				req.Header.Add(expandVar(h), expandVar(fmt.Sprintf("%v", vv)))
			}
		}
	}

	resp, err := b.client.Do(req)
	if err != nil {
		utils.GetLogger(req.Context(), "baseHttpRepo").Error(err.Error())
	}
	if resp != nil && resp.StatusCode >= http.StatusBadRequest {
		utils.GetLogger(req.Context(), "baseHttpRepo").Error("received error response from remote", "status", resp.StatusCode)
	}
	return resp, err
}

func (b *baseHttpRepo) fetchTM(ctx context.Context, tmUrl string) (string, []byte, error) {
	resp, err := b.doGet(ctx, tmUrl)
	if err != nil {
		return "", nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		value, dataType, _, err := jsonparser.Get(body, "id")
		if err != nil && dataType != jsonparser.NotExist {
			return "", body, err
		}
		switch dataType {
		case jsonparser.String:
			return string(value), body, nil
		default:
			return fmt.Sprintf("%v", value), body, fmt.Errorf("unexpected type of 'id': %v", value)
		}
	case http.StatusNotFound:
		return "", nil, model.ErrTMNotFound
	case http.StatusInternalServerError, http.StatusBadRequest:
		return "", nil, errors.New(string(body))
	default:
		return "", nil, errors.New(fmt.Sprintf("received unexpected HTTP response from remote server: %s", resp.Status))
	}

}

func (b *baseHttpRepo) fetchAttachment(ctx context.Context, reqUrl string) ([]byte, error) {
	resp, err := b.doGet(ctx, reqUrl)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return body, nil
	case http.StatusNotFound:
		var e server.ErrorResponse
		err := json.Unmarshal(body, &e)
		code := ""
		if err == nil && e.Code != nil {
			code = *e.Code
		}
		return nil, model.NewErrNotFound(code)
	case http.StatusBadRequest:
		return nil, model.ErrInvalidIdOrName
	case http.StatusInternalServerError, http.StatusUnauthorized:
		return nil, newErrorFromResponse(body)
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
