package signal

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
	log "github.com/sirupsen/logrus"
)

func ComputeGridPoints(rpBoundaryPoints []model.Coordinate, d_c float64) []model.Coordinate {

	bb := findBoundingBox(rpBoundaryPoints)

	fmt.Printf("square min point(%v, %v), max point(%v, %v)\n", bb.minLat, bb.minLng, bb.maxLat, bb.maxLng)

	latDiff := math.Abs(bb.maxLat - bb.minLat)
	lngDiff := math.Abs(bb.maxLng - bb.minLng)

	// Convert d_c from meters to degrees
	d_c_lat := utils.MetersToLatDegrees(d_c)
	avgLat := (bb.minLat + bb.maxLat) / 2.0
	d_c_lng := utils.MetersToLngDegrees(d_c, avgLat)

	// Calculate the number of grid points based on d_c
	numLatPoints := int(math.Ceil(latDiff / d_c_lat))
	numLngPoints := int(math.Ceil(lngDiff / d_c_lng))

	if numLatPoints != numLngPoints {
		log.Warnf("grid dimensions unequal %v, %v", numLatPoints, numLngPoints)
	}
	maxDim := int(math.Max(float64(numLatPoints), float64(numLngPoints)))

	gridPoints := make([]model.Coordinate, 0, numLatPoints*numLngPoints)
	for i := 0; i <= maxDim; i++ {
		for j := 0; j <= maxDim; j++ {
			lat := bb.minLat + float64(i)*d_c_lat
			lng := bb.minLng + float64(j)*d_c_lng
			gridPoints = append(gridPoints, model.Coordinate{Lat: lat, Lng: lng})
		}
	}
	return gridPoints
}

func CalculateShadowMap(gridPoints []model.Coordinate, d_c float64, sigma float64) []float64 {
	A := func(i, j int) float64 {
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
				// Compute L_ii
				L[i][i] = math.Sqrt(A(i, i) - sum)
				shadowing[i] += L[i][i] * rand.NormFloat64() * sigma
			} else {
				// Compute L_ij
				L[i][j] = (A(i, j) - sum) / L[j][j]
				shadowing[i] += L[i][j] * rand.NormFloat64() * sigma
			}
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

// Function to find the closest index
func closestIndex(arr []float64, value float64) int {
	closest := 0
	minDist := math.Abs(arr[0] - value)
	for i := 1; i < len(arr)-1; i++ {
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
	latitudes := utils.UniqueLatitudes(gridPoints)
	longitudes := utils.UniqueLongitudes(gridPoints)

	if len(latitudes) == 0 || len(longitudes) == 0 {
		return false
	}
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
func FindGridCell(point model.Coordinate, gridPoints []model.Coordinate) (int, int) {

	latitudes := utils.Latitudes(gridPoints)
	longitudes := utils.Longitudes(gridPoints)

	latIdx := closestIndex(latitudes, point.Lat)
	lngIdx := closestIndex(longitudes, point.Lng)

	return latIdx, lngIdx
}

type BoundingBox struct {
	minLat, minLng, maxLat, maxLng float64
}

// Function to find the bounding box (min and max lat/lng) of a list of coordinates
func findBoundingBox(gridPoints []model.Coordinate) (bb BoundingBox) {

	bb = BoundingBox{
		minLat: math.MaxFloat64,
		minLng: math.MaxFloat64,
		maxLat: -math.MaxFloat64,
		maxLng: -math.MaxFloat64,
	}
	for _, coord := range gridPoints {
		bb.minLat = math.Min(bb.minLat, coord.Lat)
		bb.minLng = math.Min(bb.minLng, coord.Lng)
		bb.maxLat = math.Max(bb.maxLat, coord.Lat)
		bb.maxLng = math.Max(bb.maxLng, coord.Lng)
	}
	return
}

// Function to check if a point is inside a bounding box
func isPointInsideBoundingBox(point model.Coordinate, bb BoundingBox) bool {
	return point.Lat >= bb.minLat &&
		point.Lat <= bb.maxLat &&
		point.Lng >= bb.minLng &&
		point.Lng <= bb.maxLng
}

// Function to find overlapping grid points between two grids and return index pointers
func FindOverlappingGridPoints(gridPoints1, gridPoints2 []model.Coordinate) (pointIndxsG1, pointIndxsG2 [][]int, overlapping bool) {

	pointIndxsG1 = make([][]int, 0)
	pointIndxsG2 = make([][]int, 0)
	bb := findBoundingBox(gridPoints2)
	overlapping = false

	for _, p1 := range gridPoints1 {
		if isPointInsideBoundingBox(p1, bb) {
			overlapping = true
			rowG1, colG1 := FindGridCell(p1, gridPoints1)
			rowG2, colG2 := FindGridCell(p1, gridPoints2)
			pointIndxsG1 = append(pointIndxsG1, []int{rowG1, colG1})
			pointIndxsG2 = append(pointIndxsG2, []int{rowG2, colG2})
		}
	}

	return
}

func InitShadowMap(cell *model.Cell, d_c float64) {
	log.Info("Initializing ShadowMap")

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
	log.Infof("len(rpBoundaryPoints): %d", len(rpBoundaryPoints))
	cell.GridPoints = ComputeGridPoints(rpBoundaryPoints, d_c)
	log.Infof("len(cell.GridPoints): %d", len(cell.GridPoints))
	cell.ShadowingMap = CalculateShadowMap(cell.GridPoints, d_c, sigma)
	log.Infof("len(cell.ShadowingMap): %d", len(cell.ShadowingMap))
}

func replaceOverlappingShadowMapValues(cell1 *model.Cell, cell2 *model.Cell) {
	pointIndxsG1, pointIndxsG2, overlapping := FindOverlappingGridPoints(cell1.GridPoints, cell2.GridPoints)
	if overlapping && (cell1.NCGI != cell2.NCGI) {
		for i := range pointIndxsG1 {
			log.Debugf("%d and %d overlapping: (%v) and (%v)\n", cell1.NCGI, cell2.NCGI, pointIndxsG1[i], pointIndxsG2[i])
			si2 := GetShadowMapIndex(len(cell2.ShadowingMap), pointIndxsG2[i][0], pointIndxsG2[i][1])
			cell2.ShadowingMap[si2] = GetShadowValue(cell1.ShadowingMap, pointIndxsG1[i][0], pointIndxsG1[i][1])
		}
	}
}
