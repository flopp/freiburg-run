package utils

import (
	"io"
	"os"
	"path/filepath"

	"github.com/flopp/go-filehash"
)

func MakeDir(dir string) error {
	if err := os.MkdirAll(dir, 0770); err != nil {
		return err
	}
	return nil
}

func MustMakeDir(dir string) {
	err := MakeDir(dir)
	if err != nil {
		panic(err)
	}
}

func Copy(sourceFileName, targetFileName string) error {
	source, err := os.Open(sourceFileName)
	if err != nil {
		return err
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(targetFileName), 0770); err != nil {
		return err
	}

	destination, err := os.Create(targetFileName)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func MustCopy(sourceFileName, targetFileName string) {
	err := Copy(sourceFileName, targetFileName)
	Check(err)
}

func CopyHash(src, dst string) (string, error) {
	return filehash.Copy(src, dst, "HASH")
}

func MustCopyHash(src, dst string) string {
	res, err := CopyHash(src, dst)
	if err != nil {
		panic(err)
	}
	return res
}
