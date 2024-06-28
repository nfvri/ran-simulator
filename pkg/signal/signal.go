// SPDX-FileCopyrightText: 2021-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package signal

import (
	"fmt"
	"math"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	log "github.com/sirupsen/logrus"
)

// powerFactor relates power to distance in decimal degrees
const powerFactor = 0.001

// StrengthAtLocation returns the signal strength at location relative to the specified cell.
func StrengthAtLocation(coord model.Coordinate, height float64, cell model.Cell) float64 {
	if math.IsNaN(coord.Lat) || math.IsNaN(coord.Lng) {
		err := fmt.Errorf("WTF? lat:%v lng:%v", coord.Lat, coord.Lng)
		panic(err)
	}
	strengthAfterPathloss := StrengthAfterPathloss(coord, height, cell)

	latIdx, lngIdx, inGrid := FindGridCell(coord, cell.GridPoints)
	if inGrid {
		log.Debugf("The point (%.12f, %.12f) is located in the grid cell %v with indices i: %d, j: %d and the value in faded grid is: %.5f\n", coord.Lat, coord.Lng, cell.NCGI, latIdx, lngIdx, cell.ShadowingMap[latIdx][lngIdx])
		return strengthAfterPathloss - cell.ShadowingMap[latIdx][lngIdx]
	}
	log.Debugf("The point (%.12f, %.12f) is not located in the grid cell\n", coord.Lat, coord.Lng)
	log.Debugf("strengthAfterPathloss: %v", strengthAfterPathloss)
	return strengthAfterPathloss

}

func StrengthAfterPathloss(coord model.Coordinate, height float64, cell model.Cell) float64 {
	angleAtt := angularAttenuation(coord, height, cell)
	pathLoss := GetPathLoss(coord, height, cell)
	log.Debugf("\ncell.TxPowerDB: %v \ncell.Beam.MaxGain:%v \nangleAtt:%v \npathLoss:%v \n", cell.TxPowerDB, cell.Beam.MaxGain, angleAtt, pathLoss)

	return cell.TxPowerDB + cell.Beam.MaxGain + angleAtt - pathLoss
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

// https://www.etsi.org/deliver/etsi_tr/138900_138999/138901/17.00.00_60/tr_138901v170000p.pdf
// Table 7.3-1: Radiation power pattern of a single antenna element
func angularAttenuation(coord model.Coordinate, height float64, cell model.Cell) float64 {
	log.Debug("\n======================================\n")
	azRads := utils.AzimuthToRads(cell.Sector.Azimuth)
	ueAngle := utils.GetRotationDegrees(
		&types.Point{
			Lat: cell.Sector.Center.Lat,
			Lng: cell.Sector.Center.Lng,
		},
		&types.Point{
			Lat: coord.Lat,
			Lng: coord.Lng,
		})

	ueAngleRads := utils.DegreesToRads(ueAngle)
	azimuthOffsetRads := math.Abs(azRads - ueAngleRads)
	azimuthOffset := azimuthOffsetRads * (180 / math.Pi)
	if azimuthOffset > 180 {
		azimuthOffset = 360 - azimuthOffset
	}
	horizontalCut := azimuthAttenuation(azimuthOffset, cell.Beam.H3dBAngle, cell.TxPowerDB)
	hAttformat := "\nhorizontalCut: %v \ncell.Sector.Azimuth: %v \nazpoint: %v \nazimuthOffset: %v \nazRads: %v \nazpointRads: %v"
	log.Debugf(hAttformat, horizontalCut, cell.Sector.Azimuth, ueAngleRads*(180/math.Pi), azimuthOffset, azRads, ueAngleRads)

	log.Debug("\n======================================\n")
	zenithAngle := calcZenithAngle(coord, height, cell)
	verticalCut := zenithAttenuation(zenithAngle, cell.Beam.V3dBAngle, cell.Beam.VSideLobeAttenuationDB)
	log.Debugf("\nverticalCut: %v \nzenithAngle: %v", verticalCut, zenithAngle)
	log.Debug("\n======================================\n")
	return -math.Min(-(verticalCut + horizontalCut), cell.TxPowerDB)
}

func calcZenithAngle(coord model.Coordinate, height float64, cell model.Cell) float64 {

	d2D := getSphericalDistance(coord, cell) // assume small error for small distances
	d3D := get3dEuclideanDistanceFromGPS(coord, height, cell)

	hBS := cell.Sector.Height

	var ueAngleSign float64
	if hBS >= int32(height) {
		ueAngleSign = 1
	} else {
		ueAngleSign = -1
	}
	ueAngleRads := math.Acos(d2D / d3D)
	zUERads := 90*(math.Pi/180) + ueAngleSign*ueAngleRads

	zTilt := 90 + cell.Sector.Tilt
	zTiltRads := zTilt * (math.Pi / 180)
	zAngleOffset := math.Abs(zUERads - zTiltRads)

	zenithAngle := 90 + zAngleOffset*(180/math.Pi)
	if zenithAngle > 180 {
		zenithAngle = 360 - zenithAngle
	}
	log.Debugf("\nueAngle:%v \nzUE: %v \nztilt: %v", ueAngleRads*(180/math.Pi), zUERads*(180/math.Pi), zTilt)
	log.Debugf("\nzAngleOffset: %v", zAngleOffset*(180/math.Pi))
	return zenithAngle
}

// ETSI TR 138 901 V16.1.0
// Vertical cut of the radiation power pattern (dB)
// Table 7.3-1: Radiation power pattern of a single antenna element
func zenithAttenuation(zenithAngle, theta3dB float64, slav float64) float64 {
	angleRatio := (zenithAngle - 90) / theta3dB
	a := 12 * math.Pow(angleRatio, 2)
	return -math.Min(a, slav)
}

// ETSI TR 138 901 V16.1.0
// Horizontal cut of the radiation power pattern (dB)
// Table 7.3-1: Radiation power pattern of a single antenna element
func azimuthAttenuation(azimuthAngle, phi3dB float64, aMax float64) float64 {
	angleRatio := azimuthAngle / phi3dB
	azAtt := 12 * math.Pow(angleRatio, 2)
	return -math.Min(azAtt, aMax)
}

// GetPathLoss calculates the path loss based on the environment and LOS/NLOS conditions
func GetPathLoss(coord model.Coordinate, height float64, cell model.Cell) float64 {
	var pathLoss float64

	switch cell.Channel.Environment {
	case "urban":
		if cell.Channel.LOS {
			pathLoss = getUrbanLOSPathLoss(coord, height, cell)
		} else {
			pathLoss = getUrbanNLOSPathLoss(coord, height, cell)
		}
	case "rural":
		if cell.Channel.LOS {
			pathLoss = getRuralLOSPathLoss(coord, height, cell)
		} else {
			pathLoss = getRuralNLOSPathLoss(coord, height, cell)
		}
	default:
		pathLoss = getRuralNLOSPathLoss(coord, height, cell)
	}

	return pathLoss
}

func getFreeSpacePathLoss(coord model.Coordinate, cell model.Cell) float64 {
	distanceKM := getSphericalDistance(coord, cell) / 1000
	// Assuming we're using CBRS frequency 3.6 GHz
	// 92.45 is the constant value of 20 * log10(4*pi / c) in Kilometer scale
	pathLoss := 20*math.Log10(distanceKM) + 20*math.Log10(float64(cell.Channel.SSBFrequency)/1000) + 92.45
	return pathLoss
}

// Euclidean distance function
func getSphericalDistance(coord model.Coordinate, cell model.Cell) float64 {
	earthRadius := 6378.137
	dLat := coord.Lat*math.Pi/180 - cell.Sector.Center.Lat*math.Pi/180
	dLng := coord.Lng*math.Pi/180 - cell.Sector.Center.Lng*math.Pi/180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(coord.Lat*math.Pi/180)*math.Cos(cell.Sector.Center.Lat*math.Pi/180)*
		math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c * 1000 // distance in meters
}

// 3D Euclidean distance function
func get3dEuclideanDistanceFromGPS(coord model.Coordinate, height float64, cell model.Cell) float64 {
	d2D := getSphericalDistance(coord, cell)

	heightUE := height
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
	hE := float64(1)                                   // assuming environment height is 1m
	hBS := float64(cell.Sector.Height) - hE            // base station height
	hUT := float64(1.5) - hE                           // average height of user terminal 1m <= hUT <= 10m
	fc := float64(cell.Channel.SSBFrequency) * 1000000 // frequency in Hz

	numer := (4 * hBS * hUT * fc)
	log.Debugf("\ncell.Channel.SSBFrequency:%v ssbMhz:%v \nfc: %v numer:%v", cell.Channel.SSBFrequency, float64(cell.Channel.SSBFrequency), fc, numer)
	dBP := numer / c

	return dBP
}

// getRuralLOSPathLoss calculates the RMa LOS path loss
func getRuralLOSPathLoss(coord model.Coordinate, height float64, cell model.Cell) float64 {
	d2D := getSphericalDistance(coord, cell)
	d3D := get3dEuclideanDistanceFromGPS(coord, height, cell)
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
func getRuralNLOSPathLoss(coord model.Coordinate, height float64, cell model.Cell) float64 {
	d3D := get3dEuclideanDistanceFromGPS(coord, height, cell)
	W := float64(20)                   // average street width 5m <= W <= 50m
	h := float64(5)                    // average building height 5m <= h <= 50m
	hBS := float64(cell.Sector.Height) // base station height
	hUT := height                      // average height of user terminal 1m <= hUT <= 10m

	plLOS := getRuralLOSPathLoss(coord, height, cell)
	plNLOS := 161.04 - 7.1*math.Log10(W) + 7.5*math.Log10(h) -
		(24.37-3.7*math.Pow((h/hBS), 2))*math.Log10(hBS) +
		(43.42-3.1*math.Log10(hBS))*(math.Log10(d3D)-3) +
		20*math.Log10(float64(cell.Channel.SSBFrequency)/1000) -
		(math.Pow(3.2*math.Log10(11.75*hUT), 2) - 4.97)

	return math.Max(plLOS, plNLOS)
}

// getUrbanLOSPathLoss calculates the UMa LOS path loss
func getUrbanLOSPathLoss(coord model.Coordinate, height float64, cell model.Cell) float64 {
	d2D := getSphericalDistance(coord, cell)
	d3D := get3dEuclideanDistanceFromGPS(coord, height, cell)
	dBP := getBreakpointPrimeDistance(cell)
	hBS := float64(cell.Sector.Height)              // base station height
	hUT := height                                   // average height of user terminal 1m <= hUT <= 22.5m
	fc := float64(cell.Channel.SSBFrequency) / 1000 // frequency in GHz

	log.Debugf("\ndBP:%v \nd2D:%v", dBP, d2D)
	if 10 <= d2D && d2D <= dBP {
		pl1 := 28.0 + 22*math.Log10(d3D) + 20*math.Log10(fc)
		log.Debugf("\npl1:%v", pl1)
		return pl1
	} else {
		pl2 := 28.0 + 40*math.Log10(d3D) + 20*math.Log10(fc) - 9*math.Log10(math.Pow(dBP, 2)+math.Pow(hBS-hUT, 2))
		log.Debugf("\npl2:%v", pl2)
		return pl2
	}

}

// getUrbanNLOSPathLoss calculates the UMa NLOS path loss
func getUrbanNLOSPathLoss(coord model.Coordinate, height float64, cell model.Cell) float64 {
	d3D := get3dEuclideanDistanceFromGPS(coord, height, cell)
	fc := float64(cell.Channel.SSBFrequency) / 1000 // frequency in GHz
	hUT := height                                   // average height of user terminal 1m <= W <= 22.5m

	plLOS := getUrbanLOSPathLoss(coord, height, cell)
	plNLOS := 13.54 + 39.08*math.Log10(d3D) + 20*math.Log10(fc) - 0.6*(hUT-1.5)

	log.Debugf("\nplLOS:%v \nplNLOS:%v", plLOS, plNLOS)

	return math.Max(plLOS, plNLOS)
}
