package model

import (
	"log"
	"math"
)

// Collapse uses voxel-based vertex clustering for fast mesh decimation
func (bm *BaseModel) Collapse(targetTriangleCount int) *BaseModel {
	if targetTriangleCount >= len(bm.Triangles) {
		lowRes := NewBaseModel(bm.Name + "_region")
		lowRes.Triangles = make([]Triangle, len(bm.Triangles))
		copy(lowRes.Triangles, bm.Triangles)
		return lowRes.CleanSize()
	}

	// Step 1: Calculate bounding box
	if len(bm.Triangles) == 0 {
		return bm
	}

	bounds := bm.GetBounds()
	min := Vector3{X: bounds.MinX, Y: bounds.MinY, Z: bounds.MinZ}
	max := Vector3{X: bounds.MaxX, Y: bounds.MaxY, Z: bounds.MaxZ}

	// Step 2: Calculate grid resolution
	// We want roughly the right number of voxels to achieve the target reduction
	// More triangles needed = finer grid (more voxels)
	// Estimate: each voxel contains roughly (total triangles / target triangles) vertices
	// So we need roughly targetTriangles^(1/3) voxels per dimension
	voxelsPerDim := int(math.Pow(float64(targetTriangleCount), 1.0/3.0) * 1.5)
	if voxelsPerDim < 5 {
		voxelsPerDim = 5
	} else if voxelsPerDim > 300 {
		voxelsPerDim = 300
	}

	// Step 3: Calculate voxel size
	size := Vector3{
		X: (max.X - min.X) / float64(voxelsPerDim),
		Y: (max.Y - min.Y) / float64(voxelsPerDim),
		Z: (max.Z - min.Z) / float64(voxelsPerDim),
	}

	// Add small epsilon to avoid edge cases
	size.X += 0.0001
	size.Y += 0.0001
	size.Z += 0.0001

	// Step 4: Build voxel grid - map from voxel coordinate to merged vertex
	type VoxelKey struct{ x, y, z int }
	voxelVertices := make(map[VoxelKey]Vector3)
	voxelCounts := make(map[VoxelKey]int)

	getVoxelKey := func(v Vector3) VoxelKey {
		return VoxelKey{
			x: int((v.X - min.X) / size.X),
			y: int((v.Y - min.Y) / size.Y),
			z: int((v.Z - min.Z) / size.Z),
		}
	}

	// Accumulate all vertices in each voxel
	for _, tri := range bm.Triangles {
		for _, v := range []Vector3{tri.V1, tri.V2, tri.V3} {
			key := getVoxelKey(v)
			current := voxelVertices[key]
			current.X += v.X
			current.Y += v.Y
			current.Z += v.Z
			voxelVertices[key] = current
			voxelCounts[key]++
		}
	}

	// Average vertices in each voxel
	for key, sum := range voxelVertices {
		count := float64(voxelCounts[key])
		voxelVertices[key] = Vector3{
			X: sum.X / count,
			Y: sum.Y / count,
			Z: sum.Z / count,
		}
	}

	log.Printf("Created %d voxels with merged vertices", len(voxelVertices))

	// Step 5: Rebuild triangles with merged vertices
	var result []Triangle
	for _, tri := range bm.Triangles {
		// Map each vertex to its voxel's merged vertex
		v1 := voxelVertices[getVoxelKey(tri.V1)]
		v2 := voxelVertices[getVoxelKey(tri.V2)]
		v3 := voxelVertices[getVoxelKey(tri.V3)]

		// Skip degenerate triangles (where all vertices collapsed to same point or line)
		if v1 == v2 || v2 == v3 || v1 == v3 {
			continue
		}

		// Calculate new normal
		edge1 := Vector3{X: v2.X - v1.X, Y: v2.Y - v1.Y, Z: v2.Z - v1.Z}
		edge2 := Vector3{X: v3.X - v1.X, Y: v3.Y - v1.Y, Z: v3.Z - v1.Z}
		normal := Vector3{
			X: edge1.Y*edge2.Z - edge1.Z*edge2.Y,
			Y: edge1.Z*edge2.X - edge1.X*edge2.Z,
			Z: edge1.X*edge2.Y - edge1.Y*edge2.X,
		}

		// Skip if normal is zero (degenerate triangle)
		length := math.Sqrt(float64(normal.X*normal.X + normal.Y*normal.Y + normal.Z*normal.Z))
		if length < 0.0001 {
			continue
		}

		result = append(result, Triangle{
			Normal: normal.Normalize(),
			V1:     v1,
			V2:     v2,
			V3:     v3,
		})
	}

	log.Printf("RegionDecimate: Result has %d triangles (removed %d degenerate)", len(result), len(bm.Triangles)-len(result))

	// Build result BaseModel
	lowRes := NewBaseModel(bm.Name + "_voxel")
	lowRes.Triangles = result

	return lowRes.CleanSize()
}
