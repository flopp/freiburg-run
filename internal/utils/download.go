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
	wrapErr := func(err error) error {
		return fmt.Errorf("download %s to %s: %w", url, dst, err)
	}

	// Check directory exists before attempting to create file
	err := os.MkdirAll(filepath.Dir(dst), 0770)
	if err != nil {
		return wrapErr(err)
	}

	// Create custom transport with insecure certificates
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// Make the request
	resp, err := client.Get(url)
	if err != nil {
		return wrapErr(err)
	}
	defer resp.Body.Close()

	// Check for HTTP errors before creating the file
	if resp.StatusCode != http.StatusOK {
		return wrapErr(fmt.Errorf("non-ok http status: %v", resp.Status))
	}

	// Create the output file after confirming the download is working
	out, err := os.Create(dst)
	if err != nil {
		return wrapErr(err)
	}
	defer out.Close()

	// Use a buffer for more efficient copying
	buf := make([]byte, 32*1024) // 32KB buffer
	_, err = io.CopyBuffer(out, resp.Body, buf)
	if err != nil {
		return wrapErr(err)
	}

	return nil
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
