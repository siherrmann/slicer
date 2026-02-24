package main

import (
	"log"
	"os"

	"github.com/siherrmann/slicer"
)

// DISCLAIMER: The examples are from https://ozeki.hu/p_1116-sample-stl-files-you-can-use-for-testing.html

func main() {
	slicer := slicer.NewSlicer()

	file, err := os.Open("./Eiffel_tower_sample.STL")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	stl, err := slicer.LoadSTLModel(file, "Eiffel_tower_sample.STL")
	if err != nil {
		panic(err)
	}

	log.Printf("Model statistics:")
	// #nosec G706
	log.Printf("  - Total triangles: %d", stl.GetTriangleCount())
	// #nosec G706
	log.Printf("  - Surface area: %.2f", stl.GetSurfaceArea())
	// #nosec G706
	log.Printf("  - Volume: %.2f", stl.GetVolume())
	bounds := stl.GetBounds()
	// #nosec G706
	log.Printf("  - Bounds: minX=%v, maxX=%v", bounds.MinX, bounds.MaxX)
	log.Println("")

	// Use a small tolerance appropriate for STL precision (0.001mm = 1 micron)
	// This should match vertices that differ only due to floating point errors
	tolerance := 0.00001

	issues := stl.Validate(tolerance)
	if !issues.IsWatertight || len(issues.BoundaryEdges) > 0 || len(issues.NonManifoldEdges) > 0 || len(issues.DegenerateTriangles) > 0 || len(issues.InvalidTriangles) > 0 || len(issues.DuplicateTriangles) > 0 {
		log.Printf("Issue summary:")
		// #nosec G706
		log.Printf("  - Boundary edges (holes): %d", len(issues.BoundaryEdges))
		// #nosec G706
		log.Printf("  - Non-manifold edges: %d", len(issues.NonManifoldEdges))
		// #nosec G706
		log.Printf("  - Degenerate triangles: %d", len(issues.DegenerateTriangles))
		// #nosec G706
		log.Printf("  - Invalid normals: %d", len(issues.InvalidTriangles))
		// #nosec G706
		log.Printf("  - Duplicate triangles: %d", len(issues.DuplicateTriangles))
		log.Println("")

		log.Printf("⚠️  Model has issues but will attempt to slice anyway...")
		log.Println("")
	} else {
		log.Printf("✅ The model is watertight and has no issues!")
		log.Println("")
	}

	// Perform slicing
	log.Printf("Slicing model...")
	log.Printf("  - Layer height: %.2f mm", slicer.Config.LayerHeight)
	log.Printf("  - First layer: %.2f mm", slicer.Config.FirstLayer)

	slices, err := slicer.Slice(stl)
	if err != nil {
		panic(err)
	}

	log.Printf("✅ Slicing complete!")
	// #nosec G706
	log.Printf("  - Total layers: %d", len(slices))
	// #nosec G706
	log.Printf("  - Height range: %.2f mm to %.2f mm", slices[0].Z, slices[len(slices)-1].Z)
	log.Println("")

	// Show statistics for a few layers
	log.Printf("Layer statistics:")
	layersToShow := []int{0, len(slices) / 4, len(slices) / 2, 3 * len(slices) / 4, len(slices) - 1}
	for _, idx := range layersToShow {
		if idx >= 0 && idx < len(slices) {
			slice := slices[idx]
			// #nosec G706
			log.Printf("  Layer %d (Z=%.2f mm): %d segments, %d contours",
				idx, slice.Z, len(slice.Segments), len(slice.Polygons))
		}
	}
}
