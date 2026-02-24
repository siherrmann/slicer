package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlice_BuildPolygons(t *testing.T) {
	// Let's create a simple square slice
	s := Slice{
		Z: 1.0,
		Segments: []LineSegment{
			{Start: Vector3{X: 0, Y: 0, Z: 1}, End: Vector3{X: 10, Y: 0, Z: 1}},
			{Start: Vector3{X: 10, Y: 0, Z: 1}, End: Vector3{X: 10, Y: 10, Z: 1}},
			{Start: Vector3{X: 10, Y: 10, Z: 1}, End: Vector3{X: 0, Y: 10, Z: 1}},
			{Start: Vector3{X: 0, Y: 10, Z: 1}, End: Vector3{X: 0, Y: 0, Z: 1}},
		},
	}

	s.BuildPolygons(0.001)

	assert.Equal(t, 1, len(s.Polygons), "Should build exactly 1 polygon")
	assert.True(t, s.Polygons[0].IsClosed, "Polygon should be closed")
	assert.False(t, s.Polygons[0].IsHole, "Polygon should not be a hole")
	assert.Equal(t, 5, len(s.Polygons[0].Points), "Square polygon should have 5 points (start repeated at end)")
}

func TestSlice_ClassifyPolygons(t *testing.T) {
	s := Slice{
		Z: 1.0,
		Polygons: []Polygon{
			{ // Outer square (CCW)
				Points: []Vector3{
					{X: 0, Y: 0, Z: 1}, {X: 10, Y: 0, Z: 1}, {X: 10, Y: 10, Z: 1}, {X: 0, Y: 10, Z: 1}, {X: 0, Y: 0, Z: 1},
				},
				IsClosed: true,
			},
			{ // Inner square (hole) (CW or CCW doesn't matter for containment test if ContainsPoint checks bounds)
				Points: []Vector3{
					{X: 2, Y: 2, Z: 1}, {X: 8, Y: 2, Z: 1}, {X: 8, Y: 8, Z: 1}, {X: 2, Y: 8, Z: 1}, {X: 2, Y: 2, Z: 1},
				},
				IsClosed: true,
			},
		},
	}

	s.ClassifyPolygons()

	// Outer square should not be a hole
	assert.False(t, s.Polygons[0].IsHole, "Outer polygon should not be a hole")
	// Inner square should be a hole
	assert.True(t, s.Polygons[1].IsHole, "Inner polygon should be classified as a hole")
}

func TestTriangle_IntersectZPlane(t *testing.T) {
	tests := []struct {
		name      string
		t         Triangle
		z         float64
		intersect bool
		expected  LineSegment // Start could be flipped with End depending on order, so we'll just check lengths/points
	}{
		{
			name: "triangle crossing Z plane",
			t: Triangle{
				V1: Vector3{X: 0, Y: 0, Z: 0},
				V2: Vector3{X: 10, Y: 0, Z: 0},
				V3: Vector3{X: 0, Y: 10, Z: 10},
			},
			z:         5.0,
			intersect: true,
		},
		{
			name: "triangle below Z plane",
			t: Triangle{
				V1: Vector3{X: 0, Y: 0, Z: 0},
				V2: Vector3{X: 10, Y: 0, Z: 0},
				V3: Vector3{X: 0, Y: 10, Z: 2},
			},
			z:         5.0,
			intersect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ls, ok := tt.t.IntersectZPlane(tt.z)
			assert.Equal(t, tt.intersect, ok)
			if ok {
				// verify both points of ls are on Z plane
				assert.Equal(t, tt.z, ls.Start.Z)
				assert.Equal(t, tt.z, ls.End.Z)
			}
		})
	}
}
