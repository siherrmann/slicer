package model

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVector3_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		v        Vector3
		expected string
	}{
		{
			name:     "positive coordinates",
			v:        Vector3{X: 1.2345, Y: 2.3456, Z: 3.4567},
			expected: `{"x":1.234,"y":2.346,"z":3.457}`,
		},
		{
			name:     "negative coordinates",
			v:        Vector3{X: -1.2345, Y: -2.3456, Z: -3.4567},
			expected: `{"x":-1.234,"y":-2.346,"z":-3.457}`,
		},
		{
			name:     "zero coordinates",
			v:        Vector3{X: 0, Y: 0, Z: 0},
			expected: `{"x":0.000,"y":0.000,"z":0.000}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.v)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(b))
		})
	}
}

func TestVector3_Add(t *testing.T) {
	tests := []struct {
		name     string
		v        Vector3
		other    Vector3
		expected Vector3
	}{
		{
			name:     "add positive",
			v:        Vector3{X: 1, Y: 2, Z: 3},
			other:    Vector3{X: 4, Y: 5, Z: 6},
			expected: Vector3{X: 5, Y: 7, Z: 9},
		},
		{
			name:     "add negative",
			v:        Vector3{X: 1, Y: 2, Z: 3},
			other:    Vector3{X: -4, Y: -5, Z: -6},
			expected: Vector3{X: -3, Y: -3, Z: -3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.v.Add(tt.other)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVector3_Sub(t *testing.T) {
	tests := []struct {
		name     string
		v        Vector3
		other    Vector3
		expected Vector3
	}{
		{
			name:     "subtract positive",
			v:        Vector3{X: 5, Y: 7, Z: 9},
			other:    Vector3{X: 4, Y: 5, Z: 6},
			expected: Vector3{X: 1, Y: 2, Z: 3},
		},
		{
			name:     "subtract negative",
			v:        Vector3{X: 1, Y: 2, Z: 3},
			other:    Vector3{X: -4, Y: -5, Z: -6},
			expected: Vector3{X: 5, Y: 7, Z: 9},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.v.Sub(tt.other)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVector3_Distance(t *testing.T) {
	tests := []struct {
		name     string
		p        Vector3
		other    Vector3
		expected float64
	}{
		{
			name:     "zero distance",
			p:        Vector3{X: 1, Y: 2, Z: 3},
			other:    Vector3{X: 1, Y: 2, Z: 3},
			expected: 0,
		},
		{
			name:     "positive distance",
			p:        Vector3{X: 0, Y: 0, Z: 0},
			other:    Vector3{X: 3, Y: 4, Z: 0},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.p.Distance(tt.other)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVector3_Equals(t *testing.T) {
	tests := []struct {
		name     string
		p        Vector3
		other    Vector3
		expected bool
	}{
		{
			name:     "exact equal",
			p:        Vector3{X: 1, Y: 2, Z: 3},
			other:    Vector3{X: 1, Y: 2, Z: 3},
			expected: true,
		},
		{
			name:     "within epsilon",
			p:        Vector3{X: 1, Y: 2, Z: 3},
			other:    Vector3{X: 1.0000001, Y: 2, Z: 3},
			expected: true,
		},
		{
			name:     "outside epsilon",
			p:        Vector3{X: 1, Y: 2, Z: 3},
			other:    Vector3{X: 1.00001, Y: 2, Z: 3},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.p.Equals(tt.other)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVector3_Scale(t *testing.T) {
	tests := []struct {
		name     string
		p        Vector3
		factor   float64
		expected Vector3
	}{
		{
			name:     "scale by 2",
			p:        Vector3{X: 1, Y: -2, Z: 3},
			factor:   2,
			expected: Vector3{X: 2, Y: -4, Z: 6},
		},
		{
			name:     "scale by 0",
			p:        Vector3{X: 1, Y: -2, Z: 3},
			factor:   0,
			expected: Vector3{X: 0, Y: 0, Z: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.p.Scale(tt.factor)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVector3_Rotate(t *testing.T) {
	tests := []struct {
		name     string
		p        Vector3
		angle    float64
		expected Vector3
	}{
		{
			name:     "rotate 90 degrees",
			p:        Vector3{X: 1, Y: 0, Z: 1},
			angle:    math.Pi / 2,
			expected: Vector3{X: 0, Y: 1, Z: 1},
		},
		{
			name:     "rotate 180 degrees",
			p:        Vector3{X: 1, Y: 0, Z: 1},
			angle:    math.Pi,
			expected: Vector3{X: -1, Y: 0, Z: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.p.Rotate(tt.angle)
			assert.InDelta(t, tt.expected.X, result.X, 1e-6)
			assert.InDelta(t, tt.expected.Y, result.Y, 1e-6)
			assert.InDelta(t, tt.expected.Z, result.Z, 1e-6)
		})
	}
}

func TestVector3_RotateAroundPoint(t *testing.T) {
	tests := []struct {
		name     string
		p        Vector3
		center   Vector3
		angle    float64
		expected Vector3
	}{
		{
			name:     "rotate around origin",
			p:        Vector3{X: 1, Y: 0, Z: 1},
			center:   Vector3{X: 0, Y: 0, Z: 0},
			angle:    math.Pi / 2,
			expected: Vector3{X: 0, Y: 1, Z: 1},
		},
		{
			name:     "rotate around translated point",
			p:        Vector3{X: 2, Y: 1, Z: 1},
			center:   Vector3{X: 1, Y: 1, Z: 0},
			angle:    math.Pi / 2,
			expected: Vector3{X: 1, Y: 2, Z: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.p.RotateAroundPoint(tt.center, tt.angle)
			assert.InDelta(t, tt.expected.X, result.X, 1e-6)
			assert.InDelta(t, tt.expected.Y, result.Y, 1e-6)
			assert.InDelta(t, tt.expected.Z, result.Z, 1e-6)
		})
	}
}

func TestVector3_Length(t *testing.T) {
	tests := []struct {
		name     string
		p        Vector3
		expected float64
	}{
		{
			name:     "zero length",
			p:        Vector3{X: 0, Y: 0, Z: 0},
			expected: 0,
		},
		{
			name:     "positive length",
			p:        Vector3{X: 3, Y: 4, Z: 0},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.p.Length()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVector3_Normalize(t *testing.T) {
	tests := []struct {
		name     string
		p        Vector3
		expected Vector3
	}{
		{
			name:     "normalize standard",
			p:        Vector3{X: 3, Y: 4, Z: 0},
			expected: Vector3{X: 0.6, Y: 0.8, Z: 0},
		},
		{
			name:     "normalize zero",
			p:        Vector3{0, 0, 0},
			expected: Vector3{0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.p.Normalize()
			assert.InDelta(t, tt.expected.X, result.X, 1e-6)
			assert.InDelta(t, tt.expected.Y, result.Y, 1e-6)
			assert.InDelta(t, tt.expected.Z, result.Z, 1e-6)
		})
	}
}

func TestVector3_Perpendicular(t *testing.T) {
	tests := []struct {
		name     string
		p        Vector3
		expected Vector3
	}{
		{
			name:     "basic perpendicular",
			p:        Vector3{X: 1, Y: 0, Z: 5},
			expected: Vector3{X: 0, Y: 1, Z: 5},
		},
		{
			name:     "complex perpendicular",
			p:        Vector3{X: 1, Y: 1, Z: 5},
			expected: Vector3{X: -1, Y: 1, Z: 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.p.Perpendicular()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVector3_Cross(t *testing.T) {
	tests := []struct {
		name     string
		v        Vector3
		other    Vector3
		expected Vector3
	}{
		{
			name:     "cross basic",
			v:        Vector3{X: 1, Y: 0, Z: 0},
			other:    Vector3{X: 0, Y: 1, Z: 0},
			expected: Vector3{X: 0, Y: 0, Z: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.v.Cross(tt.other)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVector3_Dot(t *testing.T) {
	tests := []struct {
		name     string
		v        Vector3
		other    Vector3
		expected float64
	}{
		{
			name:     "dot parallel",
			v:        Vector3{X: 1, Y: 0, Z: 0},
			other:    Vector3{X: 2, Y: 0, Z: 0},
			expected: 2,
		},
		{
			name:     "dot perpendicular",
			v:        Vector3{X: 1, Y: 0, Z: 0},
			other:    Vector3{X: 0, Y: 1, Z: 0},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.v.Dot(tt.other)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVector3_PerpendicularDistanceToLine(t *testing.T) {
	tests := []struct {
		name      string
		point     Vector3
		lineStart Vector3
		lineEnd   Vector3
		expected  float64
	}{
		{
			name:      "point on line",
			point:     Vector3{X: 1, Y: 1, Z: 0},
			lineStart: Vector3{X: 0, Y: 0, Z: 0},
			lineEnd:   Vector3{X: 2, Y: 2, Z: 0},
			expected:  0,
		},
		{
			name:      "point outside line segment",
			point:     Vector3{X: 3, Y: 3, Z: 0},
			lineStart: Vector3{X: 0, Y: 0, Z: 0},
			lineEnd:   Vector3{X: 1, Y: 1, Z: 0},
			expected:  math.Sqrt(8), // distance to (1,1,0)
		},
		{
			name:      "point orthogonal to line",
			point:     Vector3{X: 0, Y: 1, Z: 0},
			lineStart: Vector3{X: -1, Y: 0, Z: 0},
			lineEnd:   Vector3{X: 1, Y: 0, Z: 0},
			expected:  1,
		},
		{
			name:      "line is a point",
			point:     Vector3{X: 1, Y: 1, Z: 0},
			lineStart: Vector3{X: 0, Y: 0, Z: 0},
			lineEnd:   Vector3{X: 0, Y: 0, Z: 0},
			expected:  math.Sqrt(2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.point.PerpendicularDistanceToLine(tt.lineStart, tt.lineEnd)
			assert.InDelta(t, tt.expected, result, 1e-6)
		})
	}
}

func TestVector3_String(t *testing.T) {
	v := Vector3{X: 1.23, Y: 4.56, Z: 7.89}
	expected := "(1.230000, 4.560000, 7.890000)"
	assert.Equal(t, expected, v.String())
}
