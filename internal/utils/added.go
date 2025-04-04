package utils

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
)

type Added struct {
	items map[string]string
}

func (added *Added) SetAdded(t string, name string, date string) {
	key := fmt.Sprintf("%s:%s", t, name)
	added.items[key] = date
}

func (added Added) GetAdded(t string, name string) string {
	key := fmt.Sprintf("%s:%s", t, name)
	if date, ok := added.items[key]; ok {
		return date
	}
	return ""
}

var reLine = regexp.MustCompile(`^([^\t]+)\t([^\t]+)\t([^\t]+)\s*$`)

func ReadAdded(fileName string) (*Added, error) {
	added := &Added{make(map[string]string)}
	f, err := os.Open(fileName)
	if err != nil {
		return added, err
	}
	defer f.Close()

	fileScanner := bufio.NewScanner(f)
	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		line := fileScanner.Text()
		if match := reLine.FindStringSubmatch(line); match != nil {
			added.SetAdded(match[1], match[2], match[3])
		} else {
			return nil, fmt.Errorf("%s: cannot parse line <%s>", fileName, line)
		}
	}
	return added, nil
}

var reKey = regexp.MustCompile(`:`)

func (added Added) Write(fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	for name, date := range added.items {
		// Split the key into type and name
		parts := reKey.Split(name, 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid key format: %s", name)
		}
		entryType := parts[0]
		name = parts[1]
		_, err := fmt.Fprintf(f, "%s\t%s\t%s\n", entryType, name, date)
		if err != nil {
			return err
		}
	}

	return nil
}
