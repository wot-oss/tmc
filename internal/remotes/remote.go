package remotes

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/spf13/viper"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/config"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

const (
	KeyRemotes       = "remotes"
	KeyRemoteType    = "type"
	KeyRemoteLoc     = "loc"
	KeyRemoteAuth    = "auth"
	KeyRemoteEnabled = "enabled"

	RemoteTypeFile           = "file"
	RemoteTypeHttp           = "http"
	RemoteTypeTmc            = "tmc"
	CompletionKindNames      = "names"
	CompletionKindFetchNames = "fetchNames"
	RepoConfDir              = ".tmc"
	TOCFilename              = "tm-catalog.toc.json"
	TmNamesFile              = "tmnames.txt"
)

var ValidRemoteNameRegex = regexp.MustCompile("^[a-zA-Z0-9][\\w\\-_:]*$")

type Config map[string]map[string]any

var SupportedTypes = []string{RemoteTypeFile, RemoteTypeHttp, RemoteTypeTmc}

//go:generate mockery --name Remote --inpackage
type Remote interface {
	// Push writes the Thing Model file into the path under root that corresponds to id.
	// Returns ErrTMIDConflict if the same file is already stored with a different timestamp or
	// there is a file with the same semantic version and timestamp but different content
	Push(id model.TMID, raw []byte) error
	// Fetch retrieves the Thing Model file from remote
	// Returns the actual id of the retrieved Thing Model (it may differ in the timestamp from the id requested), the file contents, and an error
	Fetch(id string) (string, []byte, error)
	// UpdateToc updates table of contents file with data from given TM files. For ids that refer to non-existing files,
	// removes those from table of contents. Performs a full update if no updatedIds given
	UpdateToc(updatedIds ...string) error
	// List searches the catalog for TMs matching search parameters
	List(search *model.SearchParams) (model.SearchResult, error)
	// Versions lists versions of a TM with given name
	Versions(name string) ([]model.FoundVersion, error)
	// Spec returns the spec this Remote has been created from
	Spec() RepoSpec
	// Delete deletes the TM with given id from remote. Returns ErrTmNotFound if TM does not exist
	Delete(id string) error

	ListCompletions(kind string, toComplete string) ([]string, error)
}

//go:generate mockery --name RemoteManager --inpackage
type RemoteManager interface {
	// Get returns the Remote built from config with the given remoteName
	// Empty remoteName returns the sole remote, if there's only one. Otherwise, an error
	Get(spec RepoSpec) (Remote, error)
	All() ([]Remote, error)
	ReadConfig() (Config, error)
	ToggleEnabled(name string) error
	Remove(name string) error
	Rename(oldName, newName string) error
	Add(name, typ, confStr string, confFile []byte) error
	SetConfig(name, typ, confStr string, confFile []byte) error
}

type RepoSpec struct {
	remoteName string
	dir        string
}

func NewSpec(remoteName, dir string) (RepoSpec, error) {
	if remoteName != "" && dir != "" {
		return RepoSpec{}, ErrInvalidSpec
	}
	return RepoSpec{
		remoteName: remoteName,
		dir:        dir,
	}, nil
}

func NewRemoteSpec(remoteName string) RepoSpec {
	return RepoSpec{
		remoteName: remoteName,
	}
}

func NewDirSpec(dir string) RepoSpec {
	return RepoSpec{
		dir: dir,
	}
}

func NewSpecFromFoundSource(s model.FoundSource) RepoSpec {
	return RepoSpec{
		remoteName: s.RemoteName,
		dir:        s.Directory,
	}
}

func (r RepoSpec) ToFoundSource() model.FoundSource {
	return model.FoundSource{
		Directory:  r.dir,
		RemoteName: r.remoteName,
	}
}

func (r RepoSpec) String() string {
	if r.dir == "" {
		if r.remoteName == "" {
			return fmt.Sprintf("undefined repository")
		}
		return fmt.Sprintf("remote <%s>", r.remoteName)
	}
	return fmt.Sprintf("directory %s", r.dir)
}

var EmptySpec, _ = NewSpec("", "")

type remoteManager struct {
}

var defaultManager = &remoteManager{}

func DefaultManager() RemoteManager {
	return defaultManager
}
func (r *remoteManager) Get(spec RepoSpec) (Remote, error) {
	if spec.dir != "" {
		if spec.remoteName != "" {
			return nil, ErrInvalidSpec
		}
		return NewFileRemote(map[string]any{KeyRemoteType: "file", KeyRemoteLoc: spec.dir}, spec)
	}
	remotes, err := r.ReadConfig()
	if err != nil {
		return nil, err
	}
	remotes = filterEnabled(remotes)
	rc, ok := remotes[spec.remoteName]
	if spec.remoteName == "" {
		switch len(remotes) {
		case 0:
			return nil, ErrRemoteNotFound
		case 1:
			for n, v := range remotes {
				rc = v
				spec.remoteName = n
			}
		default:
			return nil, ErrAmbiguous
		}
	} else {
		if !ok {
			return nil, ErrRemoteNotFound
		}
	}
	return createRemote(rc, spec)
}

func filterEnabled(remotes Config) Config {
	res := make(Config)
	for n, rc := range remotes {
		enabled := utils.JsGetBool(rc, KeyRemoteEnabled)
		if enabled != nil && !*enabled {
			continue
		}
		res[n] = rc
	}
	return res
}

func createRemote(rc map[string]any, spec RepoSpec) (Remote, error) {
	switch t := rc[KeyRemoteType]; t {
	case RemoteTypeFile:
		return NewFileRemote(rc, spec)
	case RemoteTypeHttp:
		return NewHttpRemote(rc, spec)
	case RemoteTypeTmc:
		return NewTmcRemote(rc, spec)
	default:
		return nil, fmt.Errorf("unsupported remote type: %v. Supported types are %v", t, SupportedTypes)
	}
}

func (r *remoteManager) All() ([]Remote, error) {
	conf, err := r.ReadConfig()
	if err != nil {
		return nil, err
	}
	var rs []Remote

	for n, rc := range conf {
		en := utils.JsGetBool(rc, KeyRemoteEnabled)
		if en != nil && !*en {
			continue
		}
		r, err := createRemote(rc, NewRemoteSpec(n))
		if err != nil {
			return rs, err
		}
		rs = append(rs, r)
	}
	return rs, err
}

func (r *remoteManager) ReadConfig() (Config, error) {
	remotesConfig := viper.Get(KeyRemotes)
	remotes, ok := remotesConfig.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid remotes contig")
	}
	cp := map[string]map[string]any{}
	for k, v := range remotes {
		if cfg, ok := v.(map[string]any); ok {
			cp[k] = cfg
		} else {
			return nil, fmt.Errorf("invalid remote config: %s", k)
		}
	}
	return cp, nil
}

func (r *remoteManager) ToggleEnabled(name string) error {
	conf, err := r.ReadConfig()
	if err != nil {
		return err
	}
	c, ok := conf[name]
	if !ok {
		return ErrRemoteNotFound
	}
	if enabled, ok := c[KeyRemoteEnabled]; ok {
		if eb, ok := enabled.(bool); ok && !eb {
			delete(c, KeyRemoteEnabled)
		} else {
			c[KeyRemoteEnabled] = false
		}
	} else {
		c[KeyRemoteEnabled] = false
	}
	conf[name] = c
	return r.saveConfig(conf)
}

func (r *remoteManager) Remove(name string) error {
	conf, err := r.ReadConfig()
	if err != nil {
		return err
	}
	if _, ok := conf[name]; !ok {
		return ErrRemoteNotFound
	}
	delete(conf, name)
	return r.saveConfig(conf)
}

func (r *remoteManager) Add(name, typ, confStr string, confFile []byte) error {
	_, err := r.Get(NewRemoteSpec(name))
	if err == nil || !errors.Is(err, ErrRemoteNotFound) {
		return ErrRemoteExists
	}

	return r.setRemoteConfig(name, typ, confStr, confFile, err)
}

func (r *remoteManager) SetConfig(name, typ, confStr string, confFile []byte) error {
	_, err := r.Get(NewRemoteSpec(name))
	if err != nil && errors.Is(err, ErrRemoteNotFound) {
		return ErrRemoteNotFound
	}

	return r.setRemoteConfig(name, typ, confStr, confFile, err)
}

func (r *remoteManager) setRemoteConfig(name string, typ string, confStr string, confFile []byte, err error) error {
	var rc map[string]any
	switch typ {
	case RemoteTypeFile:
		rc, err = createFileRemoteConfig(confStr, confFile)
		if err != nil {
			return err
		}
	case RemoteTypeHttp:
		rc, err = createHttpRemoteConfig(confStr, confFile)
		if err != nil {
			return err
		}
	case RemoteTypeTmc:
		rc, err = createTmcRemoteConfig(confStr, confFile)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported remote type: %v. Supported types are %v", typ, SupportedTypes)
	}

	conf, err := r.ReadConfig()
	if err != nil {
		return err
	}

	conf[name] = rc

	return r.saveConfig(conf)
}

func (r *remoteManager) Rename(oldName, newName string) error {
	if !ValidRemoteNameRegex.MatchString(newName) {
		return ErrInvalidRemoteName
	}
	conf, err := r.ReadConfig()
	if err != nil {
		return err
	}
	if rc, ok := conf[oldName]; ok {
		conf[newName] = rc
		delete(conf, oldName)
		return r.saveConfig(conf)
	} else {
		return ErrRemoteNotFound
	}
}
func (r *remoteManager) saveConfig(conf Config) error {
	viper.Set(KeyRemotes, conf)
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		configFile = filepath.Join(config.DefaultConfigDir, "config.json")
	}
	err := os.MkdirAll(config.DefaultConfigDir, 0770)
	if err != nil {
		return err
	}

	b, err := os.ReadFile(configFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(b) == 0 {
		b = []byte("{}")
	}
	var j map[string]any
	err = json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	j[KeyRemotes] = conf

	w, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	return utils.AtomicWriteFile(configFile, w, 0660)
}

func AsRemoteConfig(bytes []byte) (map[string]any, error) {
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

// GetSpecdOrAll returns the remote specified by spec in a slice, or all remotes, if the spec is empty
func GetSpecdOrAll(manager RemoteManager, spec RepoSpec) (*Union, error) {
	if spec.remoteName != "" || spec.dir != "" {
		remote, err := manager.Get(spec)
		if err != nil {
			return nil, err
		}
		return NewUnion(remote), nil
	}
	all, err := manager.All()
	if err != nil {
		return nil, err
	}
	return NewUnion(all...), nil
}
