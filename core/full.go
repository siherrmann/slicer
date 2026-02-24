package core

import (
	"runtime"
	"sync"

	"github.com/siherrmann/slicer/model"
)

// GenerateFullSTLPath erzeugt einen zusammenhängenden Pfad für das gesamte Modell
func GenerateFullSTLPath(bm *model.BaseModel, config model.SliceConfig) []model.ContinuousPath {
	var fullPaths []model.ContinuousPath
	currentPos := config.StartPosition

	// Helper function to append paths and auto-generate travel moves between them
	appendPathsWithTravel := func(pathsToAdd []model.ContinuousPath) {
		for _, path := range pathsToAdd {
			if len(path.Segments) == 0 {
				continue
			}
			pathStart := path.Segments[0].Start
			if currentPos.Distance(pathStart) > 0.01 {
				fullPaths = append(fullPaths, model.ContinuousPath{
					Segments: []model.PathSegment{{Start: currentPos, End: pathStart, IsTravel: true}},
					PathType: model.PathTravel,
				})
			}
			fullPaths = append(fullPaths, path)
			currentPos = path.Segments[len(path.Segments)-1].End
		}
	}

	// -1. Generate Raft if needed
	if config.RaftLayers > 0 && len(bm.Slices) > 0 {
		raftPaths, _ := GenerateRaft(bm.Slices[0].Polygons, config)
		appendPathsWithTravel(raftPaths)
	}
	if len(bm.Slices) > 0 {
		skirtPaths := GenerateSkirt(bm.Slices[0].Polygons, config)
		appendPathsWithTravel(skirtPaths)
	}

	// Pre-compute support paths if enabled
	var supportPaths map[int][]model.ContinuousPath
	if config.SupportType != model.SupportNone {
		supportPaths = GenerateSupportPaths(bm, config)
	}

	// Prepare a slice to catch layer results in order
	results := make([][]model.ContinuousPath, len(bm.Slices))
	var wg sync.WaitGroup

	// Determine number of workers
	numWorkers := runtime.NumCPU()
	semaphore := make(chan struct{}, numWorkers)

	// 1. Calculate layer paths in parallel
	for i := range bm.Slices {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			slice := bm.Slices[idx]
			results[idx] = GenerateLayerPath(slice.Polygons, config, bm.Bounds, idx)
		}(i)
	}

	wg.Wait()

	// 2. Combine results with travel moves and Z-hops sequentially to maintain order and connectivity
	for i, layerPaths := range results {
		if len(layerPaths) == 0 {
			continue
		}

		// Add each path with travel moves between them
		appendPathsWithTravel(layerPaths)

		// 5. Kollisionsvermeidung: Z-Hop (Optional aber empfohlen)
		if i < len(bm.Slices)-1 {
			nextZ := bm.Slices[i+1].Z
			zHopPos := model.Vector3{X: currentPos.X, Y: currentPos.Y, Z: nextZ + config.LayerHeight}

			zHopSegments := []model.PathSegment{
				{Start: currentPos, End: zHopPos, IsTravel: true},
			}

			fullPaths = append(fullPaths, model.ContinuousPath{
				Segments: zHopSegments,
				PathType: model.PathTravel,
			})
			currentPos = zHopPos
		}

		// 6. Add support paths for this layer if any
		if paths, ok := supportPaths[i]; ok {
			appendPathsWithTravel(paths)
		}
	}

	// 6. Finaler Travel Move zur Endkoordinate
	if !currentPos.Equals(config.EndPosition) {
		fullPaths = append(fullPaths, model.ContinuousPath{
			Segments: []model.PathSegment{
				{Start: currentPos, End: config.EndPosition, IsTravel: true},
			},
			PathType: model.PathTravel,
		})
	}

	return fullPaths
}

// CleanFullPaths combines all paths and simplifies them using the Ramer-Douglas-Peucker algorithm
// epsilon controls the simplification tolerance - larger values = more aggressive simplification
// typical values: 0.01 to 0.1 mm
func CleanFullPaths(paths []model.ContinuousPath, epsilon float64) model.ContinuousPath {
	if len(paths) == 0 {
		return model.ContinuousPath{}
	}

	// Combine all segments from all paths
	var allSegments []model.PathSegment
	for _, path := range paths {
		allSegments = append(allSegments, path.Segments...)
	}

	if len(allSegments) == 0 {
		return model.ContinuousPath{}
	}

	// Build a list of all points with their travel status
	type PointWithTravel struct {
		Point    model.Vector3
		IsTravel bool
	}

	points := []PointWithTravel{{Point: allSegments[0].Start, IsTravel: allSegments[0].IsTravel}}
	for _, seg := range allSegments {
		points = append(points, PointWithTravel{Point: seg.End, IsTravel: seg.IsTravel})
	}

	// Apply RDP algorithm separately to continuous extrusion and travel sections
	simplified := []PointWithTravel{points[0]}
	currentSectionStart := 0
	currentIsTravel := points[0].IsTravel

	for i := 1; i < len(points); i++ {
		// Check if we're switching between travel and extrusion
		if points[i].IsTravel != currentIsTravel || i == len(points)-1 {
			endIdx := i
			if i == len(points)-1 {
				endIdx = i + 1
			}

			// Extract section
			section := make([]model.Vector3, endIdx-currentSectionStart)
			for j := currentSectionStart; j < endIdx; j++ {
				section[j-currentSectionStart] = points[j].Point
			}

			// Simplify section with RDP
			var simplifiedSection []model.Vector3
			if len(section) > 2 {
				// Only simplify extrusion paths, keep travel moves as-is (or use larger epsilon)
				sectionEpsilon := epsilon
				if currentIsTravel {
					sectionEpsilon = epsilon * 2 // More aggressive for travel moves
				}
				simplifiedSection = ramerDouglasPeucker(section, sectionEpsilon)
			} else {
				simplifiedSection = section
			}

			// Add simplified points (skip first if not the very first section, as it's already added)
			startIdx := 0
			if currentSectionStart > 0 {
				startIdx = 1
			}
			for j := startIdx; j < len(simplifiedSection); j++ {
				simplified = append(simplified, PointWithTravel{
					Point:    simplifiedSection[j],
					IsTravel: currentIsTravel,
				})
			}

			currentSectionStart = i
			currentIsTravel = points[i].IsTravel
		}
	}

	// Convert back to segments
	var cleanedSegments []model.PathSegment
	for i := 0; i < len(simplified)-1; i++ {
		cleanedSegments = append(cleanedSegments, model.PathSegment{
			Start:    simplified[i].Point,
			End:      simplified[i+1].Point,
			IsTravel: simplified[i].IsTravel,
		})
	}

	return model.ContinuousPath{
		Segments: cleanedSegments,
		PathType: model.PathExtrusion, // Combined path type
	}
}

// ramerDouglasPeucker implements the Ramer-Douglas-Peucker algorithm for polyline simplification
func ramerDouglasPeucker(points []model.Vector3, epsilon float64) []model.Vector3 {
	if len(points) <= 2 {
		return points
	}

	// Find the point with maximum distance from the line segment
	maxDist := 0.0
	maxIndex := 0
	start := points[0]
	end := points[len(points)-1]

	for i := 1; i < len(points)-1; i++ {
		dist := points[i].PerpendicularDistanceToLine(start, end)
		if dist > maxDist {
			maxDist = dist
			maxIndex = i
		}
	}

	// If max distance is greater than epsilon, recursively simplify
	if maxDist > epsilon {
		// Recursive call on both segments
		rec1 := ramerDouglasPeucker(points[:maxIndex+1], epsilon)
		rec2 := ramerDouglasPeucker(points[maxIndex:], epsilon)

		// Build result (remove duplicate point at connection)
		result := make([]model.Vector3, len(rec1)+len(rec2)-1)
		copy(result, rec1)
		copy(result[len(rec1):], rec2[1:])
		return result
	}

	// If max distance is less than epsilon, remove all points between start and end
	return []model.Vector3{start, end}
}
