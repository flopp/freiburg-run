package utils

import "strings"

type NameUrl struct {
	Name string
	Url  string
}

func (n NameUrl) IsRegistration() bool {
	return strings.Contains(n.Name, "Anmeldung")
}
