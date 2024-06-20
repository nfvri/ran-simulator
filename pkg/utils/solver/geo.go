package solver

import (
	"math"
	"sort"

	"github.com/nfvri/ran-simulator/pkg/model"
)

const earthRadiusKm = 6371 // Earth radius in kilometers

// GenerateEquidistPoints generates points around a central coordinate at a given radius.
func GenerateEquidistPoints(lat, lng, radiusKm float64, numPoints int) []model.Coordinate {
	points := make([]model.Coordinate, numPoints)
	latRad := toRadians(lat)
	lngRad := toRadians(lng)

	for i := 0; i < numPoints; i++ {
		angle := float64(i) * (360.0 / float64(numPoints))
		angleRad := toRadians(angle)

		newLatRad := math.Asin(math.Sin(latRad)*math.Cos(radiusKm/earthRadiusKm) +
			math.Cos(latRad)*math.Sin(radiusKm/earthRadiusKm)*math.Cos(angleRad))

		newLngRad := lngRad + math.Atan2(math.Sin(angleRad)*math.Sin(radiusKm/earthRadiusKm)*math.Cos(latRad),
			math.Cos(radiusKm/earthRadiusKm)-math.Sin(latRad)*math.Sin(newLatRad))

		newLat := toDegrees(newLatRad)
		newLng := toDegrees(newLngRad)

		points[i] = model.Coordinate{Lat: newLat, Lng: newLng}
	}

	return points
}

// toRadians converts degrees to radians.
func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

// toDegrees converts radians to degrees.
func toDegrees(radians float64) float64 {
	return radians * 180.0 / math.Pi
}

// CalculateBearing calculates the bearing angle between two geographical coordinates
func CalculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
	lat1 = toRadians(lat1)
	lon1 = toRadians(lon1)
	lat2 = toRadians(lat2)
	lon2 = toRadians(lon2)

	dLon := lon2 - lon1

	x := math.Sin(dLon) * math.Cos(lat2)
	y := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLon)

	initialBearing := math.Atan2(x, y)
	initialBearing = toDegrees(initialBearing)

	// Normalize to 0-360°
	bearing := math.Mod(initialBearing+360, 360)

	return bearing
}

// SortCoordinatesByBearing sorts a list of coordinates by their bearing angle relative to a center coordinate
func SortCoordinatesByBearing(center model.Coordinate, coordinates []model.Coordinate) []model.Coordinate {
	type BearingCoordinate struct {
		Bearing    float64
		Coordinate model.Coordinate
	}

	bearings := make([]BearingCoordinate, len(coordinates))

	for i, coord := range coordinates {
		bearing := CalculateBearing(center.Lat, center.Lng, coord.Lat, coord.Lng)
		bearings[i] = BearingCoordinate{Bearing: bearing, Coordinate: coord}
	}

	// Sort by the bearing angle
	sort.Slice(bearings, func(i, j int) bool {
		return bearings[i].Bearing < bearings[j].Bearing
	})

	// Extract the sorted coordinates
	sortedCoords := make([]model.Coordinate, len(coordinates))
	for i, bc := range bearings {
		sortedCoords[i] = bc.Coordinate
	}

	return sortedCoords
}
