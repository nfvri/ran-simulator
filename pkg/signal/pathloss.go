package signal

import (
	"math"

	bw "github.com/nfvri/ran-simulator/pkg/bandwidth"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
	log "github.com/sirupsen/logrus"
)

// GetPathLoss calculates the path loss based on the environment and LOS/NLOS conditions
func GetPathLoss(coord model.Coordinate, ueHeight float64, carrier *model.Carrier) float64 {
	var pathLoss float64

	switch carrier.Environment {
	case "urban":
		if carrier.LOS {
			pathLoss = getUrbanLOSPathLoss(coord, ueHeight, carrier)
		} else {
			pathLoss = getUrbanNLOSPathLoss(coord, ueHeight, carrier)
		}
	case "rural":
		if carrier.LOS {
			pathLoss = getRuralLOSPathLoss(coord, ueHeight, carrier)
		} else {
			pathLoss = getRuralNLOSPathLoss(coord, ueHeight, carrier)
		}
	default:
		pathLoss = getRuralNLOSPathLoss(coord, ueHeight, carrier)
	}

	return pathLoss
}

func GetFreeSpacePathLoss(coord model.Coordinate, carrier *model.Carrier) float64 {
	distanceKM := utils.GetSphericalDistance(coord, carrier.Center) / 1000
	// Assuming we're using CBRS frequency 3.6 GHz
	// 92.45 is the constant value of 20 * log10(4*pi / c) in Kilometer scale
	fc := bw.MHzToGHz(bw.CalculateFrequencyMHz(carrier.ArfcnDL)) // frequency in GHz

	pathLoss := 20*math.Log10(distanceKM) + 20*math.Log10(fc) + 92.45
	return pathLoss
}

// 3D Euclidean distance function
func get3dEuclideanDistanceFromGPS(coord model.Coordinate, ueHeight float64, carrier *model.Carrier) float64 {
	d2D := utils.GetSphericalDistance(coord, carrier.Center)

	heightUE := ueHeight
	heightDiff := math.Abs(float64(carrier.Height) - heightUE)

	// Pythagorean theorem
	d3D := math.Sqrt(math.Pow(d2D, 2) + math.Pow(heightDiff, 2))

	return d3D
}

// Breakpoint distance function
func getBreakpointDistance(carrier *model.Carrier) float64 {
	c := 3.0 * math.Pow(10, 8)
	hBS := float64(carrier.Height) // base station height
	hUT := float64(1.5)            // average height of user terminal 1m <= hUT <= 10m

	fc := bw.MHzToHz(bw.CalculateFrequencyMHz(carrier.ArfcnDL)) // frequency in Hz

	dBP := 2 * math.Pi * hBS * hUT * fc / c

	return dBP
}

// Breakpoint distance function
func getBreakpointPrimeDistance(carrier *model.Carrier) float64 {
	c := 3.0 * math.Pow(10, 8)
	hE := float64(1)                    // assuming environment height is 1m
	hBS := float64(carrier.Height) - hE // base station height
	hUT := float64(1.5) - hE            // average height of user terminal 1m <= hUT <= 10m

	fc := bw.MHzToHz(bw.CalculateFrequencyMHz(carrier.ArfcnDL)) // frequency in Hz

	numer := (4 * hBS * hUT * fc)

	dBP := numer / c

	return dBP
}

// getRuralLOSPathLoss calculates the RMa LOS path loss
func getRuralLOSPathLoss(coord model.Coordinate, ueHeight float64, carrier *model.Carrier) float64 {
	d2D := utils.GetSphericalDistance(coord, carrier.Center)
	d3D := get3dEuclideanDistanceFromGPS(coord, ueHeight, carrier)
	dBP := getBreakpointDistance(carrier)

	if 10 <= d2D && d2D <= dBP {
		return RmaLOSPL1(carrier, d3D)
	} else {
		pl2 := RmaLOSPL1(carrier, dBP) + 40*math.Log10(d3D/dBP)
		return pl2
	}
}

// calculates PL1 for RMa LOS path loss
func RmaLOSPL1(carrier *model.Carrier, d float64) float64 {
	fc := bw.MHzToGHz(bw.CalculateFrequencyMHz(carrier.ArfcnDL)) // frequency in GHz
	h := float64(5)                                              // average building height in m

	pl1 := 20*math.Log10(40*math.Pi*d*fc/3) + math.Min(0.03*math.Pow(h, 1.72), 10)*math.Log10(d) -
		math.Min(0.044*math.Pow(h, 1.72), 14.77) + 0.002*math.Log10(h)*d

	return pl1
}

// getRuralNLOSPathLoss calculates the RMa NLOS path loss
func getRuralNLOSPathLoss(coord model.Coordinate, ueHeight float64, carrier *model.Carrier) float64 {
	d3D := get3dEuclideanDistanceFromGPS(coord, ueHeight, carrier)
	W := float64(20)               // average street width 5m <= W <= 50m
	h := float64(5)                // average building height 5m <= h <= 50m
	hBS := float64(carrier.Height) // base station height
	hUT := ueHeight                // average height of user terminal 1m <= hUT <= 10m

	fc := bw.MHzToGHz(bw.CalculateFrequencyMHz(carrier.ArfcnDL)) // frequency in GHz

	plLOS := getRuralLOSPathLoss(coord, ueHeight, carrier)
	plNLOS := 161.04 - 7.1*math.Log10(W) + 7.5*math.Log10(h) -
		(24.37-3.7*math.Pow((h/hBS), 2))*math.Log10(hBS) +
		(43.42-3.1*math.Log10(hBS))*(math.Log10(d3D)-3) +
		20*math.Log10(fc) -
		(math.Pow(3.2*math.Log10(11.75*hUT), 2) - 4.97)

	return math.Max(plLOS, plNLOS)
}

// getUrbanLOSPathLoss calculates the UMa LOS path loss
func getUrbanLOSPathLoss(coord model.Coordinate, ueHeight float64, carrier *model.Carrier) float64 {
	d2D := utils.GetSphericalDistance(coord, carrier.Center)
	d3D := get3dEuclideanDistanceFromGPS(coord, ueHeight, carrier)
	dBP := getBreakpointPrimeDistance(carrier)
	hBS := float64(carrier.Height)                         // base station height
	hUT := ueHeight                                        // average height of user terminal 1m <= hUT <= 22.5m
	fc := bw.CalculateFrequencyMHz(carrier.ArfcnDL) / 1000 // frequency in GHz

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
func getUrbanNLOSPathLoss(coord model.Coordinate, ueHeight float64, carrier *model.Carrier) float64 {
	d3D := get3dEuclideanDistanceFromGPS(coord, ueHeight, carrier)

	fc := bw.MHzToGHz(bw.CalculateFrequencyMHz(carrier.ArfcnDL)) // frequency in GHz
	hUT := ueHeight                                              // average height of user terminal 1m <= W <= 22.5m

	plLOS := getUrbanLOSPathLoss(coord, ueHeight, carrier)
	plNLOS := 13.54 + 39.08*math.Log10(d3D) + 20*math.Log10(fc) - 0.6*(hUT-1.5)

	log.Debugf("\nplLOS:%v \nplNLOS:%v", plLOS, plNLOS)

	return math.Max(plLOS, plNLOS)
}
