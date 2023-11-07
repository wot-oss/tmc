package remotes

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"log/slog"
	"os"
)

type Remote interface {
	Push(model *model.ThingModel, filename string, raw []byte) error
}

func Get(name string) (Remote, error) {
	//fixme: read viper config instead
	remotesConfig := viper.Get("remotes")
	remotes, ok := remotesConfig.(map[string]any)
	if !ok {
		slog.Default().Error("invalid remotes config")
		os.Exit(1)
	}
	rc, ok := remotes[name]
	if !ok && name == "" && len(remotes) == 1 {
		for _, v := range remotes {
			rc = v
		}
	}

	remoteConfig, ok := rc.(map[string]any)
	if !ok {
		slog.Default().Error("invalid remotes config")
		os.Exit(1)
	}

	switch t := remoteConfig["type"]; t {
	case "file":
		return NewFileRemote(remoteConfig)
	default:
		return nil, fmt.Errorf("unsupported remote type: %v", t)
	}

}
