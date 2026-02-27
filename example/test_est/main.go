package main

import (
	"fmt"
	"log"
	"os"

	"github.com/siherrmann/slicer"
	"github.com/siherrmann/slicer/model"
)

func main() {
	s := slicer.NewSlicer()

	file, err := os.Open("../af86a550_f42a3fa6_rpi-3-case.stl")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	stlModel, err := s.LoadSTLModel(file, "rpi-3-case.stl")
	if err != nil {
		log.Fatal(err)
	}

	bounds := stlModel.GetBounds()
	fmt.Printf("Bounds: Min(%.2f, %.2f, %.2f) Max(%.2f, %.2f, %.2f)\n",
		bounds.MinX, bounds.MinY, bounds.MinZ, bounds.MaxX, bounds.MaxY, bounds.MaxZ)

	fmt.Printf("Size: X: %.2f Y: %.2f Z: %.2f\n",
		bounds.MaxX-bounds.MinX, bounds.MaxY-bounds.MinY, bounds.MaxZ-bounds.MinZ)

	paths, err := s.GeneratePrintPaths(s.Config)
	if err != nil {
		log.Fatal(err)
	}

	var totalResult model.PrintResult
	var totalWalls, totalInfill, totalSupport float64
	for _, p := range paths {
		r := p.GetPrintResult(s.Config)
		totalResult.PrintTime += r.PrintTime
		totalResult.ExtrusionPathLength += r.ExtrusionPathLength
		totalResult.TravelPathLength += r.TravelPathLength

		for _, seg := range p.Segments {
			if seg.IsTravel {
				continue
			}
			len := seg.Start.Distance(seg.End)
			switch seg.Category {
			case model.CategoryInnerWall, model.CategoryOuterWall:
				totalWalls += len
			case model.CategoryInfill, model.CategorySolidInfill:
				totalInfill += len
			case model.CategorySupport:
				totalSupport += len
			}
		}
	}

	fmt.Printf("Print Time: %.2f seconds (%.2f hours)\n", totalResult.PrintTime, totalResult.PrintTime/3600)
	fmt.Printf("Extrusion Length: %.2f mm\n", totalResult.ExtrusionPathLength)
	fmt.Printf("Travel Length: %.2f mm\n", totalResult.TravelPathLength)
	fmt.Printf("Walls: %.2f mm\n", totalWalls)
	fmt.Printf("Infill: %.2f mm\n", totalInfill)
	fmt.Printf("Support: %.2f mm\n", totalSupport)
}
