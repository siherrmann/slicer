package handler

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/siherrmann/slicer"
	"github.com/siherrmann/slicer/core"
	"github.com/siherrmann/slicer/model"
	"github.com/siherrmann/slicer/view"
)

type ViewerHandler struct {
	Slicer    *slicer.Slicer
	LastPaths []model.ContinuousPath // Cache paths for export
}

func NewViewerHandler() *ViewerHandler {
	return &ViewerHandler{
		Slicer: slicer.NewSlicer(),
	}
}

func (h *ViewerHandler) HandleView(w http.ResponseWriter, r *http.Request) {
	// Load default model for demo
	file, err := os.Open("./example/Stanford_Bunny_sample.stl")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open model file: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	bm, err := h.Slicer.LoadSTLModel(file, "Stanford_Bunny_sample.stl")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load model: %v", err), http.StatusInternalServerError)
		return
	}

	h.Slicer.Model = bm.CleanSize().CleanPosition().CleanBottom(1).CleanPosition()

	// Default slicing for initial view
	paths, err := h.Slicer.GeneratePrintPaths(h.Slicer.Config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate paths: %v", err), http.StatusInternalServerError)
		return
	}

	h.LastPaths = paths

	component := view.STLViewer(h.Slicer.Model, paths, h.Slicer.Config)
	component.Render(r.Context(), w)
}

func (h *ViewerHandler) HandleSlice(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Ensure model is loaded (e.g. if server restarted but page is open)
	if h.Slicer.Model == nil {
		file, err := os.Open("./example/Stanford_Bunny_sample.stl")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to open default model file: %v", err), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		bm, err := h.Slicer.LoadSTLModel(file, "Stanford_Bunny_sample.stl")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load default model: %v", err), http.StatusInternalServerError)
			return
		}
		// Clean the model just like in HandleView
		h.Slicer.Model = bm.CleanSize().CleanPosition().CleanBottom(1).CleanPosition()
	}

	// Update config from form values
	config := h.Slicer.Config

	// Float64 parsing helper
	pf := func(key string, target *float64) {
		if val := r.FormValue(key); val != "" {
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				*target = f
			}
		}
	}
	// Int parsing helper
	pi := func(key string, target *int) {
		if val := r.FormValue(key); val != "" {
			if i, err := strconv.Atoi(val); err == nil {
				*target = i
			}
		}
	}

	// Layer & Shell settings
	pf("layer_height", &config.LayerHeight)
	pf("first_layer", &config.FirstLayer)
	pi("shell_count", &config.ShellCount)
	pf("line_width", &config.LineWidth)

	// Shell order
	if val := r.FormValue("shell_order"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			config.ShellOrder = model.ShellOrder(i)
		}
	}

	// Infill settings
	pf("infill_density", &config.InfillDensity)
	pf("infill_angle", &config.InfillAngle)
	if val := r.FormValue("infill_type"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			config.InfillType = model.InfillType(i)
		}
	}

	// Speed settings
	pf("infill_speed", &config.InfillSpeed)
	pf("wall_speed", &config.WallSpeed)
	pf("outer_shell_speed", &config.OuterShellSpeed)
	pf("travel_speed", &config.TravelSpeed)
	pf("first_layer_speed", &config.FirstLayerSpeed)
	pf("min_speed", &config.MinSpeed)

	// Extrusion
	pf("flow_multiplier", &config.FlowMultiplier)

	// Temperature & Cooling
	pf("nozzle_temp", &config.NozzleTemp)
	pf("bed_temp", &config.BedTemp)
	pi("fan_speed", &config.FanSpeed)
	pi("fan_on_layer", &config.FanOnLayer)
	pf("min_layer_time", &config.MinLayerTime)

	// Retraction
	pf("retraction_dist", &config.RetractionDist)
	pf("retraction_speed", &config.RetractionSpeed)
	pf("retraction_zhop", &config.RetractionZHop)
	pf("retraction_prime", &config.RetractionPrime)

	// Machine settings
	pf("nozzle_diameter", &config.NozzleDiameter)

	// G-code settings
	if val := r.FormValue("gcode_flavor"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			config.GCodeFlavor = model.GCodeFlavor(i)
		}
	}
	if val := r.FormValue("start_gcode"); val != "" {
		config.StartGCode = val
	}
	if val := r.FormValue("end_gcode"); val != "" {
		config.EndGCode = val
	}

	// Skirt/Brim
	pi("skirt_count", &config.SkirtCount)
	pf("skirt_offset", &config.SkirtOffset)
	pi("brim_count", &config.BrimCount)

	// Support settings
	if val := r.FormValue("support_type"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			config.SupportType = model.SupportType(i)
		}
	}
	pf("support_density", &config.SupportDensity)
	pf("support_angle", &config.SupportAngle)
	pf("support_z_gap", &config.SupportZGap)
	pf("support_xy_gap", &config.SupportXYGap)
	pf("support_speed", &config.SupportSpeed)
	pf("support_tree_branch_diameter", &config.SupportTreeBranchDiameter)
	pf("support_tree_trunk_diameter", &config.SupportTreeTrunkDiameter)

	// Raft settings
	pi("raft_layers", &config.RaftLayers)
	pf("raft_offset", &config.RaftOffset)

	// Vase mode (checkbox: value="true" when checked, absent when not)
	config.VaseMode = r.FormValue("vase_mode") == "true"

	// Build volume
	pf("build_volume_x", &config.BuildVolumeX)
	pf("build_volume_y", &config.BuildVolumeY)
	pf("build_volume_z", &config.BuildVolumeZ)

	// Clear previous slices so re-slicing uses the new config
	h.Slicer.Model.Slices = nil

	// Regenerate paths
	paths, err := h.Slicer.GeneratePrintPaths(config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Slicing failed: %v", err), http.StatusInternalServerError)
		return
	}

	h.LastPaths = paths

	// Render only the sidebar form + script tags — canvas is untouched
	component := view.SlicerSidebar(h.Slicer.Model, paths, config)
	component.Render(r.Context(), w)
}

// HandleExport generates G-code from the last sliced paths and returns it as a downloadable file.
func (h *ViewerHandler) HandleExport(w http.ResponseWriter, r *http.Request) {
	if len(h.LastPaths) == 0 {
		http.Error(w, "No sliced model available. Please slice first.", http.StatusBadRequest)
		return
	}

	gcode := core.GenerateGCode(h.LastPaths, *h.Slicer.Config)

	// Set headers for file download
	filename := "print.gcode"
	if h.Slicer.Model != nil {
		filename = h.Slicer.Model.Name + ".gcode"
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(gcode)))
	w.Write([]byte(gcode))
}
