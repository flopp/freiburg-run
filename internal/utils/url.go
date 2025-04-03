package utils

import "fmt"

type Url string

func (u Url) Join(s string) string {
	if len(s) == 0 {
		return string(u)
	}
	return fmt.Sprintf("%s/%s", string(u), s)
}
