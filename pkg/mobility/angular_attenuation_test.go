package mobility

import (
	"testing"

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
		assert.Equal(t, expectedAttenuationDb[point], int(azimuthAttenuation(azimuth)))
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
		assert.Equal(t, expectedAttenuationDb[point], int(zenithAttenuation(zenithAngle)))
	}

}
