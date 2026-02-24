package core

import (
	"fmt"
	"testing"

	"github.com/siherrmann/slicer/model"
	"github.com/stretchr/testify/assert"
)

func createTestBounds() model.BoundingBox {
	return model.BoundingBox{MinX: 0, MinY: 0, MaxX: 10, MaxY: 10, MinZ: 0, MaxZ: 1}
}

func createTestConfig() model.SliceConfig {
	config := model.NewSliceConfig()
	config.LineWidth = 0.4
	config.InfillDensity = 0.2 // Spacing = 0.4 / 0.2 = 2.0
	// For solid layers
	config.InfillOverlap = 0.5 // Spacing = 0.4 * 0.5 = 0.2
	return *config
}

func TestGenerateInfill_Dispatch(t *testing.T) {
	bounds := createTestBounds()
	config := createTestConfig()
	shell := &model.Polygon{
		Points: []model.Vector3{
			{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10},
		},
		IsClosed: true,
	}

	types := []model.InfillType{
		model.InfillGrid,
		model.InfillTriHexagon,
		model.InfillCross,
		model.InfillHoneycombContinuous,
		model.InfillGyroid,
		model.InfillLineFull,
		model.InfillRectilinearFull,
		model.InfillConcentricFull,
		model.InfillLine,
	}

	for _, infType := range types {
		t.Run(fmt.Sprintf("Type%d", infType), func(t *testing.T) {
			config.InfillType = infType
			path := GenerateInfill(bounds, shell, config, 0, 0.2)
			assert.Equal(t, model.PathExtrusion, path.PathType)
			assert.Greater(t, len(path.Segments), 0, "Should generate paths for %v", infType)
		})
	}
}

func TestGenerateLineInfill(t *testing.T) {
	bounds := createTestBounds()
	config := createTestConfig()

	// Test layer 0 (no offset)
	path0 := GenerateLineInfill(bounds, config, 0, 0.2)
	assert.Greater(t, len(path0.Segments), 0)

	// Test layer 1 (offset applied)
	path1 := GenerateLineInfill(bounds, config, 1, 0.4)
	assert.Greater(t, len(path1.Segments), 0)

	// Start points should differ due to offset
	assert.NotEqual(t, path0.Segments[0].Start, path1.Segments[0].Start)
}

func TestGenerateGridInfill(t *testing.T) {
	bounds := createTestBounds()
	config := createTestConfig()
	path := GenerateGridInfill(bounds, config, 0, 0.2)

	assert.Greater(t, len(path.Segments), 0)

	// Ensure some lines are horizontal (Y constant) and some vertical (X constant)
	hasHoriz := false
	hasVert := false
	for _, seg := range path.Segments {
		if seg.Start.Y == seg.End.Y {
			hasHoriz = true
		}
		if seg.Start.X == seg.End.X {
			hasVert = true
		}
	}
	assert.True(t, hasHoriz)
	assert.True(t, hasVert)
}

func TestGenerateGyroidInfill(t *testing.T) {
	bounds := createTestBounds()
	config := createTestConfig()

	// Should produce curvy waves
	path := GenerateGyroidInfill(bounds, config, 0, 0.2)
	assert.Greater(t, len(path.Segments), 0)
}

func TestClamp(t *testing.T) {
	assert.Equal(t, 0.5, clamp(0.5, 0, 1))
	assert.Equal(t, 0.0, clamp(-0.5, 0, 1))
	assert.Equal(t, 1.0, clamp(1.5, 0, 1))
}

func TestGenerateConcentricInfillFull(t *testing.T) {
	config := createTestConfig()
	shell := &model.Polygon{
		Points: []model.Vector3{
			{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10},
		},
		IsClosed: true,
	}

	path := GenerateConcentricInfillFull(shell, config, 0)
	assert.Greater(t, len(path.Segments), 0)
}

func TestGenerateRectilinearInfillFull(t *testing.T) {
	bounds := createTestBounds()
	config := createTestConfig()

	// Layer 0 (horizontal)
	path0 := GenerateRectilinearInfillFull(bounds, config, 0, 0.2)
	assert.Greater(t, len(path0.Segments), 0)

	// Layer 1 (vertical)
	path1 := GenerateRectilinearInfillFull(bounds, config, 1, 0.4)
	assert.Greater(t, len(path1.Segments), 0)
}

func TestGyroidMath(t *testing.T) {
	// Directly test some of the gyroid helper functions to ensure no panics and expected logic.
	// We check gyroidF for vertical=true and false
	y1 := gyroidF(0, 0, 1, true, false)
	assert.NotZero(t, y1)

	y2 := gyroidF(0, 1, 0, false, false)
	assert.NotZero(t, y2)

	pts := gyroidMakeOnePeriod(10, 1, 0, true, false, 0.1)
	assert.Greater(t, len(pts), 2)
}
