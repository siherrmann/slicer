package model

import (
	"fmt"
	"math"
)

// LineSegment represents a line segment with two endpoints
type LineSegment struct {
	Start Vector3
	End   Vector3
}

// Key creates a consistent key for an edge (order-independent)
func (e *LineSegment) Key(tolerance float64) string {
	// Round vertices to tolerance to handle floating point precision
	round := func(v Vector3) Vector3 {
		scale := float64(1.0 / tolerance)
		return Vector3{
			math.Round(v.X*scale) / scale,
			math.Round(v.Y*scale) / scale,
			math.Round(v.Z*scale) / scale,
		}
	}

	r1 := round(e.Start)
	r2 := round(e.End)

	// Ensure consistent ordering
	if r1.X < r2.X || (r1.X == r2.X && r1.Y < r2.Y) || (r1.X == r2.X && r1.Y == r2.Y && r1.Z < r2.Z) {
		return fmt.Sprintf("%.6f,%.6f,%.6f-%.6f,%.6f,%.6f", r1.X, r1.Y, r1.Z, r2.X, r2.Y, r2.Z)
	}
	return fmt.Sprintf("%.6f,%.6f,%.6f-%.6f,%.6f,%.6f", r2.X, r2.Y, r2.Z, r1.X, r1.Y, r1.Z)
}

// IntersectLines finds the intersection of two infinite lines (not segments)
// Returns the parameter t along the first line and whether intersection exists
func (l LineSegment) IntersectLines(line LineSegment) (float64, bool) {
	p1, p2 := l.Start, l.End
	p3, p4 := line.Start, line.End

	denom := (p1.X-p2.X)*(p3.Y-p4.Y) - (p1.Y-p2.Y)*(p3.X-p4.X)

	if math.Abs(denom) < 1e-10 {
		return 0, false // Lines are parallel
	}

	t := ((p1.X-p3.X)*(p3.Y-p4.Y) - (p1.Y-p3.Y)*(p3.X-p4.X)) / denom

	return t, true
}

// IntersectSegments checks if two line segments intersect
// Returns the parameters (t1, t2) and whether they intersect
func (l LineSegment) IntersectSegments(seg LineSegment) (Vector3, bool) {
	p1, p2 := l.Start, l.End
	p3, p4 := seg.Start, seg.End

	denom := (p1.X-p2.X)*(p3.Y-p4.Y) - (p1.Y-p2.Y)*(p3.X-p4.X)

	if math.Abs(denom) < 1e-10 {
		return Vector3{}, false // Segments are parallel
	}

	t1 := ((p1.X-p3.X)*(p3.Y-p4.Y) - (p1.Y-p3.Y)*(p3.X-p4.X)) / denom
	t2 := -((p1.X-p2.X)*(p1.Y-p3.Y) - (p1.Y-p2.Y)*(p1.X-p3.X)) / denom

	// Check if intersection is within both segments
	if t1 >= 0 && t1 <= 1 && t2 >= 0 && t2 <= 1 {
		return Vector3{
			X: p1.X + t1*(p2.X-p1.X),
			Y: p1.Y + t1*(p2.Y-p1.Y),
			Z: p1.Z + t1*(p2.Z-p1.Z),
		}, true
	}

	return Vector3{}, false
}

// IntersectZPlane finds where an edge intersects a horizontal plane
func (l LineSegment) IntersectZPlane(z float64) (Vector3, bool) {
	// Check if edge crosses the plane (vertices on different sides)
	if (l.Start.Z < z && l.End.Z < z) || (l.Start.Z > z && l.End.Z > z) {
		return Vector3{}, false // Edge doesn't cross plane
	}

	// Handle edge exactly on plane
	if l.Start.Z == z && l.End.Z == z {
		return Vector3{}, false // Edge lies in plane (not a crossing)
	}

	// One vertex exactly on plane
	if l.Start.Z == z {
		return l.Start, true
	}
	if l.End.Z == z {
		return l.End, true
	}

	// Compute intersection point using linear interpolation
	t := (z - l.Start.Z) / (l.End.Z - l.Start.Z)
	return Vector3{
		X: l.Start.X + t*(l.End.X-l.Start.X),
		Y: l.Start.Y + t*(l.End.Y-l.Start.Y),
		Z: z,
	}, true
}
