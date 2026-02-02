package repos

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/spf13/viper"
	"github.com/wot-oss/tmc/internal/config"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

const (
	KeyRepos                  = "repos"
	keyRemotes                = "remotes" // left for compatibility
	KeyRepoType               = "type"
	KeyRepoLoc                = "loc"
	KeyRepoAuth               = "auth"
	KeyRepoHeaders            = "headers"
	KeyRepoEnabled            = "enabled"
	KeyRepoDescription        = "description"
	keySubRepo                = "keySubRepo"
	KeyRepoAWSRegion          = "aws_region"
	KeyRepoAWSBucket          = "aws_bucket"
	KeyRepoAWSEndpoint        = "aws_endpoint"
	KeyRepoAWSAccessKeyId     = "aws_access_key_id"
	KeyRepoAWSSecretAccessKey = "aws_secret_access_key"
	AuthMethodNone            = "none"
	AuthMethodBearerToken     = "bearer"
	AuthMethodBasic           = "basic"
	//AuthMethodOauthClientCredentials = "oauth-client-credentials"

	RepoTypeFile              = "file"
	RepoTypeHttp              = "http"
	RepoTypeTmc               = "tmc"
	RepoTypeS3                = "s3"
	CompletionKindNames       = "names"
	CompletionKindFetchNames  = "fetchNames"
	CompletionKindNamesOrIds  = "namesOrIds"
	CompletionKindAttachments = "attachments"
	RepoConfDir               = ".tmc"
	IndexFilename             = "tm-catalog.toc.json"
	TmNamesFile               = "tmnames.txt"
	TmIgnoreFile              = ".tmcignore"

	maxIndexingBatchSize = math.MaxInt
)

var ValidRepoNameRegex = regexp.MustCompile("^[a-zA-Z0-9][\\w\\-_:]*$")

var repoDefaultIgnore = []string{
	"# ignore any top-level files",
	"/*",
	"!/*/",
	"",
	"# ignore any top-level directories starting with a dot",
	"/.*/",
}

type Config map[string]map[string]any

var SupportedTypes = []string{RepoTypeFile, RepoTypeHttp, RepoTypeTmc, RepoTypeS3}

type ImportResultType int

const (
	ImportResultOK      = ImportResultType(iota + 1)
	ImportResultWarning // imported but with warning
	ImportResultError   // not imported because of error
)

func (t ImportResultType) String() string {
	switch t {
	case ImportResultOK:
		return "OK"
	case ImportResultWarning:
		return "warning"
	case ImportResultError:
		return "error"
	default:
		return "internal error: unknown import result type"
	}
}
func (t ImportResultType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

type ImportResult struct {
	Type ImportResultType `json:"type"`
	// TmID is not empty when the result is successful, i.e. Type is OK or Warning
	TmID    string `json:"-"`
	Message string `json:"message,omitempty"`
	// Err is not nil when there was an ID conflict or another error during import, i.e. Type is TMExists or Warning or Error
	Err error `json:"-"`
}

func ImportResultFromError(err error) (ImportResult, error) {
	return ImportResult{
		Type:    ImportResultError,
		Message: err.Error(),
		Err:     err,
	}, err
}

func (r ImportResult) String() string {
	return fmt.Sprintf("%v\t %s", r.Type, r.Message)
}

func (r ImportResult) IsSuccessful() bool {
	return r.Type == ImportResultOK || r.Type == ImportResultWarning
}

//go:generate mockery --name Repo --outpkg mocks --output mocks
type Repo interface {
	// Import writes the Thing Model file into the path under root that corresponds to id.
	// Returns ErrTMIDConflict if the same file is already stored with a different timestamp or
	// there is a file with the same semantic version and timestamp but different content
	Import(ctx context.Context, id model.TMID, raw []byte, opts ImportOptions) (ImportResult, error)
	// Fetch retrieves the Thing Model file from repo
	// Returns the actual id of the retrieved Thing Model (it may differ in the timestamp from the id requested), the file contents, and an error
	Fetch(ctx context.Context, id string) (string, []byte, error)
	// Index updates repository's index file with data from given TM files. For ids that refer to non-existing files,
	// removes those from index. Performs a full update if no updatedIds given
	Index(ctx context.Context, updatedIds ...string) error
	// CheckIntegrity checks the internal resources for integrity and consistency
	CheckIntegrity(ctx context.Context, filter model.ResourceFilter) (results []model.CheckResult, err error)
	// List searches the catalog for TMs matching search parameters
	List(ctx context.Context, search *model.Filters) (model.SearchResult, error)
	// Versions lists versions of a TM with given name
	Versions(ctx context.Context, name string) ([]model.FoundVersion, error)
	// Spec returns the spec this Repo has been created from
	Spec() model.RepoSpec
	// CanonicalRoot returns the canonical representation of the repository's root location
	CanonicalRoot() string
	// Delete deletes the TM with given id from repo. Returns ErrTMNotFound if TM does not exist
	Delete(ctx context.Context, id string) error

	ListCompletions(ctx context.Context, kind string, args []string, toComplete string) ([]string, error)

	GetTMMetadata(ctx context.Context, tmID string) ([]model.FoundVersion, error)
	ImportAttachment(ctx context.Context, container model.AttachmentContainerRef, attachment model.Attachment, content []byte, force bool) error
	FetchAttachment(ctx context.Context, container model.AttachmentContainerRef, attachmentName string) ([]byte, error)
	DeleteAttachment(ctx context.Context, container model.AttachmentContainerRef, attachmentName string) error
}

type ImportOptions struct {
	Force           bool
	OptPath         string
	IgnoreExisting  bool
	WithAttachments bool
}

var Get = func(spec model.RepoSpec) (Repo, error) {
	if spec.Dir() != "" {
		if spec.RepoName() != "" {
			return nil, fmt.Errorf("could not initialize a repo instance for %s: %w\ncheck config", spec, model.ErrInvalidSpec)
		}
		return NewFileRepo(map[string]any{KeyRepoType: "file", KeyRepoLoc: spec.Dir()}, spec)
	}
	repos, err := ReadConfig()
	if err != nil {
		return nil, fmt.Errorf("could not initialize a repo instance for %s: could not read config: %w", spec, err)
	}
	repos = filterEnabled(repos)
	parent, child := splitRepoName(spec.RepoName())
	spec = model.NewRepoSpec(parent)
	rc, ok := repos[parent]
	if parent == "" {
		switch len(repos) {
		case 0:
			return nil, fmt.Errorf("could not initialize a repo instance for %s: %w\ncheck config", spec, ErrRepoNotFound)
		case 1:
			for n, v := range repos {
				rc = v
				spec = model.NewRepoSpec(n)
			}
		default:
			return nil, fmt.Errorf("could not initialize a repo instance for %s: %w\ncheck config", spec, ErrAmbiguous)
		}
	} else {
		if !ok {
			return nil, fmt.Errorf("could not initialize a repo instance for %s: %w\ncheck config", spec, ErrRepoNotFound)
		}
	}
	if child != "" {
		rc[keySubRepo] = child
	}
	repo, err := createRepo(rc, spec)
	if err != nil {
		return nil, fmt.Errorf("could not initialize a repo instance for %s: %w\ncheck config", spec, err)
	}
	return repo, err
}

func splitRepoName(name string) (string, string) {
	before, after, _ := strings.Cut(name, "/")
	return before, after
}

func filterEnabled(repos Config) Config {
	res := make(Config)
	for n, rc := range repos {
		enabled, found := utils.JsGetBool(rc, KeyRepoEnabled)
		if found && !enabled {
			continue
		}
		res[n] = rc
	}
	return res
}

func createRepo(rc map[string]any, spec model.RepoSpec) (Repo, error) {
	switch t := rc[KeyRepoType]; t {
	case RepoTypeFile:
		return NewFileRepo(rc, spec)
	case RepoTypeHttp:
		return NewHttpRepo(rc, spec)
	case RepoTypeTmc:
		return NewTmcRepo(rc, spec)
	case RepoTypeS3:
		return NewS3Repo(rc, spec)
	default:
		return nil, fmt.Errorf("unsupported repo type: %v. Supported types are %v", t, SupportedTypes)
	}
}

var All = func() ([]Repo, error) {
	conf, err := ReadConfig()
	if err != nil {
		return nil, err
	}
	conf = filterEnabled(conf)
	var rs []Repo

	for n, rc := range conf {
		r, err := createRepo(rc, model.NewRepoSpec(n))
		if err != nil {
			return rs, err
		}
		rs = append(rs, r)
	}
	return rs, err
}

// GetDescriptions returns the list of descriptions of repositories that could be used as targets for write operations
// or be returned as "found-in" sources when reading from catalog
var GetDescriptions = func(ctx context.Context, spec model.RepoSpec) ([]model.RepoDescription, error) {
	if spec.Dir() != "" {
		r := model.RepoDescription{
			Name:        spec.Dir(),
			Type:        RepoTypeFile,
			Description: "Repo generated from the directory specified in the arguments of serve command",
		}
		return []model.RepoDescription{r}, nil
	}
	conf, err := ReadConfig()
	if err != nil {
		return nil, err
	}
	conf = filterEnabled(conf)
	var rs []model.RepoDescription
	for n, rc := range conf {
		if spec.RepoName() == "" || n == spec.RepoName() {
			ds, _ := rc[KeyRepoDescription].(string)
			r := model.RepoDescription{
				Name:        n,
				Type:        fmt.Sprintf("%v", rc[KeyRepoType]),
				Description: ds,
			}
			rs = append(rs, r)
		}
	}
	rs, err = expandTmcRepos(ctx, rs)
	return rs, err
}

func expandTmcRepos(ctx context.Context, descriptions []model.RepoDescription) ([]model.RepoDescription, error) {
	var ds []model.RepoDescription
	for _, d := range descriptions {
		if d.Type != RepoTypeTmc {
			ds = append(ds, d)
			continue
		}
		spec := model.NewRepoSpec(d.Name)
		repo, err := Get(spec)
		if err != nil {
			return nil, err
		}
		tmc, _ := repo.(*TmcRepo)
		repos, err := tmc.GetSubRepos(ctx)
		if err != nil {
			return nil, &RepoAccessError{spec, err}
		}
		if len(repos) < 2 {
			ds = append(ds, d)
		} else {
			for _, rd := range repos {
				ds = append(ds, model.RepoDescription{
					Name:        fmt.Sprintf("%s/%s", d.Name, rd.Name),
					Description: rd.Description,
				})
			}
		}
	}
	return ds, nil
}

func ReadConfig() (Config, error) {
	reposConfig := viper.Get(KeyRepos)
	if reposConfig == nil {
		remotesConfig := viper.Get(keyRemotes) // attempt to find obsolete key and convert config to new key
		if remotesConfig != nil {
			err := config.Save(KeyRepos, remotesConfig)
			if err != nil {
				return nil, err
			}
		}
		err := config.Delete(keyRemotes)
		if err != nil {
			return nil, err
		}
		reposConfig = remotesConfig
	}
	repos, ok := reposConfig.(map[string]any)
	if !ok {
		repos = map[string]any{}
	}
	return mapToConfig(repos)
}

func mapToConfig(repos map[string]any) (Config, error) {
	cp := Config{}
	for k, v := range repos {
		if cfg, ok := v.(map[string]any); ok {
			cp[k] = cfg
		} else {
			return nil, fmt.Errorf("invalid repo config: %s", k)
		}
	}
	return cp, nil
}

func ToggleEnabled(name string) error {
	conf, err := ReadConfig()
	if err != nil {
		return err
	}
	c, ok := conf[name]
	if !ok {
		return ErrRepoNotFound
	}
	if enabled, ok := c[KeyRepoEnabled]; ok {
		if eb, ok := enabled.(bool); ok && !eb {
			delete(c, KeyRepoEnabled)
		} else {
			c[KeyRepoEnabled] = false
		}
	} else {
		c[KeyRepoEnabled] = false
	}
	conf[name] = c
	return saveConfig(conf)
}

func Remove(name string) error {
	conf, err := ReadConfig()
	if err != nil {
		return err
	}
	if _, ok := conf[name]; !ok {
		return ErrRepoNotFound
	}
	// do a best-effort attempt at removing associated bleve index
	r, err := Get(model.NewRepoSpec(name))
	if err == nil {
		_ = os.RemoveAll(BleveIndexPath(r))
	}

	delete(conf, name)
	return saveConfig(conf)
}

func Add(name string, repoConf ConfigMap) error {
	_, err := Get(model.NewRepoSpec(name))
	if err == nil || !errors.Is(err, ErrRepoNotFound) {
		return ErrRepoExists
	}

	return setRepoConfig(name, repoConf)
}

func SetConfig(name string, repoConf ConfigMap) error {
	_, err := Get(model.NewRepoSpec(name))
	if err != nil && errors.Is(err, ErrRepoNotFound) {
		return ErrRepoNotFound
	}

	return setRepoConfig(name, repoConf)
}

func NewRepoConfig(typ string, confFile []byte) (ConfigMap, error) {
	var rc map[string]any
	var err error
	switch typ {
	case RepoTypeFile:
		rc, err = createFileRepoConfig(confFile)
		if err != nil {
			return nil, err
		}
	case RepoTypeHttp:
		rc, err = createHttpRepoConfig(confFile)
		if err != nil {
			return nil, err
		}
	case RepoTypeTmc:
		rc, err = createTmcRepoConfig(confFile)
		if err != nil {
			return nil, err
		}
	case RepoTypeS3:
		rc, err = createS3RepoConfig(confFile)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported repo type: %v. Supported types are %v", typ, SupportedTypes)
	}
	return rc, nil
}

func setRepoConfig(name string, rc ConfigMap) error {
	conf, err := ReadConfig()
	if err != nil {
		return err
	}

	conf[name] = rc

	return saveConfig(conf)
}

func Rename(oldName, newName string) error {
	if !ValidRepoNameRegex.MatchString(newName) {
		return ErrInvalidRepoName
	}
	conf, err := ReadConfig()
	if err != nil {
		return err
	}
	if rc, ok := conf[oldName]; ok {
		conf[newName] = rc
		delete(conf, oldName)
		return saveConfig(conf)
	} else {
		return ErrRepoNotFound
	}
}

func saveConfig(conf Config) error {
	return config.Save(KeyRepos, conf)
}

func AsRepoConfig(bytes []byte) (ConfigMap, error) {
	var js any
	err := json.Unmarshal(bytes, &js)
	if err != nil {
		return nil, fmt.Errorf("invalid json config: %w", err)
	}
	rc, ok := js.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid json config. must be a map")
	}
	return rc, nil
}

// GetUnion returns a union containing the repo specified by spec, or a union of all repos, if the spec is empty
func GetUnion(spec model.RepoSpec) (*Union, error) {
	if spec.RepoName() != "" || spec.Dir() != "" {
		repo, err := Get(spec)
		if err != nil {
			return nil, err
		}
		return NewUnion(repo), nil
	}
	all, err := All()
	if err != nil {
		return nil, err
	}
	return NewUnion(all...), nil
}

type ConfigMap map[string]any

// GetString reads a string value from the ConfigMap and expands environment variable if necessary
// It is very similar to util.JsGetString, except the latter does not expand variables
func (m ConfigMap) GetString(key string) (string, bool) {
	if m == nil {
		return "", false
	}
	s, found := utils.JsGetString(m, key)
	if !found {
		return s, false
	}
	return expandVar(s), true
}

// GetBool reads a bool value from the ConfigMap. If the value in the map is a string, it'll attempt to expand an
// environment variable and parse the result as bool.
// It is very similar to util.JsGetBool, except the latter does not expand variables
func (m ConfigMap) GetBool(key string) (bool, bool) {
	if m == nil {
		return false, false
	}
	b, found := utils.JsGetBool(m, key)
	if found {
		return b, true
	}
	s, found := m.GetString(key)
	if found {
		b, err := strconv.ParseBool(s)
		if err != nil {
			return false, false
		}
		return b, true
	}
	return false, false
}

func isEnvReference(s string) bool {
	return strings.HasPrefix(s, "$")
}

func expandVar(s string) string {
	if !isEnvReference(s) {
		return s
	}
	vName, _ := strings.CutPrefix(s, "$")
	ev, found := os.LookupEnv(vName)
	if found {
		return ev
	}
	return s
}

func BleveIndexPath(repo Repo) string {
	hasher := sha1.New()
	root := repo.CanonicalRoot()
	hasher.Write([]byte(root))
	hash := hasher.Sum(nil)
	hashStr := fmt.Sprintf("%x", hash[:6])
	return filepath.Join(config.ConfigDir, ".search-indexes", hashStr)
}

func UpdateRepoIndex(ctx context.Context, repo Repo) error {
	log := utils.GetLogger(ctx, "UpdateRepoIndex")
	searchResult, err := repo.List(ctx, nil)
	if err != nil {
		os.Exit(1)
	}
	// try to open index, if it not there create a fresh one
	indexPath := BleveIndexPath(repo)
	index, err := bleve.Open(indexPath)

	if err != nil {
		_ = os.MkdirAll(filepath.Dir(indexPath), 0755)
		// open a new index
		index, err = bleve.New(indexPath, bleve.NewIndexMapping())
		if err != nil {
			return err
		}
	}

	defer index.Close()

	contents := searchResult.Entries
	var batch *bleve.Batch

	indexedCount, batchCount, totalCount := 0, 0, 0

	for _, value := range contents {
		for _, version := range value.Versions {
			totalCount++
			// check if document is already indexed
			doc, _ := index.Document(version.TMID)
			if doc != nil {
				// already indexed -> skip
				continue
			}
			// fetch document
			id, thing, err := repo.Fetch(ctx, version.TMID)
			_ = id
			if err != nil {
				log.Warn("can't fetch TM", "error", err)
				continue
			}
			var data any
			unmErr := json.Unmarshal(thing, &data)
			if unmErr != nil {
				log.Warn("can't unmarshal TM", "error", unmErr)
				continue
			}

			if batch == nil {
				batch = index.NewBatch()
			}
			var idxErr error
			idxErr = batch.Index(version.TMID, data)
			if idxErr != nil {
				return fmt.Errorf("can't index TM: %w", idxErr)
			}
			batchCount++
			indexedCount++
			if batchCount >= maxIndexingBatchSize {
				batchCount = 0
				err = index.Batch(batch)
				if err != nil {
					return fmt.Errorf("can't run batch: %w", err)
				}
				batch = nil
			}
		}
	}
	if batch != nil {
		err = index.Batch(batch)
		if err != nil {
			return fmt.Errorf("can't run batch: %w", err)
		}
	}
	lu := searchResult.LastUpdated.Format(time.RFC3339Nano)
	err = utils.WriteFileLines([]string{lu}, filepath.Join(indexPath, "updated"), 0664)
	if err != nil {
		return err
	}
	utils.GetLogger(ctx, "create-si").Info(fmt.Sprintf("indexed %d new Thing Models out of %d\n", indexedCount, totalCount))
	return nil

}
