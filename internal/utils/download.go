package utils

import (
	"crypto/tls"
	"fmt"
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

	// check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-ok http status: %v", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func MustDownload(url string, dst string) {
	err := Download(url, dst)
	Check(err)
}

func DownloadHash(url string, dst string) (string, error) {
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

		return CopyHash(tmpfile.Name(), dst)
	} else {
		return dst, Download(url, dst)
	}
}

func MustDownloadHash(url string, dst string) string {
	res, err := DownloadHash(url, dst)
	Check(err)
	return res
}
