package core

import (
	"math"

	"github.com/siherrmann/slicer/model"
)

func GenerateInfill(bounds model.BoundingBox, shell *model.Polygon, params model.SliceConfig, layerIndex int, z float64) model.ContinuousPath {
	var infillPattern model.ContinuousPath
	switch params.InfillType {
	case model.InfillGrid:
		infillPattern = GenerateGridInfill(bounds, params, layerIndex, z)
	case model.InfillTriHexagon:
		infillPattern = GenerateTriHexagonInfill(bounds, params, layerIndex, z)
	case model.InfillCross:
		infillPattern = GenerateCrossInfill(bounds, params, layerIndex, z)
	case model.InfillHoneycombContinuous:
		infillPattern = GenerateHoneycombInfill(bounds, params, layerIndex, z)
	case model.InfillGyroid:
		infillPattern = GenerateGyroidInfill(bounds, params, layerIndex, z)
	// Full infill patterns for top/bottom layers (100% coverage with overlap)
	case model.InfillLineFull:
		infillPattern = GenerateLineInfillFull(bounds, params, layerIndex, z, 0)
	case model.InfillRectilinearFull:
		infillPattern = GenerateRectilinearInfillFull(bounds, params, layerIndex, z)
	case model.InfillConcentricFull:
		infillPattern = GenerateConcentricInfillFull(shell, params, layerIndex)
	default:
		infillPattern = GenerateLineInfill(bounds, params, layerIndex, z)
	}

	return infillPattern
}

// InfillGenerator is a function type that generates uncut infill patterns
// Patterns cover the bounding box plus margin to ensure all lines get cut
// Uses model bounds (not shell bounds) to ensure consistent alignment across layers
type InfillGenerator func(bounds model.BoundingBox, params model.SliceConfig, layerIndex int, z float64) model.ContinuousPath

// GenerateLineInfill creates a line infill pattern at the specified angle
// It shifts the lines by half spacing on alternate layers so they do not cross in the same layer
// and bonds with the previous layer correctly without creating a grid.
func GenerateLineInfill(bounds model.BoundingBox, params model.SliceConfig, layerIndex int, z float64) model.ContinuousPath {
	spacing := params.LineWidth / math.Max(0.01, params.InfillDensity)

	angleDeg := params.InfillAngle
	angleRad := angleDeg * math.Pi / 180.0

	var segments []model.PathSegment

	center := model.Vector3{
		X: (bounds.MinX + bounds.MaxX) / 2,
		Y: (bounds.MinY + bounds.MaxY) / 2,
		Z: z,
	}

	// Calculate required margin to cover bounding box when rotated
	width := bounds.MaxX - bounds.MinX
	height := bounds.MaxY - bounds.MinY
	radius := math.Sqrt(width*width+height*height)/2.0 + spacing*2.0

	// Shift half a spacing on alternate layers to bridge the lines
	offset := 0.0
	if layerIndex%2 == 1 {
		offset = spacing / 2.0
	}

	lineIndex := 0
	for y := center.Y - radius + offset; y <= center.Y+radius; y += spacing {
		var start, end model.Vector3

		// Zig-zag to minimize travel moves
		if lineIndex%2 == 0 {
			start = model.Vector3{X: center.X - radius, Y: y, Z: z}
			end = model.Vector3{X: center.X + radius, Y: y, Z: z}
		} else {
			start = model.Vector3{X: center.X + radius, Y: y, Z: z}
			end = model.Vector3{X: center.X - radius, Y: y, Z: z}
		}

		if angleRad != 0 {
			start = start.RotateAroundPoint(center, angleRad)
			end = end.RotateAroundPoint(center, angleRad)
		}

		segments = append(segments, model.PathSegment{
			Start:    start,
			End:      end,
			IsTravel: false,
			Category: model.CategoryInfill,
		})
		lineIndex++
	}

	return model.ContinuousPath{
		Segments: segments,
		PathType: model.PathExtrusion,
	}
}

// GenerateGridInfill creates a grid infill pattern (lines at 0° and 90°)
func GenerateGridInfill(bounds model.BoundingBox, params model.SliceConfig, layerIndex int, z float64) model.ContinuousPath {
	spacing := params.LineWidth / math.Max(0.01, params.InfillDensity)
	margin := spacing * 2.0

	var segments []model.PathSegment

	// Horizontal lines
	lineIndex := 0
	for y := bounds.MinY - margin; y <= bounds.MaxY+margin; y += spacing {
		var start, end model.Vector3
		if lineIndex%2 == 0 {
			start = model.Vector3{X: bounds.MinX - margin, Y: y, Z: z}
			end = model.Vector3{X: bounds.MaxX + margin, Y: y, Z: z}
		} else {
			start = model.Vector3{X: bounds.MaxX + margin, Y: y, Z: z}
			end = model.Vector3{X: bounds.MinX - margin, Y: y, Z: z}
		}
		segments = append(segments, model.PathSegment{
			Start:    start,
			End:      end,
			IsTravel: false,
			Category: model.CategoryInfill,
		})
		lineIndex++
	}

	// Vertical lines
	lineIndex = 0
	for x := bounds.MinX - margin; x <= bounds.MaxX+margin; x += spacing {
		var start, end model.Vector3
		if lineIndex%2 == 0 {
			start = model.Vector3{X: x, Y: bounds.MinY - margin, Z: z}
			end = model.Vector3{X: x, Y: bounds.MaxY + margin, Z: z}
		} else {
			start = model.Vector3{X: x, Y: bounds.MaxY + margin, Z: z}
			end = model.Vector3{X: x, Y: bounds.MinY - margin, Z: z}
		}
		segments = append(segments, model.PathSegment{
			Start:    start,
			End:      end,
			IsTravel: false,
			Category: model.CategoryInfill,
		})
		lineIndex++
	}

	return model.ContinuousPath{
		Segments: segments,
		PathType: model.PathExtrusion,
	}
}

// GenerateTriHexagonInfill creates a tri-hexagon infill pattern (lines at 0°, 60°, 120°)
func GenerateTriHexagonInfill(bounds model.BoundingBox, params model.SliceConfig, layerIndex int, z float64) model.ContinuousPath {
	spacing := params.LineWidth / math.Max(0.01, params.InfillDensity)
	margin := spacing * 2.0

	var segments []model.PathSegment

	// Three angles: 0°, 60°, 120°
	angles := []float64{0, math.Pi / 3.0, 2.0 * math.Pi / 3.0}

	for _, angle := range angles {
		// Rotate bounds to create lines at this angle
		// For simplicity, create lines in original orientation and rotate them
		for y := bounds.MinY - margin; y <= bounds.MaxY+margin; y += spacing {
			start := model.Vector3{X: bounds.MinX - margin, Y: y, Z: z}.Rotate(angle)
			end := model.Vector3{X: bounds.MaxX + margin, Y: y, Z: z}.Rotate(angle)
			segments = append(segments, model.PathSegment{
				Start:    start,
				End:      end,
				IsTravel: false,
			})
		}
	}

	return model.ContinuousPath{
		Segments: segments,
		PathType: model.PathExtrusion,
	}
}

// GenerateHoneycombInfill creates a continuous honeycomb zigzag pattern
// Pattern: \_/¯\_/ (angled down, horizontal, angled up, horizontal, repeat)
// Every second row is mirrored so horizontal parts touch each other
func GenerateHoneycombInfill(bounds model.BoundingBox, params model.SliceConfig, layerIndex int, z float64) model.ContinuousPath {
	spacing := params.LineWidth / math.Max(0.01, params.InfillDensity)

	// Honeycomb cell dimensions
	cellWidth := spacing * 2.0
	cellHeight := spacing * math.Sqrt(3) // Height of hexagon
	margin := cellWidth * 2.0

	var segments []model.PathSegment

	// Create rows of zigzag pattern
	rowNum := 0
	// Increase row spacing slightly to prevent overlap (add linewidth to spacing)
	adjustedSpacing := (cellHeight / 2.0) + (params.LineWidth / 2.0)

	for y := bounds.MinY - margin; y < bounds.MaxY+margin; y += adjustedSpacing {
		// Offset alternating rows by half cell width for honeycomb interlocking
		xStart := bounds.MinX - margin
		if rowNum%2 == 1 {
			xStart += cellWidth
		}

		var points []model.Vector3

		// Determine print direction for this row
		printLeftToRight := rowNum%2 == 0

		// Build zigzag pattern
		x := xStart
		yPos := y
		down := true

		for x < bounds.MaxX+margin+cellWidth*2 {
			if down {
				// Angled down segment
				points = append(points, model.Vector3{X: x, Y: yPos, Z: z})
				yPos += cellHeight / 2.0
				x += cellWidth / 2.0
				points = append(points, model.Vector3{X: x, Y: yPos, Z: z})

				// Horizontal segment
				x += cellWidth / 2.0
				points = append(points, model.Vector3{X: x, Y: yPos, Z: z})
			} else {
				// Angled up segment
				points = append(points, model.Vector3{X: x, Y: yPos, Z: z})
				yPos -= cellHeight / 2.0
				x += cellWidth / 2.0
				points = append(points, model.Vector3{X: x, Y: yPos, Z: z})

				// Horizontal segment
				x += cellWidth / 2.0
				points = append(points, model.Vector3{X: x, Y: yPos, Z: z})
			}
			down = !down
		}

		// Reverse points if printing right to left
		if !printLeftToRight {
			for i := 0; i < len(points)/2; i++ {
				points[i], points[len(points)-1-i] = points[len(points)-1-i], points[i]
			}
		}

		// Convert points to segments
		for i := 0; i < len(points)-1; i++ {
			segments = append(segments, model.PathSegment{
				Start:    points[i],
				End:      points[i+1],
				IsTravel: false,
			})
		}

		rowNum++
	}

	return model.ContinuousPath{
		Segments: segments,
		PathType: model.PathExtrusion,
	}
}

// GenerateCrossInfill creates a cross infill pattern
func GenerateCrossInfill(bounds model.BoundingBox, params model.SliceConfig, layerIndex int, z float64) model.ContinuousPath {
	var segments []model.PathSegment
	spacing := params.LineWidth / math.Max(0.01, params.InfillDensity)
	margin := spacing * 2.0

	for y := bounds.MinY - margin; y < bounds.MaxY+margin; y += spacing {
		for x := bounds.MinX - margin; x < bounds.MaxX+margin; x += spacing {
			// Erzeuge ein "X" oder "+" Muster innerhalb der Zelle
			points := []model.Vector3{
				{X: x, Y: y, Z: z},
				{X: x + spacing, Y: y + spacing, Z: z},
				{X: x + spacing, Y: y, Z: z},
				{X: x, Y: y + spacing, Z: z},
			}

			// Add both cross segments - cutting will handle boundaries
			segments = append(segments, model.PathSegment{
				Start:    points[0],
				End:      points[1],
				IsTravel: false,
			})
			segments = append(segments, model.PathSegment{
				Start:    points[2],
				End:      points[3],
				IsTravel: false,
			})
		}
	}

	return model.ContinuousPath{
		Segments: segments,
		PathType: model.PathExtrusion,
	}
}

// GenerateGyroidInfill creates a gyroid infill pattern based on the mathematical gyroid surface.
// The gyroid surface is defined by: sin(x)cos(y) + sin(y)cos(z) + sin(z)cos(x) = 0
// Implementation based on PrusaSlicer's FillGyroid algorithm.
// At each layer height z, the cross-section produces smooth sinusoidal waves that
// transition between horizontal and vertical orientations as z changes.
func GenerateGyroidInfill(bounds model.BoundingBox, params model.SliceConfig, layerIndex int, z float64) model.ContinuousPath {
	densityAdjusted := math.Max(0.01, params.InfillDensity)
	lineSpacing := params.LineWidth / densityAdjusted

	// Scale factor: the gyroid math works in a normalized coordinate system.
	// One full period is 2*PI in the normalized system.
	scaleFactor := lineSpacing / densityAdjusted

	// Tolerance for adaptive resolution (in model space)
	tolerance := math.Min(lineSpacing/2.0, 0.1) / scaleFactor

	// Compute z in normalized coordinates
	zNorm := z / scaleFactor
	zSin := math.Sin(zNorm)
	zCos := math.Cos(zNorm)

	// Determine if waves are primarily vertical or horizontal at this z
	vertical := math.Abs(zSin) <= math.Abs(zCos)

	// Compute width and height in normalized coordinates
	margin := lineSpacing * 2.0
	modelWidth := (bounds.MaxX - bounds.MinX) + 2*margin
	modelHeight := (bounds.MaxY - bounds.MinY) + 2*margin

	// Normalized dimensions (in units of 'distance' = lineSpacing/densityAdjusted)
	width := modelWidth / scaleFactor
	height := modelHeight / scaleFactor

	lowerBound := 0.0
	upperBound := height
	flip := true

	if vertical {
		flip = false
		lowerBound = -math.Pi
		upperBound = width - math.Pi/2.0
		width, height = height, width
	}

	// Generate one period of the wave for odd and even lines
	onePeriodOdd := gyroidMakeOnePeriod(width, zCos, zSin, vertical, flip, tolerance)
	flip = !flip
	onePeriodEven := gyroidMakeOnePeriod(width, zCos, zSin, vertical, flip, tolerance)

	// Generate all wave polylines
	type polyline struct {
		points []model.Vector3
	}
	var polylines []polyline

	originX := bounds.MinX - margin
	originY := bounds.MinY - margin

	for y0 := lowerBound; y0 < upperBound+1e-6; y0 += math.Pi {
		// Odd wave
		pts := gyroidMakeWave(onePeriodOdd, width, height, y0, scaleFactor, zCos, zSin, vertical, flip)
		if len(pts) > 1 {
			// Translate to model coordinates
			translated := make([]model.Vector3, len(pts))
			for i, p := range pts {
				translated[i] = model.Vector3{X: p.X + originX, Y: p.Y + originY, Z: z}
			}
			polylines = append(polylines, polyline{points: translated})
		}

		// Even wave
		y0 += math.Pi
		if y0 < upperBound+1e-6 {
			pts2 := gyroidMakeWave(onePeriodEven, width, height, y0, scaleFactor, zCos, zSin, vertical, flip)
			if len(pts2) > 1 {
				translated := make([]model.Vector3, len(pts2))
				for i, p := range pts2 {
					translated[i] = model.Vector3{X: p.X + originX, Y: p.Y + originY, Z: z}
				}
				polylines = append(polylines, polyline{points: translated})
			}
		}
	}

	// Convert polylines to segments with zigzag direction
	var segments []model.PathSegment
	for idx, pl := range polylines {
		pts := pl.points
		// Zigzag: reverse every other line for efficiency
		if idx%2 == 1 {
			for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
				pts[i], pts[j] = pts[j], pts[i]
			}
		}

		// Add travel move from last endpoint if needed
		if len(segments) > 0 && len(pts) > 0 {
			lastEnd := segments[len(segments)-1].End
			if lastEnd.Distance(pts[0]) > 1e-6 {
				segments = append(segments, model.PathSegment{
					Start:    lastEnd,
					End:      pts[0],
					IsTravel: true,
				})
			}
		}

		for i := 0; i < len(pts)-1; i++ {
			segments = append(segments, model.PathSegment{
				Start:    pts[i],
				End:      pts[i+1],
				IsTravel: false,
			})
		}
	}

	return model.ContinuousPath{
		Segments: segments,
		PathType: model.PathExtrusion,
	}
}

// gyroidF computes the y-value of the gyroid wave at position x for a given z.
// This is the core mathematical function derived from the gyroid implicit surface:
// sin(x)cos(y) + sin(y)cos(z) + sin(z)cos(x) = 0, solved for y.
func gyroidF(x, zSin, zCos float64, vertical, flip bool) float64 {
	if vertical {
		phaseOffset := math.Pi
		if zCos < 0 {
			phaseOffset += math.Pi
		}
		a := math.Sin(x + phaseOffset)
		b := -zCos
		flipOffset := 0.0
		if flip {
			flipOffset = math.Pi
		}
		res := zSin * math.Cos(x+phaseOffset+flipOffset)
		r := math.Sqrt(a*a + b*b)
		if r < 1e-10 {
			return math.Pi
		}
		// Clamp to [-1,1] to avoid NaN from asin
		return math.Asin(clamp(a/r, -1, 1)) + math.Asin(clamp(res/r, -1, 1)) + math.Pi
	}

	// Horizontal
	phaseOffset := 0.0
	if zSin < 0 {
		phaseOffset = math.Pi
	}
	a := math.Cos(x + phaseOffset)
	b := -zSin
	flipOffset := math.Pi
	if flip {
		flipOffset = 0.0
	}
	res := zCos * math.Sin(x+phaseOffset+flipOffset)
	r := math.Sqrt(a*a + b*b)
	if r < 1e-10 {
		return 0.5 * math.Pi
	}
	return math.Asin(clamp(a/r, -1, 1)) + math.Asin(clamp(res/r, -1, 1)) + 0.5*math.Pi
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// gyroidMakeOnePeriod generates one period of the gyroid wave with adaptive resolution.
// It starts with coarse samples at π/2 intervals, then adaptively subdivides
// until the cross-product tolerance is met.
func gyroidMakeOnePeriod(width, zCos, zSin float64, vertical, flip bool, tolerance float64) [][2]float64 {
	dx := math.Pi / 2.0
	limit := math.Min(2*math.Pi, width)

	var points [][2]float64

	// Initial coarse sampling
	for x := 0.0; x < limit-1e-10; x += dx {
		points = append(points, [2]float64{x, gyroidF(x, zSin, zCos, vertical, flip)})
	}
	points = append(points, [2]float64{limit, gyroidF(limit, zSin, zCos, vertical, flip)})

	// Adaptive refinement
	for {
		size := len(points)
		var newPoints [][2]float64
		for i := 1; i < size; i++ {
			lp := points[i-1]
			rp := points[i]
			x := lp[0] + (rp[0]-lp[0])/2.0
			y := gyroidF(x, zSin, zCos, vertical, flip)
			ip := [2]float64{x, y}

			// Cross product of (ip-lp) x (ip-rp) — tests deviation from straight line
			dx1 := ip[0] - lp[0]
			dy1 := ip[1] - lp[1]
			dx2 := ip[0] - rp[0]
			dy2 := ip[1] - rp[1]
			cross := math.Abs(dx1*dy2 - dy1*dx2)

			if cross > tolerance*tolerance {
				newPoints = append(newPoints, ip)
			}
		}

		if len(newPoints) == 0 {
			break
		}

		points = append(points, newPoints...)
		// Sort by x
		sortPoints(points)
	}

	return points
}

func sortPoints(points [][2]float64) {
	// Simple insertion sort — points are nearly sorted
	for i := 1; i < len(points); i++ {
		key := points[i]
		j := i - 1
		for j >= 0 && points[j][0] > key[0] {
			points[j+1] = points[j]
			j--
		}
		points[j+1] = key
	}
}

// gyroidMakeWave extends one period across the full width and returns model-space points.
func gyroidMakeWave(onePeriod [][2]float64, width, height, offset, scaleFactor, zCos, zSin float64, vertical, flip bool) []model.Vector3 {
	if len(onePeriod) == 0 {
		return nil
	}

	// Copy and extend the period across the width
	period := onePeriod[len(onePeriod)-1][0]
	var points [][2]float64

	if width > period+1e-6 {
		// Copy without last point (it will be the start of the next period)
		base := make([][2]float64, len(onePeriod)-1)
		copy(base, onePeriod[:len(onePeriod)-1])
		n := len(base)

		points = append(points, base...)
		for points[len(points)-1][0] < width-1e-6 {
			idx := len(points) - n
			if idx < 0 {
				break
			}
			points = append(points, [2]float64{
				points[idx][0] + period,
				points[idx][1],
			})
		}
		// Add final point at exact width
		points = append(points, [2]float64{width, gyroidF(width, zSin, zCos, vertical, flip)})
	} else {
		points = make([][2]float64, len(onePeriod))
		copy(points, onePeriod)
	}

	// Convert to model-space Vector3
	result := make([]model.Vector3, 0, len(points))
	for _, p := range points {
		px := p[0]
		py := p[1] + offset
		// Clamp y to [0, height]
		py = clamp(py, 0, height)

		if vertical {
			px, py = py, px
		}

		result = append(result, model.Vector3{
			X: px * scaleFactor,
			Y: py * scaleFactor,
			Z: 0, // Z will be set by the caller
		})
	}

	return result
}

// --- Solid Layer Infill Patterns (100% coverage with overlap) ---

// GenerateLineInfillFull creates a solid line infill pattern for top/bottom layers
// Uses overlap to ensure complete coverage
func GenerateLineInfillFull(bounds model.BoundingBox, params model.SliceConfig, layerIndex int, z float64, angle float64) model.ContinuousPath {
	// For solid layers, spacing is reduced by overlap to ensure no gaps
	spacing := params.LineWidth * (1.0 - params.InfillOverlap)

	var segments []model.PathSegment
	margin := spacing * 2.0

	for y := bounds.MinY - margin; y <= bounds.MaxY+margin; y += spacing {
		segments = append(segments, model.PathSegment{
			Start:    model.Vector3{X: bounds.MinX - margin, Y: y, Z: z},
			End:      model.Vector3{X: bounds.MaxX + margin, Y: y, Z: z},
			IsTravel: false,
		})
	}

	// If angle is specified, rotate all segments
	if angle != 0 {
		center := model.Vector3{
			X: (bounds.MinX + bounds.MaxX) / 2,
			Y: (bounds.MinY + bounds.MaxY) / 2,
			Z: z,
		}
		for i := range segments {
			segments[i].Start = segments[i].Start.RotateAroundPoint(center, angle)
			segments[i].End = segments[i].End.RotateAroundPoint(center, angle)
		}
	}

	return model.ContinuousPath{
		Segments: segments,
		PathType: model.PathExtrusion,
	}
}

// GenerateRectilinearInfillFull creates a rectilinear (alternating 0°/90°) solid infill
// Alternates direction per layer for better bonding
// Optimized to zigzag each line, eliminating travel moves
func GenerateRectilinearInfillFull(bounds model.BoundingBox, params model.SliceConfig, layerIndex int, z float64) model.ContinuousPath {
	spacing := params.LineWidth * (1.0 - params.InfillOverlap)
	margin := spacing * 2.0

	var segments []model.PathSegment

	// Determine direction based on layer index
	horizontal := layerIndex%2 == 0

	if horizontal {
		// Horizontal lines with alternating direction (zigzag)
		lineIndex := 0
		for y := bounds.MinY - margin; y <= bounds.MaxY+margin; y += spacing {
			// Alternate direction: even lines go left-to-right, odd lines go right-to-left
			if lineIndex%2 == 0 {
				segments = append(segments, model.PathSegment{
					Start:    model.Vector3{X: bounds.MinX - margin, Y: y, Z: z},
					End:      model.Vector3{X: bounds.MaxX + margin, Y: y, Z: z},
					IsTravel: false,
				})
			} else {
				segments = append(segments, model.PathSegment{
					Start:    model.Vector3{X: bounds.MaxX + margin, Y: y, Z: z},
					End:      model.Vector3{X: bounds.MinX - margin, Y: y, Z: z},
					IsTravel: false,
				})
			}
			lineIndex++
		}
	} else {
		// Vertical lines with alternating direction (zigzag)
		lineIndex := 0
		for x := bounds.MinX - margin; x <= bounds.MaxX+margin; x += spacing {
			// Alternate direction: even lines go bottom-to-top, odd lines go top-to-bottom
			if lineIndex%2 == 0 {
				segments = append(segments, model.PathSegment{
					Start:    model.Vector3{X: x, Y: bounds.MinY - margin, Z: z},
					End:      model.Vector3{X: x, Y: bounds.MaxY + margin, Z: z},
					IsTravel: false,
				})
			} else {
				segments = append(segments, model.PathSegment{
					Start:    model.Vector3{X: x, Y: bounds.MaxY + margin, Z: z},
					End:      model.Vector3{X: x, Y: bounds.MinY - margin, Z: z},
					IsTravel: false,
				})
			}
			lineIndex++
		}
	}

	return model.ContinuousPath{
		Segments: segments,
		PathType: model.PathExtrusion,
	}
}

// GenerateConcentricInfillFull creates concentric infill following the shell shape
// Creates inward offsets of the shell for complete coverage
// Note: This function still needs a shell polygon, not just bounds, so it takes the shell from context
func GenerateConcentricInfillFull(shell *model.Polygon, params model.SliceConfig, layerIndex int) model.ContinuousPath {
	spacing := params.LineWidth * (1.0 - params.InfillOverlap)
	var segments []model.PathSegment

	// Start from the shell and work inward
	currentPolygon := shell
	offset := spacing

	for currentPolygon != nil && len(currentPolygon.Points) > 2 {
		// Create segments for this concentric ring
		for j := 0; j < len(currentPolygon.Points); j++ {
			nextIdx := (j + 1) % len(currentPolygon.Points)
			segments = append(segments, model.PathSegment{
				Start:    currentPolygon.Points[j],
				End:      currentPolygon.Points[nextIdx],
				IsTravel: false,
			})
		}

		// Offset inward for next ring
		currentPolygon = currentPolygon.OffsetPolygon(-offset)
	}

	return model.ContinuousPath{
		Segments: segments,
		PathType: model.PathExtrusion,
	}
}
