package core

import (
	"testing"

	"github.com/siherrmann/slicer/model"
	"github.com/stretchr/testify/assert"
)

func TestCalculateWallCount(t *testing.T) {
	config := model.NewSliceConfig()
	config.LineWidth = 0.4

	// Test ShellCount priority
	config.ShellCount = 3
	config.WallThickness = 0.8
	assert.Equal(t, 3, CalculateWallCount(*config))

	// Test WallThickness calculation
	config.ShellCount = 0
	config.WallThickness = 1.2
	assert.Equal(t, 3, CalculateWallCount(*config)) // 1.2 / 0.4 = 3

	config.WallThickness = 1.0
	assert.Equal(t, 3, CalculateWallCount(*config)) // ceil(1.0 / 0.4) = ceil(2.5) = 3

	// Math.Max(1, ...)
	config.WallThickness = 0.0
	assert.Equal(t, 1, CalculateWallCount(*config))
}

func TestCalculateWallOffset(t *testing.T) {
	config := model.NewSliceConfig()
	config.LineWidth = 0.4
	config.InfillOverlap = 0.2 // 20%

	// Index 0: LineWidth * 0.5
	assert.InDelta(t, 0.2, CalculateWallOffset(0, *config), 1e-4)

	// Index > 0: LineWidth * 0.5 + LineWidth * InfillOverlap
	// 0.2 + 0.4 * 0.2 = 0.2 + 0.08 = 0.28
	assert.InDelta(t, 0.28, CalculateWallOffset(1, *config), 1e-4)
	assert.InDelta(t, 0.28, CalculateWallOffset(2, *config), 1e-4)
}

func TestCalculateInfillOffset(t *testing.T) {
	config := model.NewSliceConfig()
	config.LineWidth = 0.4
	config.InfillOverlap = 0.2
	config.ShellCount = 2

	// CalculateInfillOffset calls CalculateWallOffset with WallCount
	// So i = 2, which gives offset 0.28
	assert.InDelta(t, 0.28, CalculateInfillOffset(*config), 1e-4)
}

func TestGenerateWalls(t *testing.T) {
	config := model.NewSliceConfig()
	config.LineWidth = 0.4
	config.OuterShellSpeed = 30.0
	config.WallSpeed = 50.0
	config.ShellCount = 2
	config.ShellOrder = model.ShellInsideOut

	shell := model.Polygon{
		Points: []model.Vector3{
			{X: 10, Y: 10, Z: 0.2}, {X: 20, Y: 10, Z: 0.2}, {X: 20, Y: 20, Z: 0.2}, {X: 10, Y: 20, Z: 0.2},
		},
		IsClosed: true,
		IsHole:   false,
	}

	walls := GenerateWalls(shell, *config, 0)

	// Since ShellCount is 2, we expect 2 paths (outer wall and inner wall)
	assert.Equal(t, 2, len(walls))

	// By default ShellInsideOut -> order should be index 0, then 1
	// Wait, index 0 is outer wall, index 1 is inner wall?
	// The function loop goes from 0 to wallCount-1.
	// Index 0 is outer wall (CategoryOuterWall, Speed: OuterShellSpeed)
	// Index 1 is inner wall
	// So wallPaths will have [OuterWall, InnerWall] directly

	// Verify speeds and categories
	foundOuter := false
	foundInner := false
	for _, p := range walls {
		for _, seg := range p.Segments {
			if seg.Category == model.CategoryOuterWall {
				assert.Equal(t, 30.0, seg.Speed)
				foundOuter = true
			} else if seg.Category == model.CategoryInnerWall {
				assert.Equal(t, 50.0, seg.Speed)
				foundInner = true
			}
		}
	}
	assert.True(t, foundOuter)
	assert.True(t, foundInner)

	// Now test ShellOutsideIn
	config.ShellOrder = model.ShellOutsideIn
	wallsReversed := GenerateWalls(shell, *config, 0)
	assert.Equal(t, 2, len(wallsReversed))

	// Reverse order means it should be [InnerWall, OuterWall]
	// First path should be inner wall
	assert.Equal(t, model.CategoryInnerWall, wallsReversed[0].Segments[0].Category)
	// Second path should be outer wall
	assert.Equal(t, model.CategoryOuterWall, wallsReversed[1].Segments[0].Category)
}
