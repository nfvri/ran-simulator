// SPDX-FileCopyrightText: 2021-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package utils

import (
	"math"
	"sort"

	"github.com/nfvri/ran-simulator/pkg/model"
)

// Earth radius in meters
const earthRadius = 6378100

// See: http://en.wikipedia.org/wiki/Haversine_formula

// Distance returns the distance in meters between two geo coordinates
func Distance(c1 model.Coordinate, c2 model.Coordinate) float64 {
	var la1, lo1, la2, lo2 float64
	la1 = c1.Lat * math.Pi / 180
	lo1 = c1.Lng * math.Pi / 180
	la2 = c2.Lat * math.Pi / 180
	lo2 = c2.Lng * math.Pi / 180

	h := hsin(la2-la1) + math.Cos(la1)*math.Cos(la2)*hsin(lo2-lo1)

	return 2 * earthRadius * math.Asin(math.Sqrt(h))
}

// Euclidean distance function
func GetSphericalDistance(coord1 model.Coordinate, coord2 model.Coordinate) float64 {
	earthRadius := 6378.137
	dLat := coord1.Lat*math.Pi/180 - coord2.Lat*math.Pi/180
	dLng := coord1.Lng*math.Pi/180 - coord2.Lng*math.Pi/180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(coord1.Lat*math.Pi/180)*math.Cos(coord2.Lat*math.Pi/180)*
		math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c * 1000 // distance in meters
}

// TargetPoint returns the target coordinate specified distance and heading from the starting coordinate
func TargetPoint(c model.Coordinate, bearing float64, dist float64) model.Coordinate {
	var la1, lo1, la2, lo2, azimuth, d float64
	la1 = c.Lat * math.Pi / 180
	lo1 = c.Lng * math.Pi / 180
	azimuth = bearing * math.Pi / 180
	d = dist / earthRadius

	la2 = math.Asin(math.Sin(la1)*math.Cos(d) + math.Cos(la1)*math.Sin(d)*math.Cos(azimuth))
	lo2 = lo1 + math.Atan2(math.Sin(azimuth)*math.Sin(d)*math.Cos(la1), math.Cos(d)-math.Sin(la1)*math.Sin(la2))

	return model.Coordinate{Lat: la2 * 180 / math.Pi, Lng: lo2 * 180 / math.Pi}
}

// InitialBearing returns initial bearing from c1 to c2
func InitialBearing(c1 model.Coordinate, c2 model.Coordinate) float64 {
	var la1, lo1, la2, lo2 float64
	la1 = c1.Lat * math.Pi / 180
	lo1 = c1.Lng * math.Pi / 180
	la2 = c2.Lat * math.Pi / 180
	lo2 = c2.Lng * math.Pi / 180

	y := math.Sin(lo2-lo1) * math.Cos(la2)
	x := math.Cos(la1)*math.Sin(la2) - math.Sin(la1)*math.Cos(la2)*math.Cos(lo2-lo1)
	theta := math.Atan2(y, x)
	return math.Mod(theta*180/math.Pi+360, 360.0) // in degrees
}

func hsin(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
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

// Convert meters to degrees latitude
func MetersToLatDegrees(meters float64) float64 {
	return meters / 111132.954
}

// Convert meters to degrees longitude at a specific latitude
func MetersToLngDegrees(meters, latitude float64) float64 {
	return meters / (111132.954 * AspectRatio(latitude))
}

// AzimuthToRads - angle measured in degrees clockwise from north, expressed in rads from 3 o'clock anticlockwise
func AzimuthToRads(azimuth float64) float64 {
	if azimuth == 90 {
		return 0
	}
	return DegreesToRads(90 - azimuth)
}

// DegreesToRads - general conversion of degrees to rads, both starting at 3 o'clock going anticlockwise
func DegreesToRads(degrees float64) float64 {
	return 2 * math.Pi * degrees / 360
}

// AspectRatio - Compensate for the narrowing of meridians at higher latitudes
func AspectRatio(latitude float64) float64 {
	return math.Cos(DegreesToRads(latitude))
}

// Function to find the unique Latitudes
func UniqueLatitudes(points []model.Coordinate) []float64 {
	unique := make(map[float64]struct{})
	for _, point := range points {
		unique[point.Lat] = struct{}{}
	}

	latitudes := make([]float64, 0, len(unique))
	for k := range unique {
		latitudes = append(latitudes, k)
	}
	sort.Float64s(latitudes)
	return latitudes
}

// Function to find the Latitudes
func Latitudes(gridPoints []model.Coordinate) []float64 {
	size := int(math.Sqrt(float64(len(gridPoints))))
	lats := make([]float64, 0)
	for i := 0; i < len(gridPoints); i += size {
		lats = append(lats, gridPoints[i].Lat)
	}

	return lats
}

// Function to find the Longitudes
func Longitudes(gridPoints []model.Coordinate) []float64 {
	size := int(math.Sqrt(float64(len(gridPoints))))
	lngs := make([]float64, 0)
	for i := 0; i < size; i++ {
		lngs = append(lngs, gridPoints[i].Lng)
	}
	return lngs
}

// Function to find the unique Longitudes
func UniqueLongitudes(points []model.Coordinate) []float64 {
	unique := make(map[float64]struct{})
	for _, point := range points {
		unique[point.Lng] = struct{}{}
	}

	longitudes := make([]float64, 0, len(unique))
	for k := range unique {
		longitudes = append(longitudes, k)
	}
	sort.Float64s(longitudes)
	return longitudes
}
