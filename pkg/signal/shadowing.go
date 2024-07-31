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

	minLat, minLng, maxLat, maxLng := findMinMaxCoords(rpBoundaryPoints)

	fmt.Printf("square min point(%v, %v), max point(%v, %v)\n", minLat, minLng, maxLat, maxLng)

	latDiff := math.Abs(maxLat - minLat)
	lngDiff := math.Abs(maxLng - minLng)

	// Convert d_c from meters to degrees
	d_c_lat := utils.MetersToLatDegrees(d_c)
	avgLat := (minLat + maxLat) / 2.0
	d_c_lng := utils.MetersToLngDegrees(d_c, avgLat)

	// Calculate the number of grid points based on d_c
	numLatPoints := int(math.Ceil(latDiff / d_c_lat))
	numLngPoints := int(math.Ceil(lngDiff / d_c_lng))

	gridPoints := make([]model.Coordinate, 0, numLatPoints*numLngPoints)
	for i := 0; i <= numLatPoints; i++ {
		for j := 0; j <= numLngPoints; j++ {
			lat := minLat + float64(i)*d_c_lat
			lng := minLng + float64(j)*d_c_lng
			gridPoints = append(gridPoints, model.Coordinate{Lat: lat, Lng: lng})
		}
	}
	return gridPoints
}

func CalculateShadowMap(gridPoints []model.Coordinate, d_c float64, sigma float64) [][]float64 {

	log.Infof("computeCorrelationMatrix ...")
	// Compute the correlation matrix
	A := computeCorrelationMatrix(gridPoints, d_c)

	// Compute the correlated shadow fading
	log.Infof("computeCorrelatedShadowFading ...")
	shadowing := computeCorrelatedShadowFading(A, sigma)

	log.Infof("makeCorrelatedFadingGrid ...")
	mappedCorrelatedFadingGrid := makeCorrelatedFadingGrid(shadowing)

	return mappedCorrelatedFadingGrid
}

// findMinMaxCoords finds the minimum and maximum latitude and longitude from a list of coordinates.
func findMinMaxCoords(coords []model.Coordinate) (minLat, minLng, maxLat, maxLng float64) {

	minLat, minLng, maxLat, maxLng = findBoundingBox(coords)

	fmt.Printf("min point(%v, %v), max point(%v, %v)\n", minLat, minLng, maxLat, maxLng)
	minCoord := model.Coordinate{Lat: minLat, Lng: minLng}
	maxCoordLat := model.Coordinate{Lat: maxLat, Lng: minLng}
	maxCoordLng := model.Coordinate{Lat: minLat, Lng: maxLng}

	latRange := utils.GetSphericalDistance(minCoord, maxCoordLat)
	lngRange := utils.GetSphericalDistance(minCoord, maxCoordLng)

	if latRange > lngRange {
		latDiff := latRange - lngRange
		latDiffDegrees := utils.MetersToLngDegrees(latDiff/2, minLat)
		minLng = minLng - latDiffDegrees
		maxLng = maxLng + latDiffDegrees
	} else {
		lngDiff := lngRange - latRange
		lngDiffDegrees := utils.MetersToLatDegrees(lngDiff / 2)
		minLat = minLat - lngDiffDegrees
		maxLat = maxLat + lngDiffDegrees
	}
	return
}

// Function to compute the correlation matrix
func computeCorrelationMatrix(gridPoints []model.Coordinate, d_c float64) [][]float64 {
	numPoints := len(gridPoints)
	gridSize := int(math.Sqrt(float64(numPoints))) - 1
	gridNumPoints := gridSize * gridSize
	fmt.Println("----")
	fmt.Printf("numPoints: %v\n", numPoints)
	fmt.Printf("gridSize: %v\n", gridSize)
	fmt.Printf("gridNumPoints: %v\n", gridNumPoints)
	fmt.Println("----")
	A := make([][]float64, int(gridNumPoints))
	for i := range A {
		A[i] = make([]float64, int(gridNumPoints))
		for j := range A[i] {
			A[i][j] = math.Exp(-utils.GetSphericalDistance(gridPoints[i], gridPoints[j]) / d_c)
		}
	}
	return A
}

// Function to generate samples from N(0, sigma^2)
func generateNormalSamples(numSamples int, sigma float64) []float64 {
	samples := make([]float64, numSamples)
	for i := range samples {
		samples[i] = rand.NormFloat64() * sigma
	}
	return samples
}

// Function to perform Cholesky decomposition
func choleskyDecomposition(matrix [][]float64) [][]float64 {
	size := len(matrix)
	lowerTriangular := make([][]float64, size)
	for i := range lowerTriangular {
		lowerTriangular[i] = make([]float64, size)
	}

	for row := 0; row < size; row++ {
		for col := 0; col <= row; col++ {
			sum := 0.0
			for k := 0; k < col; k++ {
				sum += lowerTriangular[row][k] * lowerTriangular[col][k]
			}
			if row == col {
				lowerTriangular[row][col] = math.Sqrt(matrix[row][row] - sum)
			} else {
				lowerTriangular[row][col] = (matrix[row][col] - sum) / lowerTriangular[col][col]
			}
		}
	}
	return lowerTriangular
}

// Function to compute correlated shadow fading
func computeCorrelatedShadowFading(A [][]float64, sigma float64) []float64 {
	numPoints := len(A)

	// Draw sample S from N(0, sigma^2)
	S := generateNormalSamples(numPoints, sigma)
	// Compute the Cholesky decomposition L of A
	L := choleskyDecomposition(A)

	// Multiply L with S
	shadowing := make([]float64, numPoints)
	for i := 0; i < numPoints; i++ {
		for j := 0; j <= i; j++ {
			shadowing[i] += L[i][j] * S[j]
		}
	}
	return shadowing
}

// Function for mapping the correlated fading to the grid
func makeCorrelatedFadingGrid(shadowing []float64) [][]float64 {
	gridSize := int(math.Sqrt(float64(len(shadowing))))
	mappedCorrelatedFadingGrid := make([][]float64, gridSize)

	for i := 0; i < gridSize; i++ {
		mappedCorrelatedFadingGrid[i] = make([]float64, gridSize)
		for j := 0; j < gridSize; j++ {
			mappedCorrelatedFadingGrid[i][j] = shadowing[i*gridSize+j]
		}
	}
	return mappedCorrelatedFadingGrid
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

	latitudes := utils.UniqueLatitudes(gridPoints)
	longitudes := utils.UniqueLongitudes(gridPoints)

	latIdx := closestIndex(latitudes, point.Lat)
	lngIdx := closestIndex(longitudes, point.Lng)

	return latIdx, lngIdx
}

// Function to find the bounding box (min and max lat/lng) of a list of coordinates
func findBoundingBox(gridPoints []model.Coordinate) (minLat, minLng, maxLat, maxLng float64) {
	minLat, minLng = math.MaxFloat64, math.MaxFloat64
	maxLat, maxLng = -math.MaxFloat64, -math.MaxFloat64

	for _, coord := range gridPoints {
		minLat = math.Min(minLat, coord.Lat)
		minLng = math.Min(minLng, coord.Lng)
		maxLat = math.Max(maxLat, coord.Lng)
		maxLng = math.Max(maxLng, coord.Lng)
	}
	return minLat, minLng, maxLat, maxLng
}

// Function to check if a point is inside a bounding box
func isPointInsideBoundingBox(point model.Coordinate, minLat, minLng, maxLat, maxLng float64) bool {
	return point.Lat >= minLat && point.Lat <= maxLat && point.Lng >= minLng && point.Lng <= maxLng
}

// Function to find overlapping grid points between two grids and return index pointers
func FindOverlappingGridPoints(gridPoints1, gridPoints2 []model.Coordinate) (cell1iList, cell1jList, cell2iList, cell2jList []int, overlapping bool) {
	// minLat1, minLng1, maxLat1, maxLng1 := findBoundingBox(gridPoints1)
	minLat2, minLng2, maxLat2, maxLng2 := findBoundingBox(gridPoints2)

	overlapping = false

	gridSize1 := int(math.Sqrt(float64(len(gridPoints1)))) - 1
	gridSize2 := int(math.Sqrt(float64(len(gridPoints2)))) - 1

	// fmt.Printf("gridSize1: %v\n", gridSize1)
	// fmt.Printf("gridSize2: %v\n", gridSize2)

	cell1iList = make([]int, 0)
	cell1jList = make([]int, 0)
	cell2iList = make([]int, 0)
	cell2jList = make([]int, 0)
	// Iterate over grid points within the intersection area and check if they belong to both grids
	for _, point1 := range gridPoints1 {
		if isPointInsideBoundingBox(point1, minLat2, minLng2, maxLat2, maxLng2) {
			overlapping = true
			cell1i, cell1j := FindGridCell(point1, gridPoints1)
			cell2i, cell2j := FindGridCell(point1, gridPoints2)
			if cell1i < gridSize1 && cell1j < gridSize1 && cell2i < gridSize2 && cell2j < gridSize2 {
				cell1iList = append(cell1iList, cell1i)
				cell1jList = append(cell1jList, cell1j)
				cell2iList = append(cell2iList, cell2i)
				cell2jList = append(cell2jList, cell2j)
			}
		}
	}

	return cell1iList, cell1jList, cell2iList, cell2jList, overlapping
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
