//go:build windows

package utils

import (
	"bytes"
	"os"
	"path/filepath"
)

func convertToNativeLineEndings(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	return bytes.ReplaceAll(b, []byte{'\n'}, []byte{'\r', '\n'})
}

func atomicWriteFile(name string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(name)
	temp, err := os.CreateTemp(dir, filepath.Base(name)+".*.temp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	err = os.WriteFile(temp.Name(), data, perm)
	_ = temp.Close()
	if err != nil {
		return err
	}
	return os.Rename(temp.Name(), name)
}
