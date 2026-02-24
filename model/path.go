package model

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
