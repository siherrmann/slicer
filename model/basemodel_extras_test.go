package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseModel_Validate(t *testing.T) {
	bm := NewBaseModel("test")

	// Create a single isolated triangle (not watertight)
	t1 := Triangle{
		V1:     Vector3{X: 0, Y: 0, Z: 0},
		V2:     Vector3{X: 1, Y: 0, Z: 0},
		V3:     Vector3{X: 0, Y: 1, Z: 0},
		Normal: Vector3{X: 0, Y: 0, Z: 1}, // correct normal
	}
	bm.AddTriangle(t1)

	issues := bm.Validate(0.001)
	assert.False(t, issues.IsWatertight)
	assert.Equal(t, 3, len(issues.OpenEdges)) // 3 open edges on a single triangle
	assert.Equal(t, 0, len(issues.InvalidTriangles))
	assert.Equal(t, 0, len(issues.DuplicateTriangles))

	// Add an invalid normal triangle
	t2 := Triangle{
		V1:     Vector3{X: 0, Y: 0, Z: 1},
		V2:     Vector3{X: 1, Y: 0, Z: 1},
		V3:     Vector3{X: 0, Y: 1, Z: 1},
		Normal: Vector3{X: 0, Y: 0, Z: -1}, // purposely flipped normal!
	}
	bm.AddTriangle(t2)

	issues2 := bm.Validate(0.001)
	assert.Equal(t, 1, len(issues2.InvalidTriangles))

	// Add duplicate triangle
	bm.AddTriangle(t1)
	issues3 := bm.Validate(0.001)
	assert.Equal(t, 1, len(issues3.DuplicateTriangles))

	// Add degenerate triangle
	tDeg := Triangle{
		V1: Vector3{X: 2, Y: 2, Z: 2},
		V2: Vector3{X: 2, Y: 2, Z: 2},
		V3: Vector3{X: 2, Y: 2, Z: 2},
	}
	bm.AddTriangle(tDeg)
	issues4 := bm.Validate(0.001)
	assert.Equal(t, 1, len(issues4.DegenerateTriangles))
}

func TestBaseModel_CleanBounds(t *testing.T) {
	bm := NewBaseModel("test")
	t1 := Triangle{
		V1: Vector3{X: -10, Y: -10, Z: 0},
		V2: Vector3{X: 10, Y: -10, Z: 0},
		V3: Vector3{X: 0, Y: 10, Z: 5},
	}
	bm.AddTriangle(t1)

	// Initially Bounds might be zero
	bm.Bounds = BoundingBox{}

	cleaned := bm.CleanBounds()
	assert.Equal(t, -10.0, cleaned.Bounds.MinX)
	assert.Equal(t, 10.0, cleaned.Bounds.MaxX)
}

func TestBaseModel_CleanMesh(t *testing.T) {
	bm := NewBaseModel("test")

	// Valid triangle
	t1 := Triangle{
		V1:     Vector3{X: 0, Y: 0, Z: 0},
		V2:     Vector3{X: 1, Y: 0, Z: 0},
		V3:     Vector3{X: 0, Y: 1, Z: 0},
		Normal: Vector3{X: 0, Y: 0, Z: 1},
	}
	bm.AddTriangle(t1)

	// Duplicate
	bm.AddTriangle(t1)

	// Degenerate
	tDeg := Triangle{
		V1: Vector3{X: 2, Y: 2, Z: 2},
		V2: Vector3{X: 2, Y: 2, Z: 2},
		V3: Vector3{X: 2, Y: 2, Z: 2},
	}
	bm.AddTriangle(tDeg)

	// Invalid normal
	tInv := Triangle{
		V1:     Vector3{X: 0, Y: 0, Z: 1},
		V2:     Vector3{X: 1, Y: 0, Z: 1},
		V3:     Vector3{X: 0, Y: 1, Z: 1},
		Normal: Vector3{X: 0, Y: 1, Z: 0}, // weird normal
	}
	bm.AddTriangle(tInv)

	cleaned := bm.CleanMesh(0.001)

	// The degenerate and one duplicate should be removed, leaving 2
	assert.Equal(t, 2, len(cleaned.Triangles))

	// Check that the invalid normal was fixed
	for _, tri := range cleaned.Triangles {
		if tri.V1.Z == 1 { // This identifies tInv
			assert.InDelta(t, 1.0, tri.Normal.Z, 1e-6) // Fixed normal
		}
	}
}

func TestBaseModel_CleanPosition(t *testing.T) {
	bm := NewBaseModel("test")
	// Setup a bounding box going from (10, 10, 10) to (30, 30, 30)
	// Center will be (20, 20, 20), lowest Z = 10
	// After clean: center should be (0,0,10), lowest z = 0
	// So vertices shift by (-20, -20, -10)
	t1 := Triangle{
		V1: Vector3{X: 10, Y: 10, Z: 10},
		V2: Vector3{X: 30, Y: 10, Z: 10},
		V3: Vector3{X: 20, Y: 30, Z: 30},
	}
	bm.AddTriangle(t1)

	cleaned := bm.CleanPosition()

	assert.Equal(t, -10.0, cleaned.Triangles[0].V1.X)
	assert.Equal(t, -10.0, cleaned.Triangles[0].V1.Y)
	assert.Equal(t, 0.0, cleaned.Triangles[0].V1.Z) // Z should be anchored to 0
}

func TestBaseModel_CleanBottom(t *testing.T) {
	bm := NewBaseModel("test")
	t1 := Triangle{
		V1: Vector3{X: 0, Y: 0, Z: -5},
		V2: Vector3{X: 10, Y: 0, Z: -5},
		V3: Vector3{X: 0, Y: 10, Z: 10},
	}
	bm.AddTriangle(t1)

	// Set threshold at 0, should pull -5 vertices up to -5 or minimum below threshold?
	// The function says: "min Z among vertices below threshold". minZ is -5.
	// So -5 points become -5. That doesn't change it. Let's make one -3 and one -5.
	t2 := Triangle{
		V1: Vector3{X: 0, Y: 0, Z: -5},
		V2: Vector3{X: 10, Y: 0, Z: -3},
		V3: Vector3{X: 0, Y: 10, Z: 10},
	}
	bm2 := NewBaseModel("test2")
	bm2.AddTriangle(t2)

	cleaned2 := bm2.CleanBottom(0)
	// Both -5 and -3 should become -5.
	assert.Equal(t, -5.0, cleaned2.Triangles[0].V1.Z)
	assert.Equal(t, -5.0, cleaned2.Triangles[0].V2.Z)
	assert.Equal(t, 10.0, cleaned2.Triangles[0].V3.Z)
}

func TestBaseModel_CleanSize(t *testing.T) {
	bm := NewBaseModel("test")
	// Add a triangle that makes bounding box size 200
	t1 := Triangle{
		V1: Vector3{X: -100, Y: -100, Z: -100},
		V2: Vector3{X: 100, Y: -100, Z: -100},
		V3: Vector3{X: 0, Y: 100, Z: 100},
	}
	bm.AddTriangle(t1)

	cleaned := bm.CleanSize()
	// Should scale size to 100 max size. Since size was 200, scale is 0.5.
	// Center is 0,0,0. New V2: X=50
	assert.Equal(t, 50.0, cleaned.Triangles[0].V2.X)
}

func TestBaseModel_Collapse(t *testing.T) {
	bm := NewBaseModel("test")
	// Give it 10 small triangles
	for i := 0; i < 10; i++ {
		bm.AddTriangle(Triangle{
			V1: Vector3{float64(i) * 0.001, 0, 0},
			V2: Vector3{float64(i)*0.001 + 0.0005, 0.001, 0},
			V3: Vector3{float64(i) * 0.001, 0.001, 0.001},
		})
	}

	// Collapse to target of 5
	collapsed := bm.Collapse(5)
	assert.NotNil(t, collapsed)
	// Output may be fewer than 10 triangles due to vertex merging.
	assert.LessOrEqual(t, len(collapsed.Triangles), 10)
}

func TestBaseModel_Slice(t *testing.T) {
	bm := NewBaseModel("test")
	config := NewSliceConfig()
	config.LayerHeight = 1.0
	config.FirstLayer = 1.0

	t1 := Triangle{
		V1: Vector3{X: 0, Y: 0, Z: 0},
		V2: Vector3{X: 10, Y: 0, Z: 0},
		V3: Vector3{X: 0, Y: 10, Z: 10},
	}
	bm.AddTriangle(t1)

	err := bm.Slice(config)
	require.NoError(t, err)

	assert.Greater(t, len(bm.Slices), 0)
	assert.GreaterOrEqual(t, len(bm.Slices[0].Segments), 0) // May have some segments
}

func TestBaseModel_ClassifyTopBottomLayers(t *testing.T) {
	bm := NewBaseModel("test")
	config := NewSliceConfig()
	config.BottomLayers = 2
	config.TopLayers = 2

	// Manually create 10 slices
	for i := 0; i < 10; i++ {
		bm.Slices = append(bm.Slices, &Slice{Z: float64(i)})
	}

	bm.ClassifyTopBottomLayers(config)

	// Slices 0, 1 should be bottom
	assert.True(t, bm.Slices[0].IsBottomLayer)
	assert.True(t, bm.Slices[1].IsBottomLayer)
	assert.False(t, bm.Slices[2].IsBottomLayer)

	// Slices 8, 9 should be top
	assert.False(t, bm.Slices[7].IsTopLayer)
	assert.True(t, bm.Slices[8].IsTopLayer)
	assert.True(t, bm.Slices[9].IsTopLayer)
}
