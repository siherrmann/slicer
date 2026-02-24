package model

import (
	"fmt"
)

// Slice slices the model into horizontal layers
func (bm *BaseModel) Slice(config *SliceConfig) error {
	if len(bm.Triangles) == 0 {
		return fmt.Errorf("cannot slice empty model")
	}

	// Get model bounds to determine layer heights
	bounds := bm.GetBounds()
	layerHeights := config.CalculateLayerHeights(bounds.MinZ, bounds.MaxZ)

	// Create slices for each layer
	bm.Slices = make([]*Slice, len(layerHeights))
	for i, z := range layerHeights {
		bm.Slices[i] = &Slice{
			Z:        z,
			Segments: make([]LineSegment, 0),
			Polygons: make([]Polygon, 0),
		}
	}

	// For each triangle, find which layers it intersects and compute segments
	for _, triangle := range bm.Triangles {
		// Get triangle Z bounds
		minZ := triangle.V1.Z
		maxZ := triangle.V1.Z
		if triangle.V2.Z < minZ {
			minZ = triangle.V2.Z
		}
		if triangle.V3.Z < minZ {
			minZ = triangle.V3.Z
		}
		if triangle.V2.Z > maxZ {
			maxZ = triangle.V2.Z
		}
		if triangle.V3.Z > maxZ {
			maxZ = triangle.V3.Z
		}

		// Find which layers this triangle intersects
		for i, z := range layerHeights {
			if z >= minZ && z <= maxZ {
				// Compute intersection segment
				if segment, ok := triangle.IntersectZPlane(z); ok {
					bm.Slices[i].Segments = append(bm.Slices[i].Segments, segment)
				}
			}
		}
	}

	// Build contours from segments for each slice
	for _, slice := range bm.Slices {
		slice.BuildPolygons(config.Tolerance)
	}

	return nil
}

// Helper functions

// ClassifyTopBottomLayers determines which layers should be solid (top/bottom)
func (bm *BaseModel) ClassifyTopBottomLayers(config *SliceConfig) {
	if len(bm.Slices) == 0 {
		return
	}

	// Mark bottom layers
	for i := 0; i < config.BottomLayers && i < len(bm.Slices); i++ {
		bm.Slices[i].IsBottomLayer = true
		bm.Slices[i].LayerIndex = i
	}

	// Mark top layers
	startTop := len(bm.Slices) - config.TopLayers
	if startTop < 0 {
		startTop = 0
	}
	for i := startTop; i < len(bm.Slices); i++ {
		bm.Slices[i].IsTopLayer = true
		bm.Slices[i].LayerIndex = i
	}

	// Set layer index for all layers
	for i := range bm.Slices {
		bm.Slices[i].LayerIndex = i
	}
}
