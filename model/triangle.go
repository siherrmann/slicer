package model

import (
	"fmt"
	"math"
)

// Triangle represents a triangular facet in the STL model
type Triangle struct {
	Normal Vector3 `json:"normal"`
	V1     Vector3 `json:"v1"`
	V2     Vector3 `json:"v2"`
	V3     Vector3 `json:"v3"`
	// Attribute byte count (used in binary STL, often for color)
	Attr uint16 `json:"attr"`
}

// ComputeNormal calculates the normal vector from the vertices using the right-hand rule
func (t *Triangle) ComputeNormal() Vector3 {
	edge1 := t.V2.Sub(t.V1)
	edge2 := t.V3.Sub(t.V1)
	return edge1.Cross(edge2).Normalize()
}

// Area calculates the area of the triangle
func (t *Triangle) Area() float64 {
	edge1 := t.V2.Sub(t.V1)
	edge2 := t.V3.Sub(t.V1)
	cross := edge1.Cross(edge2)
	return cross.Length() / 2.0
}

// IsDegenerate checks if the triangle is degenerate (vertices too close or collinear)
// A triangle is degenerate if:
// 1. Any two vertices are essentially at the same location (< 1e-6 apart)
// 2. The area is effectively zero (collinear vertices)
func (t *Triangle) IsDegenerate(tolerance float64) bool {
	// Use a very small epsilon for degenerate detection (0.001mm = 1 micron)
	// This is independent of the user's tolerance parameter
	const epsilon = 1e-6

	// Check if any two vertices are too close
	edge1 := t.V2.Sub(t.V1)
	edge2 := t.V3.Sub(t.V1)
	edge3 := t.V3.Sub(t.V2)

	// If any edge is shorter than epsilon, vertices are essentially at the same location
	if edge1.Length() < epsilon || edge2.Length() < epsilon || edge3.Length() < epsilon {
		return true
	}

	// Check if area is effectively zero (collinear vertices)
	// For collinear vertices, the cross product magnitude (2*area) should be very small
	// Use a relative check: area should be at least 1e-6 times the product of edge lengths
	cross := edge1.Cross(edge2)
	crossLength := cross.Length()

	// Get maximum edge length for relative comparison
	maxEdgeLength := edge1.Length()
	if edge2.Length() > maxEdgeLength {
		maxEdgeLength = edge2.Length()
	}
	if edge3.Length() > maxEdgeLength {
		maxEdgeLength = edge3.Length()
	}

	// If cross product is extremely small relative to edge size, vertices are collinear
	// This catches collinear vertices better than absolute epsilon
	if crossLength < epsilon*maxEdgeLength {
		return true
	}

	return false
}

// Key creates a unique key for the triangle based on its vertices
func (t *Triangle) Key(tolerance float64) string {
	// Round vertices to tolerance to handle floating point precision
	round := func(v Vector3) Vector3 {
		scale := float64(1.0 / tolerance)
		return Vector3{
			math.Round(v.X*scale) / scale,
			math.Round(v.Y*scale) / scale,
			math.Round(v.Z*scale) / scale,
		}
	}

	r1 := round(t.V1)
	r2 := round(t.V2)
	r3 := round(t.V3)

	return fmt.Sprintf("%.6f,%.6f,%.6f-%.6f,%.6f,%.6f-%.6f,%.6f,%.6f",
		r1.X, r1.Y, r1.Z, r2.X, r2.Y, r2.Z, r3.X, r3.Y, r3.Z)
}

func (t Triangle) String() string {
	return fmt.Sprintf("Normal: %s, V1: %s, V2: %s, V3: %s, Attr: %d",
		t.Normal.String(), t.V1.String(), t.V2.String(), t.V3.String(), t.Attr)
}
