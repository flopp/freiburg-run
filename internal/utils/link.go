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
	return Breadcrumbs{Breadcrumb{Link: link, IsLast: true, Position: 1}}
}

func (breadcrumbs Breadcrumbs) Push(link Link) Breadcrumbs {
	res := make(Breadcrumbs, 0, len(breadcrumbs)+1)

	// Copy existing breadcrumbs, marking none as last
	for _, b := range breadcrumbs {
		res = append(res, Breadcrumb{Link: b.Link, IsLast: false, Position: b.Position})
	}

	// Add new breadcrumb as last
	res = append(res, Breadcrumb{Link: link, IsLast: true, Position: len(breadcrumbs) + 1})
	return res
}
