// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package utils

import (
	"math"
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
)

func Test_Distance(t *testing.T) {
	tests := []struct {
		name     string
		c1       model.Coordinate
		c2       model.Coordinate
		expected float64
	}{
		{
			name:     "Same Location",
			c1:       model.Coordinate{Lat: 0, Lng: 0},
			c2:       model.Coordinate{Lat: 0, Lng: 0},
			expected: 0,
		},
		{
			name:     "Approximate distance of 1000 meters ",
			c1:       model.Coordinate{Lat: 48.858844, Lng: 2.294521},
			c2:       model.Coordinate{Lat: 48.860150, Lng: 2.308033},
			expected: 1000.207797, // Approximate distance of 1000 meters
		},
		{
			name:     "Distance between New York and London",
			c1:       model.Coordinate{Lat: 40.7128, Lng: -74.0060}, // New York
			c2:       model.Coordinate{Lat: 51.5074, Lng: -0.1278},  // London
			expected: 5576429.773126,                                // ~5585 km
		},
		{
			name:     "Distance between Sydney and Melbourne",
			c1:       model.Coordinate{Lat: -33.8688, Lng: 151.2093}, // Sydney
			c2:       model.Coordinate{Lat: -37.8136, Lng: 144.9631}, // Melbourne
			expected: 714222.541953,                                  // ~714 km
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := RoundToDecimal(Distance(tt.c1, tt.c2), 6)
			if math.Abs(actual-tt.expected) > 0 {
				t.Errorf("Distance(%v, %v) = %v; want %v", tt.c1, tt.c2, actual, tt.expected)
			}
		})
	}
}

func Test_GetSphericalDistance(t *testing.T) {
	tests := []struct {
		name     string
		c1       model.Coordinate
		c2       model.Coordinate
		expected float64
	}{
		{
			name:     "Same Location",
			c1:       model.Coordinate{Lat: 0, Lng: 0},
			c2:       model.Coordinate{Lat: 0, Lng: 0},
			expected: 0,
		},
		{
			name:     "1000 meters distance",
			c1:       model.Coordinate{Lat: 48.858844, Lng: 2.294521},
			c2:       model.Coordinate{Lat: 48.860150, Lng: 2.308033},
			expected: 1000.213600, // Approximate distance of 1000 meters
		},
		{
			name:     "Distance between New York and London",
			c1:       model.Coordinate{Lat: 40.7128, Lng: -74.0060}, // New York
			c2:       model.Coordinate{Lat: 51.5074, Lng: -0.1278},  // London
			expected: 5576462.122556,                                // ~5585 km
		},
		{
			name:     "Distance between Sydney and Melbourne",
			c1:       model.Coordinate{Lat: -33.8688, Lng: 151.2093}, // Sydney
			c2:       model.Coordinate{Lat: -37.8136, Lng: 144.9631}, // Melbourne
			expected: 714226.685230,                                  // ~714 km
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RoundToDecimal(GetSphericalDistance(tt.c1, tt.c2), 6)
			if math.Abs(result-tt.expected) > 0 {
				t.Errorf("Got %f, want %f", result, tt.expected)
			}
		})
	}
}

func Test_BearingDistanceTargetPoint(t *testing.T) {
	tests := []struct {
		name string
		c1   model.Coordinate
		c2   model.Coordinate
	}{
		{
			name: "Same Location",
			c1:   model.Coordinate{Lat: 0, Lng: 0},
			c2:   model.Coordinate{Lat: 0, Lng: 0},
		},
		{
			name: "New York to London",
			c1:   model.Coordinate{Lat: 40.7128, Lng: -74.0060}, // New York
			c2:   model.Coordinate{Lat: 51.5074, Lng: -0.1278},  // London
		},
		{
			name: "Sydney to Melbourne",
			c1:   model.Coordinate{Lat: -33.8688, Lng: 151.2093}, // Sydney
			c2:   model.Coordinate{Lat: -37.8136, Lng: 144.9631}, // Melbourne
		},
		{
			name: "Test Point with 800 meters distance",
			c1:   model.Coordinate{Lat: 40.7128, Lng: -74.0060}, // New York
			c2:   model.Coordinate{Lat: 40.7175, Lng: -74.0046}, // ~800 meters northeast
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bearing := InitialBearing(tt.c1, tt.c2)
			distance := Distance(tt.c1, tt.c2)
			newPoint := TargetPoint(tt.c1, bearing, distance)

			if math.Abs(newPoint.Lat-tt.c2.Lat) > 0.001 || math.Abs(newPoint.Lng-tt.c2.Lng) > 0.001 {
				t.Errorf("TargetPoint() = %v, want %v", newPoint, tt.c2)
			}
		})
	}
}
