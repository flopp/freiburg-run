package events

import "fmt"

type Tag struct {
	Sanitized   string
	Name        string
	Description string
	Events      []*Event
	EventsOld   []*Event
	Groups      []*Event
	Shops       []*Event
}

func CreateTag(name string) *Tag {
	return &Tag{name, name, "", make([]*Event, 0), make([]*Event, 0), make([]*Event, 0), make([]*Event, 0)}
}

func (tag *Tag) Slug() string {
	return fmt.Sprintf("tag/%s.html", tag.Sanitized)
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
