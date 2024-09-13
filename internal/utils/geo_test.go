package utils

import (
	"testing"
)

func similar(x, y, delta float64) bool {
	return x >= y-delta && x <= y+delta
}

func TestDistanceBearing(t *testing.T) {
	testCases := []struct {
		lat1, lng1       float64
		lat2, lng2       float64
		expectedDistance float64
		expectedBearing  float64
	}{
		{48.0, 7.0, 49.0, 8.0, 133, 33},
		{48.0, 7.0, 48.0, 7.0, 0, 0},
		{48.0, 7.0, 48.0, 8.0, 74, 90},
		{48.0, 7.0, 48.0, 6.0, 74, 270},
		{48.0, 7.0, 47.0, 7.0, 111, 180},
		{47.0, 7.0, 48.0, 7.0, 111, 0},
	}

	for _, tc := range testCases {
		distance, bearing := DistanceBearing(tc.lat1, tc.lng1, tc.lat2, tc.lng2)
		if !similar(distance, tc.expectedDistance, 1.0) || !similar(bearing, tc.expectedBearing, 1.0) {
			t.Errorf("TC=%v dist=%v bear=%v", tc, distance, bearing)
		}
	}
}
