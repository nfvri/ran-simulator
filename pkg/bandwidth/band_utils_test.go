package bandwidth

import (
	"fmt"
	"testing"
)

func TestGetBand(t *testing.T) {
	// Expected result
	// expectedBand := "n77"
	var arfcn uint32 = 1975
	direction := UL
	band, _ := GetBand(arfcn, direction)
	fmt.Printf("Band: %v  \n", band)
	// assert.Equal(t, band, expectedBand)

}

func TestGetFR(t *testing.T) {
	// Expected result
	// expectedBand := ""
	frequency := float64(100)
	fr := GetFR(frequency)
	fmt.Printf("FR: %v  \n", fr)
	// assert.Equal(t, band, expectedBand)

}
