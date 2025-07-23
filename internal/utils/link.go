package utils

import "strings"

type Link struct {
	Name string
	Url  string
}

func CreateLink(name string, url string) *Link {
	return &Link{
		Name: name,
		Url:  url,
	}
}

func stripUrl(url string) string {
	if len(url) == 0 {
		return ""
	}

	// Remove the scheme (http:// or https://)
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")

	// Remove the "www." prefix unconditionally
	url = strings.TrimPrefix(url, "www.")

	// remove trailing hash
	if idx := strings.Index(url, "#"); idx != -1 {
		url = url[:idx]
	}
	// remove trailing query parameters
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}

	// remove trailing index.html/htm/php
	url = strings.TrimSuffix(url, "index.html")
	url = strings.TrimSuffix(url, "index.htm")
	url = strings.TrimSuffix(url, "index.php")

	// Remove trailing "/" if it exists
	url = strings.TrimSuffix(url, "/")

	return url
}

func CreateUnnamedLink(url string) *Link {
	return &Link{
		Name: stripUrl(url),
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
	Link     *Link
	IsLast   bool
	Position int
}

type Breadcrumbs []Breadcrumb

func InitBreadcrumbs(link *Link) Breadcrumbs {
	return Breadcrumbs{Breadcrumb{Link: link, IsLast: true, Position: 1}}
}

func (breadcrumbs Breadcrumbs) Push(links ...*Link) Breadcrumbs {
	res := make(Breadcrumbs, 0, len(breadcrumbs)+len(links))

	// Copy existing breadcrumbs, marking none as last
	for _, b := range breadcrumbs {
		res = append(res, Breadcrumb{Link: b.Link, IsLast: false, Position: b.Position})
	}

	// Add new breadcrumbs
	for i, link := range links {
		isLast := i == len(links)-1
		res = append(res, Breadcrumb{Link: link, IsLast: isLast, Position: len(breadcrumbs) + i + 1})
	}

	return res
}
