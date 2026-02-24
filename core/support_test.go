package core

import (
	"testing"

	"github.com/siherrmann/slicer/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSupportPaths(t *testing.T) {
	t.Run("CShapeOverhang", func(t *testing.T) {
		// Create a C-Shape model
		// Base: Z=0 to 9, X=[10, 90], Y=[10, 50]
		// Pillar: Z=10 to 19, X=[10, 30], Y=[10, 50]
		// Roof (Overhang): Z=20 to 29, X=[10, 90], Y=[10, 50]
		//
		// The overhang is at X=[30, 90] starting at Z=20.
		// It hovers directly over the base at X=[30, 90] at Z=9.
		// Tree supports should spawn at Z=19 and drop down to Z=10.

		bm := model.NewBaseModel("CShape")

		// Add dummy triangle to establish bounding box: Min(10,10,0) to Max(90,50,29)
		bm.AddTriangle(model.Triangle{
			V1: model.Vector3{X: 10, Y: 10, Z: 0},
			V2: model.Vector3{X: 90, Y: 50, Z: 29},
			V3: model.Vector3{X: 10, Y: 10, Z: 0},
		})

		for z := 0; z < 30; z++ {
			maxX := 90.0
			if z >= 10 && z < 20 {
				maxX = 30.0
			}

			poly := model.Polygon{
				Points: []model.Vector3{
					{X: 10, Y: 10, Z: float64(z)},
					{X: maxX, Y: 10, Z: float64(z)},
					{X: maxX, Y: 50, Z: float64(z)},
					{X: 10, Y: 50, Z: float64(z)},
				},
				IsClosed: true,
			}

			slice := &model.Slice{
				Z:          float64(z),
				LayerIndex: z,
				Polygons:   []model.Polygon{poly},
			}
			bm.Slices = append(bm.Slices, slice)
		}

		config := model.NewSliceConfig()
		config.LayerHeight = 1.0
		config.LineWidth = 0.4 // Realistic 0.4mm nozzle equivalent
		config.SupportType = model.SupportTree
		config.SupportPlacement = model.SupportEverywhere
		config.SupportAngle = 45.0
		config.SupportTreeBranchDiameter = 2.0
		config.SupportTreeTrunkDiameter = 12.0
		config.SupportXYGap = 0.5
		config.SupportZGap = 0.0

		// Execution
		paths := GenerateSupportPaths(bm, *config)

		// Assertions
		require.NotNil(t, paths, "Paths should not be nil")

		// We expect support to be generated under the overhang (Z=20).
		// Since the overhang goes up to X=90, branches should exist far to the right.
		foundFarRightAtLayer19 := false
		foundSupportOnBaseAtLayer11 := false

		// Restore assertions and debug logging
		for z := 0; z < 30; z++ {
			for _, path := range paths[z] {
				for _, seg := range path.Segments {
					if z == 19 {
						if seg.Start.X > 80.0 || seg.End.X > 80.0 {
							foundFarRightAtLayer19 = true
						}
					}
					// Under the new "Top-Down Z-Gap Collision" logic, branches directly over
					// a flat floor will plant firmly onto it (Layer 10/11) without sliding sideways!
					if z == 11 {
						if (seg.Start.X > 40.0 && seg.Start.X < 85.0) || (seg.End.X > 40.0 && seg.End.X < 85.0) {
							foundSupportOnBaseAtLayer11 = true
						}
					}
				}
			}
		}

		assert.True(t, foundFarRightAtLayer19, "Expected tree support branches to spawn far right (X > 80) just under the C-shape overhang at layer 19")
		assert.True(t, foundSupportOnBaseAtLayer11, "Expected tree support branches to plant firmly onto the base (X=40..85) at layer 11 without sliding off")
	})
}

func TestSupportMorphology(t *testing.T) {
	bounds := model.BoundingBox{MinX: 0, MinY: 0, MaxX: 10, MaxY: 10}
	grid := NewGrid(bounds, 1.0)

	// Set a 3x3 block
	for x := 2; x <= 4; x++ {
		for y := 2; y <= 4; y++ {
			grid.Set(x, y, true)
		}
	}

	// Add a single noise pixel
	grid.Set(8, 8, true)

	// Create another grid that just has a 1-pixel gap
	grid2 := NewGrid(bounds, 1.0)
	grid2.Set(2, 2, true)
	grid2.Set(2, 4, true)

	t.Run("Open", func(t *testing.T) {
		// Open removes noise
		opened := grid.Open(1)

		// The 3x3 block should remain (mostly)
		assert.True(t, opened.Get(3, 3))
		// The noise pixel should be gone
		assert.False(t, opened.Get(8, 8))
	})

	t.Run("Close", func(t *testing.T) {
		// Close fills gaps
		closed := grid2.Close(1)

		// The original pixels are still there
		assert.True(t, closed.Get(2, 2))
		assert.True(t, closed.Get(2, 4))

		// 1D gap gets eroded back because dilate is circular
		assert.False(t, closed.Get(2, 3))
	})

	t.Run("SetDisk", func(t *testing.T) {
		g3 := NewGrid(bounds, 1.0)
		g3.SetDisk(5, 5, 2)
		// Center is set
		assert.True(t, g3.Get(5, 5))
		assert.True(t, g3.Get(5, 7))  // radius 2
		assert.False(t, g3.Get(5, 8)) // outside radius
	})
}

func TestSupportTracing(t *testing.T) {
	bounds := model.BoundingBox{MinX: 0, MinY: 0, MaxX: 10, MaxY: 10}
	grid := NewGrid(bounds, 1.0)

	// Create a hollow square block 5x5
	for x := 2; x <= 6; x++ {
		grid.Set(x, 2, true)
		grid.Set(x, 6, true)
	}
	for y := 2; y <= 6; y++ {
		grid.Set(2, y, true)
		grid.Set(6, y, true)
	}

	paths := grid.TraceContours()

	// Should have at least the outer boundary
	assert.GreaterOrEqual(t, len(paths), 1)

	foundPoints := len(paths[0].Points)
	assert.Greater(t, foundPoints, 4) // should trace the square

	// Test simplification
	simplified := simplifyPath(paths[0].Points)

	// A square has only 5 vertices (including the closed connection),
	// so simplification shouldn't increase it.
	assert.LessOrEqual(t, len(simplified), foundPoints)
	assert.Greater(t, len(simplified), 3)
}
