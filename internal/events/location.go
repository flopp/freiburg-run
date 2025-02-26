package events

import (
	"fmt"
	"regexp"

	"github.com/flopp/freiburg-run/internal/utils"
	"github.com/flopp/go-coordsparser"
)

type Location struct {
	City      string
	Country   string
	Geo       string
	Lat       float64
	Lon       float64
	Distance  string
	Direction string
}

var reFr = regexp.MustCompile(`\s*^(.*)\s*,\s*FR\s*🇫🇷\s*$`)
var reCh = regexp.MustCompile(`\s*^(.*)\s*,\s*CH\s*🇨🇭\s*$`)

func CreateLocation(locationS, coordinatesS string) Location {
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
	if err == nil {
		coordinates = fmt.Sprintf("%.6f,%.6f", lat, lon)

		// Freiburg
		lat0 := 47.996090
		lon0 := 7.849400
		d, b := utils.DistanceBearing(lat0, lon0, lat, lon)
		distance = fmt.Sprintf("%.1fkm", d)
		direction = utils.ApproxDirection(b)
	}

	return Location{locationS, country, coordinates, lat, lon, distance, direction}
}

func (loc Location) Name() string {
	if loc.City == "" {
		return ""
	}
	if loc.Country == "Frankreich" {
		return fmt.Sprintf(`%s, FR 🇫🇷`, loc.City)
	}
	if loc.Country == "Schweiz" {
		return fmt.Sprintf(`%s, CH 🇨🇭`, loc.City)
	}
	return loc.City
}

func (loc Location) NameNoFlag() string {
	if loc.City == "" {
		return ""
	}
	if loc.Country == "Frankreich" {
		return fmt.Sprintf(`%s, FR`, loc.City)
	}
	if loc.Country == "Schweiz" {
		return fmt.Sprintf(`%s, CH`, loc.City)
	}
	return loc.City
}

func (loc Location) HasGeo() bool {
	return loc.Geo != ""
}

func (loc Location) Dir() string {
	return fmt.Sprintf(`%s %s von Freiburg`, loc.Distance, loc.Direction)
}

func (loc Location) DirLong() string {
	return fmt.Sprintf(`%s %s von Freiburg Zentrum`, loc.Distance, loc.Direction)
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
