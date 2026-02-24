package core

import (
	"github.com/siherrmann/slicer/model"
)

// GenerateRaft creates raft layers below the model for better bed adhesion.
// Returns the raft paths and the Z offset that the model should be shifted up by.
func GenerateRaft(firstLayerPolygons []model.Polygon, config model.SliceConfig) ([]model.ContinuousPath, float64) {
	if config.RaftLayers <= 0 {
		return nil, 0
	}

	var paths []model.ContinuousPath

	// Get all non-hole shells from the first layer
	var shells []model.Polygon
	for _, p := range firstLayerPolygons {
		if !p.IsHole {
			shells = append(shells, p)
		}
	}

	if len(shells) == 0 {
		return nil, 0
	}

	// Expand each shell by raft offset to create raft outline
	var raftOutlines []*model.Polygon
	for _, shell := range shells {
		expanded := shell.OffsetPolygon(config.RaftOffset)
		if expanded != nil && len(expanded.Points) > 2 {
			raftOutlines = append(raftOutlines, expanded)
		}
	}

	if len(raftOutlines) == 0 {
		return nil, 0
	}

	// Calculate Z heights for raft layers
	// Base raft layer is thicker (first layer height), subsequent layers use regular height
	raftHeight := float64(config.RaftLayers) * config.FirstLayer

	for layer := 0; layer < config.RaftLayers; layer++ {
		z := config.FirstLayer * float64(layer+1)

		// Alternate direction between layers
		isBase := layer < config.RaftLayers/2+1

		// Line spacing: base layers are wider spaced, top layers are tighter
		spacing := config.LineWidth * 3.0
		if !isBase {
			spacing = config.LineWidth * 1.5
		}

		for _, outline := range raftOutlines {
			bounds := outline.GetBounds()

			var segments []model.PathSegment

			if layer%2 == 0 {
				// Horizontal lines
				for y := bounds.MinY; y <= bounds.MaxY; y += spacing {
					start := model.Vector3{X: bounds.MinX - 0.5, Y: y, Z: z}
					end := model.Vector3{X: bounds.MaxX + 0.5, Y: y, Z: z}

					line := model.LineSegment{Start: start, End: end}
					clipped := outline.ClipLineToPolygon(line)

					for _, seg := range clipped {
						if len(segments) > 0 {
							lastEnd := segments[len(segments)-1].End
							if lastEnd.Distance(seg.Start) > 0.01 {
								segments = append(segments, model.PathSegment{
									Start:    lastEnd,
									End:      seg.Start,
									IsTravel: true,
									Category: model.CategoryTravel,
								})
							}
						}
						segments = append(segments, model.PathSegment{
							Start:    seg.Start,
							End:      seg.End,
							IsTravel: false,
							Speed:    config.FirstLayerSpeed,
							Category: model.CategorySupport,
						})
					}
				}
			} else {
				// Vertical lines
				for x := bounds.MinX; x <= bounds.MaxX; x += spacing {
					start := model.Vector3{X: x, Y: bounds.MinY - 0.5, Z: z}
					end := model.Vector3{X: x, Y: bounds.MaxY + 0.5, Z: z}

					line := model.LineSegment{Start: start, End: end}
					clipped := outline.ClipLineToPolygon(line)

					for _, seg := range clipped {
						if len(segments) > 0 {
							lastEnd := segments[len(segments)-1].End
							if lastEnd.Distance(seg.Start) > 0.01 {
								segments = append(segments, model.PathSegment{
									Start:    lastEnd,
									End:      seg.Start,
									IsTravel: true,
									Category: model.CategoryTravel,
								})
							}
						}
						segments = append(segments, model.PathSegment{
							Start:    seg.Start,
							End:      seg.End,
							IsTravel: false,
							Speed:    config.FirstLayerSpeed,
							Category: model.CategorySupport,
						})
					}
				}
			}

			if len(segments) > 0 {
				paths = append(paths, model.ContinuousPath{
					Segments: segments,
					PathType: model.PathExtrusion,
				})
			}
		}
	}

	return paths, raftHeight
}
