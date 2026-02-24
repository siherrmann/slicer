package slicer

import (
	"fmt"
	"io"

	"github.com/siherrmann/slicer/core"
	"github.com/siherrmann/slicer/model"
)

type Slicer struct {
	Config *model.SliceConfig // Slicing configuration
	Model  *model.BaseModel   // Loaded model (STL or 3MF)
}

func NewSlicer() *Slicer {
	return &Slicer{
		Config: model.NewSliceConfig(), // Default config
	}
}

// NewSlicerWithConfig creates a slicer with custom configuration
func NewSlicerWithConfig(config *model.SliceConfig) *Slicer {
	return &Slicer{
		Config: config,
	}
}

func (s *Slicer) LoadSTLModel(r io.Reader, modelName string) (*model.BaseModel, error) {
	// Load the STL model
	stl, err := model.LoadSTL(r, modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to load STL model: %w", err)
	}
	s.Model = &stl.BaseModel

	return s.Model, nil
}

// Load3mfFile loads a 3MF file from a reader
func (s *Slicer) Load3mfFile(r io.Reader, modelName string) (*model.BaseModel, error) {
	// Open the 3MF XML file via reader
	bm, err := model.Load3MF(r, modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to load 3MF file: %w", err)
	}

	s.Model = bm
	if bm.SliceConfig != nil {
		s.Config = bm.SliceConfig
	}

	return s.Model, nil
}

// Slice slices a model into layers
func (s *Slicer) Slice(bm *model.BaseModel) ([]*model.Slice, error) {
	if bm == nil {
		return nil, fmt.Errorf("cannot slice nil model")
	}

	// Perform the slicing
	err := bm.Slice(s.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to slice model: %w", err)
	}

	return bm.Slices, nil
}

// GeneratePrintPaths processes all slices to generate perimeters, infill, and classify layers
func (s *Slicer) GeneratePrintPaths(config *model.SliceConfig) ([]model.ContinuousPath, error) {
	if s.Model == nil {
		return nil, fmt.Errorf("no model loaded")
	}

	// Step 1: Clean the model to ensure it's ready for slicing
	s.Model = s.Model.CleanBounds()
	s.Model.SliceConfig = config

	// Step 2: If the model hasn't been sliced yet, slice it now
	if len(s.Model.Slices) == 0 {
		err := s.Model.Slice(config)
		if err != nil {
			return nil, err
		}
	}

	s.Model.ClassifyTopBottomLayers(config)

	// Step 3: Generate full print paths for all layers (perimeters, infill, etc.)
	paths := core.GenerateFullSTLPath(s.Model, *config)

	// Step 4: Globally simplify all geometry to prevent massive JSON payload bloats.
	// RDP simplification annihilates collinear/dense points preserving rendering and print quality.
	var cleanPaths []model.ContinuousPath
	for _, p := range paths {
		if p.PathType == model.PathExtrusion && len(p.Segments) > 3 {
			// Convert single path to array and apply RDP
			merged := core.CleanFullPaths([]model.ContinuousPath{p}, 0.05)
			if len(merged.Segments) > 0 {
				cleanPaths = append(cleanPaths, merged)
			}
		} else {
			cleanPaths = append(cleanPaths, p)
		}
	}

	return cleanPaths, nil
}
