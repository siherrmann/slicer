package model

import "math"

// MeshIssues contains information about mesh problems
type MeshIssues struct {
	IsWatertight        bool                // True if mesh is watertight
	InvalidTriangles    []Triangle          // Triangles with invalid normals
	OpenEdges           []LineSegment       // Edges that are not shared by exactly 2 triangles
	DuplicateTriangles  []Triangle          // Indices of duplicate triangles
	DegenerateTriangles []Triangle          // Indices of triangles with zero area
	EdgeUsageCount      map[string]int      // How many times each edge is used
	BoundaryEdges       []LineSegment       // Edges used only once (mesh boundary/hole)
	NonManifoldEdges    []LineSegment       // Edges used more than twice (non-manifold)
	EdgeUsageCounts     map[LineSegment]int // Maps non-manifold edges to their usage count
}

// Validate identifies issues in the mesh such as open edges,
// duplicate triangles, and degenerate triangles.
func (bm *BaseModel) Validate(tolerance float64) *MeshIssues {
	issues := &MeshIssues{
		IsWatertight:        true,
		OpenEdges:           make([]LineSegment, 0),
		DuplicateTriangles:  make([]Triangle, 0),
		DegenerateTriangles: make([]Triangle, 0),
		EdgeUsageCount:      make(map[string]int),
		BoundaryEdges:       make([]LineSegment, 0),
		NonManifoldEdges:    make([]LineSegment, 0),
		InvalidTriangles:    make([]Triangle, 0),
		EdgeUsageCounts:     make(map[LineSegment]int),
	}

	if len(bm.Triangles) == 0 {
		issues.IsWatertight = false
		return issues
	}

	// Track edge usage
	edgeCount := make(map[string]int)
	edgeOriginal := make(map[string]LineSegment)

	// Track triangle uniqueness
	triangleSet := make(map[string]bool)

	for _, t := range bm.Triangles {
		// Check if the normal is valid by comparing with computed normal
		// Use dot product to check angle between normals (should be close to 1 for same direction)
		computed := t.ComputeNormal()
		// Normalize both to be sure they're unit vectors
		storedNorm := t.Normal.Normalize()
		computedNorm := computed.Normalize()

		// Dot product of two unit vectors = cos(angle)
		// If angle > 10 degrees, consider it invalid
		// cos(10°) ≈ 0.985
		dotProduct := storedNorm.Dot(computedNorm)
		if dotProduct < 0.985 { // More than 10 degrees difference
			issues.InvalidTriangles = append(issues.InvalidTriangles, t)
		}

		// Check for degenerate triangles
		if t.IsDegenerate(tolerance) {
			issues.DegenerateTriangles = append(issues.DegenerateTriangles, t)
			issues.IsWatertight = false
			continue
		}

		// Check for duplicate triangles
		triangleKey := t.Key(tolerance)
		if triangleSet[triangleKey] {
			issues.DuplicateTriangles = append(issues.DuplicateTriangles, t)
			issues.IsWatertight = false
		}
		triangleSet[triangleKey] = true

		// Count each edge of the triangle
		edges := []LineSegment{
			{t.V1, t.V2},
			{t.V2, t.V3},
			{t.V3, t.V1},
		}
		for _, edge := range edges {
			key := edge.Key(tolerance)
			edgeCount[key]++
			if _, exists := edgeOriginal[key]; !exists {
				edgeOriginal[key] = LineSegment{Start: edge.Start, End: edge.End}
			}
		}
	}

	// Check edge usage - each edge should be used exactly twice in a watertight mesh
	for key, count := range edgeCount {
		issues.EdgeUsageCount[key] = count
		if count != 2 {
			edge := edgeOriginal[key]
			issues.OpenEdges = append(issues.OpenEdges, edge)
			issues.IsWatertight = false

			// Categorize the type of problem
			if count == 1 {
				issues.BoundaryEdges = append(issues.BoundaryEdges, edge)
			} else if count > 2 {
				issues.NonManifoldEdges = append(issues.NonManifoldEdges, edge)
				issues.EdgeUsageCounts[edge] = count
			}
		}
	}

	return issues
}

// CleanBounds recalculates the bounds of the model and updates the struct
func (bm *BaseModel) CleanBounds() *BaseModel {
	newBm := bm.Copy()
	newBm.Bounds = newBm.GetBounds()
	return newBm
}

// CleanMesh repairs common mesh issues:
// - Removes degenerate triangles (zero area)
// - Removes duplicate triangles
// - Fixes invalid normals by recomputing them
func (bm *BaseModel) CleanMesh(tolerance float64) *BaseModel {
	cleaned := bm.Copy()
	cleaned.Triangles = make([]Triangle, 0) // Reset triangles but keep other info

	// Create sets for filtering
	issues := bm.Validate(tolerance)
	degenerateSet := make(map[string]bool)
	for _, t := range issues.DegenerateTriangles {
		degenerateSet[t.Key(tolerance)] = true
	}
	duplicateSet := make(map[string]bool)
	for _, t := range issues.DuplicateTriangles {
		duplicateSet[t.Key(tolerance)] = true
	}
	invalidSet := make(map[string]bool)
	for _, t := range issues.InvalidTriangles {
		invalidSet[t.Key(tolerance)] = true
	}
	addedTriangles := make(map[string]bool)
	for _, t := range bm.Triangles {
		key := t.Key(tolerance)
		if degenerateSet[key] {
			continue
		} else if addedTriangles[key] {
			continue
		}

		cleanedTriangle := t
		if invalidSet[key] {
			cleanedTriangle.Normal = t.ComputeNormal()
		}

		cleaned.Triangles = append(cleaned.Triangles, cleanedTriangle)
		addedTriangles[key] = true
	}

	return cleaned
}

// CleanPosition recenters the model to the origin and moves it so the lowest point is at Z=0
func (bm *BaseModel) CleanPosition() *BaseModel {
	// Find center and bottom of the model
	bounds := bm.GetBounds()
	centerX := (bounds.MinX + bounds.MaxX) / 2
	centerY := (bounds.MinY + bounds.MaxY) / 2
	lowestZ := bounds.MinZ

	// Create a new BaseModel with centered triangles
	centered := bm.Copy()
	centered.Triangles = make([]Triangle, len(bm.Triangles))
	for i, t := range bm.Triangles {
		centered.Triangles[i] = Triangle{
			Normal: t.Normal,
			V1: Vector3{
				X: t.V1.X - centerX,
				Y: t.V1.Y - centerY,
				Z: t.V1.Z - lowestZ,
			},
			V2: Vector3{
				X: t.V2.X - centerX,
				Y: t.V2.Y - centerY,
				Z: t.V2.Z - lowestZ,
			},
			V3: Vector3{
				X: t.V3.X - centerX,
				Y: t.V3.Y - centerY,
				Z: t.V3.Z - lowestZ,
			},
		}
	}

	return centered
}

// CleanBottom flattens all vertices below the given zThreshold to create a flat base plane
// All vertices with Z <= zThreshold will have their Z coordinate set to the minimum Z value found
func (bm *BaseModel) CleanBottom(zThreshold float64) *BaseModel {
	if len(bm.Triangles) == 0 {
		return bm
	}

	// Find the minimum Z value among all vertices below the threshold
	minZ := float64(math.Inf(1))
	for _, tri := range bm.Triangles {
		if tri.V1.Z <= zThreshold {
			minZ = math.Min(minZ, tri.V1.Z)
		}
		if tri.V2.Z <= zThreshold {
			minZ = math.Min(minZ, tri.V2.Z)
		}
		if tri.V3.Z <= zThreshold {
			minZ = math.Min(minZ, tri.V3.Z)
		}
	}

	// If no vertices found below threshold, return unchanged
	if math.IsInf(minZ, 1) {
		return bm
	}

	// Flatten vertices below threshold
	flattened := bm.Copy()
	flattened.Triangles = make([]Triangle, len(bm.Triangles))
	for i, tri := range bm.Triangles {
		newTri := tri

		if tri.V1.Z <= zThreshold {
			newTri.V1.Z = minZ
		}
		if tri.V2.Z <= zThreshold {
			newTri.V2.Z = minZ
		}
		if tri.V3.Z <= zThreshold {
			newTri.V3.Z = minZ
		}

		// Recalculate normal after modifying vertices
		newTri.Normal = newTri.ComputeNormal()

		flattened.Triangles[i] = newTri
	}

	return flattened
}

// CleanSize returns a new BaseModel with all vertices normalized to fit in a unit cube centered at origin
func (bm *BaseModel) CleanSize() *BaseModel {
	if len(bm.Triangles) == 0 {
		return bm
	}

	// Calculate center and largest dimension
	bounds := bm.GetBounds()
	center := Vector3{
		X: (bounds.MinX + bounds.MaxX) / 2,
		Y: (bounds.MinY + bounds.MaxY) / 2,
		Z: (bounds.MinZ + bounds.MaxZ) / 2,
	}
	sizeX := bounds.MaxX - bounds.MinX
	sizeY := bounds.MaxY - bounds.MinY
	sizeZ := bounds.MaxZ - bounds.MinZ
	maxSize := sizeX
	if sizeY > maxSize {
		maxSize = sizeY
	}
	if sizeZ > maxSize {
		maxSize = sizeZ
	}

	// Avoid division by zero
	if maxSize == 0 {
		return bm
	}

	// Scale to fit in 100 units (makes it easier to work with in viewer)
	scale := 100.0 / maxSize

	// Transform all triangles
	normalized := bm.Copy()
	normalized.Triangles = make([]Triangle, len(bm.Triangles))
	for i, t := range bm.Triangles {
		normalized.Triangles[i] = Triangle{
			Normal: t.Normal, // Normal doesn't need scaling
			V1: Vector3{
				X: (t.V1.X - center.X) * scale,
				Y: (t.V1.Y - center.Y) * scale,
				Z: (t.V1.Z - center.Z) * scale,
			},
			V2: Vector3{
				X: (t.V2.X - center.X) * scale,
				Y: (t.V2.Y - center.Y) * scale,
				Z: (t.V2.Z - center.Z) * scale,
			},
			V3: Vector3{
				X: (t.V3.X - center.X) * scale,
				Y: (t.V3.Y - center.Y) * scale,
				Z: (t.V3.Z - center.Z) * scale,
			},
		}
	}

	return normalized
}
