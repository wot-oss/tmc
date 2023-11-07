package commands

import (
	"fmt"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"log/slog"
	"os"
	"path/filepath"
)

func PushToRemote(remoteName string, filename string) {
	log := slog.Default()
	remote, err := remotes.Get(remoteName)
	if err != nil {
		log.Error(fmt.Sprintf("could not ìnitialize a remote instance for %s. check config", remoteName), "error", err)
		os.Exit(1)
	}

	abs, raw := ReadRequiredFile(filename)

	tm, err := ValidateThingModel(raw)
	if err != nil {
		log.Error("validation failed", "error", err)
		os.Exit(1)
	}

	err = remote.Push(tm, filepath.Base(abs), raw)
	if err != nil {
		log.Error("error pushing to remote", "filename", abs, "error", err)
		os.Exit(1)
	}
	log.Info("pushed successfully")

}

// ReadRequiredFile reads the file. Returns expanded absolute representation of the filename and file contents.
// Removes Byte-Order-Mark from the content. Calls os.Exit(1) in case of errors
func ReadRequiredFile(name string) (string, []byte) {
	var log = slog.Default()

	filename := name
	abs, err := filepath.Abs(filename)
	if err != nil {
		log.Error("error expanding file name", "filename", filename, "error", err)
		os.Exit(1)
	}
	log.Debug("importing file", "filename", abs)

	raw, err := os.ReadFile(abs)
	if err != nil {
		log.Error("error reading file", "filename", abs, "error", err)
		os.Exit(1)
	}
	raw = removeBOM(raw)
	log.Debug(fmt.Sprintf("read %d bytes from %s beginning with %s", len(raw), abs, string(raw[:100])))
	return abs, raw
}

func removeBOM(bytes []byte) []byte {
	if len(bytes) > 2 && bytes[0] == 0xef && bytes[1] == 0xbb && bytes[2] == 0xbf {
		bytes = bytes[3:]
	}
	return bytes
}
