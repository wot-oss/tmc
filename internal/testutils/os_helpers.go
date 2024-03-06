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

func CopyDir(from, to string) {
	fds, err := os.ReadDir(from)
	if err != nil {
		fmt.Println(err)
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
				fmt.Println(err)
			}
			CopyDir(src, dst)
		} else {
			CopyFile(src, dst)
		}
	}
}

func CopyFile(from, to string) {
	from, err := filepath.Abs(from)
	if err != nil {
		fmt.Println(err)
	}
	to, err = filepath.Abs(to)
	if err != nil {
		fmt.Println(err)
	}
	err = os.MkdirAll(filepath.Dir(to), dirPermissions)
	if err != nil {
		fmt.Println(err)
	}

	fromF, err := os.OpenFile(from, os.O_RDONLY, 0)
	if err != nil {
		fmt.Println(err)
	}
	defer fromF.Close()

	toF, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE, filePermissions)
	if err != nil {
		fmt.Println(err)
	}
	defer toF.Close()

	_, err = io.Copy(toF, fromF)
	if err != nil {
		fmt.Println(err)
	}
}
