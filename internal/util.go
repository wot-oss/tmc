package internal

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// ReadRequiredFile reads the file. Returns expanded absolute representation of the filename and file contents.
// Removes Byte-Order-Mark from the content
func ReadRequiredFile(name string) (string, []byte, error) {
	var log = slog.Default()

	abs, err := filepath.Abs(name)
	if err != nil {
		log.Error("error expanding file name", "filename", name, "error", err)
		return "", nil, err
	}
	log.Debug("reading file", "filename", abs)

	stat, err := os.Stat(abs)
	if err != nil {
		log.Error("error reading file", "filename", abs, "error", err)
		return "", nil, err
	}
	if stat.IsDir() {
		err = errors.New("not a file")
		return "", nil, err
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		log.Error("error reading file", "filename", abs, "error", err)
		return "", nil, err
	}
	raw = removeBOM(raw)
	log.Debug(fmt.Sprintf("read %d bytes from %s beginning with %s", len(raw), abs, string(raw[:100])))
	return abs, raw, nil
}

func removeBOM(bytes []byte) []byte {
	if len(bytes) > 2 && bytes[0] == 0xef && bytes[1] == 0xbb && bytes[2] == 0xbf {
		bytes = bytes[3:]
	}
	return bytes
}
