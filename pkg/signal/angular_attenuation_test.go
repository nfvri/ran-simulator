package signal

import (
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/stretchr/testify/assert"
)

func Test_AzimuthAttenuation(t *testing.T) {
	azimuths := map[string]float64{
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
	zenithAngles := map[string]float64{
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

	carrier := &model.Carrier{
		Beams: []*model.Beam{
			{
				Azimuth:   90,
				H3dBAngle: 65,
				V3dBAngle: 65,
				MaxGain:   8,
			},
		},
		TxPowerDB:              45,
		Center:                 model.Coordinate{Lat: 37.979207, Lng: 23.716702},
		Height:                 30,
		VSideLobeAttenuationDB: 30,
	}

	// Test horizontal -3dB Coordinate
	// ue.Location = model.Coordinate{Lat: 37.976707, Lng: 23.720902}
	// assert.Equal(t, -3, int(angularAttenuation(ue.Location, ue.Height, cell)))

	// Test symmetric vertical -3dB Points
	expectedHAttenuation := 0
	expectedVAttenuation := -3

	ue.Location = model.Coordinate{Lat: 37.979207, Lng: 23.720989} // 4 degree vertical angle from cell center
	carrier.Beams[0].Tilt = -29
	assert.Equal(t, expectedHAttenuation+expectedVAttenuation, int(angularAttenuation(ue.Location, ue.Height, carrier, 0)))

	carrier.Beams[0].Tilt = 37
	assert.Equal(t, expectedHAttenuation+expectedVAttenuation, int(angularAttenuation(ue.Location, ue.Height, carrier, 0)))

	// Test horizon
	carrier.Beams[0].Tilt = 4 // target ue
	expectedVAttenuation = 0
	assert.Equal(t, expectedHAttenuation+expectedVAttenuation, int(angularAttenuation(ue.Location, ue.Height, carrier, 0)))

}
