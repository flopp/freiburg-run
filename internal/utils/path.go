package utils

import "path/filepath"

type Path string

func NewPath(path string) Path {
	return Path(path)
}

func (p Path) String() string {
	return string(p)
}

func (p Path) Join(s string) string {
	if len(s) == 0 {
		return string(p)
	}
	return filepath.Join(string(p), s)
}
