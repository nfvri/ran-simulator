package model

import (
	"fmt"
	"math"
	"math/rand"
	"time"
	"sort"
)

// Function to compute the gridPoints based on cell coordinates
func ComputeGridPoints(gridSize int, m *Model) []Coordinate {
	var minLat, minLng, maxLat, maxLng float64
	// fmt.Print(m.Cells)
	for _, cell := range m.Cells {
		minLat = cell.Sector.Center.Lat
		maxLat = cell.Sector.Center.Lat
		minLng = cell.Sector.Center.Lng
		maxLng = cell.Sector.Center.Lng
		break
	}

	for _, cell := range m.Cells {
		if cell.Sector.Center.Lat < minLat{
			minLat = cell.Sector.Center.Lat
		}
		if cell.Sector.Center.Lng < minLng{
			minLng = cell.Sector.Center.Lng
		}
		if cell.Sector.Center.Lat > maxLat{
			maxLat = cell.Sector.Center.Lat
		}
		if cell.Sector.Center.Lng > maxLng{
			maxLng = cell.Sector.Center.Lng
		}
	}

	latDiff := math.Abs(maxLat - minLat)
	lngDiff := math.Abs(maxLng - minLng)

	fmt.Printf("------------------\n minLat: %f\n maxLat: %f\n latDiff: %f\n minLng: %f\n maxLng: %f\n lngDiff: %f\n",minLat,maxLat,latDiff,minLng,maxLng,lngDiff)

	gridPoints := make([]Coordinate, 0, gridSize*gridSize)
	for i := 0; i < gridSize; i++ {
		for j := 0; j < gridSize; j++ {
			lat := minLat + float64(i)*(float64(latDiff)/float64(gridSize-1))
			lng := minLng + float64(j)*(float64(lngDiff)/float64(gridSize-1))
			gridPoints = append(gridPoints, Coordinate{Lat: lat, Lng: lng})
		}
	}
	return gridPoints
}

func CalculateShadowMap(d_c float64, sigma float64, gridSize int, m *Model) [][]float64{
	gridPoints := ComputeGridPoints(gridSize,m)
	
	fmt.Println("Grid Points:")
	fmt.Println(gridPoints)

	// Compute the correlation matrix
	A := computeCorrelationMatrix(gridPoints, d_c)

	// Print the correlation matrix
	fmt.Println("Correlation matrix (A):")
	for _, row := range A {
		for _, val := range row {
			fmt.Printf("%.4f ", val)
		}
		fmt.Println()
	}

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
func getEuclideanDistanceFromCoordinates(coord1 Coordinate, coord2 Coordinate) float64 {
	earthRadius := 6378.137
	dLat := coord1.Lat*math.Pi/180 - coord2.Lat*math.Pi/180
	dLng := coord1.Lng*math.Pi/180 - coord2.Lng*math.Pi/180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(coord1.Lat*math.Pi/180)*math.Cos(coord2.Lat*math.Pi/180)*
		math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c *1000 // distance in meters
}

// Function to compute the correlation matrix
func computeCorrelationMatrix(gridPoints []Coordinate, d_c float64) [][]float64 {
	numPoints := len(gridPoints)
	A := make([][]float64, int(numPoints))
	for i := range A {
		A[i] = make([]float64, int(numPoints))
		for j := range A[i] {
			A[i][j] = math.Exp(-getEuclideanDistanceFromCoordinates(gridPoints[i], gridPoints[j]) / d_c)
		}
	}
	return A
}

// Function to generate samples from N(0, sigma^2)
func generateNormalSamples(numSamples int, sigma float64) []float64 {
	rand.Seed(time.Now().UnixNano())
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
func makeCorrelatedFadingGrid(shadowing []float64) [][]float64{
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
func uniqueLatitudes(points []Coordinate) []float64 {
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
func uniqueLongitudes(points []Coordinate) []float64 {
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

// Function to find the grid cell containing the given point
func findGridCell(point Coordinate, gridPoints []Coordinate) (int, int) {
	latitudes := uniqueLatitudes(gridPoints)
	longitudes := uniqueLongitudes(gridPoints)

	latIdx := closestIndex(latitudes, point.Lat)
	lngIdx := closestIndex(longitudes, point.Lng)

	return latIdx, lngIdx
}

