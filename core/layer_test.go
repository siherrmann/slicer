package core

import (
	"testing"

	"github.com/siherrmann/slicer/model"
	"github.com/stretchr/testify/assert"
)

func TestGenerateLayerPath_VaseMode(t *testing.T) {
	config := model.NewSliceConfig()
	config.VaseMode = true
	config.LayerHeight = 0.2
	config.LineWidth = 0.4

	polygons := []model.Polygon{
		{
			Points: []model.Vector3{
				{X: 0, Y: 0, Z: 0.2}, {X: 10, Y: 0, Z: 0.2}, {X: 10, Y: 10, Z: 0.2}, {X: 0, Y: 10, Z: 0.2},
			},
			IsClosed: true,
			IsHole:   false,
		},
	}
	bounds := model.BoundingBox{MinX: 0, MinY: 0, MaxX: 10, MaxY: 10, MinZ: 0, MaxZ: 10}

	// Layer 0 - standard generation for base
	pathsL0 := GenerateLayerPath(polygons, nil, nil, *config, bounds, 0)
	assert.Greater(t, len(pathsL0), 0)

	// Layer 1 - Vase mode interpolation
	pathsL1 := GenerateLayerPath(polygons, nil, nil, *config, bounds, 1)
	assert.Greater(t, len(pathsL1), 0)

	// In vase mode, Z should strictly increase across the segment endpoints
	// Let's verify that
	foundZIncrease := false
	for _, p := range pathsL1 {
		for _, seg := range p.Segments {
			if seg.End.Z > seg.Start.Z {
				foundZIncrease = true
				break
			}
		}
	}
	assert.True(t, foundZIncrease, "Vase mode should have Z-interpolated moves")
}

func TestGenerateLayerPath_Standard(t *testing.T) {
	config := model.NewSliceConfig()
	config.VaseMode = false
	config.InfillDensity = 0.2
	config.ShellCount = 2
	config.TopLayers = 0
	config.BottomLayers = 0

	polygons := []model.Polygon{
		// Shell
		{
			Points: []model.Vector3{
				{X: 0, Y: 0, Z: 0.2}, {X: 20, Y: 0, Z: 0.2}, {X: 20, Y: 20, Z: 0.2}, {X: 0, Y: 20, Z: 0.2},
			},
			IsClosed: true,
			IsHole:   false,
		},
		// Hole
		{
			Points: []model.Vector3{
				{X: 5, Y: 5, Z: 0.2}, {X: 15, Y: 5, Z: 0.2}, {X: 15, Y: 15, Z: 0.2}, {X: 5, Y: 15, Z: 0.2},
			},
			IsClosed: true,
			IsHole:   true,
		},
	}
	bounds := model.BoundingBox{MinX: 0, MinY: 0, MaxX: 20, MaxY: 20, MinZ: 0, MaxZ: 10}

	paths := GenerateLayerPath(polygons, nil, nil, *config, bounds, 1)

	assert.Greater(t, len(paths), 0)

	hasInfill := false
	hasWalls := false
	for _, p := range paths {
		for _, seg := range p.Segments {
			if seg.Category == model.CategoryInfill {
				hasInfill = true
			}
			if seg.Category == model.CategoryInnerWall || seg.Category == model.CategoryOuterWall {
				hasWalls = true
			}
		}
	}

	assert.True(t, hasInfill)
	assert.True(t, hasWalls)
}

func TestCutInfill(t *testing.T) {
	// A simple line going straight across
	pattern := model.ContinuousPath{
		Segments: []model.PathSegment{
			{Start: model.Vector3{X: -5, Y: 10}, End: model.Vector3{X: 25, Y: 10}}, // Across from X=-5 to 25
		},
		PathType: model.PathExtrusion,
	}

	shell := &model.Polygon{
		Points: []model.Vector3{
			{X: 0, Y: 0}, {X: 20, Y: 0}, {X: 20, Y: 20}, {X: 0, Y: 20},
		},
		IsClosed: true,
	}

	holes := []model.Polygon{
		{
			Points: []model.Vector3{
				{X: 5, Y: 5}, {X: 15, Y: 5}, {X: 15, Y: 15}, {X: 5, Y: 15},
			},
			IsClosed: true,
		},
	}

	cutPaths := cutInfill(pattern, shell, holes)

	// Expectations:
	// Line is y=10, x from -5 to 25.
	// Shell limits to x from 0 to 20.
	// Hole limits to exclude x from 5 to 15.
	// So we expect segments: (0,10) -> (5,10) and (15,10) -> (20,10)

	assert.Greater(t, len(cutPaths), 0)

	var allExtrusionSegs []model.PathSegment
	for _, p := range cutPaths {
		for _, s := range p.Segments {
			if !s.IsTravel {
				allExtrusionSegs = append(allExtrusionSegs, s)
			}
		}
	}

	assert.Equal(t, 2, len(allExtrusionSegs))
	// segment 1
	assert.InDelta(t, 0.0, allExtrusionSegs[0].Start.X, 1e-4)
	assert.InDelta(t, 5.0, allExtrusionSegs[0].End.X, 1e-4)
	// segment 2
	assert.InDelta(t, 15.0, allExtrusionSegs[1].Start.X, 1e-4)
	assert.InDelta(t, 20.0, allExtrusionSegs[1].End.X, 1e-4)
}

func TestCutInfill_OutsideSegment(t *testing.T) {
	pattern := model.ContinuousPath{
		Segments: []model.PathSegment{
			{Start: model.Vector3{X: -5, Y: -5}, End: model.Vector3{X: -1, Y: -1}},
		},
	}

	shell := &model.Polygon{
		Points: []model.Vector3{
			{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10},
		},
		IsClosed: true,
	}

	cutPaths := cutInfill(pattern, shell, nil)
	assert.Nil(t, cutPaths)
}

func TestCutInfill_FullyInsideSegment(t *testing.T) {
	pattern := model.ContinuousPath{
		Segments: []model.PathSegment{
			{Start: model.Vector3{X: 2, Y: 2}, End: model.Vector3{X: 8, Y: 8}},
		},
	}

	shell := &model.Polygon{
		Points: []model.Vector3{
			{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10},
		},
		IsClosed: true,
	}

	cutPaths := cutInfill(pattern, shell, nil)
	assert.NotNil(t, cutPaths)
	assert.Equal(t, 1, len(cutPaths[0].Segments))
	assert.Equal(t, 2.0, cutPaths[0].Segments[0].Start.X)
	assert.Equal(t, 8.0, cutPaths[0].Segments[0].End.X)
}
