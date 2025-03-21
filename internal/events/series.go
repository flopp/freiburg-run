package events

import (
	"fmt"
	"html/template"

	"github.com/flopp/freiburg-run/internal/utils"
)

type Serie struct {
	Sanitized   string
	Name        string
	Description template.HTML
	Links       []utils.NameUrl
	Events      []*Event
	EventsOld   []*Event
	Groups      []*Event
	Shops       []*Event
}

func (s Serie) IsOld() bool {
	return len(s.Events) == 0 && len(s.Groups) == 0 && len(s.Shops) == 0
}

func (s Serie) Num() int {
	return NonSeparators(s.Events) + NonSeparators(s.EventsOld) + NonSeparators(s.Groups) + NonSeparators(s.Shops)
}

func CreateSerie(id string, name string) *Serie {
	return &Serie{id, name, "", make([]utils.NameUrl, 0), make([]*Event, 0), make([]*Event, 0), make([]*Event, 0), make([]*Event, 0)}
}

func (serie *Serie) Slug() string {
	return fmt.Sprintf("serie/%s.html", serie.Sanitized)
}

func (serie *Serie) ImageSlug() string {
	return fmt.Sprintf("serie/%s.png", serie.Sanitized)
}

func GetSerie(series map[string]*Serie, name string) *Serie {
	id := utils.SanitizeName(name)
	if s, found := series[id]; found {
		return s
	}
	serie := CreateSerie(id, name)
	series[id] = serie
	return serie
}
