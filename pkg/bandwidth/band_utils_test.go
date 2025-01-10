package bandwidth

import (
	"fmt"
	"testing"
)

func TestGetBand(t *testing.T) {
	// Expected result
	// expectedBand := "n77"
	frequency := float64(1975)
	direction := UL
	band := GetBand(frequency, direction)
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
