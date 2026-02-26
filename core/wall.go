package core

import (
	"math"

	"github.com/siherrmann/slicer/model"
)

// CalaculateWallOffset calculates the offset distance for a given wall index based on line width and overlap.
func CalculateWallOffset(index int, params model.SliceConfig) float64 {
	if index <= 0 {
		return params.LineWidth * 0.5
	} else {
		return (params.LineWidth * 0.5) + (params.LineWidth * params.InfillOverlap)
	}
}

// CalculateWallCount calculates how many wall lines are needed.
// It prioritizes ShellCount if it's greater than 0, otherwise it uses WallThickness.
func CalculateWallCount(params model.SliceConfig) int {
	if params.ShellCount > 0 {
		return params.ShellCount
	}
	return int(math.Max(1, math.Ceil(params.WallThickness/params.LineWidth)))
}

// CalculateInfillOffset calculates the offset distance for the infill area based on the number of walls and line width.
func CalculateInfillOffset(params model.SliceConfig) float64 {
	return CalculateWallOffset(CalculateWallCount(params), params)
}

// GenerateWalls generates the wall paths for a given shell polygon based on the slicing configuration.
func GenerateWalls(shell model.Polygon, params model.SliceConfig, layerIndex int) []model.ContinuousPath {
	wallCount := CalculateWallCount(params)
	wallPaths := make([]model.ContinuousPath, 0, wallCount)

	for i := range wallCount {
		offsetDist := CalculateWallOffset(i, params)
		wall := shell.OffsetPolygon(-offsetDist)
		if wall != nil && len(wall.Points) > 2 {
			// Determine if this is the outermost wall
			isOuterWall := (i == 0)

			// Set speed based on wall position
			speed := params.WallSpeed
			category := model.CategoryInnerWall
			if isOuterWall {
				speed = params.OuterShellSpeed
				category = model.CategoryOuterWall
			}

			// Create a continuous path for this wall (closed loop)
			wall.IsClosed = true // Offset polygons are inherently closed
			wallPaths = append(wallPaths, wall.ToContinuousPath(speed, category, layerIndex))
		}
	}

	// Apply shell order: reverse if outside-in (outer wall first)
	if params.ShellOrder == model.ShellOutsideIn && len(wallPaths) > 1 {
		for i, j := 0, len(wallPaths)-1; i < j; i, j = i+1, j-1 {
			wallPaths[i], wallPaths[j] = wallPaths[j], wallPaths[i]
		}
	}

	return wallPaths
}
