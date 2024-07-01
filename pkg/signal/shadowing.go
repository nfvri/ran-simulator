package signal

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/nfvri/ran-simulator/pkg/model"
)

// Convert meters to degrees latitude
func metersToLatDegrees(meters float64) float64 {
	return meters / 111132.954
}

// Convert meters to degrees longitude at a specific latitude
func metersToLngDegrees(meters, latitude float64) float64 {
	return meters / (111132.954 * math.Cos(latitude*math.Pi/180.0))
}

func ComputeGridPoints(coverageCoordinates []model.Coordinate, d_c float64) []model.Coordinate {

	minLat, minLng, maxLat, maxLng := findMinMaxCoords(coverageCoordinates)
	// fmt.Println(coverageCoordinates)
	fmt.Printf("square min point(%v, %v), max point(%v, %v)\n", minLat, minLng, maxLat, maxLng)

	latDiff := math.Abs(maxLat - minLat)
	lngDiff := math.Abs(maxLng - minLng)

	// Convert d_c from meters to degrees
	d_c_lat := metersToLatDegrees(d_c)
	fmt.Println(d_c_lat)
	avgLat := (minLat + maxLat) / 2.0
	d_c_lng := metersToLngDegrees(d_c, avgLat)
	fmt.Println(d_c_lat)

	// Calculate the number of grid points based on d_c
	numLatPoints := int(math.Ceil(latDiff / d_c_lat))
	numLngPoints := int(math.Ceil(lngDiff / d_c_lng))
	// fmt.Println("*******************")
	// fmt.Println(cell.NCGI)
	// fmt.Println("*******************")
	// fmt.Printf("------------------\n minLat: %f\n maxLat: %f\n latDiff: %f\n minLng: %f\n maxLng: %f\n lngDiff: %f\n", minLat, maxLat, latDiff, minLng, maxLng, lngDiff)

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
	gridSize := int(math.Sqrt(float64(len(gridPoints)))) - 1
	fmt.Println("gridSize:")
	fmt.Println(gridSize)
	// Compute the correlation matrix
	A := computeCorrelationMatrix(gridPoints, d_c)

	// Compute the correlated shadow fading
	shadowing := computeCorrelatedShadowFading(A, sigma)

	mappedCorrelatedFadingGrid := makeCorrelatedFadingGrid(shadowing)

	fmt.Println("Mapped Correlated Fading to the Grid:")
	for i := 0; i < gridSize; i++ {
		for j := 0; j < gridSize; j++ {

			fmt.Printf(" %8.4f |", mappedCorrelatedFadingGrid[i][j])
		}
		fmt.Println()
		for j := 0; j < gridSize; j++ {
			fmt.Printf("----------|")
		}
		fmt.Println()
	}

	return mappedCorrelatedFadingGrid
}

// Function to compute the Euclidean distance from GPS coordinates
func getEuclideanDistanceFromCoordinates(coord1 model.Coordinate, coord2 model.Coordinate) float64 {
	earthRadius := 6378.137
	dLat := coord1.Lat*math.Pi/180 - coord2.Lat*math.Pi/180
	dLng := coord1.Lng*math.Pi/180 - coord2.Lng*math.Pi/180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(coord1.Lat*math.Pi/180)*math.Cos(coord2.Lat*math.Pi/180)*
		math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c * 1000 // distance in meters
}

// findMinMaxCoords finds the minimum and maximum latitude and longitude from a list of coordinates.
func findMinMaxCoords(coords []model.Coordinate) (float64, float64, float64, float64) {
	// Initialize min and max with extreme values
	minLat := math.MaxFloat64
	minLng := math.MaxFloat64
	maxLat := -math.MaxFloat64
	maxLng := -math.MaxFloat64

	for _, coord := range coords {
		if coord.Lat < minLat {
			minLat = coord.Lat
		}
		if coord.Lng < minLng {
			minLng = coord.Lng
		}
		if coord.Lat > maxLat {
			maxLat = coord.Lat
		}
		if coord.Lng > maxLng {
			maxLng = coord.Lng
		}
	}
	fmt.Printf("min point(%v, %v), max point(%v, %v)\n", minLat, minLng, maxLat, maxLng)
	minCoord := model.Coordinate{Lat: minLat, Lng: minLng}
	maxCoordLat := model.Coordinate{Lat: maxLat, Lng: minLng}
	maxCoordLng := model.Coordinate{Lat: minLat, Lng: maxLng}

	latRange := getEuclideanDistanceFromCoordinates(minCoord, maxCoordLat)
	lngRange := getEuclideanDistanceFromCoordinates(minCoord, maxCoordLng)

	if latRange > lngRange {
		latDiff := latRange - lngRange
		latDiffDegrees := metersToLngDegrees(latDiff/2, minLat)
		minLng = minLng - latDiffDegrees
		maxLng = maxLng + latDiffDegrees
	} else {
		lngDiff := lngRange - latRange
		lngDiffDegrees := metersToLatDegrees(lngDiff / 2)
		minLat = minLat - lngDiffDegrees
		maxLat = maxLat + lngDiffDegrees
	}

	return minLat, minLng, maxLat, maxLng
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
			A[i][j] = math.Exp(-getEuclideanDistanceFromCoordinates(gridPoints[i], gridPoints[j]) / d_c)
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
func FindGridCell(point model.Coordinate, gridPoints []model.Coordinate) (int, int, bool) {
	if !isPointInsideGrid(point, gridPoints) {
		return -1, -1, false
	}

	latitudes := uniqueLatitudes(gridPoints)
	longitudes := uniqueLongitudes(gridPoints)

	latIdx := closestIndex(latitudes, point.Lat)
	lngIdx := closestIndex(longitudes, point.Lng)

	return latIdx, lngIdx, true
}

// Function to find the bounding box (min and max lat/lng) of a list of coordinates
func findBoundingBox(gridPoints []model.Coordinate) (minLat, minLng, maxLat, maxLng float64) {
	minLat, minLng = math.MaxFloat64, math.MaxFloat64
	maxLat, maxLng = -math.MaxFloat64, -math.MaxFloat64

	for _, point := range gridPoints {
		if point.Lat < minLat {
			minLat = point.Lat
		}
		if point.Lat > maxLat {
			maxLat = point.Lat
		}
		if point.Lng < minLng {
			minLng = point.Lng
		}
		if point.Lng > maxLng {
			maxLng = point.Lng
		}
	}
	return minLat, minLng, maxLat, maxLng
}

// Function to check if a point is inside a bounding box
func isPointInsideBoundingBox(point model.Coordinate, minLat, minLng, maxLat, maxLng float64) bool {
	return point.Lat >= minLat && point.Lat <= maxLat && point.Lng >= minLng && point.Lng <= maxLng
}

// // Function to find overlapping grid points between two grids
// func findOverlappingGridPoints(gridPoints1, gridPoints2 []model.Coordinate) []model.Coordinate {
// 	minLat1, minLng1, maxLat1, maxLng1 := findBoundingBox(gridPoints1)
// 	minLat2, minLng2, maxLat2, maxLng2 := findBoundingBox(gridPoints2)

// 	// Calculate the intersection of bounding boxes
// 	intersectMinLat := math.Max(minLat1, minLat2)
// 	intersectMinLng := math.Max(minLng1, minLng2)
// 	intersectMaxLat := math.Min(maxLat1, maxLat2)
// 	intersectMaxLng := math.Min(maxLng1, maxLng2)

// 	// Map to store already visited points
// 	visited := make(map[model.Coordinate]bool)

// 	// Iterate over grid points within the intersection area and check if they belong to both grids
// 	overlappingPoints := make([]model.Coordinate, 0)
// 	for lat := intersectMinLat; lat <= intersectMaxLat; lat += 0.001 { // Adjust the step size as needed
// 		for lng := intersectMinLng; lng <= intersectMaxLng; lng += 0.001 { // Adjust the step size as needed
// 			point := model.Coordinate{Lat: lat, Lng: lng}
// 			if isPointInsideBoundingBox(point, minLat1, minLng1, maxLat1, maxLng1) &&
// 				isPointInsideBoundingBox(point, minLat2, minLng2, maxLat2, maxLng2) &&
// 				!visited[point] {
// 				overlappingPoints = append(overlappingPoints, point)
// 				visited[point] = true
// 			}
// 		}
// 	}

// 	return overlappingPoints
// }

// Function to find overlapping grid points between two grids and return index pointers
func FindOverlappingGridPoints(gridPoints1, gridPoints2 []model.Coordinate) (cell1iList, cell1jList, cell2iList, cell2jList []int, overlapping bool) {
	// minLat1, minLng1, maxLat1, maxLng1 := findBoundingBox(gridPoints1)
	minLat2, minLng2, maxLat2, maxLng2 := findBoundingBox(gridPoints2)

	overlapping = false

	gridSize1 := int(math.Sqrt(float64(len(gridPoints1)))) - 1
	gridSize2 := int(math.Sqrt(float64(len(gridPoints2)))) - 1

	fmt.Printf("gridSize1: %v\n", gridSize1)
	fmt.Printf("gridSize2: %v\n", gridSize2)

	cell1iList = make([]int, 0)
	cell1jList = make([]int, 0)
	cell2iList = make([]int, 0)
	cell2jList = make([]int, 0)
	// Iterate over grid points within the intersection area and check if they belong to both grids
	for _, point1 := range gridPoints1 {
		if isPointInsideBoundingBox(point1, minLat2, minLng2, maxLat2, maxLng2) {
			overlapping = true
			cell1i, cell1j, _ := FindGridCell(point1, gridPoints1)
			cell2i, cell2j, _ := FindGridCell(point1, gridPoints2)
			if cell1i < gridSize1 && cell1j < gridSize1 && cell2i < gridSize2 && cell2j < gridSize2 {
				cell1iList = append(cell1iList, cell1i)
				cell1jList = append(cell1jList, cell1j)
				cell2iList = append(cell2iList, cell2i)
				cell2jList = append(cell2jList, cell2j)
			}
			// overlapIndices2 = append(overlapIndices2, (cell2i, cell2j))

			// for j, point2 := range gridPoints2 {
			// 	if isPointInsideBoundingBox(point2, minLat1, minLng1, maxLat1, maxLng1) {
			// 		fmt.Printf("point1: %v, point2: %v\n", point1, point2)
			// 		overlapIndices1 = append(overlapIndices1, i)
			// 		overlapIndices2 = append(overlapIndices2, j)
			// 		break // Break to avoid duplicates
			// 	}
			// }
		}
	}

	return cell1iList, cell1jList, cell2iList, cell2jList, overlapping
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

	cell.GridPoints = ComputeGridPoints(*cell, d_c)
	cell.ShadowingMap = CalculateShadowMap(cell.GridPoints, d_c, sigma)
}

func ReplaceOverlappingShadowMap(cell1 *model.Cell, cell2 *model.Cell, d_c float64) {
	cell1iList, cell1jList, cell2iList, cell2jList, overlapping := FindOverlappingGridPoints(cell1.GridPoints, cell2.GridPoints)
	if overlapping {
		if cell1.NCGI == cell2.NCGI {
			fmt.Printf("%d and %d overlapping but is the same cell\n", cell1.NCGI, cell2.NCGI)
		} else {
			for i := range cell1iList {
				fmt.Printf("%d and %d overlapping: (%d,%d) and (%d,%d)\n", cell1.NCGI, cell2.NCGI, cell1iList[i], cell1jList[i], cell2iList[i], cell2jList[i])
				cell2.ShadowingMap[cell2iList[i]][cell2jList[i]] = cell1.ShadowingMap[cell1iList[i]][cell1jList[i]]
			}
		}
	} else {
		fmt.Printf("%d and %d does not overlap\n", cell1.NCGI, cell2.NCGI)
	}

}
