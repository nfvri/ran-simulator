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
		CellType:  types.CellType_MACRO,
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
		CellType:  types.CellType_MACRO,
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
			SSBFrequency: 900,
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
		CellType:  types.CellType_MACRO,
		Sector: model.Sector{
			Center:  model.Coordinate{Lat: 37.976548691741705, Lng: 23.757677078247074},
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
	cell.RPCoverageBoundaries = []model.CoverageBoundary{
		{
			RefSignalStrength: -90,
			BoundaryPoints:    GetRPBoundaryPoints(1.5, &cell, -90),
		},
	}
	cell1.RPCoverageBoundaries = []model.CoverageBoundary{
		{
			RefSignalStrength: -90,
			BoundaryPoints:    GetRPBoundaryPoints(1.5, &cell1, -90),
		},
	}
	cell2.RPCoverageBoundaries = []model.CoverageBoundary{
		{
			RefSignalStrength: -90,
			BoundaryPoints:    GetRPBoundaryPoints(1.5, &cell2, -90),
		},
	}
	InitShadowMap(&cell, 150)
	InitShadowMap(&cell1, 150)
	InitShadowMap(&cell2, 150)
	sinr := Sinr(model.Coordinate{Lat: 37.981419843336816, Lng: 23.7590503692627}, ueHeight, &cell1, []*model.Cell{&cell2})
	t.Logf("[%f], \n", sinr)

	// Output:
	//
	// Root: (x, y) = (1.00, 2.00)
	// Function value: (-0.00, 0.00)
}
