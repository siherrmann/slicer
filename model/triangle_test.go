package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTriangle_ComputeNormal(t *testing.T) {
	tests := []struct {
		name     string
		t        Triangle
		expected Vector3
	}{
		{
			name: "xy plane triangle",
			t: Triangle{
				V1: Vector3{X: 0, Y: 0, Z: 0},
				V2: Vector3{X: 1, Y: 0, Z: 0},
				V3: Vector3{X: 0, Y: 1, Z: 0},
			},
			expected: Vector3{X: 0, Y: 0, Z: 1}, // CCW order gives +Z normal
		},
		{
			name: "xz plane triangle",
			t: Triangle{
				V1: Vector3{X: 0, Y: 0, Z: 0},
				V2: Vector3{X: 1, Y: 0, Z: 0},
				V3: Vector3{X: 0, Y: 0, Z: 1},
			},
			expected: Vector3{X: 0, Y: -1, Z: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.t.ComputeNormal()
			assert.InDelta(t, tt.expected.X, result.X, 1e-6)
			assert.InDelta(t, tt.expected.Y, result.Y, 1e-6)
			assert.InDelta(t, tt.expected.Z, result.Z, 1e-6)
		})
	}
}

func TestTriangle_Area(t *testing.T) {
	tests := []struct {
		name     string
		t        Triangle
		expected float64
	}{
		{
			name: "right triangle",
			t: Triangle{
				V1: Vector3{X: 0, Y: 0, Z: 0},
				V2: Vector3{X: 3, Y: 0, Z: 0},
				V3: Vector3{X: 0, Y: 4, Z: 0},
			},
			expected: 6.0,
		},
		{
			name: "zero area",
			t: Triangle{
				V1: Vector3{X: 0, Y: 0, Z: 0},
				V2: Vector3{X: 1, Y: 1, Z: 1},
				V3: Vector3{X: 2, Y: 2, Z: 2},
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.t.Area()
			assert.InDelta(t, tt.expected, result, 1e-6)
		})
	}
}

func TestTriangle_IsDegenerate(t *testing.T) {
	tests := []struct {
		name      string
		t         Triangle
		tolerance float64
		expected  bool
	}{
		{
			name: "valid triangle",
			t: Triangle{
				V1: Vector3{X: 0, Y: 0, Z: 0},
				V2: Vector3{X: 1, Y: 0, Z: 0},
				V3: Vector3{X: 0, Y: 1, Z: 0},
			},
			tolerance: 0.001,
			expected:  false,
		},
		{
			name: "two identical vertices",
			t: Triangle{
				V1: Vector3{X: 0, Y: 0, Z: 0},
				V2: Vector3{X: 0, Y: 0, Z: 0},
				V3: Vector3{X: 0, Y: 1, Z: 0},
			},
			tolerance: 0.001,
			expected:  true,
		},
		{
			name: "collinear vertices",
			t: Triangle{
				V1: Vector3{X: 0, Y: 0, Z: 0},
				V2: Vector3{X: 1, Y: 1, Z: 0},
				V3: Vector3{X: 2, Y: 2, Z: 0},
			},
			tolerance: 0.001,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.t.IsDegenerate(tt.tolerance)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTriangle_Key(t *testing.T) {
	tests := []struct {
		name      string
		t         Triangle
		tolerance float64
		expected  string
	}{
		{
			name: "basic key generation",
			t: Triangle{
				V1: Vector3{X: 1.234567, Y: 2.345678, Z: 3.456789},
				V2: Vector3{X: 4.567890, Y: 5.678901, Z: 6.789012},
				V3: Vector3{X: 7.890123, Y: 8.901234, Z: 9.012345},
			},
			tolerance: 0.001, // rounds to 3 decimal places conceptually
			expected:  "1.235000,2.346000,3.457000-4.568000,5.679000,6.789000-7.890000,8.901000,9.012000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.t.Key(tt.tolerance)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTriangle_String(t *testing.T) {
	tri := Triangle{
		Normal: Vector3{X: 0, Y: 0, Z: 1},
		V1:     Vector3{X: 1, Y: 0, Z: 0},
		V2:     Vector3{X: 0, Y: 1, Z: 0},
		V3:     Vector3{X: 0, Y: 0, Z: 0},
		Attr:   15,
	}
	expected := "Normal: (0.000000, 0.000000, 1.000000), V1: (1.000000, 0.000000, 0.000000), V2: (0.000000, 1.000000, 0.000000), V3: (0.000000, 0.000000, 0.000000), Attr: 15"
	assert.Equal(t, expected, tri.String())
}
