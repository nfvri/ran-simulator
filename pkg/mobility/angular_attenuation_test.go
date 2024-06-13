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
		assert.Equal(t, expectedAttenuationDb[point], int(azimuthAttenuation(azimuth, 65)))
	}
}

func Test_ZenithAttenuation(t *testing.T) {
	zenithAngles := map[string]int32{
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
		assert.Equal(t, expectedAttenuationDb[point], int(zenithAttenuation(zenithAngle, 65)))
	}

}

func Test_AngularAttenuation(t *testing.T) {
	cell := model.Cell{
		Sector: model.Sector{
			Azimuth: 21,
			Center:  model.Coordinate{Lat: 37.979207, Lng: 23.716702},
			Height:  30,
		},
		Channel: model.Channel{
			Environment:  "urban",
			LOS:          true,
			SSBFrequency: 3600,
		},
	}

	ueLocation_azimuth_33 := model.Coordinate{Lat: 37.985168, Lng: 23.720989}
	assert.Equal(t, -3, int(angularAttenuation(ueLocation_azimuth_33, cell)))

	cell.Sector.Tilt = 4
	ueLocation_zenith_33 := model.Coordinate{Lat: 37.979207, Lng: 23.720989}
	assert.Equal(t, -1, int(angularAttenuation(ueLocation_zenith_33, cell)))

}
