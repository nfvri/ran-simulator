// SPDX-FileCopyrightText: 2021-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package signal

import (
	"math"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
	log "github.com/sirupsen/logrus"
)

// powerFactor relates power to distance in decimal degrees
const (
	powerFactor        = 0.001
	RICEAN_K_MEAN      = 9.0
	RICEAN_K_STD_MICRO = 5.0
	RICEAN_K_STD_MACRO = 3.5
)

// Strength returns the signal strength at location relative to the specified cell.
func Strength(coord model.Coordinate, height, mpf float64, cell model.Cell) float64 {
	if math.IsNaN(coord.Lat) || math.IsNaN(coord.Lng) || !isPointInsideGrid(coord, cell.GridPoints) {
		return math.Inf(-1)
	}

	latIdx, lngIdx := FindGridCell(coord, cell.GridPoints)

	radiatedStrength := RadiatedStrength(coord, height, cell)
	if radiatedStrength == math.Inf(-1) {
		return math.Inf(-1)
	}

	shadowing := 0.0
	if len(cell.ShadowingMap) > 0 {
		shadowing = GetShadowValue(cell.ShadowingMap, latIdx, lngIdx)
	}

	return (radiatedStrength + shadowing) * mpf

}

func RadiatedStrength(coord model.Coordinate, height float64, cell model.Cell) float64 {
	if math.IsNaN(coord.Lat) || math.IsNaN(coord.Lng) {
		return math.Inf(-1)
	}
	angleAtt := angularAttenuation(coord, height, cell)
	pathLoss := GetPathLoss(coord, height, cell)

	antenaGain := cell.Beam.MaxGain + angleAtt

	return cell.TxPowerDB + antenaGain - pathLoss

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
	return utils.DbToMw(gain * math.Sqrt(powerFactor/r))
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
	ueAngle := utils.CalculateBearing(cell.Sector.Center.Lat, cell.Sector.Center.Lng, coord.Lat, coord.Lng)

	azimuthOffset := math.Abs(cell.Sector.Azimuth - ueAngle)
	if azimuthOffset > 180 {
		azimuthOffset = 360 - azimuthOffset
	}
	horizontalCut := azimuthAttenuation(azimuthOffset, cell.Beam.H3dBAngle, cell.TxPowerDB)

	log.Debugf(
		`
		horizontalCut: %v 
		cell.Sector.Azimuth: %v
		ueAngle: %v 
		azimuthOffset: %v 
		`,
		horizontalCut,
		cell.Sector.Azimuth,
		ueAngle,
		azimuthOffset,
	)

	log.Debug("\n======================================\n")
	zenithAngle := calcZenithAngle(coord, height, cell)
	verticalCut := zenithAttenuation(zenithAngle, cell.Beam.V3dBAngle, cell.Beam.VSideLobeAttenuationDB)
	log.Debugf("\nverticalCut: %v \nzenithAngle: %v", verticalCut, zenithAngle)
	log.Debug("\n======================================\n")
	return -math.Min(-(verticalCut + horizontalCut), cell.TxPowerDB)
}

func calcZenithAngle(coord model.Coordinate, height float64, cell model.Cell) float64 {

	d2D := utils.GetSphericalDistance(coord, cell.Sector.Center) // assume small error for small distances
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
	log.Debugf("[azimuth attenuatuation]: %v, [max]: %v", azAtt, aMax)
	return -math.Min(azAtt, aMax)
}
