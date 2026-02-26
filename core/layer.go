package core

import (
	"math"
	"sort"

	"github.com/siherrmann/slicer/model"
)

// GenerateLayerPath berechnet den optimierten Werkzeugpfad
func GenerateLayerPath(polygons []model.Polygon, params model.SliceConfig, modelBounds model.BoundingBox, layerIndex int) []model.ContinuousPath {
	var paths []model.ContinuousPath
	var shells []model.Polygon
	var holes []model.Polygon

	for _, p := range polygons {
		if p.IsHole {
			holes = append(holes, p)
		} else {
			shells = append(shells, p)
		}
	}

	// Vase mode: single outer wall, no infill, spiral Z interpolation
	if params.VaseMode && layerIndex > 0 {
		for _, shell := range shells {
			// Generate single outer wall
			wall := shell.OffsetPolygon(-params.LineWidth * 0.5)
			if wall == nil || len(wall.Points) < 3 {
				continue
			}

			z := wall.Points[0].Z
			nextZ := z + params.LayerHeight

			var segments []model.PathSegment
			totalPoints := len(wall.Points)
			for j := 0; j < totalPoints; j++ {
				nextIdx := (j + 1) % totalPoints
				// Interpolate Z across the perimeter for spiral effect
				t := float64(j) / float64(totalPoints)
				startZ := z + t*(nextZ-z)
				t2 := float64(j+1) / float64(totalPoints)
				endZ := z + t2*(nextZ-z)

				start := model.Vector3{X: wall.Points[j].X, Y: wall.Points[j].Y, Z: startZ}
				end := model.Vector3{X: wall.Points[nextIdx].X, Y: wall.Points[nextIdx].Y, Z: endZ}

				segments = append(segments, model.PathSegment{
					Start:    start,
					End:      end,
					IsTravel: false,
					Speed:    params.OuterShellSpeed,
					Category: model.CategoryOuterWall,
				})
			}

			if len(segments) > 0 {
				paths = append(paths, model.ContinuousPath{
					Segments:   segments,
					PathType:   model.PathExtrusion,
					LayerIndex: layerIndex,
				})
			}
		}
		return paths
	}

	// Process each shell completely (walls + infill) before moving to next shell
	for _, shell := range shells {
		// 1. Print all walls		// 2. Standard mode walls
		walls := GenerateWalls(shell, params, layerIndex)
		paths = append(paths, walls...)

		// 2. Print infill for this shell
		infillArea := shell.OffsetPolygon(-CalculateInfillOffset(params) + (params.LineWidth*params.InfillOverlap)/2)
		if infillArea == nil || len(infillArea.Points) < 3 {
			continue
		}

		// Generate the uncut infill pattern
		// Calculate Z position for this layer
		z := shell.Points[0].Z
		infillPattern := GenerateInfill(modelBounds, &shell, params, layerIndex, z)

		// Cut the infill pattern to fit within the shell and avoid holes (if needed)
		if len(infillPattern.Segments) > 0 {
			infillPaths := cutInfill(infillPattern, infillArea, holes)
			paths = append(paths, infillPaths...)
		}
	}

	return paths
}

// --- Infill Cutting ---

// intersectionPoint represents an intersection with its position along a segment
type intersectionPoint struct {
	Point model.Vector3
	T     float64 // Parameter t (0 to 1) along the segment
}

// cutInfill takes an uncut infill pattern and trims it to fit within the shell and avoid holes.
// All coordinates are in model space — no rotation is applied.
func cutInfill(pattern model.ContinuousPath, shell *model.Polygon, holes []model.Polygon) []model.ContinuousPath {
	var resultSegments []model.PathSegment
	var lastEndPoint *model.Vector3

	// Get shell bounds for quick skipping
	shellBounds := shell.GetBounds()

	// Pre-calculate hole bounds
	holeBounds := make([]model.BoundingBox, len(holes))
	for i, hole := range holes {
		holeBounds[i] = hole.GetBounds()
	}

	// Get shell lines once
	shellLines := shell.GetLines()

	// Process each segment in the pattern
	for _, segment := range pattern.Segments {
		// Quick bounding box check against shell
		segMinX := math.Min(segment.Start.X, segment.End.X)
		segMaxX := math.Max(segment.Start.X, segment.End.X)
		segMinY := math.Min(segment.Start.Y, segment.End.Y)
		segMaxY := math.Max(segment.Start.Y, segment.End.Y)

		if segMaxX < shellBounds.MinX || segMinX > shellBounds.MaxX ||
			segMaxY < shellBounds.MinY || segMinY > shellBounds.MaxY {
			continue // Segment is completely outside shell bounds
		}

		infillLine := model.LineSegment{
			Start: segment.Start,
			End:   segment.End,
		}

		var intersections []intersectionPoint

		// Helper to extract intersection parameter T and store correctly
		addIntersections := func(lines []model.LineSegment) {
			for _, line := range lines {
				intersection, ok := infillLine.IntersectSegments(line)
				if ok {
					segmentVec := segment.End.Sub(segment.Start)
					intersectVec := intersection.Sub(segment.Start)
					segmentLength := math.Sqrt(segmentVec.X*segmentVec.X + segmentVec.Y*segmentVec.Y)
					if segmentLength < 1e-10 {
						continue
					}
					intersectLength := math.Sqrt(intersectVec.X*intersectVec.X + intersectVec.Y*intersectVec.Y)
					t := intersectLength / segmentLength
					// Check sign (intersectVec should be in same direction as segmentVec)
					if segmentVec.X*intersectVec.X+segmentVec.Y*intersectVec.Y < 0 {
						t = -t
					}
					if t >= -1e-6 && t <= 1.0+1e-6 {
						t = math.Max(0, math.Min(1, t))
						intersections = append(intersections, intersectionPoint{Point: intersection, T: t})
					}
				}
			}
		}

		// Find intersections with shell edges
		addIntersections(shellLines)

		// Find intersections with holes
		for i, hole := range holes {
			hB := holeBounds[i]
			if segMaxX < hB.MinX || segMinX > hB.MaxX ||
				segMaxY < hB.MinY || segMinY > hB.MaxY {
				continue
			}
			addIntersections(hole.GetLines())
		}

		// If no intersections, check if the entire segment is inside the shell
		if len(intersections) == 0 {
			segmentInside := shell.ContainsPoint(segment.Start) && shell.ContainsPoint(segment.End)

			if segmentInside {
				// Check if it's not in any hole
				inHole := false
				for _, hole := range holes {
					if hole.ContainsPoint(segment.Start) || hole.ContainsPoint(segment.End) {
						inHole = true
						break
					}
				}

				if !inHole {
					// Add travel move if needed
					if lastEndPoint != nil && lastEndPoint.Distance(segment.Start) > 1e-6 {
						resultSegments = append(resultSegments, model.PathSegment{
							Start:    *lastEndPoint,
							End:      segment.Start,
							IsTravel: true,
							Category: model.CategoryTravel,
						})
					}

					resultSegments = append(resultSegments, segment)
					endPoint := segment.End
					lastEndPoint = &endPoint
				}
			}
			continue
		}

		// Sort intersections by parameter t (position along the segment)
		sort.Slice(intersections, func(i, j int) bool {
			return intersections[i].T < intersections[j].T
		})

		// Remove duplicate intersections (too close together)
		var deduped []intersectionPoint
		for i, ip := range intersections {
			if i == 0 || ip.T-deduped[len(deduped)-1].T > 1e-6 {
				deduped = append(deduped, ip)
			}
		}
		intersections = deduped

		// Build candidate sub-segments by testing midpoints

		// Create boundary points: [0 (start), intersections..., 1 (end)]
		tValues := []float64{0}
		for _, ip := range intersections {
			tValues = append(tValues, ip.T)
		}
		tValues = append(tValues, 1)

		segVec := segment.End.Sub(segment.Start)

		for i := 0; i < len(tValues)-1; i++ {
			t1 := tValues[i]
			t2 := tValues[i+1]
			if t2-t1 < 1e-8 {
				continue
			}

			// Test midpoint
			midT := (t1 + t2) / 2.0
			midPoint := segment.Start.Add(segVec.Scale(midT))

			// Check if midpoint is inside shell and outside all holes
			if !shell.ContainsPoint(midPoint) {
				continue
			}

			inHole := false
			for _, hole := range holes {
				if hole.ContainsPoint(midPoint) {
					inHole = true
					break
				}
			}
			if inHole {
				continue
			}

			// This sub-segment is valid
			p1 := segment.Start.Add(segVec.Scale(t1))
			p2 := segment.Start.Add(segVec.Scale(t2))

			if lastEndPoint != nil && lastEndPoint.Distance(p1) > 1e-6 {
				resultSegments = append(resultSegments, model.PathSegment{
					Start:    *lastEndPoint,
					End:      p1,
					IsTravel: true,
					Category: model.CategoryTravel,
				})
			}

			resultSegments = append(resultSegments, model.PathSegment{
				Start:    p1,
				End:      p2,
				IsTravel: false,
				Category: segment.Category,
				Speed:    segment.Speed,
				FlowRate: segment.FlowRate,
			})

			lastEndPoint = &p2
		}
	}

	if len(resultSegments) == 0 {
		return nil
	}

	return []model.ContinuousPath{{
		Segments:   resultSegments,
		PathType:   model.PathExtrusion,
		LayerIndex: pattern.LayerIndex, // preserve layer index
	}}
}
