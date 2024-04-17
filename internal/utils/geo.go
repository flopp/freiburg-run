package utils

import (
	"math"
)

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
		return "östl."
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
