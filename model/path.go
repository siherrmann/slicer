package model

import "math"

// PathSegment represents a single segment in a path
type PathSegment struct {
	Start    Vector3      `json:"start"`
	End      Vector3      `json:"end"`
	IsTravel bool         `json:"is_travel,omitempty"`
	Speed    float64      `json:"speed,omitempty"`     // Intended print speed (mm/s), 0 = use default
	FlowRate float64      `json:"flow_rate,omitempty"` // Flow rate multiplier, 0 = use config default
	Category PathCategory `json:"category,omitempty"`  // Category for per-type speed/visibility
}

type PathType int

const (
	PathTravel PathType = iota
	PathExtrusion
)

// ContinuousPath represents a single continuous extrusion path
type ContinuousPath struct {
	Segments   []PathSegment `json:"segments"`
	PathType   PathType      `json:"path_type"`
	LayerIndex int           `json:"layer_index,omitempty"` // Layer index for G-code export
}

// PrintResult holds statistics about a print path
type PrintResult struct {
	PrintTime           float64 // Estimated print time in seconds
	ExtrusionPathLength float64 // Total length of extrusion moves in mm
	TravelPathLength    float64 // Total length of travel moves in mm
	MovementPath        float64 // Total length of all moves in mm
}

// GetPrintResult calculates the length and time statistics for the path
func (p *ContinuousPath) GetPrintResult(config *SliceConfig) PrintResult {
	var result PrintResult
	for _, seg := range p.Segments {
		length := seg.Start.Distance(seg.End)

		if seg.IsTravel {
			result.TravelPathLength += length
			if config.TravelSpeed > 0 {
				result.PrintTime += length / config.TravelSpeed
			}
		} else {
			result.ExtrusionPathLength += length
			speed := p.getSegmentSpeed(seg, config)
			if speed > 0 {
				result.PrintTime += length / speed
			}
		}
	}
	result.MovementPath = result.ExtrusionPathLength + result.TravelPathLength
	return result
}

func (p *ContinuousPath) getSegmentSpeed(seg PathSegment, config *SliceConfig) float64 {
	var speed float64
	if seg.Speed > 0 {
		speed = seg.Speed
	} else {
		switch seg.Category {
		case CategoryOuterWall:
			speed = config.OuterShellSpeed
		case CategoryInnerWall:
			speed = config.WallSpeed
		case CategoryInfill:
			speed = config.InfillSpeed
		case CategorySolidInfill:
			speed = config.InfillSpeed * 0.8
		case CategorySupport:
			speed = config.SupportSpeed
		case CategorySkirt, CategoryBrim:
			speed = config.WallSpeed
		default:
			speed = config.InfillSpeed
		}
	}

	if p.LayerIndex == 0 && config.FirstLayerSpeed > 0 {
		speed = math.Min(speed, config.FirstLayerSpeed)
	}

	return math.Max(speed, config.MinSpeed)
}
