package mobility

import (
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/stretchr/testify/assert"
)

func Test_AzimuthAttenuation(t *testing.T) {
	azimuths := map[string]int32{
		"boresight":  0,
		"halfPower1": 33,
		"halfPower2": -33,
	}

	expectedAttenuationDb := map[string]int{
		"boresight":  0,
		"halfPower1": -3,
		"halfPower2": -3,
	}

	for point, azimuth := range azimuths {
		assert.Equal(t, expectedAttenuationDb[point], int(azimuthAttenuation(azimuth, 65, 30)))
	}
}

func Test_ZenithAttenuation(t *testing.T) {
	zenithAngles := map[string]uint32{
		"boresight":  90,
		"halfPower1": 57,  // 90 - 65/2
		"halfPower2": 123, // 90 + 65/2
	}

	expectedAttenuationDb := map[string]int{
		"boresight":  0,
		"halfPower1": -3,
		"halfPower2": -3,
	}

	for point, zenithAngle := range zenithAngles {
		assert.Equal(t, expectedAttenuationDb[point], int(zenithAttenuation(zenithAngle, 65, 30)))
	}

}

func Test_AngularAttenuation(t *testing.T) {

	ue := model.UE{
		Height: 1.5,
	}

	cell := model.Cell{
		Sector: model.Sector{
			Azimuth: 21,
			Center:  model.Coordinate{Lat: 37.979207, Lng: 23.716702},
			Height:  30,
		},
		Beam: model.Beam{
			H3dBAngle:              65,
			V3dBAngle:              65,
			MaxGain:                8,
			MaxAttenuationDB:       30,
			VSideLobeAttenuationDB: 30,
		},
	}

	ue.Location = model.Coordinate{Lat: 37.985168, Lng: 23.720989}
	assert.Equal(t, -3, int(angularAttenuation(ue, cell)))

	cell.Sector.Tilt = 37
	ue.Location = model.Coordinate{Lat: 37.979207, Lng: 23.720989}
	expectedHAttenuation := -1
	expectedVAttenuation := -3
	assert.Equal(t, expectedHAttenuation+expectedVAttenuation, int(angularAttenuation(ue, cell)))

}
