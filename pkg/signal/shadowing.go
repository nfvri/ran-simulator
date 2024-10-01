package signal

import (
	"math"
	"math/rand"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	log "github.com/sirupsen/logrus"
)

func ComputeGridPoints(bb model.BoundingBox, d_c float64, ncgi types.NCGI) []model.Coordinate {

	log.Debugf("square min point(%v, %v), max point(%v, %v)\n", bb.MinLat, bb.MinLng, bb.MaxLat, bb.MaxLng)

	latDiff := math.Abs(bb.MaxLat - bb.MinLat)
	lngDiff := math.Abs(bb.MaxLng - bb.MinLng)

	// Convert d_c from meters to degrees
	d_c_lat := utils.MetersToLatDegrees(d_c)
	avgLat := (bb.MinLat + bb.MaxLat) / 2.0
	d_c_lng := utils.MetersToLngDegrees(d_c, avgLat)

	// Calculate the number of grid points based on d_c
	numLatPoints := int(math.Ceil(latDiff / d_c_lat))
	numLngPoints := int(math.Ceil(lngDiff / d_c_lng))

	if numLatPoints != numLngPoints {
		log.Warnf("NCGI: %v: grid dimensions unequal: lat:%v, lng:%v", ncgi, numLatPoints, numLngPoints)
	}
	maxDim := int(math.Max(float64(numLatPoints), float64(numLngPoints)))

	gridPoints := make([]model.Coordinate, 0, maxDim*maxDim)
	for i := 0; i <= maxDim; i++ {
		lat := bb.MinLat + float64(i)*d_c_lat
		for j := 0; j <= maxDim; j++ {
			lng := bb.MinLng + float64(j)*d_c_lng
			gridPoints = append(gridPoints, model.Coordinate{Lat: lat, Lng: lng})
		}
	}
	return gridPoints
}

func CalculateShadowMap(gridPoints []model.Coordinate, d_c float64, sigma float64) []float64 {
	A := func(i, j int) float64 {
		if i == j {
			return 1
		}
		return math.Exp(-utils.GetSphericalDistance(gridPoints[i], gridPoints[j]) / d_c)
	}

	n := len(gridPoints)
	L := make([][]float64, n)
	for i := range L {
		L[i] = make([]float64, i+1)
	}
	shadowing := make([]float64, n)
	// Cholesky
	// Compute entries of L
	for i := 0; i < n; i++ {
		for j := 0; j <= i; j++ {
			sum := 0.0
			for k := 0; k < j; k++ {
				sum += L[i][k] * L[j][k]
			}

			if i == j {
				L[i][j] = math.Sqrt(A(i, j) - sum)
			} else {
				L[i][j] = (A(i, j) - sum) / L[j][j]
			}

			shadowing[i] += L[i][j] * rand.NormFloat64() * sigma
		}
	}

	return shadowing
}

func GetShadowMapIndex(shadowingLen, i, j int) int {
	gridSize := int(math.Sqrt(float64(shadowingLen)))
	return i*gridSize + j
}

func GetShadowValue(shadowing []float64, i, j int) float64 {
	gridSize := int(math.Sqrt(float64(len(shadowing))))
	return shadowing[i*gridSize+j]
}

// Function to find the grid cell containing the given point
func FindGridCell(point model.Coordinate, gridPoints []model.Coordinate) (int, int) {

	size := int(math.Sqrt(float64(len(gridPoints)))) - 1

	closestLat := 0
	closestLng := 0

	minDistLat := math.Abs(gridPoints[0].Lat - point.Lat)
	minDistLng := math.Abs(gridPoints[0].Lng - point.Lng)

	for i := 1; i < size; i++ {
		distLat := math.Abs(gridPoints[i].Lat - point.Lat)
		distLng := math.Abs(gridPoints[i].Lng - point.Lng)
		if distLat < minDistLat {
			closestLat = i
			minDistLat = distLat
		}
		if distLng < minDistLng {
			closestLng = i
			minDistLng = distLng
		}
	}

	return closestLat, closestLng
}

// Function to find the bounding box (min and max lat/lng) of a list of coordinates
func FindBoundingBox(gridPoints []model.Coordinate) (bb model.BoundingBox) {

	bb = model.BoundingBox{
		MinLat: math.MaxFloat64,
		MinLng: math.MaxFloat64,
		MaxLat: -math.MaxFloat64,
		MaxLng: -math.MaxFloat64,
	}
	for _, coord := range gridPoints {
		bb.MinLat = math.Min(bb.MinLat, coord.Lat)
		bb.MinLng = math.Min(bb.MinLng, coord.Lng)
		bb.MaxLat = math.Max(bb.MaxLat, coord.Lat)
		bb.MaxLng = math.Max(bb.MaxLng, coord.Lng)
	}
	return
}

// Function to check if a point is inside a bounding box
func IsPointInsideBoundingBox(point model.Coordinate, bb model.BoundingBox) bool {
	return point.Lat >= bb.MinLat &&
		point.Lat <= bb.MaxLat &&
		point.Lng >= bb.MinLng &&
		point.Lng <= bb.MaxLng
}

// Function to find overlapping grid points between two grids and return index pointers
func FindOverlappingGridPoints(cell1, cell2 *model.Cell) (pointIndxsG1, pointIndxsG2 [][]int, overlapping bool) {

	pointIndxsG1 = make([][]int, 0)
	pointIndxsG2 = make([][]int, 0)
	overlapping = false

	for _, p1 := range cell1.GridPoints {
		if IsPointInsideBoundingBox(p1, cell2.BoundingBox) {
			overlapping = true
			rowG1, colG1 := FindGridCell(p1, cell1.GridPoints)
			rowG2, colG2 := FindGridCell(p1, cell2.GridPoints)
			pointIndxsG1 = append(pointIndxsG1, []int{rowG1, colG1})
			pointIndxsG2 = append(pointIndxsG2, []int{rowG2, colG2})
		}
	}

	return
}

func InitShadowMap(cell *model.Cell, d_c float64) {

	sigma := 6.0
	switch {
	case cell.Channel.Environment == "urban" && cell.Channel.LOS:
		sigma = 4.0
	case cell.Channel.Environment == "urban" && !cell.Channel.LOS:
		sigma = 6.0
	case cell.Channel.Environment == "rural" && cell.Channel.LOS:
		sigma = 4.0
	case cell.Channel.Environment != "rural" && !cell.Channel.LOS:
		sigma = 8.0
	}
	rpBoundaryPoints := cell.RPCoverageBoundaries[0].BoundaryPoints
	if len(rpBoundaryPoints) == 0 {
		return
	}
	log.Infof("NCGI: %v: len(rpBoundaryPoints): %d", cell.NCGI, len(rpBoundaryPoints))
	cell.BoundingBox = FindBoundingBox(rpBoundaryPoints)
	cell.GridPoints = ComputeGridPoints(cell.BoundingBox, d_c, cell.NCGI)

	log.Infof("NCGI: %v: len(gridPoints): %d", cell.NCGI, len(cell.GridPoints))
	cell.ShadowingMap = CalculateShadowMap(cell.GridPoints, d_c, sigma)
	log.Infof("NCGI: %v: len(ShadowingMap): %d", cell.NCGI, len(cell.ShadowingMap))
}

func replaceOverlappingShadowMapValues(cell1 *model.Cell, cell2 *model.Cell) {
	pointIndxsG1, pointIndxsG2, overlapping := FindOverlappingGridPoints(cell1, cell2)
	if overlapping && (cell1.NCGI != cell2.NCGI) {
		for i := range pointIndxsG1 {
			log.Debugf("%d and %d overlapping: (%v) and (%v)\n", cell1.NCGI, cell2.NCGI, pointIndxsG1[i], pointIndxsG2[i])
			si2 := GetShadowMapIndex(len(cell2.ShadowingMap), pointIndxsG2[i][0], pointIndxsG2[i][1])
			cell2.ShadowingMap[si2] = GetShadowValue(cell1.ShadowingMap, pointIndxsG1[i][0], pointIndxsG1[i][1])
		}
	}
}
