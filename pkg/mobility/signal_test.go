package mobility

import (
	"github.com/onosproject/ran-simulator/pkg/model"
	"math"
	"testing"
	"fmt"
)


// getPathLoss calculates the path loss based on the environment and LOS/NLOS conditions
func getPathLoss(coord model.Coordinate, cell model.Cell) float64 {
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
	pathLoss := 20*math.Log10(distanceKM) + 20*math.Log10(float64(cell.Channel.SSBFrequency) / 1000) + 92.45
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
	return earthRadius * c * 1000// distance in meters
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
	c := 3.0*math.Pow(10,8)
	hBS := float64(cell.Sector.Height) // base station height
	hUT := float64(1.5) // average height of user terminal 1m <= hUT <= 10m 
	fc := float64(cell.Channel.SSBFrequency) * 1000 // frequency in Hz

	dBP := 2*math.Pi*hBS*hUT*fc/c

	return dBP
}

// Breakpoint distance function
func getBreakpointPrimeDistance(cell model.Cell) float64 {
	c := 3.0*math.Pow(10,8)
	hE := float64(1) // assuming enviroment height is 1m
	hBS := float64(cell.Sector.Height) - hE // base station height  
	hUT := float64(1.5) - hE // average height of user terminal 1m <= hUT <= 10m 
	fc := float64(cell.Channel.SSBFrequency) * 1000 // frequency in Hz

	dBP := 4*hBS*hUT*fc/c

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
	h := float64(5) // average building height in m

	pl1 := 20*math.Log10(40*math.Pi*d*fc/3) + math.Min(0.03*math.Pow(h,1.72), 10)*math.Log10(d) -
		math.Min(0.044*math.Pow(h,1.72), 14.77) + 0.002*math.Log10(float64(h))*d

	return pl1
}

// getRuralNLOSPathLoss calculates the RMa NLOS path loss
func getRuralNLOSPathLoss(coord model.Coordinate, cell model.Cell) float64 {
	d3D := get3dEuclideanDistanceFromGPS(coord, cell)
	W := float64(20) // average street width 5m <= W <= 50m
	h := float64(5) // average building height 5m <= h <= 50m 
	hBS := float64(cell.Sector.Height) // base station height
	hUT := float64(1.5) // average height of user terminal 1m <= hUT <= 10m 

	plLOS := getRuralLOSPathLoss(coord, cell)
	plNLOS := 161.04 - 7.1*math.Log10(W) + 7.5*math.Log10(h) -
		(24.37 - 3.7*math.Pow((h/hBS),2))*math.Log10(hBS) +
		(43.42 - 3.1*math.Log10(hBS))*(math.Log10(d3D) - 3) +
		20*math.Log10(float64(cell.Channel.SSBFrequency)/1000) - 
		(math.Pow(3.2*math.Log10(11.75*hUT),2) - 4.97)

	return math.Max(plLOS, plNLOS)
}

// getUrbanLOSPathLoss calculates the UMa LOS path loss
func getUrbanLOSPathLoss(coord model.Coordinate, cell model.Cell) float64 {
	d2D := getEuclideanDistanceFromGPS(coord, cell)
	d3D := get3dEuclideanDistanceFromGPS(coord, cell)
	dBP := getBreakpointPrimeDistance(cell)
	hBS := float64(cell.Sector.Height) // base station height
	hUT := float64(5) // average height of user terminal 1m <= hUT <= 22.5m 
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
	hUT := float64(5) // average height of user terminal 1m <= W <= 22.5m 

	plLOS := getUrbanLOSPathLoss(coord, cell)
	plNLOS := 13.54 + 39.08*math.Log10(d3D) + 20*math.Log10(fc) -
		0.6*(hUT-1.5)

	return math.Max(plLOS, plNLOS)
}

func TestGetPathLossUrbanLOS(t *testing.T) {
	cell := model.Cell{
		Sector: model.Sector{
			Center: model.Coordinate{Lat: 0, Lng: 0},
			Height: 30,
		},
		Channel: model.Channel{
			Environment:  "urban",
			LOS:          true,
			SSBFrequency: 3600,
		},
	}

	coord := model.Coordinate{Lat: 1, Lng: 1}
	expectedPathLoss := getUrbanLOSPathLoss(coord, cell)

	pathLoss := getPathLoss(coord, cell)
	fmt.Println("UrbanLOS")
	fmt.Printf("expectedPathLoss: %f\n",expectedPathLoss)
	fmt.Printf("pathLoss: %f\n",pathLoss)
	if pathLoss != expectedPathLoss {
		t.Errorf("Expected %f but got %f", expectedPathLoss, pathLoss)
	}
}

func TestGetPathLossUrbanNLOS(t *testing.T) {
	cell := model.Cell{
		Sector: model.Sector{
			Center: model.Coordinate{Lat: 0, Lng: 0},
			Height: 30,
		},
		Channel: model.Channel{
			Environment:  "urban",
			LOS:          false,
			SSBFrequency: 3600,
		},
	}

	coord := model.Coordinate{Lat: 1, Lng: 1}
	expectedPathLoss := getUrbanNLOSPathLoss(coord, cell)

	pathLoss := getPathLoss(coord, cell)
	fmt.Println("UrbanNLOS")
	fmt.Printf("expectedPathLoss: %f\n",expectedPathLoss)
	fmt.Printf("pathLoss: %f\n",pathLoss)
	if pathLoss != expectedPathLoss {
		t.Errorf("Expected %f but got %f", expectedPathLoss, pathLoss)
	}
}

func TestGetPathLossRuralLOS(t *testing.T) {
	cell := model.Cell{
		Sector: model.Sector{
			Center: model.Coordinate{Lat: 0, Lng: 0},
			Height: 30,
		},
		Channel: model.Channel{
			Environment:  "rural",
			LOS:          true,
			SSBFrequency: 3600,
		},
	}

	coord := model.Coordinate{Lat: 1, Lng: 1}
	expectedPathLoss := getRuralLOSPathLoss(coord, cell)

	pathLoss := getPathLoss(coord, cell)
	fmt.Println("RuralLOS")
	fmt.Printf("expectedPathLoss: %f\n",expectedPathLoss)
	fmt.Printf("pathLoss: %f\n",pathLoss)
	if pathLoss != expectedPathLoss {
		t.Errorf("Expected %f but got %f", expectedPathLoss, pathLoss)
	}
}

func TestGetPathLossRuralNLOS(t *testing.T) {
	cell := model.Cell{
		Sector: model.Sector{
			Center: model.Coordinate{Lat: 0, Lng: 0},
			Height: 30,
		},
		Channel: model.Channel{
			Environment:  "rural",
			LOS:          false,
			SSBFrequency: 3600,
		},
	}

	coord := model.Coordinate{Lat: 1, Lng: 1}
	expectedPathLoss := getRuralNLOSPathLoss(coord, cell)

	pathLoss := getPathLoss(coord, cell)
	fmt.Println("RuralNLOS")
	fmt.Printf("expectedPathLoss: %f\n",expectedPathLoss)
	fmt.Printf("pathLoss: %f\n",pathLoss)
	if pathLoss != expectedPathLoss {
		t.Errorf("Expected %f but got %f", expectedPathLoss, pathLoss)
	}
}

func TestGetPathLossUnknownEnvironment(t *testing.T) {
	cell := model.Cell{
		Sector: model.Sector{
			Center: model.Coordinate{Lat: 0, Lng: 0},
			Height: 30,
		},
		Channel: model.Channel{
			Environment:  "unknown",
			LOS:          true,
			SSBFrequency: 3600,
		},
	}

	coord := model.Coordinate{Lat: 1, Lng: 1}
	expectedPathLoss := getFreeSpacePathLoss(coord, cell)

	pathLoss := getPathLoss(coord, cell)
	fmt.Println("Unknown")
	fmt.Printf("expectedPathLoss: %f\n",expectedPathLoss)
	fmt.Printf("pathLoss: %f\n",pathLoss)
	if pathLoss != expectedPathLoss {
		t.Errorf("Expected %f but got %f", expectedPathLoss, pathLoss)
	}
}

func TestGetChangingPositionPathLossUrbanLOS(t *testing.T) {
	cell := model.Cell{
		Sector: model.Sector{
			Center: model.Coordinate{Lat: 0, Lng: 0},
			Height: 30,
		},
		Channel: model.Channel{
			Environment:  "urban",
			LOS:          true,
			SSBFrequency: 3600,
		},
	}

	firstCoord := model.Coordinate{Lat: 0.0001, Lng: 0.0001}
	firstPathLoss := getPathLoss(firstCoord, cell)

	fmt.Println("UrbanLOS")
	fmt.Printf("(1,1) pathLoss: %f\n",firstPathLoss)

	secondCoord := model.Coordinate{Lat: 0.0002, Lng: 0.0002}
	secondPathLoss := getPathLoss(secondCoord, cell)

	fmt.Printf("(2,2) pathLoss: %f\n",secondPathLoss)

	if firstPathLoss > secondPathLoss {
		t.Errorf("Expected %f but got %f", firstPathLoss, secondPathLoss)
	}
}

func TestGetChangingPositionPathLossUrbanNLOS(t *testing.T) {
	cell := model.Cell{
		Sector: model.Sector{
			Center: model.Coordinate{Lat: 0, Lng: 0},
			Height: 30,
		},
		Channel: model.Channel{
			Environment:  "urban",
			LOS:          false,
			SSBFrequency: 3600,
		},
	}

	firstCoord := model.Coordinate{Lat: 0.0001, Lng: 0.0001}
	firstPathLoss := getPathLoss(firstCoord, cell)

	fmt.Println("UrbanNLOS")
	fmt.Printf("(1,1) pathLoss: %f\n",firstPathLoss)

	secondCoord := model.Coordinate{Lat: 0.0002, Lng: 0.0002}
	secondPathLoss := getPathLoss(secondCoord, cell)

	fmt.Printf("(2,2) pathLoss: %f\n",secondPathLoss)

	if firstPathLoss > secondPathLoss {
		t.Errorf("Expected %f but got %f", firstPathLoss, secondPathLoss)
	}
}

func TestGetChangingPositionPathLossRuralLOS(t *testing.T) {
	cell := model.Cell{
		Sector: model.Sector{
			Center: model.Coordinate{Lat: 0, Lng: 0},
			Height: 30,
		},
		Channel: model.Channel{
			Environment:  "rural",
			LOS:          true,
			SSBFrequency: 3600,
		},
	}

	firstCoord := model.Coordinate{Lat: 0.0001, Lng: 0.0001}
	firstPathLoss := getPathLoss(firstCoord, cell)

	fmt.Println("RuralLOS")
	fmt.Printf("(1,1) pathLoss: %f\n",firstPathLoss)

	secondCoord := model.Coordinate{Lat: 0.0002, Lng: 0.0002}
	secondPathLoss := getPathLoss(secondCoord, cell)

	fmt.Printf("(2,2) pathLoss: %f\n",secondPathLoss)

	if firstPathLoss > secondPathLoss {
		t.Errorf("Expected %f but got %f", firstPathLoss, secondPathLoss)
	}
}

func TestGetChangingPositionPathLossRuralNLOS(t *testing.T) {
	cell := model.Cell{
		Sector: model.Sector{
			Center: model.Coordinate{Lat: 0, Lng: 0},
			Height: 30,
		},
		Channel: model.Channel{
			Environment:  "rural",
			LOS:          false,
			SSBFrequency: 3600,
		},
	}

	firstCoord := model.Coordinate{Lat: 0.0001, Lng: 0.0001}
	firstPathLoss := getPathLoss(firstCoord, cell)

	fmt.Println("RuralNLOS")
	fmt.Printf("(1,1) pathLoss: %f\n",firstPathLoss)

	secondCoord := model.Coordinate{Lat: 0.0002, Lng: 0.0002}
	secondPathLoss := getPathLoss(secondCoord, cell)

	fmt.Printf("(2,2) pathLoss: %f\n",secondPathLoss)

	if firstPathLoss > secondPathLoss {
		t.Errorf("Expected %f but got %f", firstPathLoss, secondPathLoss)
	}
}