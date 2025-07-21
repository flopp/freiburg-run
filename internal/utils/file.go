package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/flopp/go-filehash"
)

func GetMtime(filePath string) (time.Time, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}, err
	}

	return stat.ModTime(), nil
}

func MakeDir(dir string) error {
	return os.MkdirAll(dir, 0770)
}

func Copy(sourceFileName, targetFileName string) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("copy %s to %s: %w", sourceFileName, targetFileName, err)
	}

	source, err := os.Open(sourceFileName)
	if err != nil {
		return wrapErr(err)
	}
	defer source.Close()

	if err := MakeDir(filepath.Dir(targetFileName)); err != nil {
		return wrapErr(err)
	}

	destination, err := os.Create(targetFileName)
	if err != nil {
		return wrapErr(err)
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		return wrapErr(err)
	}
	return nil
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
