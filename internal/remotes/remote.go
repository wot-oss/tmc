package remotes

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
	"os"
	"path/filepath"
	"regexp"

	"github.com/spf13/viper"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/config"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

const (
	KeyRemotes       = "remotes"
	KeyRemoteType    = "type"
	KeyRemoteLoc     = "loc"
	KeyRemoteDefault = "default"

	RemoteTypeFile = "file"
	RemoteTypeHttp = "http"
)

var ValidRemoteNameRegex = regexp.MustCompile("^[a-zA-Z0-9][\\w\\-_:]*$")

type Config map[string]map[string]any

var ErrNoDefault = errors.New("no default remote found")
var ErrRemoteNotFound = errors.New("named remote not found")
var ErrInvalidRemoteName = errors.New("invalid remote name")
var ErrRemoteExists = errors.New("named remote already exists")
var ErrTMAlreadyExists = errors.New("given Thing Model already exists as")
var ErrTMNotExists = errors.New("given Thing Model does not exist")
var ErrNotSupported = errors.New("method not supported")
var SupportedTypes = []string{RemoteTypeFile, RemoteTypeHttp}

type Remote interface {
	// Push writes the Thing Model file into the path under root that corresponds to id.
	// Returns ErrTMAlreadyExists if the same file is already stored with a different timestamp
	Push(id model.TMID, raw []byte) (model.TMID, error)
	Fetch(id model.TMID) ([]byte, error)
	CreateToC() error
	List(filter string) (model.TOC, error)
	Versions(name string) (model.TOCEntry, error)
}

// Get returns the Remote built from config with the given name
// Empty name returns the default remote
func Get(name string) (Remote, error) {
	remotes, err := ReadConfig()
	if err != nil {
		return nil, err
	}
	rc, ok := remotes[name]
	if name == "" {
		if len(remotes) == 1 {
			for _, v := range remotes {
				rc = v
			}
		} else {
			found := false
			for _, v := range remotes {
				if def, ok := v[KeyRemoteDefault]; ok {
					if d, ok := def.(bool); ok && d {
						rc = v
						found = true
						break
					}
				}
			}
			if !found {
				return nil, utils.NewClientErr(ErrNoDefault, "", nil)
			}
		}
	} else {
		if !ok {
			return nil, utils.NewClientErr(ErrRemoteNotFound, name, nil)
		}
	}

	switch t := rc[KeyRemoteType]; t {
	case RemoteTypeFile:
		return NewFileRemote(rc)
	case RemoteTypeHttp:
		return NewHttpRemote(rc)
	default:
		return nil, utils.NewClientErr(fmt.Errorf("unsupported remote type"),
			fmt.Sprintf("remote name=%s, type=%s, supported types=%v", name, t, SupportedTypes), nil)
	}

}

func ReadConfig() (Config, error) {
	remotesConfig := viper.Get(KeyRemotes)
	remotes, ok := remotesConfig.(map[string]any)
	if !ok {
		return nil, utils.NewClientErr(fmt.Errorf("invalid remotes config"), "", nil)
	}
	cp := map[string]map[string]any{}
	for k, v := range remotes {
		if cfg, ok := v.(map[string]any); ok {
			cp[k] = cfg
		} else {
			return nil, utils.NewClientErr(fmt.Errorf("invalid remote config"), k, nil)
		}
	}
	return cp, nil
}

func SetDefault(name string) error {
	conf, err := ReadConfig()
	if err != nil {
		return err
	}
	if _, ok := conf[name]; !ok {
		return utils.NewClientErr(ErrRemoteNotFound, name, nil)
	}
	for n, rc := range conf {
		if n == name {
			rc[KeyRemoteDefault] = true
		} else {
			delete(rc, KeyRemoteDefault)
		}
	}
	return saveConfig(conf)
}
func Remove(name string) error {
	conf, err := ReadConfig()
	if err != nil {
		return err
	}
	if _, ok := conf[name]; !ok {
		return utils.NewClientErr(ErrRemoteNotFound, name, nil)
	}
	delete(conf, name)
	return saveConfig(conf)
}

func Add(name, typ, confStr string, confFile []byte) error {
	_, err := Get(name)
	if err == nil || !errors.Is(err, ErrRemoteNotFound) {
		return utils.NewClientErr(ErrRemoteExists, name, nil)
	}

	return setRemoteConfig(name, typ, confStr, confFile, err)
}
func SetConfig(name, typ, confStr string, confFile []byte) error {
	_, err := Get(name)
	if err != nil && errors.Is(err, ErrRemoteNotFound) {
		return utils.NewClientErr(ErrRemoteNotFound, name, nil)
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
	default:
		return utils.NewClientErr(fmt.Errorf("unsupported remote type"),
			fmt.Sprintf("remote name=%s, type=%s, supported types=%v", name, typ, SupportedTypes), nil)
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
		return utils.NewClientErr(ErrInvalidRemoteName, newName, nil)
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
		return utils.NewClientErr(ErrRemoteNotFound, oldName, nil)
	}
}
func saveConfig(conf Config) error {
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
