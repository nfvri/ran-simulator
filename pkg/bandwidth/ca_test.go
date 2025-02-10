package bandwidth

import (
	"testing"

	art "github.com/plar/go-adaptive-radix-tree/v2"
	"gotest.tools/assert"
)

func Test_CarrierAggregator_IsValidBandCombination(t *testing.T) {

	caNR := NewCarrierAggregatorNR()

	for _, bandCombo := range CABandCombinationsNR {
		_, found := caNR.bandCombinationsNRTree.Search(art.Key(bandCombo))
		assert.Equal(t, found, true)
	}

}
