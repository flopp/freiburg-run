package utils

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
)

var geoRe1 = regexp.MustCompile(`^\s*(\d*\.?\d*)\s*,\s*(\d*\.?\d*)\s*$`)
var geoRe2 = regexp.MustCompile(`^\s*N\s*(\d*\.?\d*)\s*E\s*(\d*\.?\d*)\s*$`)

func NormalizeGeo(s string) string {
	m := geoRe1.FindStringSubmatch(s)
	if m != nil {
		return fmt.Sprintf("%s,%s", m[1], m[2])
	}
	m = geoRe2.FindStringSubmatch(s)
	if m != nil {
		return fmt.Sprintf("%s,%s", m[1], m[2])
	}
	return ""
}

func LatLon(s string) (float64, float64, error) {
	m := geoRe1.FindStringSubmatch(s)
	if m != nil {
		lat, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, 0, err
		}
		lon, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			return 0, 0, err
		}
		return lat, lon, nil
	}
	m = geoRe2.FindStringSubmatch(s)
	if m != nil {
		lat, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, 0, err
		}
		lon, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			return 0, 0, err
		}
		return lat, lon, nil
	}
	return 0, 0, fmt.Errorf("cannot parse coordinates: %s", s)
}

func DistanceBearing(lat1deg, lon1deg, lat2deg, lon2deg float64) (float64, float64) {
	const earthRadiusKM float64 = 6371.0

	lat1 := lat1deg * math.Pi / 180.0
	lon1 := lon1deg * math.Pi / 180.0
	lat2 := lat2deg * math.Pi / 180.0
	lon2 := lon2deg * math.Pi / 180.0

	dlat := lat2 - lat1
	dlon := lon2 - lon1

	a := math.Pow(math.Sin(dlat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(dlon/2), 2)
	distance := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a)) * earthRadiusKM

	y := math.Sin(dlon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) -
		math.Sin(lat1)*math.Cos(lat2)*math.Cos(dlon)
	t := math.Atan2(y, x)
	bearing := t * 180.0 / math.Pi
	for bearing < 0 {
		bearing = bearing + 360.0
	}
	for bearing > 360.0 {
		bearing = bearing - 360.0
	}

	return distance, bearing
}

func ApproxDirection(deg float64) string {
	d := 22.5

	if deg <= d {
		return "nördl."
	}
	if deg <= 3*d {
		return "nordöstl."
	}
	if deg <= 5*d {
		return "östli."
	}
	if deg <= 7*d {
		return "südostl."
	}
	if deg <= 9*d {
		return "südl."
	}
	if deg <= 11*d {
		return "südwestl."
	}
	if deg <= 13*d {
		return "westl."
	}
	if deg <= 15*d {
		return "nordwestl."
	}
	return ""
}
