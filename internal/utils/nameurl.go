package utils

import "strings"

type NameUrl struct {
	Name string
	Url  string
}

func CreateNameUrl(name string, url string) NameUrl {
	return NameUrl{
		Name: name,
		Url:  url,
	}
}

func (n NameUrl) IsRegistration() bool {
	return strings.Contains(n.Name, "Anmeldung")
}
