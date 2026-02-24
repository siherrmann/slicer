package model

import "fmt"

type InfillType int

const (
	InfillLine InfillType = iota
	InfillGrid
	InfillTriHexagon
	InfillCross
	InfillHoneycombContinuous
	InfillGyroid
	// Full infills for top and bottom
	InfillLineFull
	InfillRectilinearFull
	InfillConcentricFull
)

// ShellOrder controls the order in which wall perimeters are printed
type ShellOrder int

const (
	ShellInsideOut ShellOrder = iota // Print inner walls first, then outer (default)
	ShellOutsideIn                   // Print outer wall first, then inner
)

// StartPointStrategy controls where each layer's printing starts
type StartPointStrategy int

const (
	StartPointNearest StartPointStrategy = iota // Start nearest to previous end position
	StartPointRandom                            // Random start point per layer
)

// GCodeFlavor selects the G-code dialect for export
type GCodeFlavor int

const (
	GCodeMarlin  GCodeFlavor = iota // Marlin / Prusa firmware
	GCodeRepRap                     // RepRap firmware
	GCodeKlipper                    // Klipper firmware
)

// SupportType controls whether support structures are generated
type SupportType int

const (
	SupportNone SupportType = iota // No support generation
	SupportAuto                    // Automatic linear support
	SupportTree                    // Tree/Stump support (widens at base)
)

// SupportPlacement defines where supports can be placed
type SupportPlacement int

const (
	SupportEverywhere SupportPlacement = iota
	SupportTouchingBuildplate
)

// PathCategory distinguishes path types for per-category speed/color/visibility
type PathCategory int

const (
	CategoryOuterWall   PathCategory = iota // Outermost perimeter
	CategoryInnerWall                       // Inner perimeters
	CategoryInfill                          // Sparse infill
	CategorySolidInfill                     // Top/bottom solid infill
	CategorySupport                         // Support structures (future)
	CategorySkirt                           // Skirt lines
	CategoryBrim                            // Brim lines
	CategoryTravel                          // Non-extrusion travel moves
)

// SliceConfig contains parameters for slicing
type SliceConfig struct {
	LayerHeight float64 `json:"layer_height"` // Height of each layer in mm
	FirstLayer  float64 `json:"first_layer"`  // Height of first layer (often thicker)
	Tolerance   float64 `json:"tolerance"`    // Tolerance for connecting segments

	// Wall/Perimeter settings
	WallThickness float64    `json:"wall_thickness"` // Total wall thickness in mm
	ShellCount    int        `json:"shell_count"`    // Number of perimeters
	LineWidth     float64    `json:"line_width"`     // Width of a single extrusion line in mm
	ShellOrder    ShellOrder `json:"shell_order"`    // Inside-out or outside-in wall order

	// Top/Bottom settings
	TopLayers       int        `json:"top_layers"`        // Number of solid top layers
	TopLayerType    InfillType `json:"top_layer_type"`    // Infill pattern for top layers
	BottomLayers    int        `json:"bottom_layers"`     // Number of solid bottom layers
	BottomLayerType InfillType `json:"bottom_layer_type"` // Infill pattern for bottom layers

	// Infill settings
	InfillDensity float64    `json:"infill_density"` // Infill density 0.0 to 1.0 (0% to 100%)
	InfillAngle   float64    `json:"infill_angle"`   // Angle of infill lines in degrees
	InfillType    InfillType `json:"infill_type"`    // Pattern: "lines", "grid", "triangles", "honeycomb"

	// Printer Speed settings (mm/s)
	InfillSpeed     float64 `json:"infill_speed"`
	WallSpeed       float64 `json:"wall_speed"`
	OuterShellSpeed float64 `json:"outer_shell_speed"` // Slower outer perimeter speed
	TravelSpeed     float64 `json:"travel_speed"`
	FirstLayerSpeed float64 `json:"first_layer_speed"` // Speed for first layer
	MinSpeed        float64 `json:"min_speed"`         // Minimum print speed (for min layer time)

	// Extrusion settings
	FlowMultiplier float64 `json:"flow_multiplier"` // Extrusion multiplier (1.0 = 100%)

	// Retraction settings
	RetractionDist  float64 `json:"retraction_dist"`  // mm
	RetractionSpeed float64 `json:"retraction_speed"` // mm/s
	RetractionZHop  float64 `json:"retraction_zhop"`  // Z lift on retract (mm)
	RetractionPrime float64 `json:"retraction_prime"` // Extra prime after unretract (mm)

	// Cooling settings
	FanSpeed     int     `json:"fan_speed"`      // 0-255
	FanOnLayer   int     `json:"fan_on_layer"`   // Layer number when fan activates
	MinLayerTime float64 `json:"min_layer_time"` // Min seconds per layer (slows down if too fast)

	// Skirt/Brim settings
	SkirtCount  int     `json:"skirt_count"`  // Number of skirt lines
	SkirtOffset float64 `json:"skirt_offset"` // Distance from model (mm)
	BrimCount   int     `json:"brim_count"`   // Number of brim lines (directly attached)

	// Support settings
	SupportType               SupportType      `json:"support_type"`                 // None or Auto
	SupportPlacement          SupportPlacement `json:"support_placement"`            // Everywhere or Touching Buildplate
	SupportDensity            float64          `json:"support_density"`              // 0.0-1.0 density of support infill
	SupportAngle              float64          `json:"support_angle"`                // Overhang threshold in degrees
	SupportZGap               float64          `json:"support_z_gap"`                // Gap between support top and model (mm)
	SupportXYGap              float64          `json:"support_xy_gap"`               // XY gap between support and model (mm)
	SupportSpeed              float64          `json:"support_speed"`                // Print speed for support (mm/s)
	SupportInterfaceLayers    int              `json:"support_interface_layers"`     // Number of interface layers
	SupportTreeBranchDiameter float64          `json:"support_tree_branch_diameter"` // Diameter of the spawned tips (mm)
	SupportTreeTrunkDiameter  float64          `json:"support_tree_trunk_diameter"`  // Max diameter of merged trunk (mm)

	// Raft settings
	RaftLayers int     `json:"raft_layers"` // Number of raft layers (0 = disabled)
	RaftOffset float64 `json:"raft_offset"` // Extra offset around model for raft (mm)

	// Vase mode
	VaseMode bool `json:"vase_mode"` // Spiral single-wall mode

	// Machine settings
	PrinterModel   string             `json:"printer_model"`   // Target printer model
	NozzleDiameter float64            `json:"nozzle_diameter"` // Nozzle diameter in mm
	StartPoint     StartPointStrategy `json:"start_point"`     // Seam placement strategy
	BuildVolumeX   float64            `json:"build_volume_x"`  // Build volume X in mm
	BuildVolumeY   float64            `json:"build_volume_y"`  // Build volume Y in mm
	BuildVolumeZ   float64            `json:"build_volume_z"`  // Build volume Z in mm

	// Material settings
	MaterialName  string  `json:"material_name"`  // e.g. PLA, ABS
	MaterialColor string  `json:"material_color"` // HEX or name
	NozzleTemp    float64 `json:"nozzle_temp"`
	BedTemp       float64 `json:"bed_temp"`

	// G-code settings
	GCodeFlavor GCodeFlavor `json:"gcode_flavor"` // G-code dialect
	StartGCode  string      `json:"start_gcode"`  // Custom start G-code
	EndGCode    string      `json:"end_gcode"`    // Custom end G-code

	// Positions
	StartPosition Vector3 `json:"start_position"` // Starting position for printing
	EndPosition   Vector3 `json:"end_position"`   // Ending position for printing

	// Advanced / Slicing Details
	InfillOverlap       float64 `json:"infill_overlap"`         // Overlap percentage for infill (0.0 to 1.0)
	FirstLayerLineWidth float64 `json:"first_layer_line_width"` // Line width for first layer in mm
}

// DefaultStartGCode returns template start G-code for Marlin-compatible printers
func DefaultStartGCode(config *SliceConfig) string {
	return "; Start G-code\n" +
		"G28 ; Home all axes\n" +
		"G92 E0 ; Reset extruder\n" +
		"G1 Z5 F3000 ; Lift nozzle\n" +
		"M104 S" + floatStr(config.NozzleTemp) + " ; Set nozzle temp\n" +
		"M140 S" + floatStr(config.BedTemp) + " ; Set bed temp\n" +
		"M190 S" + floatStr(config.BedTemp) + " ; Wait for bed temp\n" +
		"M109 S" + floatStr(config.NozzleTemp) + " ; Wait for nozzle temp\n" +
		"G1 Z0.3 F3000 ; Move to start height\n" +
		"G92 E0 ; Reset extruder\n"
}

// DefaultEndGCode returns template end G-code for Marlin-compatible printers
func DefaultEndGCode() string {
	return "; End G-code\n" +
		"G91 ; Relative positioning\n" +
		"G1 E-2 F2700 ; Retract\n" +
		"G1 Z10 F3000 ; Lift nozzle\n" +
		"G90 ; Absolute positioning\n" +
		"G28 X Y ; Home X and Y\n" +
		"M104 S0 ; Turn off nozzle\n" +
		"M140 S0 ; Turn off bed\n" +
		"M106 S0 ; Turn off fan\n" +
		"M84 ; Disable motors\n"
}

// floatStr formats a float64 to a clean string for G-code
func floatStr(f float64) string {
	s := fmt.Sprintf("%.1f", f)
	return s
}

// NewSliceConfig creates a default slicing configuration
func NewSliceConfig() *SliceConfig {
	config := &SliceConfig{
		LayerHeight:               0.2,    // 0.2mm standard layer height
		FirstLayer:                0.3,    // 0.3mm first layer
		Tolerance:                 0.0001, // 0.1 micron tolerance
		WallThickness:             1.2,    // 1.2mm total width
		ShellCount:                3,      // 3 perimeters
		LineWidth:                 0.4,    // 0.4mm standard nozzle width
		ShellOrder:                ShellInsideOut,
		TopLayers:                 4,    // 4 solid top layers
		BottomLayers:              3,    // 3 solid bottom layers
		InfillDensity:             0.2,  // 20% infill
		InfillAngle:               45.0, // 45 degree infill angle
		InfillType:                InfillLine,
		InfillSpeed:               60.0,  // 60 mm/s
		WallSpeed:                 40.0,  // 40 mm/s
		OuterShellSpeed:           25.0,  // 25 mm/s (slower for quality)
		TravelSpeed:               120.0, // 120 mm/s
		FirstLayerSpeed:           20.0,  // 20 mm/s (slow first layer)
		MinSpeed:                  10.0,  // 10 mm/s minimum
		FlowMultiplier:            1.0,   // 100% flow
		NozzleTemp:                200.0, // 200°C (PLA)
		BedTemp:                   60.0,  // 60°C
		RetractionDist:            5.0,   // 5mm
		RetractionSpeed:           40.0,  // 40 mm/s
		RetractionZHop:            0.0,   // No Z-hop by default
		RetractionPrime:           0.0,   // No extra prime by default
		FanSpeed:                  255,   // Full fan speed
		FanOnLayer:                2,     // Fan on from layer 2
		MinLayerTime:              5.0,   // 5 seconds minimum
		SkirtCount:                3,     // 3 skirt lines
		SkirtOffset:               3.0,   // 3mm from model
		BrimCount:                 0,     // Default to no brim
		SupportType:               SupportTree,
		SupportPlacement:          SupportEverywhere,
		SupportDensity:            0.15, // 15% support infill
		SupportAngle:              45.0, // 45 degree overhang threshold
		SupportZGap:               0.2,  // 0.2mm gap at top of support
		SupportXYGap:              0.4,  // 0.4mm XY gap
		SupportSpeed:              40.0, // 40 mm/s for support
		SupportInterfaceLayers:    3,
		SupportTreeBranchDiameter: 2.0,
		SupportTreeTrunkDiameter:  12.0,
		RaftLayers:                0,     // No raft by default
		RaftOffset:                3.0,   // 3mm raft offset
		VaseMode:                  false, // Vase mode off by default
		PrinterModel:              "Generic FDM",
		NozzleDiameter:            0.4, // 0.4mm standard nozzle
		StartPoint:                StartPointNearest,
		BuildVolumeX:              220.0, // 220mm X
		BuildVolumeY:              220.0, // 220mm Y
		BuildVolumeZ:              250.0, // 250mm Z
		MaterialName:              "PLA",
		MaterialColor:             "#FFFFFF",
		GCodeFlavor:               GCodeMarlin,
		StartPosition:             Vector3{X: 0, Y: 0, Z: 1000},
		EndPosition:               Vector3{X: 0, Y: 0, Z: 1000},
		InfillOverlap:             0.15, // 15% overlap
		FirstLayerLineWidth:       0.4,
	}
	config.StartGCode = DefaultStartGCode(config)
	config.EndGCode = DefaultEndGCode()
	return config
}

// CalculateLayerHeights computes the Z heights for each layer
func (s *SliceConfig) CalculateLayerHeights(minZ, maxZ float64) []float64 {
	heights := make([]float64, 0)

	// Start with first layer
	z := minZ + s.FirstLayer
	heights = append(heights, z)

	// Add remaining layers
	for z < maxZ {
		z += s.LayerHeight
		if z <= maxZ {
			heights = append(heights, z)
		}
	}

	return heights
}
