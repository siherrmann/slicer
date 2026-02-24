[![Go Reference](https://pkg.go.dev/badge/github.com/siherrmann/slicer.svg)](https://pkg.go.dev/github.com/siherrmann/slicer)
[![Go Coverage](https://github.com/siherrmann/slicer/wiki/coverage.svg)](https://raw.githack.com/wiki/siherrmann/slicer/coverage.html)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/siherrmann/slicer/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/siherrmann/slicer)](http://goreportcard.com/report/siherrmann/slicer)

# slicer

## 💡 Goal of this package
Slicer is a robust, lightweight, and embeddable 3D slicing engine written entirely in Go. It is designed to be easily integrated into any backend service or CLI application, allowing you to parse 3D models (STL, 3MF) directly from `io.Reader` streams, generate high-quality G-code, and serve interactive 3D layer visuals—all without relying on heavy external C++ binaries.

---

## 🛠️ Installation
To integrate the slicer package into your Go project, use the standard `go get` command:

```bash
go get github.com/siherrmann/slicer
```

---

## 🚀 Getting started
To read a 3D model and generate G-code, you just need a few lines of code. Slicer natively accepts `io.Reader`, so you can stream your files from disk, S3, or HTTP uploads directly into the engine.

```go
package main

import (
	"fmt"
	"os"
	"github.com/siherrmann/slicer"
)

func main() {
	// 1. Create a new slicer instance with default configuration
	s := slicer.NewSlicer()

	// 2. Open an STL or 3MF file (or use any io.Reader)
	file, err := os.Open("example.stl")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// 3. Load the model into the slicer
	bm, err := s.LoadSTLModel(file, "example.stl")
	if err != nil {
		panic(err)
	}
	
	// Optional: Auto-orient and drop to the build plate
	s.Model = bm.CleanSize().CleanBottom(1).CleanPosition()

	// 4. Generate print paths (Perimeters, Infill, Supports, etc.)
	paths, err := s.GeneratePrintPaths(s.Config)
	if err != nil {
		panic(err)
	}

	// 5. Export to G-code
	gcode := s.GenerateGCode(paths)
	fmt.Printf("Successfully generated %d bytes of G-code!\n", len(gcode))
}
```

---

## ⚙️ Configuration
Slicing parameters are easily customized by modifying the `SliceConfig` object attached to the engine before generating paths.

```go
s.Config.LayerHeight = 0.2         // Layer height in mm
s.Config.LineWidth = 0.4           // Extrusion line width in mm
s.Config.ShellCount = 3            // Number of outer perimeters
s.Config.InfillDensity = 0.15      // 15% infill
s.Config.SupportType = model.SupportTree   // Enable organic tree supports
```

---

## ⭐ Features
* **Natively Embeddable**: Pure Go implementation. No CGO, no Python wrappers, no heavy sub-processes.
* **Stream Processing**: Load meshes directly from any `io.Reader` out-of-the-box (Zero-FS footprints).
* **Multi-Format Support**: Reads raw ASCII/Binary `STL` meshes and modern compressed `3MF` packages.
* **Organic Tree Supports**: Built-in dynamic vector-particle physics engine that generates incredibly smooth, flaring tree-supports targeting localized barycenters automatically.
* **Advanced Infills**: Supports complex infill patterns, including mathematical sinusoidal Gyroid structures.
* **Automated Mesh Healing**: Handles floating-point precision errors, clipping, and manifold adjustments automatically during poly-segment boolean operations.
* **Built-in HTTP Debugger**: Ships with a fast internal HTMX viewer for rendering specific layers and debugging toolpaths live in your browser.
