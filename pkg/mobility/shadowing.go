package mobility

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/nfvri/ran-simulator/pkg/model"
)

// Convert meters to degrees latitude
func MetersToLatDegrees(meters float64) float64 {
	return meters / 111132.954
}

// Convert meters to degrees longitude at a specific latitude
func MetersToLngDegrees(meters, latitude float64) float64 {
	return meters / (111132.954 * math.Cos(latitude*math.Pi/180.0))
}

// Function to expand diagonally and check path loss until it reaches 90
func getMinMaxPoints(cell model.Cell, d_c float64) (float64, float64, float64, float64) {
	latStep := MetersToLatDegrees(d_c)
	lngStep := MetersToLngDegrees(d_c, cell.Sector.Center.Lat)
	fmt.Println("latStep:")
	fmt.Println(latStep)
	fmt.Println("lngStep:")
	fmt.Println(lngStep)

	maxLat := cell.Sector.Center.Lat
	maxLng := cell.Sector.Center.Lng

	// Expand in the positive direction
	for {
		coord := model.Coordinate{Lat: maxLat, Lng: maxLng}
		pathLoss := GetPathLoss(coord, cell)
		fmt.Printf("Coordinate: (%.6f, %.6f), signalStrength: %.2f, Path Loss: %.2f\n", maxLat, maxLng, cell.TxPowerDB, pathLoss)
		if pathLoss >= cell.TxPowerDB {
			break
		}
		maxLat += latStep
		maxLng += lngStep
	}

	minLat := cell.Sector.Center.Lat
	minLng := cell.Sector.Center.Lng

	// Expand in the negative direction
	for {
		coord := model.Coordinate{Lat: minLat, Lng: minLng}
		pathLoss := GetPathLoss(coord, cell)
		fmt.Printf("Coordinate: (%.6f, %.6f), signalStrength: %.2f, Path Loss: %.2f\n", minLat, minLng, cell.TxPowerDB, pathLoss)
		if pathLoss >= cell.TxPowerDB {
			break
		}
		minLat -= latStep
		minLng -= lngStep
	}
	return minLat, minLng, maxLat, maxLng
}

func ComputeGridPoints(cell model.Cell, d_c float64) []model.Coordinate {

	minLat, minLng, maxLat, maxLng := getMinMaxPoints(cell, d_c)

	latDiff := math.Abs(maxLat - minLat)
	lngDiff := math.Abs(maxLng - minLng)

	// Convert d_c from meters to degrees
	d_c_lat := MetersToLatDegrees(d_c)
	avgLat := (minLat + maxLat) / 2.0
	d_c_lng := MetersToLngDegrees(d_c, avgLat)

	// Calculate the number of grid points based on d_c
	numLatPoints := int(math.Ceil(latDiff / d_c_lat))
	numLngPoints := int(math.Ceil(lngDiff / d_c_lng))

	fmt.Printf("------------------\n minLat: %f\n maxLat: %f\n latDiff: %f\n minLng: %f\n maxLng: %f\n lngDiff: %f\n", minLat, maxLat, latDiff, minLng, maxLng, lngDiff)

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
	gridSize := int(math.Sqrt(float64(len(gridPoints))))

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
func getEuclideanDistanceFromCoordinates(coord1 model.Coordinate, coord2 model.Coordinate) float64 {
	earthRadius := 6378.137
	dLat := coord1.Lat*math.Pi/180 - coord2.Lat*math.Pi/180
	dLng := coord1.Lng*math.Pi/180 - coord2.Lng*math.Pi/180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(coord1.Lat*math.Pi/180)*math.Cos(coord2.Lat*math.Pi/180)*
		math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c * 1000 // distance in meters
}

// Function to compute the correlation matrix
func computeCorrelationMatrix(gridPoints []model.Coordinate, d_c float64) [][]float64 {
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
