package signal

import (
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
)

func TestSinrAtLocationNewtonKrylov(t *testing.T) {

	cell := model.Cell{
		NCGI:      17660905553922,
		TxPowerDB: 40,
		CellType:  types.CellType_FEMTO,
		Sector: model.Sector{
			Center:  model.Coordinate{Lat: 38.024973, Lng: 23.767187},
			Azimuth: 150,
			Arc:     90,
			Tilt:    20,
			Height:  0,
		},
		Channel: model.Channel{
			Environment:  "urban",
			LOS:          true,
			SSBFrequency: 3425,
		},
		Beam: model.Beam{
			H3dBAngle:              90,
			V3dBAngle:              65,
			MaxGain:                8,
			MaxAttenuationDB:       40,
			VSideLobeAttenuationDB: 40,
		},
		Neighbors: []types.NCGI{17660905537537, 17660905570307},
	}
	cell1 := model.Cell{
		NCGI:      17660905537537,
		TxPowerDB: 40,
		CellType:  types.CellType_FEMTO,
		Sector: model.Sector{
			Center:  model.Coordinate{Lat: 37.981629, Lng: 23.743353},
			Azimuth: 90,
			Arc:     90,
			Tilt:    20,
			Height:  0,
		},
		Neighbors: []types.NCGI{17660905570307, cell.NCGI},
		Channel: model.Channel{
			Environment:  "urban",
			LOS:          true,
			SSBFrequency: 3425,
		},
		Beam: model.Beam{
			H3dBAngle:              90,
			V3dBAngle:              65,
			MaxGain:                8,
			MaxAttenuationDB:       40,
			VSideLobeAttenuationDB: 40,
		},
	}

	cell2 := model.Cell{
		NCGI:      17660905570307,
		TxPowerDB: 40,
		CellType:  types.CellType_FEMTO,
		Sector: model.Sector{
			Center:  model.Coordinate{Lat: 37.969502, Lng: 23.796955},
			Azimuth: 0,
			Arc:     120,
			Tilt:    20,
			Height:  0,
		},
		Neighbors: []types.NCGI{cell1.NCGI, cell.NCGI},
		Channel: model.Channel{
			Environment:  "urban",
			LOS:          true,
			SSBFrequency: 900,
		},
		Beam: model.Beam{
			H3dBAngle:              120,
			V3dBAngle:              65,
			MaxGain:                8,
			MaxAttenuationDB:       40,
			VSideLobeAttenuationDB: 40,
		},
	}
	ueHeight := 1.5
	sinr := Sinr(model.Coordinate{Lat: 38.031206366546336, Lng: 23.76793681488556}, ueHeight, &cell, []*model.Cell{&cell1, &cell2})
	t.Logf("[%f], \n", sinr)

	// Output:
	//
	// Root: (x, y) = (1.00, 2.00)
	// Function value: (-0.00, 0.00)
}

func TestCalculateNoisePower(t *testing.T) {
	bandwidth := 10e6 // 4.096e6 // 20e6 // 20 MHz bandwidth
	noise := CalculateNoisePower(bandwidth, types.CellType_ENTERPRISE)

	t.Logf("noise: %f \n", noise)
}
