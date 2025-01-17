package signal

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
)

func TestGetPathLossUrbanLOS(t *testing.T) {
	carrier := &model.Carrier{
		Center:      model.Coordinate{Lat: 0, Lng: 0},
		Height:      30,
		Environment: "urban",
		LOS:         true,
		ArfcnDL:     640000,
	}

	coord := model.Coordinate{Lat: 1, Lng: 1}
	ueHeight := 1.5
	expectedPathLoss := getUrbanLOSPathLoss(coord, ueHeight, carrier)

	pathLoss := GetPathLoss(coord, ueHeight, carrier)
	fmt.Println("UrbanLOS")
	fmt.Printf("expectedPathLoss: %f\n", expectedPathLoss)
	fmt.Printf("pathLoss: %f\n", pathLoss)
	if pathLoss != expectedPathLoss {
		t.Errorf("Expected %f but got %f", expectedPathLoss, pathLoss)
	}
}

func TestGetPathLossUrbanNLOS(t *testing.T) {
	carrier := &model.Carrier{
		Center:      model.Coordinate{Lat: 0, Lng: 0},
		Height:      30,
		Environment: "urban",
		LOS:         false,
		ArfcnDL:     640000,
	}

	coord := model.Coordinate{Lat: 1, Lng: 1}
	ueHeight := 1.5
	expectedPathLoss := getUrbanNLOSPathLoss(coord, ueHeight, carrier)

	pathLoss := GetPathLoss(coord, ueHeight, carrier)
	fmt.Println("UrbanNLOS")
	fmt.Printf("expectedPathLoss: %f\n", expectedPathLoss)
	fmt.Printf("pathLoss: %f\n", pathLoss)
	if pathLoss != expectedPathLoss {
		t.Errorf("Expected %f but got %f", expectedPathLoss, pathLoss)
	}
}

func TestGetPathLossRuralLOS(t *testing.T) {
	carrier := &model.Carrier{
		Center:      model.Coordinate{Lat: 0, Lng: 0},
		Height:      30,
		Environment: "rural",
		LOS:         true,
		ArfcnDL:     640000,
	}

	coord := model.Coordinate{Lat: 1, Lng: 1}
	ueHeight := 1.5
	expectedPathLoss := getRuralLOSPathLoss(coord, ueHeight, carrier)

	pathLoss := GetPathLoss(coord, ueHeight, carrier)
	fmt.Println("RuralLOS")
	fmt.Printf("expectedPathLoss: %f\n", expectedPathLoss)
	fmt.Printf("pathLoss: %f\n", pathLoss)
	if pathLoss != expectedPathLoss {
		t.Errorf("Expected %f but got %f", expectedPathLoss, pathLoss)
	}
}

func TestGetPathLossRuralNLOS(t *testing.T) {
	carrier := &model.Carrier{
		Center:      model.Coordinate{Lat: 0, Lng: 0},
		Height:      30,
		Environment: "rural",
		LOS:         false,
		ArfcnDL:     640000,
	}

	coord := model.Coordinate{Lat: 1, Lng: 1}
	ueHeight := 1.5
	expectedPathLoss := getRuralNLOSPathLoss(coord, ueHeight, carrier)

	pathLoss := GetPathLoss(coord, ueHeight, carrier)
	fmt.Println("RuralNLOS")
	fmt.Printf("expectedPathLoss: %f\n", expectedPathLoss)
	fmt.Printf("pathLoss: %f\n", pathLoss)
	if pathLoss != expectedPathLoss {
		t.Errorf("Expected %f but got %f", expectedPathLoss, pathLoss)
	}
}

func TestGetPathLossUnknownEnvironment(t *testing.T) {
	carrier := &model.Carrier{
		Center:      model.Coordinate{Lat: 0, Lng: 0},
		Height:      30,
		Environment: "unknown",
		LOS:         true,
		ArfcnDL:     640000,
	}

	coord := model.Coordinate{Lat: 1, Lng: 1}
	ueHeight := 1.5
	expectedPathLoss := getRuralNLOSPathLoss(coord, ueHeight, carrier)

	pathLoss := GetPathLoss(coord, ueHeight, carrier)
	fmt.Println("Unknown")
	fmt.Printf("expectedPathLoss: %f\n", expectedPathLoss)
	fmt.Printf("pathLoss: %f\n", pathLoss)
	if pathLoss != expectedPathLoss {
		t.Errorf("Expected %f but got %f", expectedPathLoss, pathLoss)
	}
}

func TestGetChangingPositionPathLossUrbanLOS(t *testing.T) {

	carrier := &model.Carrier{
		Center:      model.Coordinate{Lat: 0, Lng: 0},
		Height:      30,
		Environment: "urban",
		LOS:         true,
		ArfcnDL:     640000,
	}

	firstCoord := model.Coordinate{Lat: 0.0001, Lng: 0.0001}
	ueHeight := 1.5
	firstPathLoss := GetPathLoss(firstCoord, ueHeight, carrier)

	fmt.Println("UrbanLOS")
	fmt.Printf("(1,1) pathLoss: %f\n", firstPathLoss)

	secondCoord := model.Coordinate{Lat: 0.0002, Lng: 0.0002}

	secondPathLoss := GetPathLoss(secondCoord, ueHeight, carrier)

	fmt.Printf("(2,2) pathLoss: %f\n", secondPathLoss)

	if firstPathLoss > secondPathLoss {
		t.Errorf("Expected %f but got %f", firstPathLoss, secondPathLoss)
	}
}

func TestGetChangingPositionPathLossUrbanNLOS(t *testing.T) {
	carrier := &model.Carrier{
		Center:      model.Coordinate{Lat: 0, Lng: 0},
		Height:      30,
		Environment: "urban",
		LOS:         false,
		ArfcnDL:     640000,
	}

	firstCoord := model.Coordinate{Lat: 0.0001, Lng: 0.0001}
	ueHeight := 1.5
	firstPathLoss := GetPathLoss(firstCoord, ueHeight, carrier)

	fmt.Println("UrbanNLOS")
	fmt.Printf("(1,1) pathLoss: %f\n", firstPathLoss)

	secondCoord := model.Coordinate{Lat: 0.0002, Lng: 0.0002}
	secondPathLoss := GetPathLoss(secondCoord, ueHeight, carrier)

	fmt.Printf("(2,2) pathLoss: %f\n", secondPathLoss)

	if firstPathLoss > secondPathLoss {
		t.Errorf("Expected %f but got %f", firstPathLoss, secondPathLoss)
	}
}

func TestGetChangingPositionPathLossRuralLOS(t *testing.T) {
	carrier := &model.Carrier{
		Center:      model.Coordinate{Lat: 0, Lng: 0},
		Height:      30,
		Environment: "rural",
		LOS:         true,
		ArfcnDL:     640000,
	}

	ueHeight := 1.5
	firstCoord := model.Coordinate{Lat: 0.0001, Lng: 0.0001}
	firstPathLoss := GetPathLoss(firstCoord, ueHeight, carrier)

	fmt.Println("RuralLOS")
	fmt.Printf("(1,1) pathLoss: %f\n", firstPathLoss)

	secondCoord := model.Coordinate{Lat: 0.0002, Lng: 0.0002}
	secondPathLoss := GetPathLoss(secondCoord, ueHeight, carrier)

	fmt.Printf("(2,2) pathLoss: %f\n", secondPathLoss)

	if firstPathLoss > secondPathLoss {
		t.Errorf("Expected %f but got %f", firstPathLoss, secondPathLoss)
	}
}

func TestGetChangingPositionPathLossRuralNLOS(t *testing.T) {
	carrier := &model.Carrier{
		Center:      model.Coordinate{Lat: 0, Lng: 0},
		Height:      30,
		Environment: "rural",
		LOS:         false,
		ArfcnDL:     640000,
	}

	ueHeight := 1.5
	firstCoord := model.Coordinate{Lat: 0.0001, Lng: 0.0001}
	firstPathLoss := GetPathLoss(firstCoord, ueHeight, carrier)

	fmt.Println("RuralNLOS")
	fmt.Printf("(1,1) pathLoss: %f\n", firstPathLoss)

	secondCoord := model.Coordinate{Lat: 0.0002, Lng: 0.0002}
	secondPathLoss := GetPathLoss(secondCoord, ueHeight, carrier)

	fmt.Printf("(2,2) pathLoss: %f\n", secondPathLoss)

	if firstPathLoss > secondPathLoss {
		t.Errorf("Expected %f but got %f", firstPathLoss, secondPathLoss)
	}
}

func TestPathloss(t *testing.T) {
	carrier := &model.Carrier{
		Center:  model.Coordinate{Lat: 0, Lng: 0},
		Height:  35,
		ArfcnDL: 620000,
	}

	ueHeight := 1.5
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
		dist3d := get3dEuclideanDistanceFromGPS(coord, ueHeight, carrier)
		urbanLOSPathLoss := getUrbanLOSPathLoss(coord, ueHeight, carrier)
		urbanNLOSPathLoss := getUrbanNLOSPathLoss(coord, ueHeight, carrier)
		ruralLOSPathLoss := getRuralLOSPathLoss(coord, ueHeight, carrier)
		ruralNLOSPathLoss := getRuralNLOSPathLoss(coord, ueHeight, carrier)
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
