package core

import (
	"github.com/siherrmann/slicer/model"
)

// GenerateSkirt generates skirt or brim paths around the base of the model
func GenerateSkirt(firstLayer []model.Polygon, params model.SliceConfig) []model.ContinuousPath {
	if params.SkirtCount <= 0 && params.BrimCount <= 0 {
		return nil
	}

	var paths []model.ContinuousPath

	// 1. Identify all outer shells on the first layer
	var shells []model.Polygon
	for _, p := range firstLayer {
		if !p.IsHole {
			shells = append(shells, p)
		}
	}

	if len(shells) == 0 {
		return nil
	}

	// 2. Generate Brim (directly attached)
	if params.BrimCount > 0 {
		for _, shell := range shells {
			for i := 1; i <= params.BrimCount; i++ {
				offset := float64(i) * params.LineWidth
				brimLine := shell.OffsetPolygon(offset)
				if brimLine != nil && len(brimLine.Points) > 2 {
					brimLine.IsClosed = true
					paths = append(paths, brimLine.ToContinuousPath(params.FirstLayerSpeed, model.CategoryOuterWall))
				}
			}
		}
	}

	// 3. Generate Skirt (offset from model)
	if params.SkirtCount > 0 {
		for _, shell := range shells {
			for i := 0; i < params.SkirtCount; i++ {
				// Base offset + line index * width
				offset := params.SkirtOffset + float64(i)*params.LineWidth
				skirtLine := shell.OffsetPolygon(offset)
				if skirtLine != nil && len(skirtLine.Points) > 2 {
					skirtLine.IsClosed = true
					paths = append(paths, skirtLine.ToContinuousPath(params.FirstLayerSpeed, model.CategoryOuterWall))
				}
			}
		}
	}

	return paths
}
