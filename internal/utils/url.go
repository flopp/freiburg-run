package utils

import "fmt"

type Url string

func (u Url) Join(s string) string {
	if len(s) == 0 {
		return string(u)
	}
	return fmt.Sprintf("%s/%s", string(u), s)
}

func ExtractDomain(url string) string {
	if len(url) == 0 {
		return ""
	}

	// Remove the scheme (http:// or https://) if present
	if len(url) > 7 && url[:7] == "http://" {
		url = url[7:]
	} else if len(url) > 8 && url[:8] == "https://" {
		url = url[8:]
	}

	// find the first slash or question mark or hash
	// and remove everything after it
	for idx := 0; idx < len(url); idx++ {
		if url[idx] == '/' || url[idx] == '?' || url[idx] == '#' {
			url = url[:idx]
			break
		}
	}

	return url
}
