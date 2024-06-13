// SPDX-FileCopyrightText: 2021-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package mobility

import (
	"fmt"
	"math"
	"sort"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
)

// powerFactor relates power to distance in decimal degrees
const powerFactor = 0.001
const MAX_ATTENUATION_DB = 30.0
const VERTICAL_SIDELOBE_ATTENUATION_DB = 30.0

// StrengthAtLocation returns the signal strength at location relative to the specified cell.
func StrengthAtLocation(coord model.Coordinate, cell model.Cell) float64 {
	distAtt := DistanceAttenuation(coord, cell)
	angleAtt := AngleAttenuation(coord, cell)
	pathLoss := GetPathLoss(coord, cell)

	latIdx, lngIdx, inGrid := findGridCell(coord, cell.GridPoints)
	if inGrid {
		fmt.Printf("The point (%.12f, %.12f) is located in the grid cell with indices i: %d, j: %d and the value in faded grid is: %.5f\n", coord.Lat, coord.Lng, latIdx, lngIdx, cell.ShadowingMap[latIdx][lngIdx])
		return cell.TxPowerDB + distAtt + angleAtt - pathLoss - cell.ShadowingMap[latIdx][lngIdx]
	}
	fmt.Printf("The point (%.12f, %.12f) is not located in the grid cell\n", coord.Lat, coord.Lng)
	return cell.TxPowerDB + distAtt + angleAtt - pathLoss

}

// distanceAttenuation is the antenna Gain as a function of the dist
// a very rough approximation to take in to account the width of
// the antenna beam. A 120° wide beam with 30° height will span ≅ 2x0.5 = 1 steradians
// A 60° wide beam will be half that and so will have double the gain
// https://en.wikipedia.org/wiki/Sector_antenna
// https://en.wikipedia.org/wiki/Steradian
func DistanceAttenuation(coord model.Coordinate, cell model.Cell) float64 {
	latDist := coord.Lat - cell.Sector.Center.Lat
	realLngDist := (coord.Lng - cell.Sector.Center.Lng) / utils.AspectRatio(cell.Sector.Center.Lat)
	r := math.Hypot(latDist, realLngDist)
	gain := 120.0 / float64(cell.Sector.Arc)
	return 10 * math.Log10(gain*math.Sqrt(powerFactor/r))
}

// angleAttenuation is the attenuation of power reaching a UE due to its
// position off the centre of the beam in dB
// It is an approximation of the directivity of the antenna
// https://en.wikipedia.org/wiki/Radiation_pattern
// https://en.wikipedia.org/wiki/Sector_antenna
// https://www.etsi.org/deliver/etsi_tr/138900_138999/138901/14.03.00_60/tr_138901v140300p.pdf
// Table 7.3-1: Radiation power pattern of a single antenna element
func AngleAttenuation(coord model.Coordinate, cell model.Cell) float64 {
	azRads := utils.AzimuthToRads(float64(cell.Sector.Azimuth))
	pointRads := math.Atan2(coord.Lat-cell.Sector.Center.Lat, coord.Lng-cell.Sector.Center.Lng)
	angularOffset := math.Abs(azRads - pointRads)
	angleScaling := float64(cell.Sector.Arc) / 120.0 // Compensate for narrower beams

	// We just use a simple linear formula 0 => no loss
	// 33° => -3dB for a 120° sector according to [2]
	// assume this is 1:1 rads:attenuation e.g. 0.50 rads = 0.5 = -3dB attenuation
	//return 10 * math.Log10(1-(angularOffset/math.Pi/angleScaling))
	return -math.Min(12*math.Pow((angularOffset/(math.Pi*2/3)/angleScaling), 2), 30)
}

// ETSI TR 138 901 V16.1.0
// Vertical cut of the radiation power pattern (dB)
// Table 7.3-1: Radiation power pattern of a single antenna element
func zenithAttenuation(zenithAngle int32) float64 {
	halfPowerAngle := 65.0
	angleRatio := float64(zenithAngle-90) / halfPowerAngle
	a := 12 * math.Pow(angleRatio, 2)
	return -math.Min(a, VERTICAL_SIDELOBE_ATTENUATION_DB)
}

// ETSI TR 138 901 V16.1.0
// Horizontal cut of the radiation power pattern (dB)
// Table 7.3-1: Radiation power pattern of a single antenna element
func azimuthAttenuation(azimuth int32) float64 {
	halfPowerAngle := 65.0
	angleRatio := float64(azimuth) / halfPowerAngle
	azAtt := 12 * math.Pow(angleRatio, 2)
	return -math.Min(azAtt, MAX_ATTENUATION_DB)
}

// GetPathLoss calculates the path loss based on the environment and LOS/NLOS conditions
func GetPathLoss(coord model.Coordinate, cell model.Cell) float64 {
	var pathLoss float64

	switch cell.Channel.Environment {
	case "urban":
		if cell.Channel.LOS {
			pathLoss = getUrbanLOSPathLoss(coord, cell)
		} else {
			pathLoss = getUrbanNLOSPathLoss(coord, cell)
		}
	case "rural":
		if cell.Channel.LOS {
			pathLoss = getRuralLOSPathLoss(coord, cell)
		} else {
			pathLoss = getRuralNLOSPathLoss(coord, cell)
		}
	default:
		pathLoss = getFreeSpacePathLoss(coord, cell)
	}

	return pathLoss
}

func getFreeSpacePathLoss(coord model.Coordinate, cell model.Cell) float64 {
	distanceKM := getEuclideanDistanceFromGPS(coord, cell) / 1000
	// Assuming we're using CBRS frequency 3.6 GHz
	// 92.45 is the constant value of 20 * log10(4*pi / c) in Kilometer scale
	pathLoss := 20*math.Log10(distanceKM) + 20*math.Log10(float64(cell.Channel.SSBFrequency)/1000) + 92.45
	return pathLoss
}

// Euclidean distance function
func getEuclideanDistanceFromGPS(coord model.Coordinate, cell model.Cell) float64 {
	earthRadius := 6378.137
	dLat := coord.Lat*math.Pi/180 - cell.Sector.Center.Lat*math.Pi/180
	dLng := coord.Lng*math.Pi/180 - cell.Sector.Center.Lng*math.Pi/180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(coord.Lat*math.Pi/180)*math.Cos(cell.Sector.Center.Lat*math.Pi/180)*
		math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c * 1000 // distance in meters
}

// 3D Euclidean distance function
func get3dEuclideanDistanceFromGPS(coord model.Coordinate, cell model.Cell) float64 {
	d2D := getEuclideanDistanceFromGPS(coord, cell)

	heightUE := float64(1.5)
	heightDiff := math.Abs(float64(cell.Sector.Height) - heightUE)

	// Pythagorean theorem
	d3D := math.Sqrt(math.Pow(d2D, 2) + math.Pow(heightDiff, 2))

	return d3D
}

// Breakpoint distance function
func getBreakpointDistance(cell model.Cell) float64 {
	c := 3.0 * math.Pow(10, 8)
	hBS := float64(cell.Sector.Height)              // base station height
	hUT := float64(1.5)                             // average height of user terminal 1m <= hUT <= 10m
	fc := float64(cell.Channel.SSBFrequency) * 1000 // frequency in Hz

	dBP := 2 * math.Pi * hBS * hUT * fc / c

	return dBP
}

// Breakpoint distance function
func getBreakpointPrimeDistance(cell model.Cell) float64 {
	c := 3.0 * math.Pow(10, 8)
	hE := float64(1)                                // assuming environment height is 1m
	hBS := float64(cell.Sector.Height) - hE         // base station height
	hUT := float64(1.5) - hE                        // average height of user terminal 1m <= hUT <= 10m
	fc := float64(cell.Channel.SSBFrequency) * 1000 // frequency in Hz

	dBP := 4 * hBS * hUT * fc / c

	return dBP
}

// getRuralLOSPathLoss calculates the RMa LOS path loss
func getRuralLOSPathLoss(coord model.Coordinate, cell model.Cell) float64 {
	d2D := getEuclideanDistanceFromGPS(coord, cell)
	d3D := get3dEuclideanDistanceFromGPS(coord, cell)
	dBP := getBreakpointDistance(cell)

	if 10 <= d2D && d2D <= dBP {
		return RmaLOSPL1(cell, d3D)
	} else {
		pl2 := RmaLOSPL1(cell, dBP) + 40*math.Log10(d3D/dBP)
		return pl2
	}
}

// calculates PL1 for RMa LOS path loss
func RmaLOSPL1(cell model.Cell, d float64) float64 {
	fc := float64(cell.Channel.SSBFrequency) / 1000 // frequency in GHz
	h := float64(5)                                 // average building height in m

	pl1 := 20*math.Log10(40*math.Pi*d*fc/3) + math.Min(0.03*math.Pow(h, 1.72), 10)*math.Log10(d) -
		math.Min(0.044*math.Pow(h, 1.72), 14.77) + 0.002*math.Log10(h)*d

	return pl1
}

// getRuralNLOSPathLoss calculates the RMa NLOS path loss
func getRuralNLOSPathLoss(coord model.Coordinate, cell model.Cell) float64 {
	d3D := get3dEuclideanDistanceFromGPS(coord, cell)
	W := float64(20)                   // average street width 5m <= W <= 50m
	h := float64(5)                    // average building height 5m <= h <= 50m
	hBS := float64(cell.Sector.Height) // base station height
	hUT := float64(1.5)                // average height of user terminal 1m <= hUT <= 10m

	plLOS := getRuralLOSPathLoss(coord, cell)
	plNLOS := 161.04 - 7.1*math.Log10(W) + 7.5*math.Log10(h) -
		(24.37-3.7*math.Pow((h/hBS), 2))*math.Log10(hBS) +
		(43.42-3.1*math.Log10(hBS))*(math.Log10(d3D)-3) +
		20*math.Log10(float64(cell.Channel.SSBFrequency)/1000) -
		(math.Pow(3.2*math.Log10(11.75*hUT), 2) - 4.97)

	return math.Max(plLOS, plNLOS)
}

// getUrbanLOSPathLoss calculates the UMa LOS path loss
func getUrbanLOSPathLoss(coord model.Coordinate, cell model.Cell) float64 {
	d2D := getEuclideanDistanceFromGPS(coord, cell)
	d3D := get3dEuclideanDistanceFromGPS(coord, cell)
	dBP := getBreakpointPrimeDistance(cell)
	hBS := float64(cell.Sector.Height)              // base station height
	hUT := float64(5)                               // average height of user terminal 1m <= hUT <= 22.5m
	fc := float64(cell.Channel.SSBFrequency) / 1000 // frequency in GHz

	if 10 <= d2D && d2D <= dBP {
		pl1 := 28.0 + 22*math.Log10(d3D) + 20*math.Log10(fc)
		return pl1
	} else {
		pl2 := 28.0 + 40*math.Log10(d3D) + 20*math.Log10(fc) - 9*math.Log10(math.Pow(dBP, 2)+math.Pow(hBS-hUT, 2))
		return pl2
	}
}

// getUrbanNLOSPathLoss calculates the UMa NLOS path loss
func getUrbanNLOSPathLoss(coord model.Coordinate, cell model.Cell) float64 {
	d3D := get3dEuclideanDistanceFromGPS(coord, cell)
	fc := float64(cell.Channel.SSBFrequency) / 1000 // frequency in GHz
	hUT := float64(5)                               // average height of user terminal 1m <= W <= 22.5m

	plLOS := getUrbanLOSPathLoss(coord, cell)
	plNLOS := 13.54 + 39.08*math.Log10(d3D) + 20*math.Log10(fc) -
		0.6*(hUT-1.5)

	return math.Max(plLOS, plNLOS)
}

// Function to find the unique Latitudes
func uniqueLatitudes(points []model.Coordinate) []float64 {
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

// Function to find the unique Longitudes
func uniqueLongitudes(points []model.Coordinate) []float64 {
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

// Function to find the closest index
func closestIndex(arr []float64, value float64) int {
	closest := 0
	minDist := math.Abs(arr[0] - value)
	for i := 1; i < len(arr); i++ {
		dist := math.Abs(arr[i] - value)
		if dist < minDist {
			closest = i
			minDist = dist
		}
	}
	return closest
}

// Function to check if a point is inside the grid
func isPointInsideGrid(point model.Coordinate, gridPoints []model.Coordinate) bool {
	latitudes := uniqueLatitudes(gridPoints)
	longitudes := uniqueLongitudes(gridPoints)

	// Check if point's latitude is within the range of grid latitudes
	if point.Lat < latitudes[0] || point.Lat > latitudes[len(latitudes)-1] {
		return false
	}

	// Check if point's longitude is within the range of grid longitudes
	if point.Lng < longitudes[0] || point.Lng > longitudes[len(longitudes)-1] {
		return false
	}

	return true
}

// Function to find the grid cell containing the given point
func findGridCell(point model.Coordinate, gridPoints []model.Coordinate) (int, int, bool) {
	if !isPointInsideGrid(point, gridPoints) {
		return -1, -1, false
	}

	latitudes := uniqueLatitudes(gridPoints)
	longitudes := uniqueLongitudes(gridPoints)

	latIdx := closestIndex(latitudes, point.Lat)
	lngIdx := closestIndex(longitudes, point.Lng)

	return latIdx, lngIdx, true
}
