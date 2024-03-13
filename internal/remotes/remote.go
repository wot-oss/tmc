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

//go:generate mockery --name Remote --outpkg mocks --output mocks
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
	Spec() model.RepoSpec
	// Delete deletes the TM with given id from remote. Returns ErrTmNotFound if TM does not exist
	Delete(id string) error

	ListCompletions(kind string, toComplete string) ([]string, error)
}

var Get = func(spec model.RepoSpec) (Remote, error) {
	if spec.Dir() != "" {
		if spec.RemoteName() != "" {
			return nil, model.ErrInvalidSpec
		}
		return NewFileRemote(map[string]any{KeyRemoteType: "file", KeyRemoteLoc: spec.Dir()}, spec)
	}
	remotes, err := ReadConfig()
	if err != nil {
		return nil, err
	}
	remotes = filterEnabled(remotes)
	rc, ok := remotes[spec.RemoteName()]
	if spec.RemoteName() == "" {
		switch len(remotes) {
		case 0:
			return nil, ErrRemoteNotFound
		case 1:
			for n, v := range remotes {
				rc = v
				spec = model.NewRemoteSpec(n)
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

func createRemote(rc map[string]any, spec model.RepoSpec) (Remote, error) {
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

var All = func() ([]Remote, error) {
	conf, err := ReadConfig()
	if err != nil {
		return nil, err
	}
	var rs []Remote

	for n, rc := range conf {
		en := utils.JsGetBool(rc, KeyRemoteEnabled)
		if en != nil && !*en {
			continue
		}
		r, err := createRemote(rc, model.NewRemoteSpec(n))
		if err != nil {
			return rs, err
		}
		rs = append(rs, r)
	}
	return rs, err
}

func ReadConfig() (Config, error) {
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

func ToggleEnabled(name string) error {
	conf, err := ReadConfig()
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
	return saveConfig(conf)
}

func Remove(name string) error {
	conf, err := ReadConfig()
	if err != nil {
		return err
	}
	if _, ok := conf[name]; !ok {
		return ErrRemoteNotFound
	}
	delete(conf, name)
	return saveConfig(conf)
}

func Add(name, typ, confStr string, confFile []byte) error {
	_, err := Get(model.NewRemoteSpec(name))
	if err == nil || !errors.Is(err, ErrRemoteNotFound) {
		return ErrRemoteExists
	}

	return setRemoteConfig(name, typ, confStr, confFile, err)
}

func SetConfig(name, typ, confStr string, confFile []byte) error {
	_, err := Get(model.NewRemoteSpec(name))
	if err != nil && errors.Is(err, ErrRemoteNotFound) {
		return ErrRemoteNotFound
	}

	return setRemoteConfig(name, typ, confStr, confFile, err)
}

func setRemoteConfig(name string, typ string, confStr string, confFile []byte, err error) error {
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

	conf, err := ReadConfig()
	if err != nil {
		return err
	}

	conf[name] = rc

	return saveConfig(conf)
}

func Rename(oldName, newName string) error {
	if !ValidRemoteNameRegex.MatchString(newName) {
		return ErrInvalidRemoteName
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
		return ErrRemoteNotFound
	}
}
func saveConfig(conf Config) error {
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
func GetSpecdOrAll(spec model.RepoSpec) (*Union, error) {
	if spec.RemoteName() != "" || spec.Dir() != "" {
		remote, err := Get(spec)
		if err != nil {
			return nil, err
		}
		return NewUnion(remote), nil
	}
	all, err := All()
	if err != nil {
		return nil, err
	}
	return NewUnion(all...), nil
}

func MockRemotesAll(t interface {
	Cleanup(func())
}, mock func() ([]Remote, error)) {
	org := All
	All = mock
	t.Cleanup(func() {
		All = org
	})
}

func MockRemotesGet(t interface {
	Cleanup(func())
}, mock func(spec model.RepoSpec) (Remote, error)) {
	org := Get
	Get = mock
	t.Cleanup(func() {
		Get = org
	})
}

func MockFail(t interface {
	Fail()
	Error(args ...any)
}, args ...any) {
	t.Error(args...)
	t.Fail()
}
