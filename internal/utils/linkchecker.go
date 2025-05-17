package utils

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

type LinkChecker struct {
	client       *http.Client
	linksChecked int
	issuesFound  int
}

func NewLinkChecker() *LinkChecker {
	return &LinkChecker{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		linksChecked: 0,
		issuesFound:  0,
	}
}

// Check validates that the given URL is reachable (by sending a GET request).
// It returns nil if the link is valid, otherwise an appropriate error.
func (lc *LinkChecker) Check(url string) error {
	lc.linksChecked++

	// check that the url starts with http or https
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		lc.issuesFound++
		return fmt.Errorf("invalid URL (no http:// or https://)")
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		lc.issuesFound++
		return err
	}
	req.Header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36")

	resp, err := lc.client.Do(req)
	if err != nil {
		lc.issuesFound++
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		lc.issuesFound++
		return fmt.Errorf("invalid URL (status code %d)", resp.StatusCode)
	}

	return nil
}

// Stats returns the number of links checked and the number of issues found.
func (lc *LinkChecker) Stats() (int, int) {
	return lc.linksChecked, lc.issuesFound
}
