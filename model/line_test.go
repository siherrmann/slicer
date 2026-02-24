package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLineSegment_Key(t *testing.T) {
	tests := []struct {
		name      string
		ls        LineSegment
		tolerance float64
		expected  string
	}{
		{
			name: "basic key",
			ls: LineSegment{
				Start: Vector3{X: 1.2345, Y: 2.3456, Z: 3.4567},
				End:   Vector3{X: 4.5678, Y: 5.6789, Z: 6.7890},
			},
			tolerance: 0.001,
			expected:  "1.235000,2.346000,3.457000-4.568000,5.679000,6.789000",
		},
		{
			name: "reversed order gives same key",
			ls: LineSegment{
				Start: Vector3{X: 4.5678, Y: 5.6789, Z: 6.7890},
				End:   Vector3{X: 1.2345, Y: 2.3456, Z: 3.4567},
			},
			tolerance: 0.001,
			expected:  "1.235000,2.346000,3.457000-4.568000,5.679000,6.789000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ls.Key(tt.tolerance)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLineSegment_IntersectLines(t *testing.T) {
	tests := []struct {
		name      string
		l1        LineSegment
		l2        LineSegment
		expectedT float64
		intersect bool
	}{
		{
			name:      "intersecting lines",
			l1:        LineSegment{Start: Vector3{X: 0, Y: 0, Z: 0}, End: Vector3{X: 2, Y: 2, Z: 0}},
			l2:        LineSegment{Start: Vector3{X: 0, Y: 2, Z: 0}, End: Vector3{X: 2, Y: 0, Z: 0}},
			expectedT: 0.5,
			intersect: true,
		},
		{
			name:      "parallel lines",
			l1:        LineSegment{Start: Vector3{X: 0, Y: 0, Z: 0}, End: Vector3{X: 2, Y: 0, Z: 0}},
			l2:        LineSegment{Start: Vector3{X: 0, Y: 1, Z: 0}, End: Vector3{X: 2, Y: 1, Z: 0}},
			expectedT: 0,
			intersect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resT, ok := tt.l1.IntersectLines(tt.l2)
			assert.Equal(t, tt.intersect, ok)
			if ok {
				assert.InDelta(t, tt.expectedT, resT, 1e-6)
			}
		})
	}
}

func TestLineSegment_IntersectSegments(t *testing.T) {
	tests := []struct {
		name      string
		l1        LineSegment
		l2        LineSegment
		intersect bool
		expected  Vector3
	}{
		{
			name:      "intersecting segments",
			l1:        LineSegment{Start: Vector3{X: 0, Y: 0, Z: 0}, End: Vector3{X: 2, Y: 2, Z: 0}},
			l2:        LineSegment{Start: Vector3{X: 0, Y: 2, Z: 0}, End: Vector3{X: 2, Y: 0, Z: 0}},
			intersect: true,
			expected:  Vector3{X: 1, Y: 1, Z: 0},
		},
		{
			name:      "non-intersecting segments",
			l1:        LineSegment{Start: Vector3{X: 0, Y: 0, Z: 0}, End: Vector3{X: 0.5, Y: 0.5, Z: 0}},
			l2:        LineSegment{Start: Vector3{X: 0, Y: 2, Z: 0}, End: Vector3{X: 2, Y: 0, Z: 0}},
			intersect: false,
		},
		{
			name:      "parallel segments",
			l1:        LineSegment{Start: Vector3{X: 0, Y: 0, Z: 0}, End: Vector3{X: 2, Y: 0, Z: 0}},
			l2:        LineSegment{Start: Vector3{X: 0, Y: 1, Z: 0}, End: Vector3{X: 2, Y: 1, Z: 0}},
			intersect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, ok := tt.l1.IntersectSegments(tt.l2)
			assert.Equal(t, tt.intersect, ok)
			if ok {
				assert.InDelta(t, tt.expected.X, res.X, 1e-6)
				assert.InDelta(t, tt.expected.Y, res.Y, 1e-6)
				assert.InDelta(t, tt.expected.Z, res.Z, 1e-6)
			}
		})
	}
}

func TestLineSegment_IntersectZPlane(t *testing.T) {
	tests := []struct {
		name      string
		l         LineSegment
		z         float64
		intersect bool
		expected  Vector3
	}{
		{
			name:      "crossing z plane",
			l:         LineSegment{Start: Vector3{X: 0, Y: 0, Z: 0}, End: Vector3{X: 2, Y: 2, Z: 2}},
			z:         1.0,
			intersect: true,
			expected:  Vector3{X: 1, Y: 1, Z: 1},
		},
		{
			name:      "not crossing z plane",
			l:         LineSegment{Start: Vector3{X: 0, Y: 0, Z: 0}, End: Vector3{X: 2, Y: 2, Z: 2}},
			z:         3.0,
			intersect: false,
		},
		{
			name:      "lying on z plane",
			l:         LineSegment{Start: Vector3{X: 0, Y: 0, Z: 1}, End: Vector3{X: 2, Y: 2, Z: 1}},
			z:         1.0,
			intersect: false,
		},
		{
			name:      "start point on z plane",
			l:         LineSegment{Start: Vector3{X: 0, Y: 0, Z: 1}, End: Vector3{X: 2, Y: 2, Z: 2}},
			z:         1.0,
			intersect: true,
			expected:  Vector3{X: 0, Y: 0, Z: 1},
		},
		{
			name:      "end point on z plane",
			l:         LineSegment{Start: Vector3{X: 0, Y: 0, Z: 0}, End: Vector3{X: 2, Y: 2, Z: 1}},
			z:         1.0,
			intersect: true,
			expected:  Vector3{X: 2, Y: 2, Z: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, ok := tt.l.IntersectZPlane(tt.z)
			assert.Equal(t, tt.intersect, ok)
			if ok {
				assert.InDelta(t, tt.expected.X, res.X, 1e-6)
				assert.InDelta(t, tt.expected.Y, res.Y, 1e-6)
				assert.InDelta(t, tt.expected.Z, res.Z, 1e-6)
			}
		})
	}
}
