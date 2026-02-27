package model

import (
	"math"
	"sort"
)

// Slice represents a single horizontal layer of the sliced model
type Slice struct {
	Z          float64       // Z height of this slice
	LayerIndex int           // Index of this layer (0-based)
	Segments   []LineSegment // Line segments from triangle intersections
	Polygons   []Polygon     // Closed polygons formed from segments

	// Processed print paths
	Perimeters      [][]Vector3      // Outer walls (multiple perimeters)
	InfillLines     []LineSegment    // Infill line segments
	ContinuousPaths []ContinuousPath // Optimized continuous paths (perimeters + infill)
	IsTopLayer      bool             // True if this is a top solid layer
	IsBottomLayer   bool             // True if this is a bottom solid layer
}

// BuildPolygons connects segments into closed polygons
func (s *Slice) BuildPolygons(tolerance float64) {
	if len(s.Segments) == 0 {
		return
	}

	// Copy segments as we'll be removing them as we build polygons
	remainingSegments := make([]LineSegment, len(s.Segments))
	copy(remainingSegments, s.Segments)

	// Build polygons by connecting segments
	for len(remainingSegments) > 0 {
		// Start a new contour with the first remaining segment
		contour := Polygon{
			Points:   make([]Vector3, 0),
			IsClosed: false,
			IsHole:   false,
		}

		// Take first segment
		current := remainingSegments[0]
		remainingSegments = remainingSegments[1:]

		contour.Points = append(contour.Points, current.Start, current.End)

		// Try to extend the contour
		maxIterations := len(remainingSegments) + 1
		iterations := 0
		for len(remainingSegments) > 0 && iterations < maxIterations {
			iterations++

			// Try to find a segment that connects to the end of our contour
			lastPoint := contour.Points[len(contour.Points)-1]
			firstPoint := contour.Points[0]

			foundConnection := false
			for i, seg := range remainingSegments {
				// Check if segment connects to end of contour
				if seg.Start.Equals(lastPoint) {
					contour.Points = append(contour.Points, seg.End)
					remainingSegments = append(remainingSegments[:i], remainingSegments[i+1:]...)
					foundConnection = true
					break
				} else if seg.End.Equals(lastPoint) {
					contour.Points = append(contour.Points, seg.Start)
					remainingSegments = append(remainingSegments[:i], remainingSegments[i+1:]...)
					foundConnection = true
					break
				}
			}

			// Check if contour is closed
			if len(contour.Points) > 2 {
				if lastPoint.Equals(firstPoint) ||
					contour.Points[len(contour.Points)-1].Equals(firstPoint) {
					contour.IsClosed = true
					break
				}
			}

			if !foundConnection {
				break
			}
		}

		// Add contour if it has enough points
		if len(contour.Points) >= 3 {
			// Ensure consistent winding order - make all polygons counter-clockwise (positive area)
			area := contour.GetArea()
			if area < 0 {
				// Clockwise - reverse to make counter-clockwise
				contour.Reverse()
			}
			s.Polygons = append(s.Polygons, contour)
		}
	}

	// Determine which contours are holes (inside other contours)
	s.ClassifyPolygons()
}

// ClassifyPolygons determines which contours are holes based on containment
func (s *Slice) ClassifyPolygons() {
	// Sort contours by absolute area (largest first)
	sort.Slice(s.Polygons, func(i, j int) bool {
		return math.Abs(s.Polygons[i].GetArea()) > math.Abs(s.Polygons[j].GetArea())
	})

	// A polygon is a hole if it's contained within another polygon
	// We check containment by testing if a point from the polygon is inside another
	for i := range s.Polygons {
		if !s.Polygons[i].IsClosed || len(s.Polygons[i].Points) < 3 {
			continue
		}

		s.Polygons[i].IsHole = false
		testPoint := s.Polygons[i].Points[0]

		// Check if this polygon is contained in any larger polygon
		for j := range s.Polygons {
			if i == j || !s.Polygons[j].IsClosed || len(s.Polygons[i].Points) < 3 || s.Polygons[i].Points[0].Z != s.Polygons[j].Points[0].Z {
				continue
			}

			// Only check larger polygons (already sorted by area)
			if j < i && s.Polygons[j].ContainsPoint(testPoint) {
				s.Polygons[i].IsHole = true
				break
			}
		}
	}
}

// ===== Helper functions =====

// IntersectZPlane finds where a triangle intersects a horizontal plane at height z
// Returns a line segment if the triangle crosses the plane (2 edge intersections)
func (t Triangle) IntersectZPlane(z float64) (LineSegment, bool) {
	// Check each edge of the triangle for intersection with plane
	edges := []LineSegment{
		{t.V1, t.V2},
		{t.V2, t.V3},
		{t.V3, t.V1},
	}

	// Check if edge crosses the plane
	intersections := make([]Vector3, 0, 2)
	for _, edge := range edges {
		if point, ok := edge.IntersectZPlane(z); ok {
			intersections = append(intersections, point)
		}
	}

	// A valid intersection should have exactly 2 points
	if len(intersections) == 2 {
		return LineSegment{
			Start: intersections[0],
			End:   intersections[1],
		}, true
	}

	return LineSegment{}, false
}

// ContainsPoint checks if a point is inside the slice's solid geometry
func (s *Slice) ContainsPoint(p Vector3) bool {
	inShell := false
	for _, poly := range s.Polygons {
		if !poly.IsHole && poly.ContainsPoint(p) {
			inShell = true
			break
		}
	}
	if !inShell {
		return false
	}
	for _, poly := range s.Polygons {
		if poly.IsHole && poly.ContainsPoint(p) {
			return false
		}
	}
	return true
}
