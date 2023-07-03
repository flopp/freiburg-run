package utils

import (
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func Download(url string, dst string) error {
	err := os.MkdirAll(filepath.Dir(dst), 0770)
	if err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	// temporarily skip insecure certificates
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func DownloadHash(url string, dst, dstDir string) (string, error) {
	if strings.Contains(dst, "HASH") {
		tmpfile, err := os.CreateTemp("", "")
		if err != nil {
			return "", err
		}
		defer os.Remove(tmpfile.Name())

		err = Download(url, tmpfile.Name())
		if err != nil {
			return "", err
		}

		return CopyHash(tmpfile.Name(), dst, dstDir)
	} else {
		dst2 := filepath.Join(dstDir, dst)

		err := Download(url, dst2)
		if err != nil {
			return "", err
		}

		return dst, nil
	}
}

func MustDownloadHash(url string, dst, dstDir string) string {
	res, err := DownloadHash(url, dst, dstDir)
	if err != nil {
		panic(err)
	}
	return res
}
