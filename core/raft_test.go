package core

import (
	"testing"

	"github.com/siherrmann/slicer/model"
	"github.com/stretchr/testify/assert"
)

func TestGenerateRaft(t *testing.T) {
	config := model.NewSliceConfig()
	config.RaftLayers = 2
	config.FirstLayer = 0.3
	config.LineWidth = 0.4
	config.RaftOffset = 2.0

	// Create a square shell
	polygons := []model.Polygon{
		{
			Points: []model.Vector3{
				{X: 10, Y: 10}, {X: 20, Y: 10}, {X: 20, Y: 20}, {X: 10, Y: 20},
			},
			IsClosed: true,
			IsHole:   false,
		},
		{
			// Add a hole, should be ignored by raft
			Points: []model.Vector3{
				{X: 12, Y: 12}, {X: 18, Y: 12}, {X: 18, Y: 18}, {X: 12, Y: 18},
			},
			IsClosed: true,
			IsHole:   true,
		},
	}

	paths, zOffset := GenerateRaft(polygons, *config)

	// Since RaftLayers = 2 and FirstLayer = 0.3, zOffset should be 0.6
	assert.Equal(t, 0.6, zOffset)
	assert.Greater(t, len(paths), 0)

	// Layer 0 is Horizontal lines, Layer 1 is Vertical lines
	hasHoriz := false
	hasVert := false

	for _, p := range paths {
		for _, seg := range p.Segments {
			if !seg.IsTravel {
				if seg.Start.Y == seg.End.Y {
					hasHoriz = true
				}
				if seg.Start.X == seg.End.X {
					hasVert = true
				}
			}
		}
	}

	assert.True(t, hasHoriz, "Should have horizontal lines in layer 0")
	assert.True(t, hasVert, "Should have vertical lines in layer 1")
}

func TestGenerateRaft_Disabled(t *testing.T) {
	config := model.NewSliceConfig()
	config.RaftLayers = 0

	polygons := []model.Polygon{{Points: []model.Vector3{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 5, Y: 10}}, IsClosed: true}}
	paths, zOffset := GenerateRaft(polygons, *config)

	assert.Nil(t, paths)
	assert.Equal(t, 0.0, zOffset)
}

func TestGenerateRaft_EmptyPolygons(t *testing.T) {
	config := model.NewSliceConfig()
	config.RaftLayers = 2

	paths, zOffset := GenerateRaft([]model.Polygon{}, *config)

	assert.Nil(t, paths)
	assert.Equal(t, 0.0, zOffset)
}
