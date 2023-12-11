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
	KeyRemoteDefault = "default"
	KeyRemoteEnabled = "enabled"

	RemoteTypeFile = "file"
	RemoteTypeHttp = "http"
)

var ValidRemoteNameRegex = regexp.MustCompile("^[a-zA-Z0-9][\\w\\-_:]*$")

type Config map[string]map[string]any

var ErrNoDefault = errors.New("no default remote config found")
var ErrRemoteNotFound = errors.New("named remote not found")
var ErrInvalidRemoteName = errors.New("invalid remote name")
var ErrRemoteExists = errors.New("named remote already exists")
var ErrEntryNotFound = errors.New("entry not found")

var SupportedTypes = []string{RemoteTypeFile, RemoteTypeHttp}

type Remote interface {
	// Push writes the Thing Model file into the path under root that corresponds to id.
	// Returns ErrTMExists if the same file is already stored with a different timestamp
	Push(id model.TMID, raw []byte) error
	Fetch(id model.TMID) ([]byte, error)
	CreateToC() error
	List(filter string) (model.TOC, error)
	Versions(name string) (model.TOCEntry, error)
	Name() string
}

type RemoteManager interface {
	// Get returns the Remote built from config with the given name
	// Empty name returns the default remote
	Get(name string) (Remote, error)
	All() ([]Remote, error)
	ReadConfig() (Config, error)
	SetDefault(name string) error
	ToggleEnabled(name string) error
	Remove(name string) error
	Rename(oldName, newName string) error
	Add(name, typ, confStr string, confFile []byte) error
	SetConfig(name, typ, confStr string, confFile []byte) error
}

type remoteManager struct {
}

var defaultManager = &remoteManager{}

func DefaultManager() RemoteManager {
	return defaultManager
}
func (r *remoteManager) Get(name string) (Remote, error) {
	remotes, err := r.ReadConfig()
	if err != nil {
		return nil, err
	}
	rc, ok := remotes[name]
	if name == "" {
		if len(remotes) == 1 {
			for n, v := range remotes {
				rc = v
				name = n
			}
		} else {
			found := false
			for n, v := range remotes {
				def := utils.JsGetBool(v, KeyRemoteDefault)
				if def != nil && *def {
					rc = v
					name = n
					found = true
					break
				}
			}
			if !found {
				return nil, ErrNoDefault
			}
		}
	} else {
		if !ok {
			return nil, ErrRemoteNotFound
		}
	}

	enabled := utils.JsGetBool(rc, KeyRemoteEnabled)
	if enabled != nil && !*enabled {
		return nil, ErrRemoteNotFound
	}
	return createRemote(rc, name)
}

func createRemote(rc map[string]any, name string) (Remote, error) {
	switch t := rc[KeyRemoteType]; t {
	case RemoteTypeFile:
		return NewFileRemote(rc, name)
	case RemoteTypeHttp:
		return NewHttpRemote(rc, name)
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
		r, err := createRemote(rc, n)
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

func (r *remoteManager) SetDefault(name string) error {
	conf, err := r.ReadConfig()
	if err != nil {
		return err
	}
	if _, ok := conf[name]; !ok {
		return ErrRemoteNotFound
	}
	for n, rc := range conf {
		if n == name {
			rc[KeyRemoteDefault] = true
		} else {
			delete(rc, KeyRemoteDefault)
		}
	}
	return r.saveConfig(conf)
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
	_, err := r.Get(name)
	if err == nil || !errors.Is(err, ErrRemoteNotFound) {
		return ErrRemoteExists
	}

	return r.setRemoteConfig(name, typ, confStr, confFile, err)
}

func (r *remoteManager) SetConfig(name, typ, confStr string, confFile []byte) error {
	_, err := r.Get(name)
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
	dc := 0
	for _, rc := range conf {
		d := rc[KeyRemoteDefault]
		if b, ok := d.(bool); ok && b {
			dc++
		}
	}
	if dc > 1 {
		return fmt.Errorf("too many default remotes. can accept at most one")
	}

	viper.Set(KeyRemotes, conf)
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		configFile = filepath.Join(config.DefaultConfigDir, "config.json")
	}
	err := os.MkdirAll(config.DefaultConfigDir, 0770)
	if err != nil {
		return err
	}
	return viper.WriteConfigAs(configFile)
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

// GetNamedOrAll returns the remote with name remoteName in a slice, or all remotes, in case remoteName is empty string
func GetNamedOrAll(manager RemoteManager, remoteName string) ([]Remote, error) {
	if remoteName != "" {
		remote, err := manager.Get(remoteName)
		if err != nil {
			return nil, err
		}
		return []Remote{remote}, nil
	} else {
		return manager.All()
	}
}
