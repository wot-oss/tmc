package remotes

import (
	"fmt"

	"github.com/spf13/viper"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

type Remote interface {
	// Push writes the Thing Model file into the path under root that corresponds to id.
	// Returns ErrTMExists if the same file is already stored with a different timestamp
	Push(id model.TMID, raw []byte) error
	Fetch(id model.TMID) ([]byte, error)
	CreateToC() error
	List(filter string) (model.Toc, error)
}

func Get(name string) (Remote, error) {
	remotesConfig := viper.Get("remotes")
	remotes, ok := remotesConfig.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid remotes contig")
	}
	rc, ok := remotes[name]
	if !ok && name == "" && len(remotes) == 1 {
		for _, v := range remotes {
			rc = v
		}
	}

	remoteConfig, ok := rc.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid config of remote \"%s\"", name)
	}

	switch t := remoteConfig["type"]; t {
	case "file":
		return NewFileRemote(remoteConfig)
	default:
		return nil, fmt.Errorf("unsupported remote type: %v", t)
	}

}
