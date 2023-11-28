package config

import (
	"os"
	"path/filepath"
)

var HomeDir string
var DefaultConfigDir string

func init() {
	var err error
	HomeDir, err = os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	DefaultConfigDir = filepath.Join(HomeDir, ".tm-catalog")
}
