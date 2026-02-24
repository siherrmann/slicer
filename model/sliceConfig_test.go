package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliceConfig_DefaultStartGCode(t *testing.T) {
	config := NewSliceConfig()
	config.NozzleTemp = 210.0
	config.BedTemp = 65.0

	gcode := DefaultStartGCode(config)

	assert.True(t, strings.Contains(gcode, "M104 S210.0"), "Start G-code should contain correct nozzle temp")
	assert.True(t, strings.Contains(gcode, "M140 S65.0"), "Start G-code should contain correct bed temp")
	assert.True(t, strings.Contains(gcode, "G28 ; Home all axes"), "Start G-code should contain homing command")
}

func TestSliceConfig_DefaultEndGCode(t *testing.T) {
	gcode := DefaultEndGCode()

	assert.True(t, strings.Contains(gcode, "M104 S0"), "End G-code should turn off nozzle")
	assert.True(t, strings.Contains(gcode, "M140 S0"), "End G-code should turn off bed")
}

func TestSliceConfig_floatStr(t *testing.T) {
	tests := []struct {
		name     string
		f        float64
		expected string
	}{
		{name: "integer float", f: 200.0, expected: "200.0"},
		{name: "float with decimals", f: 200.45, expected: "200.4"}, // rounding (round half to even)
		{name: "zero", f: 0.0, expected: "0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := floatStr(tt.f)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSliceConfig_NewSliceConfig(t *testing.T) {
	config := NewSliceConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 0.2, config.LayerHeight)
	assert.Equal(t, 0.4, config.LineWidth)
	assert.Equal(t, "PLA", config.MaterialName)
	assert.NotEmpty(t, config.StartGCode)
	assert.NotEmpty(t, config.EndGCode)
}

func TestSliceConfig_CalculateLayerHeights(t *testing.T) {
	config := NewSliceConfig()
	config.FirstLayer = 0.3
	config.LayerHeight = 0.2

	tests := []struct {
		name     string
		minZ     float64
		maxZ     float64
		expected []float64
	}{
		{
			name: "standard slice",
			minZ: 0.0,
			maxZ: 1.0,
			// 0.3 (first), 0.5, 0.7, 0.9
			expected: []float64{0.3, 0.5, 0.7, 0.9},
		},
		{
			name: "small object",
			minZ: 0.0,
			maxZ: 0.2,
			// only first layer at 0.3, but maxZ is 0.2?
			// Wait, the logic is: z = minZ + FirstLayer => 0.3. Over maxZ! Still appended.
			expected: []float64{0.3},
		},
		{
			name: "exact boundary",
			minZ: 0.0,
			maxZ: 0.5,
			// 0.3, 0.5
			expected: []float64{0.3, 0.5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.CalculateLayerHeights(tt.minZ, tt.maxZ)
			assert.Equal(t, len(tt.expected), len(result))
			for i := range tt.expected {
				assert.InDelta(t, tt.expected[i], result[i], 1e-6)
			}
		})
	}
}
