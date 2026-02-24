package core

import (
	"testing"

	"github.com/siherrmann/slicer/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanFullPaths(t *testing.T) {
	// Let's create a continuous path with a mix of points that can be simplified
	paths := []model.ContinuousPath{
		{
			Segments: []model.PathSegment{
				{Start: model.Vector3{X: 0, Y: 0, Z: 0}, End: model.Vector3{X: 1, Y: 0.001, Z: 0}, IsTravel: false},
				{Start: model.Vector3{X: 1, Y: 0.001, Z: 0}, End: model.Vector3{X: 2, Y: -0.001, Z: 0}, IsTravel: false},
				{Start: model.Vector3{X: 2, Y: -0.001, Z: 0}, End: model.Vector3{X: 3, Y: 0, Z: 0}, IsTravel: false}, // Collinear line X=0 to 3
				// Travel move
				{Start: model.Vector3{X: 3, Y: 0, Z: 0}, End: model.Vector3{X: 5, Y: 5, Z: 0}, IsTravel: true},
				// Another segment
				{Start: model.Vector3{X: 5, Y: 5, Z: 0}, End: model.Vector3{X: 5, Y: 6, Z: 0}, IsTravel: false},
			},
		},
	}

	// Clean with a tolerance of 0.1 mm
	cleaned := CleanFullPaths(paths, 0.1)

	// Since the Y variations are 0.001 mm and epsilon is 0.1, the points (1, 0.001) and (2, -0.001) should be removed
	// The travel move connects 3,0 to 5,5.
	// We expect segments: (0,0)->(3,0), (3,0)->(5,5), (5,5)->(5,6)
	// That's 3 segments.
	assert.Equal(t, 3, len(cleaned.Segments))

	// Check the simplified points
	assert.Equal(t, model.Vector3{X: 0, Y: 0, Z: 0}, cleaned.Segments[0].Start)
	assert.Equal(t, model.Vector3{X: 3, Y: 0, Z: 0}, cleaned.Segments[0].End)
	assert.False(t, cleaned.Segments[0].IsTravel)

	assert.Equal(t, model.Vector3{X: 3, Y: 0, Z: 0}, cleaned.Segments[1].Start)
	assert.Equal(t, model.Vector3{X: 5, Y: 5, Z: 0}, cleaned.Segments[1].End)
	assert.True(t, cleaned.Segments[1].IsTravel)

	assert.Equal(t, model.Vector3{X: 5, Y: 5, Z: 0}, cleaned.Segments[2].Start)
	assert.Equal(t, model.Vector3{X: 5, Y: 6, Z: 0}, cleaned.Segments[2].End)
	assert.False(t, cleaned.Segments[2].IsTravel)
}

func TestRamerDouglasPeucker(t *testing.T) {
	points := []model.Vector3{
		{X: 0, Y: 0, Z: 0},
		{X: 1, Y: 0.1, Z: 0},  // distance from line is 0.1
		{X: 2, Y: -0.1, Z: 0}, // distance from line is 0.1
		{X: 3, Y: 0, Z: 0},
	}

	// Test case 1: tight epsilon (no points removed)
	result1 := ramerDouglasPeucker(points, 0.05)
	assert.Equal(t, 4, len(result1))

	// Test case 2: loose epsilon (intermediate points removed)
	result2 := ramerDouglasPeucker(points, 0.2)
	assert.Equal(t, 2, len(result2))
	assert.Equal(t, model.Vector3{X: 0, Y: 0, Z: 0}, result2[0])
	assert.Equal(t, model.Vector3{X: 3, Y: 0, Z: 0}, result2[1])

	// Test case 3: edge peak (should keep the peak)
	pointsPeak := []model.Vector3{
		{X: 0, Y: 0, Z: 0},
		{X: 5, Y: 5, Z: 0}, // peak
		{X: 10, Y: 0, Z: 0},
	}
	result3 := ramerDouglasPeucker(pointsPeak, 1.0)
	assert.Equal(t, 3, len(result3))

	// Test case 4: short array
	short := []model.Vector3{{X: 0, Y: 0}, {X: 1, Y: 1}}
	assert.Equal(t, 2, len(ramerDouglasPeucker(short, 1.0)))
}

func TestGenerateFullSTLPath(t *testing.T) {
	// Create a minimal BaseModel with 1 slice
	bm := model.NewBaseModel("test")
	bm.Bounds = model.BoundingBox{MinX: 0, MinY: 0, MaxX: 10, MaxY: 10, MinZ: 0, MaxZ: 1}
	slice0 := &model.Slice{
		Z: 0.2,
		Polygons: []model.Polygon{
			{
				Points:   []model.Vector3{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}},
				IsClosed: true,
			},
		},
		LayerIndex:    0,
		IsBottomLayer: true, // Generate layer will fill it if bottom
	}
	bm.Slices = append(bm.Slices, slice0)

	config := model.NewSliceConfig()
	config.LayerHeight = 0.2
	config.WallThickness = 0.4
	config.ShellCount = 1
	config.TopLayers = 0
	config.BottomLayers = 1
	config.InfillDensity = 0
	config.RaftLayers = 0
	config.SupportType = model.SupportNone
	config.StartPosition = model.Vector3{X: -10, Y: -10, Z: 0}
	config.EndPosition = model.Vector3{X: 50, Y: 50, Z: 50}

	// Make sure walls/infill run without panic
	paths := GenerateFullSTLPath(bm, *config)

	// Since we mocked minimal polygons, there should be travel moves and layer moves
	assert.Greater(t, len(paths), 0, "Should generate some paths")

	hasTravel := false
	hasExtrusion := false
	for _, p := range paths {
		if p.PathType == model.PathTravel {
			hasTravel = true
		} else if p.PathType == model.PathExtrusion {
			hasExtrusion = true
		}
	}
	assert.True(t, hasTravel, "Should have travel moves connecting sequences")
	assert.True(t, hasExtrusion, "Should have extrusion paths")

	// Verify travel ends up at EndPosition
	lastPath := paths[len(paths)-1]
	require.Greater(t, len(lastPath.Segments), 0)
	lastSegment := lastPath.Segments[len(lastPath.Segments)-1]
	assert.Equal(t, config.EndPosition, lastSegment.End)
}
