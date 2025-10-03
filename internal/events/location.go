package events

import (
	"fmt"
	"regexp"

	"github.com/flopp/freiburg-run/internal/config"
	"github.com/flopp/freiburg-run/internal/utils"
	"github.com/flopp/go-coordsparser"
)

type Location struct {
	City         string
	Country      string
	Geo          string
	Lat          float64
	Lon          float64
	Distance     string
	Direction    string
	DistDirFancy string
}

var reFr = regexp.MustCompile(`\s*^(.*)\s*,\s*FR\s*(🇫🇷)?\s*$`)
var reCh = regexp.MustCompile(`\s*^(.*)\s*,\s*CH\s*(🇨🇭)?\s*$`)

func CreateLocation(config config.Config, locationS, coordinatesS string) Location {
	country := ""
	if m := reFr.FindStringSubmatch(locationS); m != nil {
		country = "Frankreich"
		locationS = m[1]
	} else if m := reCh.FindStringSubmatch(locationS); m != nil {
		country = "Schweiz"
		locationS = m[1]
	}

	lat, lon, err := coordsparser.Parse(coordinatesS)
	coordinates := ""
	distance := ""
	direction := ""
	distDirFancy := ""
	if err == nil {
		coordinates = fmt.Sprintf("%.6f,%.6f", lat, lon)
		d, b := utils.DistanceBearing(config.City.Lat, config.City.Lon, lat, lon)
		distance = fmt.Sprintf("%.1fkm", d)
		direction = utils.ApproxDirection(b)

		distDirFancy = fmt.Sprintf("%s %s von %s", distance, direction, config.City.Name)
	}

	return Location{locationS, country, coordinates, lat, lon, distance, direction, distDirFancy}
}

func (loc Location) Name() string {
	if loc.City == "" {
		return ""
	}
	switch loc.Country {
	case "Frankreich":
		return fmt.Sprintf(`%s, FR 🇫🇷`, loc.City)
	case "Schweiz":
		return fmt.Sprintf(`%s, CH 🇨🇭`, loc.City)
	default:
		return loc.City
	}
}

func (loc Location) NameNoFlag() string {
	if loc.City == "" {
		return ""
	}
	switch loc.Country {
	case "Frankreich":
		return fmt.Sprintf(`%s, FR`, loc.City)
	case "Schweiz":
		return fmt.Sprintf(`%s, CH`, loc.City)
	default:
		return loc.City
	}
}

func (loc Location) HasGeo() bool {
	return loc.Geo != ""
}

func (loc Location) GoogleMaps() string {
	return fmt.Sprintf(`https://www.google.com/maps/place/%s`, loc.Geo)
}

func (loc Location) Tags() []string {
	tags := make([]string, 0)
	if loc.Country != "" {
		tags = append(tags, utils.SanitizeName(loc.Country))
	}
	// tags = append(tags, utils.SplitAndSanitize(loc.City)...)

	return tags
}
