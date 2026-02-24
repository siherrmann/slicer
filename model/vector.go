package model

import (
	"fmt"
	"math"
)

// Vector3 represents a 3D vector with X, Y, Z coordinates
type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// MarshalJSON truncates floating point coordinates to 3 decimal places to drastically reduce JSON byte footprint.
func (v Vector3) MarshalJSON() ([]byte, error) {
	// 3 decimal places = 1 micrometer precision.
	// We do not need more than this for gcode or 3D rendering.
	str := fmt.Sprintf(`{"x":%.3f,"y":%.3f,"z":%.3f}`, v.X, v.Y, v.Z)
	return []byte(str), nil
}

// Add adds two vectors
func (v Vector3) Add(other Vector3) Vector3 {
	return Vector3{v.X + other.X, v.Y + other.Y, v.Z + other.Z}
}

// Sub subtracts two vectors
func (v Vector3) Sub(other Vector3) Vector3 {
	return Vector3{v.X - other.X, v.Y - other.Y, v.Z - other.Z}
}

// Distance calculates the distance between two points
func (p Vector3) Distance(other Vector3) float64 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	dz := p.Z - other.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func (p Vector3) Equals(other Vector3) bool {
	const epsilon = 1e-6
	return p.Distance(other) < epsilon
}

// Scale multiplies the vector by a scalar
func (p Vector3) Scale(factor float64) Vector3 {
	return Vector3{X: p.X * factor, Y: p.Y * factor, Z: p.Z * factor}
}

func (p Vector3) Rotate(angle float64) Vector3 {
	s, c := math.Sincos(angle)
	return Vector3{
		X: p.X*c - p.Y*s,
		Y: p.X*s + p.Y*c,
		Z: p.Z,
	}
}

// RotateAroundPoint rotates the vector around a center point by the given angle
func (p Vector3) RotateAroundPoint(center Vector3, angle float64) Vector3 {
	// Translate to origin
	dx := p.X - center.X
	dy := p.Y - center.Y

	// Rotate
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	rotatedX := dx*cos - dy*sin
	rotatedY := dx*sin + dy*cos

	// Translate back
	return Vector3{
		X: rotatedX + center.X,
		Y: rotatedY + center.Y,
		Z: p.Z,
	}
}

// Length returns the length of the vector
func (p Vector3) Length() float64 {
	return math.Sqrt(p.X*p.X + p.Y*p.Y + p.Z*p.Z)
}

// Normalize returns a unit vector
func (p Vector3) Normalize() Vector3 {
	length := p.Length()
	if length == 0 {
		return Vector3{0, 0, 0}
	}
	return Vector3{X: p.X / length, Y: p.Y / length, Z: p.Z / length}
}

// Perpendicular returns a perpendicular vector (rotated 90° counterclockwise)
func (p Vector3) Perpendicular() Vector3 {
	return Vector3{X: -p.Y, Y: p.X, Z: p.Z}
}

// Cross computes the cross product of two vectors
func (v Vector3) Cross(other Vector3) Vector3 {
	return Vector3{
		v.Y*other.Z - v.Z*other.Y,
		v.Z*other.X - v.X*other.Z,
		v.X*other.Y - v.Y*other.X,
	}
}

// Dot computes the dot product of two vectors
func (v Vector3) Dot(other Vector3) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

// PerpendicularDistanceToLine calculates the perpendicular distance from this point to a line segment
func (point Vector3) PerpendicularDistanceToLine(lineStart, lineEnd Vector3) float64 {
	// Vector from lineStart to lineEnd
	dx := lineEnd.X - lineStart.X
	dy := lineEnd.Y - lineStart.Y
	dz := lineEnd.Z - lineStart.Z

	// If the line segment is actually a point
	if dx == 0 && dy == 0 && dz == 0 {
		return point.Distance(lineStart)
	}

	// Calculate the parameter t that represents the projection of point onto the line
	// t = [(P-A) · (B-A)] / |B-A|²
	t := ((point.X-lineStart.X)*dx + (point.Y-lineStart.Y)*dy + (point.Z-lineStart.Z)*dz) / (dx*dx + dy*dy + dz*dz)

	// Clamp t to [0, 1] to ensure we're measuring to the line segment, not the infinite line
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	// Find the closest point on the line segment
	closestPoint := Vector3{
		X: lineStart.X + t*dx,
		Y: lineStart.Y + t*dy,
		Z: lineStart.Z + t*dz,
	}

	// Return the distance from the point to the closest point on the segment
	return point.Distance(closestPoint)
}

func (v Vector3) String() string {
	return fmt.Sprintf("(%.6f, %.6f, %.6f)", v.X, v.Y, v.Z)
}
