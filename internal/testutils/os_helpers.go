package testutils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	dirPermissions  = 0775
	filePermissions = 0700
)

func CopyDir(from, to string) error {
	fds, err := os.ReadDir(from)
	if err != nil {
		return err
	}
	for _, fd := range fds {
		src := filepath.Join(from, fd.Name())
		dst := filepath.Join(to, fd.Name())

		if fd.IsDir() {
			absDst, err := filepath.Abs(dst)
			if err != nil {
				fmt.Println(err)
			}
			err = os.MkdirAll(absDst, dirPermissions)
			if err != nil {
				return err
			}
			err = CopyDir(src, dst)
			if err != nil {
				return err
			}
		} else {
			err = CopyFile(src, dst)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func CopyFile(from, to string) error {
	from, err := filepath.Abs(from)
	if err != nil {
		return err
	}
	to, err = filepath.Abs(to)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(to), dirPermissions)
	if err != nil {
		return err
	}

	fromF, err := os.OpenFile(from, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer fromF.Close()

	toF, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE, filePermissions)
	if err != nil {
		return err
	}
	defer toF.Close()

	_, err = io.Copy(toF, fromF)
	if err != nil {
		return err
	}
	return nil
}

func CreateDir(rootDir, dir string) error {
	path := filepath.Join(rootDir, dir)
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	err = os.MkdirAll(path, dirPermissions)
	if err != nil {
		return err
	}
	return nil
}

func CreateFile(rootDir, filePath string, content []byte) error {
	path := filepath.Join(rootDir, filePath)
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(path), dirPermissions)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, content, filePermissions)
	if err != nil {
		return err
	}
	return nil
}

// ReplaceStdout temporarily replaces os.Stdout with a buffer and captures the standard output in a string.
// Use `defer restore()` to restore the original os.Stdout. Call getOutput once done writing to standard output.
func ReplaceStdout() (restore func(), getOutput func() string) {
	return replaceFileDescriptor(&os.Stdout)
}

// ReplaceStderr temporarily replaces os.Stderr with a buffer and captures the standard error in a string.
// Use `defer restore()` to restore the original os.Stderr. Call getOutput once done writing to standard error.
func ReplaceStderr() (restore func(), getOutput func() string) {
	return replaceFileDescriptor(&os.Stderr)
}

func replaceFileDescriptor(fd **os.File) (func(), func() string) {
	old := *fd
	restore := func() {
		*fd = old
	}

	rr, w, _ := os.Pipe()
	*fd = w
	out := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rr)
		out <- buf.String()
	}()
	getOutput := func() string {
		_ = w.Close()
		return <-out
	}
	return restore, getOutput
}
