package core

import (
	"testing"

	"github.com/siherrmann/slicer/model"
	"github.com/stretchr/testify/assert"
)

func TestGenerateSkirt(t *testing.T) {
	config := model.NewSliceConfig()
	config.LineWidth = 0.4
	config.FirstLayerSpeed = 20.0

	shell := model.Polygon{
		Points: []model.Vector3{
			{X: 10, Y: 10, Z: 0.2}, {X: 20, Y: 10, Z: 0.2}, {X: 20, Y: 20, Z: 0.2}, {X: 10, Y: 20, Z: 0.2},
		},
		IsClosed: true,
		IsHole:   false,
	}

	hole := model.Polygon{
		Points: []model.Vector3{
			{X: 12, Y: 12, Z: 0.2}, {X: 18, Y: 12, Z: 0.2}, {X: 18, Y: 18, Z: 0.2}, {X: 12, Y: 18, Z: 0.2},
		},
		IsClosed: true,
		IsHole:   true,
	}

	layer := []model.Polygon{shell, hole}

	t.Run("Disabled", func(t *testing.T) {
		cfg := *config
		cfg.BrimCount = 0
		cfg.SkirtCount = 0
		paths := GenerateSkirt(layer, cfg)
		assert.Nil(t, paths)
	})

	t.Run("Only Brim", func(t *testing.T) {
		cfg := *config
		cfg.BrimCount = 2
		cfg.SkirtCount = 0

		paths := GenerateSkirt(layer, cfg)

		// Brim runs along outer shell, so 2 paths for 2 counts
		assert.Equal(t, 2, len(paths))

		// First brim path is separated by 1 * LineWidth
		firstBrimOffset := 0.4
		// The original boundary goes from 10 to 20
		// MinX should now be 10 - 0.4 = 9.6

		var minXBrim float64 = 100
		for _, seg := range paths[0].Segments {
			if seg.Start.X < minXBrim {
				minXBrim = seg.Start.X
			}
		}

		assert.InDelta(t, 10.0-firstBrimOffset, minXBrim, 1e-4)
	})

	t.Run("Only Skirt", func(t *testing.T) {
		cfg := *config
		cfg.BrimCount = 0
		cfg.SkirtCount = 2
		cfg.SkirtOffset = 3.0

		paths := GenerateSkirt(layer, cfg)

		assert.Equal(t, 2, len(paths))

		// Skirt starts at SkirtOffset + 0*LineWidth
		firstSkirtOffset := 3.0
		var minXSkirt float64 = 100
		for _, seg := range paths[0].Segments {
			if seg.Start.X < minXSkirt {
				minXSkirt = seg.Start.X
			}
		}
		assert.InDelta(t, 10.0-firstSkirtOffset, minXSkirt, 1e-4)
	})

	t.Run("Brim and Skirt", func(t *testing.T) {
		cfg := *config
		cfg.BrimCount = 1
		cfg.SkirtCount = 1
		cfg.SkirtOffset = 5.0

		paths := GenerateSkirt(layer, cfg)

		// 1 brim + 1 skirt = 2 paths
		assert.Equal(t, 2, len(paths))
	})

	t.Run("Empty Layer", func(t *testing.T) {
		cfg := *config
		cfg.BrimCount = 1
		paths := GenerateSkirt([]model.Polygon{}, cfg)
		assert.Nil(t, paths)
	})
}
