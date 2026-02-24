package model

import (
	"math"
)

// BaseModel contains common data and methods for 3D models (STL, 3MF, etc.)
type BaseModel struct {
	Name        string       `json:"name"`
	SliceConfig *SliceConfig `json:"slice_config"`
	Triangles   []Triangle   `json:"triangles"`
	Slices      []*Slice     `json:"slices"`
	Bounds      BoundingBox  `json:"bounds"`
}

// NewBaseModel creates a new empty base model
func NewBaseModel(name string) *BaseModel {
	return &BaseModel{
		Name:      name,
		Triangles: make([]Triangle, 0),
		Bounds:    BoundingBox{},
	}
}

// Copy returns a deep copy of the BaseModel
func (bm *BaseModel) Copy() *BaseModel {
	newBm := &BaseModel{
		Name:        bm.Name,
		SliceConfig: bm.SliceConfig,
		Triangles:   make([]Triangle, len(bm.Triangles)),
		Slices:      make([]*Slice, len(bm.Slices)),
		Bounds:      bm.Bounds,
	}

	copy(newBm.Triangles, bm.Triangles)
	copy(newBm.Slices, bm.Slices)

	return newBm
}

// AddTriangle adds a triangle to the model
func (bm *BaseModel) AddTriangle(t Triangle) {
	bm.Triangles = append(bm.Triangles, t)
}

// GetTriangleCount returns the number of triangles in the model
func (bm *BaseModel) GetTriangleCount() int {
	return len(bm.Triangles)
}

// GetBounds calculates the bounding box of the model
func (bm *BaseModel) GetBounds() BoundingBox {
	if len(bm.Triangles) == 0 {
		return BoundingBox{}
	}

	// Initialize with first vertex
	min := bm.Triangles[0].V1
	max := bm.Triangles[0].V1

	// Check all vertices
	for _, t := range bm.Triangles {
		vertices := []Vector3{t.V1, t.V2, t.V3}
		for _, v := range vertices {
			// Update min
			if v.X < min.X {
				min.X = v.X
			}
			if v.Y < min.Y {
				min.Y = v.Y
			}
			if v.Z < min.Z {
				min.Z = v.Z
			}
			// Update max
			if v.X > max.X {
				max.X = v.X
			}
			if v.Y > max.Y {
				max.Y = v.Y
			}
			if v.Z > max.Z {
				max.Z = v.Z
			}
		}
	}

	return BoundingBox{
		MinX: min.X,
		MinY: min.Y,
		MaxX: max.X,
		MaxY: max.Y,
		MinZ: min.Z,
		MaxZ: max.Z,
	}
}

// GetVolume calculates the volume of the model (assuming it's a closed mesh)
func (bm *BaseModel) GetVolume() float64 {
	var volume float64 = 0.0

	for _, t := range bm.Triangles {
		// Using the divergence theorem
		volume += t.V1.X * (t.V2.Y*t.V3.Z - t.V3.Y*t.V2.Z)
	}

	return math.Abs(volume) / 6.0
}

// GetSurfaceArea calculates the total surface area of the model
func (bm *BaseModel) GetSurfaceArea() float64 {
	var area float64 = 0.0
	for _, t := range bm.Triangles {
		area += t.Area()
	}
	return area
}
