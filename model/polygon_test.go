package model

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPolygon_GetArea(t *testing.T) {
	// A CCW square of 10x10 area = 100
	p1 := &Polygon{
		Points: []Vector3{
			{X: 0, Y: 0},
			{X: 10, Y: 0},
			{X: 10, Y: 10},
			{X: 0, Y: 10},
		},
	}
	assert.Equal(t, 100.0, p1.GetArea())

	// CW square area = -100
	p2 := &Polygon{
		Points: []Vector3{
			{X: 0, Y: 0},
			{X: 0, Y: 10},
			{X: 10, Y: 10},
			{X: 10, Y: 0},
		},
	}
	assert.Equal(t, -100.0, p2.GetArea())

	p3 := &Polygon{
		Points: []Vector3{
			{X: 0, Y: 0},
		},
	}
	assert.Equal(t, 0.0, p3.GetArea(), "Area of <3 points is 0")
}

func TestPolygon_GetBounds(t *testing.T) {
	p1 := &Polygon{
		Points: []Vector3{
			{X: 1, Y: 2},
			{X: 10, Y: -5},
			{X: -3, Y: 10},
		},
	}
	bounds := p1.GetBounds()
	assert.Equal(t, -3.0, bounds.MinX)
	assert.Equal(t, 10.0, bounds.MaxX)
	assert.Equal(t, -5.0, bounds.MinY)
	assert.Equal(t, 10.0, bounds.MaxY)

	pEmpty := &Polygon{}
	assert.Equal(t, BoundingBox{}, pEmpty.GetBounds())
}

func TestPolygon_GetLines(t *testing.T) {
	p := &Polygon{
		Points: []Vector3{
			{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10},
		},
	}
	lines := p.GetLines()
	assert.Equal(t, 3, len(lines))
	assert.Equal(t, Vector3{X: 0, Y: 0}, lines[0].Start)
	assert.Equal(t, Vector3{X: 10, Y: 0}, lines[0].End)
}

func TestPolygon_IsClockwise(t *testing.T) {
	// CCW
	p1 := &Polygon{Points: []Vector3{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}}}
	assert.False(t, p1.IsClockwise())

	// CW
	p2 := &Polygon{Points: []Vector3{{X: 0, Y: 0}, {X: 0, Y: 10}, {X: 10, Y: 10}}}
	assert.True(t, p2.IsClockwise())
}

func TestPolygon_Reverse(t *testing.T) {
	p := &Polygon{Points: []Vector3{{X: 1, Y: 1}, {X: 2, Y: 2}, {X: 3, Y: 3}}}
	p.Reverse()
	assert.Equal(t, Vector3{X: 3, Y: 3}, p.Points[0])
	assert.Equal(t, Vector3{X: 1, Y: 1}, p.Points[2])
}

func TestPolygon_Rotate(t *testing.T) {
	p := &Polygon{Points: []Vector3{{X: 1, Y: 0, Z: 5}}}
	rotated := p.Rotate(math.Pi / 2) // 90 deg
	assert.InDelta(t, 0.0, rotated.Points[0].X, 1e-6)
	assert.InDelta(t, 1.0, rotated.Points[0].Y, 1e-6)
	assert.InDelta(t, 5.0, rotated.Points[0].Z, 1e-6) // Z should be unchanged
}

func TestPolygon_SetZ(t *testing.T) {
	p := &Polygon{Points: []Vector3{{X: 1, Y: 1, Z: 0}, {X: 2, Y: 2, Z: 0}}}
	p.SetZ(5.0)
	assert.Equal(t, 5.0, p.Points[0].Z)
	assert.Equal(t, 5.0, p.Points[1].Z)
}

func TestPolygon_ToContinuousPath(t *testing.T) {
	p := &Polygon{
		Points:   []Vector3{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}},
		IsClosed: true,
	}
	path := p.ToContinuousPath(50.0, CategoryOuterWall, 0)

	assert.Equal(t, 3, len(path.Segments))
	assert.Equal(t, PathExtrusion, path.PathType)
	assert.Equal(t, 50.0, path.Segments[0].Speed)
	assert.Equal(t, CategoryOuterWall, path.Segments[0].Category)

	pOpen := &Polygon{
		Points:   []Vector3{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}},
		IsClosed: false,
	}
	pathOpen := pOpen.ToContinuousPath(50.0, CategoryOuterWall, 0)
	assert.Equal(t, 2, len(pathOpen.Segments), "Open polygon should not have closing segment")
}

func TestPolygon_ContainsPoint(t *testing.T) {
	p := &Polygon{
		Points: []Vector3{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}},
	}
	assert.True(t, p.ContainsPoint(Vector3{X: 5, Y: 5}))    // Inside
	assert.False(t, p.ContainsPoint(Vector3{X: -1, Y: -1})) // Outside
}

func TestPolygon_IntersectLine(t *testing.T) {
	p := &Polygon{
		Points: []Vector3{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}},
	}

	intersections := p.IntersectLine(5.0) // horizontal line at Y=5
	assert.Equal(t, 2, len(intersections))

	// Assuming an unsorted return, but since vertices are (0,0)->(10,0)->(10,10)->(0,10)
	// Extrapolating edges: (10,0) to (10,10) crosses Y=5 at X=10
	// (0,10) to (0,0) crosses Y=5 at X=0
	assert.Contains(t, intersections, 0.0)
	assert.Contains(t, intersections, 10.0)
}

func TestPolygon_ClipLineToPolygon(t *testing.T) {
	p := &Polygon{
		Points: []Vector3{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}},
	}

	line := LineSegment{Start: Vector3{X: -5, Y: 5}, End: Vector3{X: 15, Y: 5}}
	clipped := p.ClipLineToPolygon(line)

	assert.Equal(t, 1, len(clipped), "Line passing completely through should result in 1 internal segment")
	if len(clipped) > 0 {
		assert.Equal(t, 0.0, clipped[0].Start.X)
		assert.Equal(t, 10.0, clipped[0].End.X)
	}

	lineOutside := LineSegment{Start: Vector3{X: -5, Y: 15}, End: Vector3{X: 15, Y: 15}}
	clippedOutside := p.ClipLineToPolygon(lineOutside)
	assert.Equal(t, 0, len(clippedOutside), "Line outside should result in 0 segments")

	lineInside := LineSegment{Start: Vector3{X: 2, Y: 5}, End: Vector3{X: 8, Y: 5}}
	clippedInside := p.ClipLineToPolygon(lineInside)
	assert.Equal(t, 1, len(clippedInside))
}

func TestPolygon_OffsetPolygon(t *testing.T) {
	p := &Polygon{
		Points: []Vector3{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}},
	}
	// offset outward by 1
	offsetP := p.OffsetPolygon(1.0)
	assert.NotNil(t, offsetP)

	// Result should effectively be a larger square: min (-1,-1) max (11,11) roughly
	bounds := offsetP.GetBounds()
	assert.InDelta(t, -1.0, bounds.MinX, 0.1)
	assert.InDelta(t, 11.0, bounds.MaxX, 0.1)
	assert.InDelta(t, -1.0, bounds.MinY, 0.1)
	assert.InDelta(t, 11.0, bounds.MaxY, 0.1)
}
