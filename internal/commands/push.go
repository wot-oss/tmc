package commands

import (
	"fmt"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"log/slog"
	"path/filepath"
)

func PushToRemote(remoteName string, filename string) error {
	log := slog.Default()
	remote, err := remotes.Get(remoteName)
	if err != nil {
		log.Error(fmt.Sprintf("could not ìnitialize a remote instance for %s. check config", remoteName), "error", err)
		return err
	}

	abs, raw, err := internal.ReadRequiredFile(filename)
	if err != nil {
		log.Error("couldn't read file", "error", err)
		return err
	}

	tm, err := ValidateThingModel(raw)
	if err != nil {
		log.Error("validation failed", "error", err)
		return err
	}

	err = remote.Push(tm, filepath.Base(abs), raw)
	if err != nil {
		log.Error("error pushing to remote", "filename", abs, "error", err)
		return err
	}
	log.Info("pushed successfully")
	return nil
}
