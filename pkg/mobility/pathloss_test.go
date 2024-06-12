package mobility

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"

	"github.com/nfvri/ran-simulator/pkg/model"
)

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

	pathLoss := GetPathLoss(coord, cell)
	fmt.Println("UrbanLOS")
	fmt.Printf("expectedPathLoss: %f\n", expectedPathLoss)
	fmt.Printf("pathLoss: %f\n", pathLoss)
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

	pathLoss := GetPathLoss(coord, cell)
	fmt.Println("UrbanNLOS")
	fmt.Printf("expectedPathLoss: %f\n", expectedPathLoss)
	fmt.Printf("pathLoss: %f\n", pathLoss)
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

	pathLoss := GetPathLoss(coord, cell)
	fmt.Println("RuralLOS")
	fmt.Printf("expectedPathLoss: %f\n", expectedPathLoss)
	fmt.Printf("pathLoss: %f\n", pathLoss)
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

	pathLoss := GetPathLoss(coord, cell)
	fmt.Println("RuralNLOS")
	fmt.Printf("expectedPathLoss: %f\n", expectedPathLoss)
	fmt.Printf("pathLoss: %f\n", pathLoss)
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

	pathLoss := GetPathLoss(coord, cell)
	fmt.Println("Unknown")
	fmt.Printf("expectedPathLoss: %f\n", expectedPathLoss)
	fmt.Printf("pathLoss: %f\n", pathLoss)
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
	firstPathLoss := GetPathLoss(firstCoord, cell)

	fmt.Println("UrbanLOS")
	fmt.Printf("(1,1) pathLoss: %f\n", firstPathLoss)

	secondCoord := model.Coordinate{Lat: 0.0002, Lng: 0.0002}
	secondPathLoss := GetPathLoss(secondCoord, cell)

	fmt.Printf("(2,2) pathLoss: %f\n", secondPathLoss)

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
	firstPathLoss := GetPathLoss(firstCoord, cell)

	fmt.Println("UrbanNLOS")
	fmt.Printf("(1,1) pathLoss: %f\n", firstPathLoss)

	secondCoord := model.Coordinate{Lat: 0.0002, Lng: 0.0002}
	secondPathLoss := GetPathLoss(secondCoord, cell)

	fmt.Printf("(2,2) pathLoss: %f\n", secondPathLoss)

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
	firstPathLoss := GetPathLoss(firstCoord, cell)

	fmt.Println("RuralLOS")
	fmt.Printf("(1,1) pathLoss: %f\n", firstPathLoss)

	secondCoord := model.Coordinate{Lat: 0.0002, Lng: 0.0002}
	secondPathLoss := GetPathLoss(secondCoord, cell)

	fmt.Printf("(2,2) pathLoss: %f\n", secondPathLoss)

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
	firstPathLoss := GetPathLoss(firstCoord, cell)

	fmt.Println("RuralNLOS")
	fmt.Printf("(1,1) pathLoss: %f\n", firstPathLoss)

	secondCoord := model.Coordinate{Lat: 0.0002, Lng: 0.0002}
	secondPathLoss := GetPathLoss(secondCoord, cell)

	fmt.Printf("(2,2) pathLoss: %f\n", secondPathLoss)

	if firstPathLoss > secondPathLoss {
		t.Errorf("Expected %f but got %f", firstPathLoss, secondPathLoss)
	}
}

func TestPathloss(t *testing.T) {
	cell := model.Cell{
		Sector: model.Sector{
			Center: model.Coordinate{Lat: 0, Lng: 0},
			Height: 35,
		},
		Channel: model.Channel{
			SSBFrequency: 3000,
		},
	}

	file, err := os.Create("pathloss.csv")
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"Index", "UE position", "distance3D", "pathlossUmaLOS", "pathlossUmaNLOS", "pathlossRmaLOS", "pathlossRmaNLOS"}
	if err := writer.Write(header); err != nil {
		t.Fatalf("failed to write header to CSV: %v", err)
	}

	fmt.Printf("Index \t| UE position \t\t| distance3D \t| pathlossUmaLOS \t| pathlossUmaNLOS \t| pathlossRmaLOS \t| pathlossRmaNLOS |\n")
	for i := 0; i <= 20; i++ {
		lng := 0 + (float64(i) * math.Pow(10, -3))

		coord := model.Coordinate{Lat: 0.0001, Lng: lng}
		dist3d := get3dEuclideanDistanceFromGPS(coord, cell)
		urbanLOSPathLoss := getUrbanLOSPathLoss(coord, cell)
		urbanNLOSPathLoss := getUrbanNLOSPathLoss(coord, cell)
		ruralLOSPathLoss := getRuralLOSPathLoss(coord, cell)
		ruralNLOSPathLoss := getRuralNLOSPathLoss(coord, cell)
		fmt.Printf("%d \t| %.3v \t| %f \t| %f \t\t| %f \t\t| %f \t\t| %f \t|\n", i, coord, dist3d, urbanLOSPathLoss, urbanNLOSPathLoss, ruralLOSPathLoss, ruralNLOSPathLoss)

		row := []string{
			fmt.Sprintf("%d", i),
			fmt.Sprintf("%.3v", coord),
			fmt.Sprintf("%f", dist3d),
			fmt.Sprintf("%f", urbanLOSPathLoss),
			fmt.Sprintf("%f", urbanNLOSPathLoss),
			fmt.Sprintf("%f", ruralLOSPathLoss),
			fmt.Sprintf("%f", ruralNLOSPathLoss),
		}
		if err := writer.Write(row); err != nil {
			t.Fatalf("failed to write row to CSV: %v", err)
		}
	}
}
