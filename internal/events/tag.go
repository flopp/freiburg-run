package events

import (
	"fmt"

	"github.com/flopp/freiburg-run/internal/utils"
)

type Tag struct {
	Name        utils.Name
	Description string
	Events      []*Event
	EventsOld   []*Event
	Groups      []*Event
	Shops       []*Event
}

func CreateTag(name string) *Tag {
	return &Tag{utils.NewName(name), "", make([]*Event, 0), make([]*Event, 0), make([]*Event, 0), make([]*Event, 0)}
}

func (tag *Tag) Slug() string {
	return fmt.Sprintf("tag/%s.html", tag.Name.Sanitized)
}

func (tag *Tag) NumEvents() int {
	return NonSeparators(tag.Events)
}

func (tag *Tag) NumOldEvents() int {
	return NonSeparators(tag.EventsOld)
}

func (tag *Tag) NumGroups() int {
	return NonSeparators(tag.Groups)
}

func (tag *Tag) NumShops() int {
	return NonSeparators(tag.Shops)
}

func GetTag(tags map[string]*Tag, name string) *Tag {
	if tag, found := tags[name]; found {
		return tag
	}
	tag := CreateTag(name)
	tags[name] = tag
	return tag
}
