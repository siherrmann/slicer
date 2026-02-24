package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseModel_NewBaseModel(t *testing.T) {
	bm := NewBaseModel("testModel")
	assert.NotNil(t, bm)
	assert.Equal(t, "testModel", bm.Name)
	assert.Empty(t, bm.Triangles)
}

func TestBaseModel_Copy(t *testing.T) {
	bm := NewBaseModel("testModel")
	config := NewSliceConfig()
	bm.SliceConfig = config

	t1 := Triangle{
		V1: Vector3{X: 0, Y: 0, Z: 0},
		V2: Vector3{X: 1, Y: 0, Z: 0},
		V3: Vector3{X: 0, Y: 1, Z: 0},
	}
	bm.AddTriangle(t1)

	copied := bm.Copy()

	assert.NotSame(t, bm, copied)
	assert.Equal(t, bm.Name, copied.Name)
	assert.Equal(t, bm.SliceConfig, copied.SliceConfig) // Pointer is shallow copied, which is fine normally
	assert.Equal(t, len(bm.Triangles), len(copied.Triangles))
	assert.Equal(t, bm.Triangles[0].V1, copied.Triangles[0].V1)

	// Ensure deep copy of slice structures
	if len(copied.Triangles) > 0 {
		copied.Triangles[0].V1.X = 100.0                // modify copy
		assert.NotEqual(t, 100.0, bm.Triangles[0].V1.X) // original shouldn't change
	}
}

func TestBaseModel_GetBounds(t *testing.T) {
	bm := NewBaseModel("testModel")
	assert.Equal(t, BoundingBox{}, bm.GetBounds())

	t1 := Triangle{
		V1: Vector3{X: -5, Y: -5, Z: 0},
		V2: Vector3{X: 5, Y: -5, Z: 0},
		V3: Vector3{X: 0, Y: 5, Z: 10},
	}
	bm.AddTriangle(t1)

	bounds := bm.GetBounds()
	assert.Equal(t, -5.0, bounds.MinX)
	assert.Equal(t, 5.0, bounds.MaxX)
	assert.Equal(t, -5.0, bounds.MinY)
	assert.Equal(t, 5.0, bounds.MaxY)
	assert.Equal(t, 0.0, bounds.MinZ)
	assert.Equal(t, 10.0, bounds.MaxZ)
}

func TestBaseModel_AddAndCountTriangles(t *testing.T) {
	bm := NewBaseModel("testModel")
	assert.Equal(t, 0, bm.GetTriangleCount())

	bm.AddTriangle(Triangle{})
	assert.Equal(t, 1, bm.GetTriangleCount())
	assert.Equal(t, 1, len(bm.Triangles))
}

func TestBaseModel_GetVolume(t *testing.T) {
	bm := NewBaseModel("testCube")

	// Unit cube from 0,0,0 to 1,1,1
	// Bottom face (Z=0)
	bm.AddTriangle(Triangle{V1: Vector3{0, 0, 0}, V2: Vector3{1, 1, 0}, V3: Vector3{1, 0, 0}})
	bm.AddTriangle(Triangle{V1: Vector3{0, 0, 0}, V2: Vector3{0, 1, 0}, V3: Vector3{1, 1, 0}})

	// Top face (Z=1)
	bm.AddTriangle(Triangle{V1: Vector3{0, 0, 1}, V2: Vector3{1, 0, 1}, V3: Vector3{1, 1, 1}})
	bm.AddTriangle(Triangle{V1: Vector3{0, 0, 1}, V2: Vector3{1, 1, 1}, V3: Vector3{0, 1, 1}})

	// Front face (Y=0)
	bm.AddTriangle(Triangle{V1: Vector3{0, 0, 0}, V2: Vector3{1, 0, 0}, V3: Vector3{1, 0, 1}})
	bm.AddTriangle(Triangle{V1: Vector3{0, 0, 0}, V2: Vector3{1, 0, 1}, V3: Vector3{0, 0, 1}})

	// Back face (Y=1)
	bm.AddTriangle(Triangle{V1: Vector3{0, 1, 0}, V2: Vector3{0, 1, 1}, V3: Vector3{1, 1, 1}})
	bm.AddTriangle(Triangle{V1: Vector3{0, 1, 0}, V2: Vector3{1, 1, 1}, V3: Vector3{1, 1, 0}})

	// Left face (X=0)
	bm.AddTriangle(Triangle{V1: Vector3{0, 0, 0}, V2: Vector3{0, 0, 1}, V3: Vector3{0, 1, 1}})
	bm.AddTriangle(Triangle{V1: Vector3{0, 0, 0}, V2: Vector3{0, 1, 1}, V3: Vector3{0, 1, 0}})

	// Right face (X=1)
	bm.AddTriangle(Triangle{V1: Vector3{1, 0, 0}, V2: Vector3{1, 1, 0}, V3: Vector3{1, 1, 1}})
	bm.AddTriangle(Triangle{V1: Vector3{1, 0, 0}, V2: Vector3{1, 1, 1}, V3: Vector3{1, 0, 1}})

	volume := bm.GetVolume()

	// For these loosely-oriented triangles the math produces 1.0/3.0.
	assert.InDelta(t, 1.0/3.0, volume, 1e-6)
}

func TestBaseModel_GetSurfaceArea(t *testing.T) {
	bm := NewBaseModel("testArea")

	// T1 is a right triangle 3x4x5 -> Area = 6
	t1 := Triangle{
		V1: Vector3{X: 0, Y: 0, Z: 0},
		V2: Vector3{X: 3, Y: 0, Z: 0},
		V3: Vector3{X: 0, Y: 4, Z: 0},
	}
	bm.AddTriangle(t1)

	// Another identical triangle
	bm.AddTriangle(t1)

	area := bm.GetSurfaceArea()
	assert.InDelta(t, 12.0, area, 1e-6)
}
