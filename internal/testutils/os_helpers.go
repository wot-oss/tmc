package testutils

import (
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
