package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContinuousPath_GetPrintResult(t *testing.T) {
	config := NewSliceConfig() // provides default speeds

	tests := []struct {
		name     string
		path     ContinuousPath
		config   *SliceConfig
		expected PrintResult
	}{
		{
			name: "mixed segments",
			path: ContinuousPath{
				Segments: []PathSegment{
					{
						Start:    Vector3{X: 0, Y: 0, Z: 0},
						End:      Vector3{X: 10, Y: 0, Z: 0},
						IsTravel: true,
					}, // Length 10, travel speed 120 -> time = 10/120 = 0.08333...
					{
						Start:    Vector3{X: 10, Y: 0, Z: 0},
						End:      Vector3{X: 10, Y: 10, Z: 0},
						IsTravel: false,
						Category: CategoryOuterWall,
					}, // Length 10, outer wall speed 25 -> time = 10/25 = 0.4
				},
				LayerIndex: 1,
			},
			config: config,
			expected: PrintResult{
				PrintTime:           10.0/120.0 + 10.0/25.0,
				ExtrusionPathLength: 10.0,
				TravelPathLength:    10.0,
				MovementPath:        20.0,
			},
		},
		{
			name: "first layer speed limit",
			path: ContinuousPath{
				Segments: []PathSegment{
					{
						Start:    Vector3{X: 0, Y: 0, Z: 0},
						End:      Vector3{X: 10, Y: 0, Z: 0},
						IsTravel: false,
						Category: CategoryOuterWall,
					}, // Length 10, outer wall speed 25, but first layer limits to 20 -> time = 10/20 = 0.5
				},
				LayerIndex: 0, // First layer!
			},
			config: config,
			expected: PrintResult{
				PrintTime:           10.0 / 20.0, // default FirstLayerSpeed is 20.0
				ExtrusionPathLength: 10.0,
				TravelPathLength:    0.0,
				MovementPath:        10.0,
			},
		},
		{
			name: "custom segment speed override",
			path: ContinuousPath{
				Segments: []PathSegment{
					{
						Start:    Vector3{X: 0, Y: 0, Z: 0},
						End:      Vector3{X: 10, Y: 0, Z: 0},
						IsTravel: false,
						Category: CategoryOuterWall,
						Speed:    100.0, // Should override
					},
				},
				LayerIndex: 1,
			},
			config: config,
			expected: PrintResult{
				PrintTime:           10.0 / 100.0,
				ExtrusionPathLength: 10.0,
				TravelPathLength:    0.0,
				MovementPath:        10.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.path.GetPrintResult(tt.config)
			assert.InDelta(t, tt.expected.PrintTime, result.PrintTime, 1e-6)
			assert.InDelta(t, tt.expected.ExtrusionPathLength, result.ExtrusionPathLength, 1e-6)
			assert.InDelta(t, tt.expected.TravelPathLength, result.TravelPathLength, 1e-6)
			assert.InDelta(t, tt.expected.MovementPath, result.MovementPath, 1e-6)
		})
	}
}

func TestContinuousPath_getSegmentSpeed(t *testing.T) {
	config := NewSliceConfig()

	tests := []struct {
		name     string
		path     ContinuousPath
		seg      PathSegment
		expected float64
	}{
		{
			name:     "category inner wall",
			path:     ContinuousPath{LayerIndex: 1},
			seg:      PathSegment{Category: CategoryInnerWall},
			expected: config.WallSpeed,
		},
		{
			name:     "category solid infill",
			path:     ContinuousPath{LayerIndex: 1},
			seg:      PathSegment{Category: CategorySolidInfill},
			expected: config.InfillSpeed * 0.8,
		},
		{
			name:     "min speed enforcement",
			path:     ContinuousPath{LayerIndex: 1},
			seg:      PathSegment{Category: CategoryInnerWall, Speed: 1.0}, // explicitly set low speed
			expected: config.MinSpeed,                                      // Should be bumped up to MinSpeed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.path.getSegmentSpeed(tt.seg, config)
			assert.InDelta(t, tt.expected, result, 1e-6)
		})
	}
}
