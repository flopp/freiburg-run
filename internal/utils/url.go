package utils

import "fmt"

type Url string

func (u Url) Join(s string) string {
	return fmt.Sprintf("%s/%s", string(u), s)
}
