package model

import (
	"bytes"
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSTL(t *testing.T) {
	stl := NewSTL("test")
	assert.NotNil(t, stl)
	assert.Equal(t, "test", stl.Name)
	assert.False(t, stl.IsBinary)
}

func TestDetectASCIIFormat(t *testing.T) {
	// Standard ASCII
	asciiData := []byte("solid cube\n  facet normal 0 0 1\n    outer loop\n      vertex 0 0 0\n      vertex 1 0 0\n      vertex 0 1 0\n    endloop\n  endfacet\nendsolid\n")
	assert.True(t, detectASCIIFormat(asciiData))

	// Binary (starts with random bytes, no "solid")
	binaryHeader := make([]byte, 84)
	binaryHeader[0] = 'B'
	binaryHeader[1] = 'I'
	assert.False(t, detectASCIIFormat(binaryHeader))

	// Tricky binary: starts with "solid" but has exact binary size for 1 triangle
	// 80 bytes header + 4 byte count + 50 bytes triangle = 134 bytes total
	trickyBinary := make([]byte, 134)
	copy(trickyBinary[0:5], "solid")
	// Set count to 1
	binary.LittleEndian.PutUint32(trickyBinary[80:84], 1)
	assert.False(t, detectASCIIFormat(trickyBinary), "Should detect exact binary size even if it starts with 'solid'")
}

func TestLoadSTL_ASCII(t *testing.T) {
	asciiSTL := `solid testCube
  facet normal 0.0 0.0 1.0
    outer loop
      vertex 0.0 0.0 0.0
      vertex 10.0 0.0 0.0
      vertex 0.0 10.0 0.0
    endloop
  endfacet
endsolid testCube`

	r := strings.NewReader(asciiSTL)
	stl, err := LoadSTL(r, "testCube")

	require.NoError(t, err)
	assert.NotNil(t, stl)
	assert.False(t, stl.IsBinary)
	assert.Equal(t, "testCube", stl.Name)
	assert.Equal(t, 1, stl.GetTriangleCount())

	tri := stl.Triangles[0]
	assert.Equal(t, 1.0, tri.Normal.Z)
	assert.Equal(t, 10.0, tri.V2.X)
	assert.Equal(t, 10.0, tri.V3.Y)
}

func TestLoadSTL_Binary(t *testing.T) {
	// 80 byte header + 4 byte count + 50 byte triangle = 134 bytes
	data := make([]byte, 134)
	copy(data[:80], "Binary STL Test")
	binary.LittleEndian.PutUint32(data[80:84], 1) // 1 triangle

	offset := 84
	// Normal: (0, 0, 1)
	binary.LittleEndian.PutUint32(data[offset:offset+4], math.Float32bits(0.0))
	binary.LittleEndian.PutUint32(data[offset+4:offset+8], math.Float32bits(0.0))
	binary.LittleEndian.PutUint32(data[offset+8:offset+12], math.Float32bits(1.0))
	offset += 12

	// V1: (0, 0, 0)
	binary.LittleEndian.PutUint32(data[offset:offset+4], math.Float32bits(0.0))
	binary.LittleEndian.PutUint32(data[offset+4:offset+8], math.Float32bits(0.0))
	binary.LittleEndian.PutUint32(data[offset+8:offset+12], math.Float32bits(0.0))
	offset += 12

	// V2: (10, 0, 0)
	binary.LittleEndian.PutUint32(data[offset:offset+4], math.Float32bits(10.0))
	binary.LittleEndian.PutUint32(data[offset+4:offset+8], math.Float32bits(0.0))
	binary.LittleEndian.PutUint32(data[offset+8:offset+12], math.Float32bits(0.0))
	offset += 12

	// V3: (0, 10, 0)
	binary.LittleEndian.PutUint32(data[offset:offset+4], math.Float32bits(0.0))
	binary.LittleEndian.PutUint32(data[offset+4:offset+8], math.Float32bits(10.0))
	binary.LittleEndian.PutUint32(data[offset+8:offset+12], math.Float32bits(0.0))
	offset += 12

	// Attr: 0
	binary.LittleEndian.PutUint16(data[offset:offset+2], 0)

	r := bytes.NewReader(data)
	stl, err := LoadSTL(r, "binaryTest")

	require.NoError(t, err)
	assert.NotNil(t, stl)
	assert.True(t, stl.IsBinary)
	assert.Equal(t, 1, stl.GetTriangleCount())

	tri := stl.Triangles[0]
	assert.Equal(t, 1.0, tri.Normal.Z)
	assert.Equal(t, 10.0, tri.V2.X)
	assert.Equal(t, 10.0, tri.V3.Y)
}

func TestSTL_GetBounds(t *testing.T) {
	stl := NewSTL("test")

	t1 := Triangle{
		V1: Vector3{X: -10, Y: -10, Z: 0},
		V2: Vector3{X: 10, Y: -10, Z: 0},
		V3: Vector3{X: 0, Y: 10, Z: 5},
	}
	stl.AddTriangle(t1)

	bounds := stl.GetBounds() // Internally overrides BaseModel's GetBounds to trigger recalc
	assert.Equal(t, -10.0, bounds.MinX)
	assert.Equal(t, 10.0, bounds.MaxX)
	assert.Equal(t, -10.0, bounds.MinY)
	assert.Equal(t, 10.0, bounds.MaxY)
	assert.Equal(t, 0.0, bounds.MinZ)
	assert.Equal(t, 5.0, bounds.MaxZ)
}
