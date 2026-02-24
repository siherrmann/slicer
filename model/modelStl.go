package model

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/siherrmann/slicer/helper"
)

// STL represents a 3D model in STL format
type STL struct {
	BaseModel
	IsBinary bool     `json:"is_binary"`
	Header   [80]byte `json:"header"`
}

// NewSTL creates a new empty STL model
func NewSTL(name string) *STL {
	return &STL{
		BaseModel: *NewBaseModel(name),
		IsBinary:  false,
	}
}

// LoadSTL loads an STL file from a reader (supports both ASCII and binary formats)
func LoadSTL(r io.Reader, name string) (*STL, error) {
	// Read all data into buffer to detect format
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read STL data: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty STL file")
	}

	// Detect format: ASCII files start with "solid"
	// But be careful - some binary files might have "solid" in the header
	var stl *STL
	if detectASCIIFormat(data) {
		stl, err = loadASCIISTL(data, name)
		if err != nil {
			return nil, err
		}
		return stl, nil
	} else {
		stl, err = loadBinarySTL(data, name)
		if err != nil {
			return nil, err
		}
	}

	stl.Bounds = stl.GetBounds()

	return stl, nil
}

// detectASCIIFormat determines if the STL is ASCII or binary
func detectASCIIFormat(data []byte) bool {
	// ASCII STL files start with "solid"
	if !bytes.HasPrefix(data, []byte("solid")) {
		return false
	}

	// Binary files might have "solid" in header, but they have a specific structure
	// Binary format: 80-byte header + 4-byte count + triangles (50 bytes each)
	if len(data) >= 84 {
		// Try to read triangle count from binary format
		triangleCount := binary.LittleEndian.Uint32(data[80:84])
		expectedSize := 84 + int(triangleCount)*50

		// If the file size matches binary format exactly, it's binary
		if len(data) == expectedSize {
			return false
		}
	}

	return true
}

// loadBinarySTL loads a binary STL file
func loadBinarySTL(data []byte, name string) (*STL, error) {
	if len(data) < 84 {
		return nil, fmt.Errorf("binary STL file too small (minimum 84 bytes)")
	}

	stl := NewSTL(name)
	stl.IsBinary = true

	// Read 80-byte header
	copy(stl.Header[:], data[0:80])

	// Read triangle count
	triangleCount := binary.LittleEndian.Uint32(data[80:84])

	// Validate file size
	expectedSize := 84 + int(triangleCount)*50
	if len(data) != expectedSize {
		return nil, fmt.Errorf("invalid binary STL: expected %d bytes, got %d bytes", expectedSize, len(data))
	}

	// Read triangles
	offset := 84
	for i := 0; i < int(triangleCount); i++ {
		if offset+50 > len(data) {
			return nil, fmt.Errorf("unexpected end of file at triangle %d", i)
		}

		triangle := Triangle{}

		// Read normal vector (3 float32)
		triangle.Normal.X = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset : offset+4])))
		triangle.Normal.Y = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset+4 : offset+8])))
		triangle.Normal.Z = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset+8 : offset+12])))
		offset += 12

		// Read vertex 1 (3 float32)
		triangle.V1.X = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset : offset+4])))
		triangle.V1.Y = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset+4 : offset+8])))
		triangle.V1.Z = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset+8 : offset+12])))
		offset += 12

		// Read vertex 2 (3 float32)
		triangle.V2.X = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset : offset+4])))
		triangle.V2.Y = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset+4 : offset+8])))
		triangle.V2.Z = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset+8 : offset+12])))
		offset += 12

		// Read vertex 3 (3 float32)
		triangle.V3.X = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset : offset+4])))
		triangle.V3.Y = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset+4 : offset+8])))
		triangle.V3.Z = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset+8 : offset+12])))
		offset += 12

		// Read attribute byte count (uint16)
		triangle.Attr = binary.LittleEndian.Uint16(data[offset : offset+2])
		offset += 2

		stl.AddTriangle(triangle)
	}

	return stl, nil
}

// loadASCIISTL loads an ASCII STL file
func loadASCIISTL(data []byte, name string) (*STL, error) {
	stl := NewSTL(name)
	stl.IsBinary = false

	scanner := bufio.NewScanner(bytes.NewReader(data))

	var currentTriangle *Triangle
	var vertexCount int

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)

		if line == "" || len(fields) == 0 {
			continue
		}

		keyword := fields[0]

		switch keyword {
		case "solid":
			// Optional name after "solid"
			if len(fields) > 1 && stl.Name == name {
				stl.Name = strings.Join(fields[1:], " ")
			}

		case "facet":
			if len(fields) < 5 || fields[1] != "normal" {
				return nil, fmt.Errorf("line %d: invalid facet normal declaration", lineNum)
			}
			currentTriangle = &Triangle{}
			var err error
			currentTriangle.Normal.X, err = helper.ParseFloat64(fields[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid normal X: %w", lineNum, err)
			}
			currentTriangle.Normal.Y, err = helper.ParseFloat64(fields[3])
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid normal Y: %w", lineNum, err)
			}
			currentTriangle.Normal.Z, err = helper.ParseFloat64(fields[4])
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid normal Z: %w", lineNum, err)
			}
			vertexCount = 0

		case "vertex":
			if currentTriangle == nil {
				return nil, fmt.Errorf("line %d: vertex outside facet", lineNum)
			}
			if len(fields) < 4 {
				return nil, fmt.Errorf("line %d: invalid vertex declaration", lineNum)
			}

			var v Vector3
			var err error
			v.X, err = helper.ParseFloat64(fields[1])
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid vertex X: %w", lineNum, err)
			}
			v.Y, err = helper.ParseFloat64(fields[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid vertex Y: %w", lineNum, err)
			}
			v.Z, err = helper.ParseFloat64(fields[3])
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid vertex Z: %w", lineNum, err)
			}

			switch vertexCount {
			case 0:
				currentTriangle.V1 = v
			case 1:
				currentTriangle.V2 = v
			case 2:
				currentTriangle.V3 = v
			default:
				return nil, fmt.Errorf("line %d: too many vertices in facet", lineNum)
			}
			vertexCount++

		case "endfacet":
			if currentTriangle == nil {
				return nil, fmt.Errorf("line %d: endfacet without facet", lineNum)
			}
			if vertexCount != 3 {
				return nil, fmt.Errorf("line %d: facet has %d vertices, expected 3", lineNum, vertexCount)
			}
			stl.AddTriangle(*currentTriangle)
			currentTriangle = nil

		case "endsolid":
			// End of file - just continue parsing in case there's more content
			continue
		case "outer", "loop", "endloop":
			// These keywords are expected but don't require action
			continue
		default:
			// Ignore unknown keywords (for compatibility)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading STL: %w", err)
	}

	if len(stl.Triangles) == 0 {
		return nil, fmt.Errorf("no triangles found in STL file")
	}

	return stl, nil
}

func (stl *STL) GetBounds() BoundingBox {
	return stl.BaseModel.GetBounds()
}
