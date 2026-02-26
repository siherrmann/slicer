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
					Segments:   []model.PathSegment{{Start: currentPos, End: pathStart, IsTravel: true}},
					PathType:   model.PathTravel,
					LayerIndex: path.LayerIndex,
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
				Segments:   zHopSegments,
				PathType:   model.PathTravel,
				LayerIndex: i,
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
			PathType:   model.PathTravel,
			LayerIndex: len(bm.Slices) - 1,
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

	var allSegments []model.PathSegment
	for _, path := range paths {
		allSegments = append(allSegments, path.Segments...)
	}

	if len(allSegments) == 0 {
		return model.ContinuousPath{}
	}

	var cleanedSegments []model.PathSegment

	currentSection := []model.Vector3{allSegments[0].Start}
	currentIsTravel := allSegments[0].IsTravel

	flushSection := func(section []model.Vector3, isTravel bool) {
		if len(section) < 2 {
			return
		}

		// Simplify
		secEpsilon := epsilon
		if isTravel {
			secEpsilon = epsilon * 2
		}

		simplified := ramerDouglasPeucker(section, secEpsilon)
		for i := 0; i < len(simplified)-1; i++ {
			cleanedSegments = append(cleanedSegments, model.PathSegment{
				Start:    simplified[i],
				End:      simplified[i+1],
				IsTravel: isTravel,
			})
		}
	}

	for _, seg := range allSegments {
		if seg.IsTravel != currentIsTravel {
			flushSection(currentSection, currentIsTravel)
			currentSection = []model.Vector3{seg.Start}
			currentIsTravel = seg.IsTravel
		}
		currentSection = append(currentSection, seg.End)
	}
	flushSection(currentSection, currentIsTravel)

	return model.ContinuousPath{
		Segments:   cleanedSegments,
		PathType:   model.PathExtrusion,
		LayerIndex: paths[0].LayerIndex, // preserve layer index from the first path
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
