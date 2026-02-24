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
