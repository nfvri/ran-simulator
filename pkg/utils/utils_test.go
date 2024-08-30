// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package utils

import (
	"bytes"
	"math"
	"testing"

	"github.com/onosproject/onos-api/go/onos/ransim/types"
	"gotest.tools/assert"
)

const (
	PosCenLat = 52.12345
	PosCenLng = 13.12345
	Pos1Lat   = 52.12350
	Pos1Lng   = 13.12350
	Pos2Lat   = 52.12340
	Pos2Lng   = 13.12340
)

func Test_GetRotationDegrees(t *testing.T) {
	centre := types.Point{
		Lat: PosCenLat,
		Lng: PosCenLng,
	}
	p1 := types.Point{
		Lat: Pos1Lat,
		Lng: Pos1Lng,
	}
	p2 := types.Point{
		Lat: Pos2Lat,
		Lng: Pos2Lng,
	}
	p3 := types.Point{
		Lat: Pos2Lat,
		Lng: Pos1Lng,
	}
	p4 := types.Point{
		Lat: Pos1Lat,
		Lng: Pos2Lng,
	}
	r1 := GetRotationDegrees(&centre, &p1)
	assert.Equal(t, 45.0, math.Round(r1), "Unexpected r1")

	r2 := GetRotationDegrees(&centre, &p2)
	assert.Equal(t, -135.0, math.Round(r2), "Unexpected r2")

	r3 := GetRotationDegrees(&centre, &p3)
	assert.Equal(t, -45.0, math.Round(r3), "Unexpected r3")

	r4 := GetRotationDegrees(&centre, &p4)
	assert.Equal(t, 135.0, math.Round(r4), "Unexpected r4")
}

func Test_RandomColor(t *testing.T) {
	c := RandomColor()
	assert.Equal(t, 7, len(c))
	assert.Equal(t, uint8('#'), c[0])
}

func Test_GetRandomLngLat(t *testing.T) {
	const radius = 0.2
	for i := 0; i < 100; i++ {
		pt := RandomLatLng(0.0, 0.0, radius, 1)
		assert.Assert(t, pt.GetLat() < radius, "Expecting position %f to be within radius", pt.GetLat())
	}
}

func Test_RoundToDecimal(t *testing.T) {
	floatValue := 1.123456789
	roundedValue := RoundToDecimal(floatValue, 4)
	assert.Equal(t, roundedValue, 1.1235)

}

func Test_AzimuthToRads(t *testing.T) {
	assert.Equal(t, math.Pi/2, AzimuthToRads(0))
	assert.Equal(t, 0.0, AzimuthToRads(90))
	assert.Equal(t, -math.Pi/2, AzimuthToRads(180))
	assert.Equal(t, -math.Pi, AzimuthToRads(270))
	assert.Equal(t, math.Round(10e6*math.Pi/3), math.Round(10e6*AzimuthToRads(30)))
}

func Test_AspectRatio(t *testing.T) {
	ar := AspectRatio(52.52)
	assert.Equal(t, 608, int(math.Round(ar*1e3)))
}

func Test_ImsiGenerator(t *testing.T) {
	imsi := ImsiGenerator(50)
	assert.Equal(t, imsi, types.IMSI(315010999900051))
}

func Test_Uint64ToBitString(t *testing.T) {
	tests := []struct {
		value    uint64
		bitCount int
		expected []byte
	}{
		{value: 0xFF, bitCount: 8, expected: []byte{0x0, 0xFF}},
		{value: 0x1234, bitCount: 16, expected: []byte{0x0, 0x12, 0x34}},
		{value: 0x1234, bitCount: 12, expected: []byte{0x23, 0x40}},
		{value: 0x123456, bitCount: 24, expected: []byte{0x0, 0x12, 0x34, 0x56}},
		{value: 0x123456789ABCDEF, bitCount: 60, expected: []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			actual := Uint64ToBitString(tt.value, tt.bitCount)
			if !bytes.Equal(actual, tt.expected) {
				t.Errorf("Uint64ToBitString(%v, %v) = %v; want %v", tt.value, tt.bitCount, actual, tt.expected)
			}
		})
	}
}

func Test_BitStringToUint64(t *testing.T) {
	tests := []struct {
		bitString []byte
		bitCount  int
		expected  uint64
	}{
		{bitString: []byte{0xFF}, bitCount: 8, expected: 0xFF},
		{bitString: []byte{0x12, 0x34}, bitCount: 16, expected: 0x1234},
		{bitString: []byte{0x12, 0x30}, bitCount: 12, expected: 0x123},
		{bitString: []byte{0x12, 0x34, 0x56}, bitCount: 24, expected: 0x123456},
		{bitString: []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}, bitCount: 60, expected: 0x123456789ABCDEF},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			actual := BitStringToUint64(tt.bitString, tt.bitCount)
			if actual != tt.expected {
				t.Errorf("BitStringToUint64(%v, %v) = %v; want %v", tt.bitString, tt.bitCount, actual, tt.expected)
			}
		})
	}
}
