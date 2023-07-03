package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func MustCopyHash(src, dst, dstDir string) string {
	res, err := CopyHash(src, dst, dstDir)
	if err != nil {
		panic(err)
	}
	return res
}

func CopyHash(src, dst, dstDir string) (string, error) {
	dir := filepath.Join(dstDir, filepath.Dir(dst))
	err := os.MkdirAll(dir, 0770)
	if err != nil {
		return "", err
	}

	hash, err := ComputeHash(src)
	if err != nil {
		return "", err
	}

	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return "", err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer source.Close()

	dstHash := strings.Replace(dst, "HASH", hash, -1)
	dstHash2 := filepath.Join(dstDir, dstHash)
	destination, err := os.Create(dstHash2)
	if err != nil {
		return "", err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	if err != nil {
		return "", err
	}

	return dstHash, nil
}

func ComputeHash(fileName string) (string, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%.8x", h.Sum(nil)), nil
}
