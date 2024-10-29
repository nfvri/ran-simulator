package bandwidth

import (
	"testing"

	"gotest.tools/assert"
)

func TestGetNumUEsPerCQIByCell(t *testing.T) {
	var c1Ncgi uint64 = 12345
	var c2Ncgi uint64 = 12346
	numUEsByCell := map[uint64]map[string]int{
		c1Ncgi: {
			ACTIVE_UES_DL_METRIC: 150,
		},
		c2Ncgi: {
			ACTIVE_UES_DL_METRIC:        5000,
			ACTIVE_UES_DL_METRIC + ".1": 50,
			ACTIVE_UES_DL_METRIC + ".5": 50,
		},
	}

	// Expected result
	expectedMap := map[uint64]map[int]int{
		c1Ncgi: {},
		c2Ncgi: {
			1: 50,
			5: 50,
		},
	}
	for cqi := 1; cqi <= 15; cqi++ {
		expectedMap[c1Ncgi][cqi] = 10
	}

	assert.Equal(t, expectedMap, GetNumUEsPerCQIByCell(numUEsByCell))
}
