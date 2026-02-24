package model

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// 3MF XML Structures
type modelXML struct {
	XMLName   xml.Name      `xml:"model"`
	Unit      string        `xml:"unit,attr"`
	Metadata  []metadataXML `xml:"metadata"`
	Resources resourcesXML  `xml:"resources"`
	Build     buildXML      `xml:"build"`
}

type metadataXML struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

type resourcesXML struct {
	Objects []objectXML `xml:"object"`
}

type objectXML struct {
	ID   int     `xml:"id,attr"`
	Type string  `xml:"type,attr"`
	Mesh meshXML `xml:"mesh"`
}

type meshXML struct {
	Vertices []vertexXML  `xml:"vertices>vertex"`
	Indices  []polygonXML `xml:"triangles>triangle"`
}

type vertexXML struct {
	X float64 `xml:"x,attr"`
	Y float64 `xml:"y,attr"`
	Z float64 `xml:"z,attr"`
}

type polygonXML struct {
	V1 int `xml:"v1,attr"`
	V2 int `xml:"v2,attr"`
	V3 int `xml:"v3,attr"`
}

type buildXML struct {
	Items []itemXML `xml:"item"`
}

type itemXML struct {
	ObjectID  int    `xml:"objectid,attr"`
	Transform string `xml:"transform,attr"`
}

// Load3MF loads a 3MF file from an io.Reader and returns a BaseModel
func Load3MF(r io.Reader, modelName string) (*BaseModel, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	var modelData *modelXML

	// Look for the model file
	for _, f := range zr.File {
		if f.Name == "3D/3dmodel.model" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			decoder := xml.NewDecoder(rc)
			if err := decoder.Decode(&modelData); err != nil {
				return nil, err
			}
			break
		}
	}

	if modelData == nil {
		return nil, fmt.Errorf("could not find 3D/3dmodel.model in 3MF package")
	}

	bm := NewBaseModel(modelName)
	bm.SliceConfig = NewSliceConfig()

	// Parse Metadata into SliceConfig
	parse3MFMetadata(modelData.Metadata, bm.SliceConfig)

	// Map unit to scale
	unitScale := 1.0
	switch strings.ToLower(modelData.Unit) {
	case "inch":
		unitScale = 25.4
	case "millimeter":
		unitScale = 1.0
	case "centimeter":
		unitScale = 10.0
	case "meter":
		unitScale = 1000.0
	}

	// Create objects map for quick lookup
	objects := make(map[int]objectXML)
	for _, obj := range modelData.Resources.Objects {
		objects[obj.ID] = obj
	}

	// Build the model from items
	for _, item := range modelData.Build.Items {
		obj, ok := objects[item.ObjectID]
		if !ok {
			continue
		}

		// TODO: Apply transform matrix from item.Transform
		// For now, assume identity or simple translation if present

		for _, tri := range obj.Mesh.Indices {
			v1 := obj.Mesh.Vertices[tri.V1]
			v2 := obj.Mesh.Vertices[tri.V2]
			v3 := obj.Mesh.Vertices[tri.V3]

			triangle := Triangle{
				V1: Vector3{X: v1.X * unitScale, Y: v1.Y * unitScale, Z: v1.Z * unitScale},
				V2: Vector3{X: v2.X * unitScale, Y: v2.Y * unitScale, Z: v2.Z * unitScale},
				V3: Vector3{X: v3.X * unitScale, Y: v3.Y * unitScale, Z: v3.Z * unitScale},
			}
			triangle.Normal = triangle.ComputeNormal()
			bm.AddTriangle(triangle)
		}
	}

	bm.Bounds = bm.GetBounds()
	return bm, nil
}

func parse3MFMetadata(metadata []metadataXML, config *SliceConfig) {
	for _, meta := range metadata {
		val := meta.Value
		switch strings.ToLower(meta.Name) {
		case "title":
			// skip
		case "designer":
			// skip
		case "layerheight", "slic3rpe:layer_height", "cura:layer_height":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				config.LayerHeight = f
			}
		case "infilldensity", "slic3rpe:fill_density", "cura:infill_sparse_density":
			// 3mf often uses percentage (e.g. 20) while we use 0.0-1.0
			if f, err := strconv.ParseFloat(strings.TrimSuffix(val, "%"), 64); err == nil {
				if f > 1.0 {
					config.InfillDensity = f / 100.0
				} else {
					config.InfillDensity = f
				}
			}
		case "nozzletemp", "slic3rpe:temperature", "cura:material_print_temperature":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				config.NozzleTemp = f
			}
		case "bedtemp", "slic3rpe:bed_temperature", "cura:material_bed_temperature":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				config.BedTemp = f
			}
		}
	}
}
