//go:build !windows

package utils

import (
	"os"

	"github.com/google/renameio"
)

func convertToNativeLineEndings(b []byte) []byte {
	return b
}

func atomicWriteFile(name string, data []byte, perm os.FileMode) error {
	return renameio.WriteFile(name, data, perm)
}
