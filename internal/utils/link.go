package utils

import "strings"

type Link struct {
	Name string
	Url  string
}

func (link Link) IsExternal() bool {
	return strings.HasPrefix(link.Url, "http:") || strings.HasPrefix(link.Url, "https:")
}

func (link Link) IsEmail() bool {
	return strings.HasPrefix(link.Url, "mailto:")
}

type Breadcrumb struct {
	Link     Link
	IsLast   bool
	Position int
}

func InitBreadcrumbs(link Link) []Breadcrumb {
	res := make([]Breadcrumb, 0, 1)
	res = append(res, Breadcrumb{link, true, 1})
	return res
}

func PushBreadcrumb(breadcrumbs []Breadcrumb, link Link) []Breadcrumb {
	res := make([]Breadcrumb, 0, len(breadcrumbs)+1)
	for _, b := range breadcrumbs {
		res = append(res, Breadcrumb{b.Link, false, b.Position})
	}
	res = append(res, Breadcrumb{link, true, len(breadcrumbs) + 1})
	return res
}
