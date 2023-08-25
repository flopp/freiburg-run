package utils

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
)

type Added struct {
	events map[string]string
	groups map[string]string
	shops  map[string]string
}

func (added *Added) SetAdded(t string, name string, date string) error {
	if t == "event" {
		added.events[name] = date
	} else if t == "group" {
		added.groups[name] = date
	} else if t == "shop" {
		added.shops[name] = date
	} else {
		return fmt.Errorf("invalid event type: %s", t)
	}
	return nil
}

func (added Added) GetAdded(t string, name string) (string, error) {
	if t == "event" {
		if date, ok := added.events[name]; ok {
			return date, nil
		} else {
			return "", nil
		}
	} else if t == "group" {
		if date, ok := added.groups[name]; ok {
			return date, nil
		} else {
			return "", nil
		}
	} else if t == "shop" {
		if date, ok := added.shops[name]; ok {
			return date, nil
		} else {
			return "", nil
		}
	}

	return "", fmt.Errorf("invalid event type: %s", t)
}

func ReadAdded(fileName string) (*Added, error) {
	added := &Added{make(map[string]string), make(map[string]string), make(map[string]string)}
	f, err := os.Open(fileName)
	if err != nil {
		return added, err
	}
	defer f.Close()

	fileScanner := bufio.NewScanner(f)
	fileScanner.Split(bufio.ScanLines)

	r := regexp.MustCompile(`^([^\t]+)\t([^\t]+)\t([^\t]+)\s*$`)
	for fileScanner.Scan() {
		line := fileScanner.Text()
		if match := r.FindStringSubmatch(line); match != nil {
			err = added.SetAdded(match[1], match[2], match[3])
			if err != nil {
				log.Printf("%s: cannot process line <%s> - %v", fileName, line, err)
			}
		} else {
			log.Printf("%s: cannot parse line <%s>", fileName, line)
		}
	}
	return added, nil
}

func (added Added) Write(fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	for name, date := range added.events {
		f.WriteString(fmt.Sprintf("event\t%s\t%s\n", name, date))
	}

	for name, date := range added.groups {
		f.WriteString(fmt.Sprintf("group\t%s\t%s\n", name, date))
	}

	for name, date := range added.shops {
		f.WriteString(fmt.Sprintf("shop\t%s\t%s\n", name, date))
	}

	return nil
}
