package utils

import "strings"

type Link struct {
	Name string
	Url  string
}

func CreateLink(name string, url string) Link {
	return Link{
		Name: name,
		Url:  url,
	}
}

func (link Link) IsExternal() bool {
	return strings.HasPrefix(link.Url, "http:") || strings.HasPrefix(link.Url, "https:")
}

func (link Link) IsEmail() bool {
	return strings.HasPrefix(link.Url, "mailto:")
}

func (link Link) IsRegistration() bool {
	return strings.Contains(link.Name, "Anmeldung")
}

type Breadcrumb struct {
	Link     Link
	IsLast   bool
	Position int
}

type Breadcrumbs []Breadcrumb

func InitBreadcrumbs(link Link) Breadcrumbs {
	res := make([]Breadcrumb, 0, 1)
	res = append(res, Breadcrumb{link, true, 1})
	return res
}

func (breadcrumbs Breadcrumbs) Push(link Link) Breadcrumbs {
	res := make([]Breadcrumb, 0, len(breadcrumbs)+1)
	for _, b := range breadcrumbs {
		res = append(res, Breadcrumb{b.Link, false, b.Position})
	}
	res = append(res, Breadcrumb{link, true, len(breadcrumbs) + 1})
	return res
}
