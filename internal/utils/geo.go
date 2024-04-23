package utils

import (
	"math"

	"github.com/flopp/go-compass"
)

func deg2rad(d float64) float64 {
	return d * math.Pi / 180.0
}

func rad2deg(r float64) float64 {
	return r * 180.0 / math.Pi
}

func normalizeAngle(deg float64) float64 {
	deg = math.Mod(deg, 360.0)
	if deg < 0 {
		deg += 360.0
	}
	return deg
}

func DistanceBearing(lat1deg, lon1deg, lat2deg, lon2deg float64) (float64, float64) {
	const earthRadiusKM float64 = 6371.0

	lat1 := deg2rad(lat1deg)
	lon1 := deg2rad(lon1deg)
	lat2 := deg2rad(lat2deg)
	lon2 := deg2rad(lon2deg)

	dlat := lat2 - lat1
	dlon := lon2 - lon1

	a := math.Pow(math.Sin(dlat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(dlon/2), 2)
	distance := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a)) * earthRadiusKM

	y := math.Sin(dlon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) -
		math.Sin(lat1)*math.Cos(lat2)*math.Cos(dlon)
	t := math.Atan2(y, x)

	bearing := normalizeAngle(rad2deg(t))
	return distance, bearing
}

func ApproxDirection(deg float64) string {
	switch compass.GetDirection(deg, compass.Resolution8) {
	case compass.N:
		return "nördl."
	case compass.NE:
		return "nordöstl."
	case compass.E:
		return "östl."
	case compass.SE:
		return "südostl."
	case compass.S:
		return "südl."
	case compass.SW:
		return "südwestl."
	case compass.W:
		return "westl."
	case compass.NW:
		return "nordwestl."
	}
	return "???"
}
