package model

import (
	"math"
)

// Polygon represents a 2D polygon
type Polygon struct {
	Points   []Vector3
	IsClosed bool // Whether the contour forms a closed loop
	IsHole   bool // Whether this contour is a hole (inner boundary)
}

// BoundingBox represents a 2D bounding box (XY plane)
type BoundingBox struct {
	MinX float64 `json:"min_x"`
	MinY float64 `json:"min_y"`
	MaxX float64 `json:"max_x"`
	MaxY float64 `json:"max_y"`
	MinZ float64 `json:"min_z"`
	MaxZ float64 `json:"max_z"`
}

// GetArea calculates the signed area of a polygon
func (p *Polygon) GetArea() float64 {
	if len(p.Points) < 3 {
		return 0
	}

	var area float64
	n := len(p.Points)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += p.Points[i].X * p.Points[j].Y
		area -= p.Points[j].X * p.Points[i].Y
	}
	return area / 2
}

// GetBounds calculates the 2D bounding box of a contour
func (p *Polygon) GetBounds() BoundingBox {
	if len(p.Points) == 0 {
		return BoundingBox{}
	}

	bounds := BoundingBox{
		MinX: p.Points[0].X,
		MinY: p.Points[0].Y,
		MaxX: p.Points[0].X,
		MaxY: p.Points[0].Y,
	}

	for _, p := range p.Points[1:] {
		if p.X < bounds.MinX {
			bounds.MinX = p.X
		}
		if p.X > bounds.MaxX {
			bounds.MaxX = p.X
		}
		if p.Y < bounds.MinY {
			bounds.MinY = p.Y
		}
		if p.Y > bounds.MaxY {
			bounds.MaxY = p.Y
		}
	}

	return bounds
}

// GetLines returns the edges of the polygon as line segments
func (p *Polygon) GetLines() []LineSegment {
	lines := make([]LineSegment, len(p.Points))
	for i, vec := range p.Points {
		if i%2 == 0 && i < len(p.Points)-1 {
			lines[i] = LineSegment{Start: vec, End: p.Points[(i+1)%len(p.Points)]}
		} else if i%2 == 0 {
			lines[i] = LineSegment{Start: vec, End: p.Points[0]}
		} else {
			lines[i] = LineSegment{Start: p.Points[(i+1)%len(p.Points)], End: vec}
		}
	}
	return lines
}

// IsClockwise returns true if polygon is oriented clockwise
func (p *Polygon) IsClockwise() bool {
	return p.GetArea() < 0
}

// Reverse reverses the order of points
func (p *Polygon) Reverse() {
	n := len(p.Points)
	for i := 0; i < n/2; i++ {
		p.Points[i], p.Points[n-1-i] = p.Points[n-1-i], p.Points[i]
	}
}

func (p *Polygon) Rotate(angle float64) *Polygon {
	newPoints := make([]Vector3, len(p.Points))
	for i, pt := range p.Points {
		newPoints[i] = pt.Rotate(angle)
	}
	return &Polygon{
		Points:   newPoints,
		IsClosed: p.IsClosed,
		IsHole:   p.IsHole,
	}
}

// SetZ recursively updates the Z coordinate of all points in the polygon
func (p *Polygon) SetZ(z float64) {
	for i := range p.Points {
		p.Points[i].Z = z
	}
}

// ToContinuousPath converts a Polygon into a ContinuousPath with the given speed and category.
func (p *Polygon) ToContinuousPath(speed float64, category PathCategory, layerIndex int) ContinuousPath {
	if len(p.Points) < 2 {
		return ContinuousPath{}
	}
	var segments []PathSegment
	for j := 0; j < len(p.Points); j++ {
		nextIdx := (j + 1) % len(p.Points)

		// If the polygon is not closed, don't connect the last point to the first
		if !p.IsClosed && j == len(p.Points)-1 {
			continue
		}

		segments = append(segments, PathSegment{
			Start:    p.Points[j],
			End:      p.Points[nextIdx],
			IsTravel: false,
			Speed:    speed,
			Category: category,
		})
	}

	return ContinuousPath{
		Segments:   segments,
		PathType:   PathExtrusion,
		LayerIndex: layerIndex,
	}
}

// OffsetPolygon creates a new polygon offset by the given distance
// Positive distance offsets outward, negative offsets inward
func (p *Polygon) OffsetPolygon(distance float64) *Polygon {
	if len(p.Points) < 3 {
		return nil
	}

	// Step 0: Pre-clean the polygon to remove microscopic or collinear segments.
	// This prevents numerical instability leading to millions of mm of wall paths.
	cleanedPoints := ramerDouglasPeuckerPolygon(p.Points, 0.001)
	if len(cleanedPoints) < 3 {
		return nil
	}

	// Step 1: Offset each edge by the perpendicular distance
	n := len(cleanedPoints)
	offsetEdges := make([]LineSegment, n)
	for i := range n {
		j := (i + 1) % n

		p1 := cleanedPoints[i]
		p2 := cleanedPoints[j]

		// Edge vector
		edge := p2.Sub(p1)
		edgeLen := edge.Length()

		var offset Vector3
		if edgeLen < 1e-10 {
			// For degenerate edges, use zero offset to preserve Z coordinate
			offset = Vector3{X: 0, Y: 0, Z: 0}
		} else {
			// Perpendicular vector (rotate 90° counterclockwise)
			perp := edge.Perpendicular()
			perp = perp.Normalize()
			// Offset the edge
			offset = perp.Scale(-distance)
		}

		offsetEdges[i] = LineSegment{
			Start: p1.Add(offset),
			End:   p2.Add(offset),
		}
	}

	// Step 2: Find intersections at corners
	newPoints := make([]Vector3, 0, n)
	for i := range n {
		prevEdge := offsetEdges[(i-1+n)%n]
		currEdge := offsetEdges[i]

		// Find intersection of the two offset edges
		t, valid := prevEdge.IntersectLines(currEdge)
		if valid {
			// Calculate intersection point
			intersection := Vector3{
				X: prevEdge.Start.X + t*(prevEdge.End.X-prevEdge.Start.X),
				Y: prevEdge.Start.Y + t*(prevEdge.End.Y-prevEdge.Start.Y),
				Z: prevEdge.Start.Z + t*(prevEdge.End.Z-prevEdge.Start.Z),
			}
			newPoints = append(newPoints, intersection)
		} else {
			// Parallel edges - use the end point of previous edge
			newPoints = append(newPoints, prevEdge.End)
		}
	}

	if len(newPoints) < 3 {
		return nil
	}

	result := &Polygon{Points: newPoints}

	// Step 3: Remove self-intersections (simplified approach)
	result = removeSelfIntersections(result)

	// Step 4: Simplify perfectly collinear or overly dense vertices to fix 200MB JSON bloat
	if result != nil && len(result.Points) > 3 {
		// Use a conservative simplification tolerance (e.g. 0.05mm) to keep curves round
		// but annihilate microscopic 0.001mm segments inherited from dense STL files
		simplified := ramerDouglasPeuckerPolygon(result.Points, 0.05)
		if len(simplified) > 3 {
			result.Points = simplified
		}
	}

	return result
}

// Private helper to simplify closed polygon loops
func ramerDouglasPeuckerPolygon(points []Vector3, epsilon float64) []Vector3 {
	// For closed loops, we don't want to accidentally delete the entire shape.
	// We'll just simplify it as an open path, and then if the start != end, that's fine.
	// We just ensure we don't reduce a circle to a triangle.
	if len(points) <= 3 {
		return points
	}

	// We need to implement a simple collinear decimation instead to be safe,
	// or RDP if we want true geometric simplification.
	// Since RDP is already in `core`, let's just do collinear here to be fast and safe.
	var decimated []Vector3
	decimated = append(decimated, points[0])

	for k := 1; k < len(points)-1; k++ {
		pPrev := decimated[len(decimated)-1]
		pCurr := points[k]
		pNext := points[k+1]

		dx1 := pCurr.X - pPrev.X
		dy1 := pCurr.Y - pPrev.Y
		dx2 := pNext.X - pCurr.X
		dy2 := pNext.Y - pCurr.Y
		cross := dx1*dy2 - dy1*dx2

		// If points are perfectly straight, skip adding pCurr (0.01mm tolerance)
		if math.Abs(cross) > 0.001 {
			decimated = append(decimated, pCurr)
		}
	}
	decimated = append(decimated, points[len(points)-1])

	return decimated
}

// ContainsPoint checks if a point is inside a polygon using ray casting
func (poly *Polygon) ContainsPoint(p Vector3) bool {
	if len(poly.Points) < 3 {
		return false
	}

	inside := false
	n := len(poly.Points)

	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		xi, yi := poly.Points[i].X, poly.Points[i].Y
		xj, yj := poly.Points[j].X, poly.Points[j].Y

		if ((yi > p.Y) != (yj > p.Y)) &&
			(p.X < (xj-xi)*(p.Y-yi)/(yj-yi)+xi) {
			inside = !inside
		}
	}

	return inside
}

// ClipPolygonToPolygon clips subject polygon to clip polygon using Sutherland-Hodgman algorithm
func (p *Polygon) ClipPolygonToPolygon(subject *Polygon) *Polygon {
	if len(subject.Points) < 3 || len(p.Points) < 3 {
		return nil
	}

	output := &Polygon{Points: make([]Vector3, len(subject.Points))}
	copy(output.Points, subject.Points)

	// Clip against each edge of the clip polygon
	n := len(p.Points)
	for i := 0; i < n; i++ {
		if len(output.Points) == 0 {
			return nil
		}

		j := (i + 1) % n
		edge := LineSegment{Start: p.Points[i], End: p.Points[j]}
		output = output.ClipPolygonByEdge(edge)
	}

	if len(output.Points) < 3 {
		return nil
	}

	return output
}

// ClipLineToPolygon clips a line segment to a polygon boundary
// Returns all segments that lie within the polygon
func (poly *Polygon) ClipLineToPolygon(line LineSegment) []LineSegment {
	if len(poly.Points) < 3 {
		return nil
	}

	// Find all intersection points between the line and polygon edges
	type intersection struct {
		point Vector3
		t     float64 // Parameter along line (0 = start, 1 = end)
	}

	intersections := []intersection{}
	n := len(poly.Points)

	// Check intersections with each polygon edge
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		edge := LineSegment{Start: poly.Points[i], End: poly.Points[j]}

		// Find intersection between line and edge
		p1, p2 := line.Start, line.End
		p3, p4 := edge.Start, edge.End

		denom := (p1.X-p2.X)*(p3.Y-p4.Y) - (p1.Y-p2.Y)*(p3.X-p4.X)
		if math.Abs(denom) < 1e-10 {
			continue // Parallel or coincident
		}

		t := ((p1.X-p3.X)*(p3.Y-p4.Y) - (p1.Y-p3.Y)*(p3.X-p4.X)) / denom
		u := -((p1.X-p2.X)*(p1.Y-p3.Y) - (p1.Y-p2.Y)*(p1.X-p3.X)) / denom

		// Check if intersection is within both segments
		if t >= 0 && t <= 1 && u >= 0 && u <= 1 {
			intersections = append(intersections, intersection{
				point: Vector3{
					X: p1.X + t*(p2.X-p1.X),
					Y: p1.Y + t*(p2.Y-p1.Y),
					Z: p1.Z + t*(p2.Z-p1.Z), // Preserve Z along the line
				},
				t: t,
			})
		}
	}

	// Check if start and end points are inside
	startInside := poly.ContainsPoint(line.Start)
	endInside := poly.ContainsPoint(line.End)

	// Build result segments
	var result []LineSegment

	if len(intersections) == 0 {
		// No intersections
		if startInside && endInside {
			// Entire line is inside
			result = append(result, line)
		}
		// Otherwise, line is entirely outside
		return result
	}

	// Sort intersections by t parameter
	for i := 0; i < len(intersections)-1; i++ {
		for j := i + 1; j < len(intersections); j++ {
			if intersections[i].t > intersections[j].t {
				intersections[i], intersections[j] = intersections[j], intersections[i]
			}
		}
	}

	// Build segments from intersections
	points := []Vector3{}
	if startInside {
		points = append(points, line.Start)
	}
	for _, inter := range intersections {
		points = append(points, inter.point)
	}
	if endInside {
		points = append(points, line.End)
	}

	// Create segments from consecutive pairs of points
	// Only include segments whose midpoint is inside the polygon
	for i := 0; i < len(points)-1; i++ {
		seg := LineSegment{Start: points[i], End: points[i+1]}

		// Check midpoint
		mid := Vector3{
			X: (seg.Start.X + seg.End.X) / 2,
			Y: (seg.Start.Y + seg.End.Y) / 2,
		}

		if poly.ContainsPoint(mid) {
			result = append(result, seg)
		}
	}

	return result
}

// ClipPolygonByEdge clips a polygon against a single edge
func (poly *Polygon) ClipPolygonByEdge(edge LineSegment) *Polygon {
	if len(poly.Points) == 0 {
		return poly
	}

	result := &Polygon{Points: []Vector3{}}

	// Edge normal (perpendicular, pointing inward to clip region)
	edgeVec := edge.End.Sub(edge.Start)
	normal := Vector3{X: -edgeVec.Y, Y: edgeVec.X}

	n := len(poly.Points)
	for i := 0; i < n; i++ {
		current := poly.Points[i]
		next := poly.Points[(i+1)%n]

		// Check which side of the edge each point is on
		currentVec := current.Sub(edge.Start)
		nextVec := next.Sub(edge.Start)

		currentInside := normal.Dot(currentVec) >= 0
		nextInside := normal.Dot(nextVec) >= 0

		if currentInside {
			result.Points = append(result.Points, current)
		}

		// If edge crosses the clip edge, find intersection
		if currentInside != nextInside {
			// Find intersection point
			segLine := LineSegment{Start: current, End: next}
			t, valid := segLine.IntersectLines(edge)
			if valid && t >= 0 && t <= 1 {
				intersection := Vector3{
					X: current.X + t*(next.X-current.X),
					Y: current.Y + t*(next.Y-current.Y),
				}
				result.Points = append(result.Points, intersection)
			}
		}
	}

	return result
}

// removeSelfIntersections removes self-intersecting parts of a polygon
// This is a simplified implementation - a full version would use a sweep-line algorithm
func removeSelfIntersections(poly *Polygon) *Polygon {
	if len(poly.Points) < 3 {
		return poly
	}

	n := len(poly.Points)
	edges := make([]LineSegment, n)

	for i := 0; i < n; i++ {
		j := (i + 1) % n
		edges[i] = LineSegment{Start: poly.Points[i], End: poly.Points[j]}
	}

	// Find any self-intersections
	// For simplicity, just check each edge against all non-adjacent edges
	intersectionFound := false

	for i := 0; i < n && !intersectionFound; i++ {
		for j := i + 2; j < n && !intersectionFound; j++ {
			// Skip adjacent edges
			if (j+1)%n == i {
				continue
			}

			_, intersects := edges[i].IntersectSegments(edges[j])
			if intersects {
				intersectionFound = true
				break
			}
		}
	}

	// If no self-intersections, return as-is
	if !intersectionFound {
		return poly
	}

	// If self-intersections found, try to recover by removing problematic vertices
	// This is a very simplified approach - just keep points that maintain a valid polygon
	filtered := []Vector3{poly.Points[0]}
	for i := 1; i < n; i++ {
		// Only add point if it's not too close to the previous point
		prev := filtered[len(filtered)-1]
		curr := poly.Points[i]
		dist := prev.Distance(curr)
		if dist > 1e-6 {
			filtered = append(filtered, curr)
		}
	}

	if len(filtered) < 3 {
		return poly // Return original if filtering made it invalid
	}

	return &Polygon{Points: filtered}
}

// IntersectLine finds all x-coordinates where a horizontal line at height y intersects the polygon
func (p *Polygon) IntersectLine(y float64) []float64 {
	var intersections []float64
	n := len(p.Points)

	if n < 2 {
		return intersections
	}

	for i := 0; i < n; i++ {
		j := (i + 1) % n
		p1 := p.Points[i]
		p2 := p.Points[j]

		// Check if the edge crosses the horizontal line at y
		// Use < on one side and >= on the other to avoid double-counting vertices
		if (p1.Y <= y && p2.Y > y) || (p2.Y <= y && p1.Y > y) {
			// Skip horizontal edges (both points at approximately same Y)
			if math.Abs(p2.Y-p1.Y) < 1e-10 {
				continue
			}

			// Calculate x-coordinate of intersection
			t := (y - p1.Y) / (p2.Y - p1.Y)
			x := p1.X + t*(p2.X-p1.X)
			intersections = append(intersections, x)
		}
	}

	return intersections
}
