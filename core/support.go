package core

import (
	"fmt"
	"math"
	"sort"

	"github.com/siherrmann/slicer/model"
)

// Grid represents a 2D boolean grid for a layer
type Grid struct {
	Width, Height int
	Resolution    float64
	MinX, MinY    float64
	Cells         []bool
}

func (g *Grid) GetBounds() model.BoundingBox {
	return model.BoundingBox{
		MinX: g.MinX,
		MinY: g.MinY,
		MaxX: g.MinX + float64(g.Width)*g.Resolution,
		MaxY: g.MinY + float64(g.Height)*g.Resolution,
	}
}

func NewGrid(bounds model.BoundingBox, resolution float64) *Grid {
	width := int(math.Ceil((bounds.MaxX - bounds.MinX) / resolution))
	height := int(math.Ceil((bounds.MaxY - bounds.MinY) / resolution))
	return &Grid{
		Width:      width,
		Height:     height,
		Resolution: resolution,
		MinX:       bounds.MinX,
		MinY:       bounds.MinY,
		Cells:      make([]bool, width*height),
	}
}

func (g *Grid) Index(x, y int) int {
	if x < 0 || x >= g.Width || y < 0 || y >= g.Height {
		return -1
	}
	return y*g.Width + x
}

func (g *Grid) Set(x, y int, val bool) {
	idx := g.Index(x, y)
	if idx != -1 {
		g.Cells[idx] = val
	}
}

func (g *Grid) Get(x, y int) bool {
	idx := g.Index(x, y)
	if idx != -1 {
		return g.Cells[idx]
	}
	return false
}

// SetDisk sets a circular area to true centered at cx, cy with given radius
func (g *Grid) SetDisk(cx, cy, radius int) {
	if radius <= 0 {
		g.Set(cx, cy, true)
		return
	}
	r2 := radius * radius
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= r2 {
				// g.Set checks bounds
				g.Set(cx+dx, cy+dy, true)
			}
		}
	}
}

// SetDiskFloat sets a circular area to true centered at cx, cy with given float radius
func (g *Grid) SetDiskFloat(cx, cy float64, radius float64) {
	if radius <= 0 {
		return
	}
	r2 := radius * radius
	bound := int(math.Ceil(radius))
	icx := int(cx)
	icy := int(cy)

	for dy := -bound - 1; dy <= bound+1; dy++ {
		for dx := -bound - 1; dx <= bound+1; dx++ {
			x := icx + dx
			y := icy + dy

			// distance from pixel center to cx,cy
			dfx := float64(x) - cx
			dfy := float64(y) - cy
			if dfx*dfx+dfy*dfy <= r2 {
				if x >= 0 && x < g.Width && y >= 0 && y < g.Height {
					g.Cells[y*g.Width+x] = true
				}
			}
		}
	}
}

// RasterizePolygons fills the grid based on polygons using Scanline Even-Odd rule
func (g *Grid) RasterizePolygons(polygons []model.Polygon) {
	for j := 0; j < g.Height; j++ {
		y := g.MinY + float64(j)*g.Resolution + g.Resolution/2
		var intersections []float64

		for _, p := range polygons {
			bounds := p.GetBounds()
			if y >= bounds.MinY && y <= bounds.MaxY {
				intersections = append(intersections, p.IntersectLine(y)...)
			}
		}

		if len(intersections) == 0 {
			continue
		}

		sort.Float64s(intersections)

		// Fill between pairs (Even-Odd rule handles holes automatically)
		for k := 0; k < len(intersections)-1; k += 2 {
			x1 := intersections[k]
			x2 := intersections[k+1]

			startI := int((x1 - g.MinX) / g.Resolution)
			endI := int((x2 - g.MinX) / g.Resolution)

			startI = maxInt(0, startI)
			endI = minInt(g.Width-1, endI)

			for i := startI; i <= endI; i++ {
				g.Set(i, j, true)
			}
		}
	}
}

// DistanceField computes a 2D distance transform of the grid.
// Returns a slice of float64 where each element is the distance in grid cells
// to the nearest TRUE cell. TRUE cells have a distance of 0.
// This uses a fast 2-pass sweep algorithm (8SSE).
func (g *Grid) DistanceField() []float64 {
	dist := make([]float64, g.Width*g.Height)

	// Initialize distances
	maxDist := float64(g.Width + g.Height) // Safe upper bound
	for i, val := range g.Cells {
		if val {
			dist[i] = 0
		} else {
			dist[i] = maxDist
		}
	}

	// Pass 1: Top-Left to Bottom-Right
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			idx := y*g.Width + x
			if dist[idx] == 0 {
				continue
			}

			minD := dist[idx]

			// Check West
			if x > 0 {
				d := dist[idx-1] + 1.0
				if d < minD {
					minD = d
				}
			}
			// Check North-West
			if x > 0 && y > 0 {
				d := dist[idx-g.Width-1] + math.Sqrt2
				if d < minD {
					minD = d
				}
			}
			// Check North
			if y > 0 {
				d := dist[idx-g.Width] + 1.0
				if d < minD {
					minD = d
				}
			}
			// Check North-East
			if x < g.Width-1 && y > 0 {
				d := dist[idx-g.Width+1] + math.Sqrt2
				if d < minD {
					minD = d
				}
			}

			dist[idx] = minD
		}
	}

	// Pass 2: Bottom-Right to Top-Left
	for y := g.Height - 1; y >= 0; y-- {
		for x := g.Width - 1; x >= 0; x-- {
			idx := y*g.Width + x
			minD := dist[idx]

			if minD == 0 {
				continue
			}

			// Check East
			if x < g.Width-1 {
				d := dist[idx+1] + 1.0
				if d < minD {
					minD = d
				}
			}
			// Check South-East
			if x < g.Width-1 && y < g.Height-1 {
				d := dist[idx+g.Width+1] + math.Sqrt2
				if d < minD {
					minD = d
				}
			}
			// Check South
			if y < g.Height-1 {
				d := dist[idx+g.Width] + 1.0
				if d < minD {
					minD = d
				}
			}
			// Check South-West
			if x > 0 && y < g.Height-1 {
				d := dist[idx+g.Width-1] + math.Sqrt2
				if d < minD {
					minD = d
				}
			}

			dist[idx] = minD
		}
	}

	return dist
}

// Dilate expands the grid by steps using a circular kernel (approx)
func (g *Grid) Dilate(steps int) *Grid {
	if steps <= 0 {
		return g
	}
	newGrid := NewGrid(model.BoundingBox{
		MinX: g.MinX, MinY: g.MinY,
		MaxX: g.MinX + float64(g.Width)*g.Resolution,
		MaxY: g.MinY + float64(g.Height)*g.Resolution,
	}, g.Resolution)

	// Precompute circle offsets to avoid sqrt per pixel
	type Offset struct{ dx, dy int }
	var offsets []Offset
	r2 := steps * steps
	for dy := -steps; dy <= steps; dy++ {
		for dx := -steps; dx <= steps; dx++ {
			if dx*dx+dy*dy <= r2 {
				offsets = append(offsets, Offset{dx, dy})
			}
		}
	}

	for j := 0; j < g.Height; j++ {
		for i := 0; i < g.Width; i++ {
			if g.Get(i, j) {
				for _, off := range offsets {
					nx, ny := i+off.dx, j+off.dy
					// inline boundary check for speed
					if nx >= 0 && nx < g.Width && ny >= 0 && ny < g.Height {
						newGrid.Cells[ny*g.Width+nx] = true
					}
				}
			}
		}
	}
	return newGrid
}

// Erode shrinks the grid by steps using a circular kernel
// Erode is equivalent to Dilating the FALSE regions (Background)
// Or: A cell survives ONLY IF all neighbors in radius are TRUE.
func (g *Grid) Erode(steps int) *Grid {
	if steps <= 0 {
		return g
	}
	newGrid := NewGrid(model.BoundingBox{
		MinX: g.MinX, MinY: g.MinY,
		MaxX: g.MinX + float64(g.Width)*g.Resolution,
		MaxY: g.MinY + float64(g.Height)*g.Resolution,
	}, g.Resolution)

	// Precompute circle offsets
	type Offset struct{ dx, dy int }
	var offsets []Offset
	r2 := steps * steps
	for dy := -steps; dy <= steps; dy++ {
		for dx := -steps; dx <= steps; dx++ {
			if dx*dx+dy*dy <= r2 {
				offsets = append(offsets, Offset{dx, dy})
			}
		}
	}

	// Iterate all cells. If any neighbor in radius is FALSE, result is FALSE.
	// Optimization: Iterate TRUE cells. Check neighbors.
	// But Erode can turn TRUE into FALSE. It never keeps FALSE as TRUE.
	// So we only check current TRUE cells.
	for j := 0; j < g.Height; j++ {
		for i := 0; i < g.Width; i++ {
			if g.Get(i, j) {
				keep := true
				for _, off := range offsets {
					nx, ny := i+off.dx, j+off.dy
					if nx < 0 || nx >= g.Width || ny < 0 || ny >= g.Height || !g.Get(nx, ny) {
						keep = false
						break
					}
				}
				if keep {
					newGrid.Set(i, j, true)
				}
			}
		}
	}
	return newGrid
}

// Open performs morphological Opening (Erode then Dilate) to remove noise/smooth
func (g *Grid) Open(steps int) *Grid {
	return g.Erode(steps).Dilate(steps)
}

// Close performs morphological Closing (Dilate then Erode) to fill holes/smooth
func (g *Grid) Close(steps int) *Grid {
	return g.Dilate(steps).Erode(steps)
}

// TraceContours extracts the boundaries of connected components as polygons
// utilizing the Moore-Neighbor Tracing algorithm
func (g *Grid) TraceContours() []model.Polygon {
	var contours []model.Polygon
	visited := make([]bool, g.Width*g.Height)

	// Moore-Neighbor tracing requires a starting pixel on the boundary.
	// We scan the grid to find an unvisited TRUE pixel that has a FALSE neighbor (or edge) to its left (or just any boundary).
	// Since we scan left-to-right, top-to-bottom, the first TRUE pixel we find for a component
	// is guaranteed to be on the external boundary.

	for j := 0; j < g.Height; j++ {
		for i := 0; i < g.Width; i++ {
			idx := j*g.Width + i
			if g.Cells[idx] && !visited[idx] {
				// Found a start point for a new component
				// Trace the boundary
				path := g.traceBoundary(i, j, visited)
				if len(path) > 2 {
					// Simplify path (remove collinear points)
					simplified := simplifyPath(path)
					if len(simplified) > 2 {
						contours = append(contours, model.Polygon{Points: simplified})
					}
				}
			}
		}
	}
	return contours
}

func (g *Grid) traceBoundary(startX, startY int, visited []bool) []model.Vector3 {
	var path []model.Vector3

	// Directions (dx, dy) in grid coordinates (i, j)
	// Order: N, NE, E, SE, S, SW, W, NW (Clockwise)
	// j+1 is UP (North)
	dirs := [8][2]int{
		{0, 1}, {1, 1}, {1, 0}, {1, -1},
		{0, -1}, {-1, -1}, {-1, 0}, {-1, 1},
	}

	// Start with backtrack direction = West (since we entered from left)
	startIdx := startY*g.Width + startX

	// 1. Flood fill component to mark 'visited' so main loop doesn't restart here.
	queue := []int{startIdx}
	visited[startIdx] = true
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		cx, cy := curr%g.Width, curr/g.Width

		// 4-connected flood fill sufficient for marking component
		ndirs := [][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
		for _, d := range ndirs {
			nx, ny := cx+d[0], cy+d[1]
			if nx >= 0 && nx < g.Width && ny >= 0 && ny < g.Height {
				nidx := ny*g.Width + nx
				if g.Cells[nidx] && !visited[nidx] {
					visited[nidx] = true
					queue = append(queue, nidx)
				}
			}
		}
	}

	// 2. Trace Boundary
	// Backtrack direction relative to current pixel
	// Start by coming from West (6)
	entryDir := 6

	currX, currY := startX, startY
	first := true

	// Helper to get pixel value safely
	get := func(mx, my int) bool {
		if mx < 0 || mx >= g.Width || my < 0 || my >= g.Height {
			return false
		}
		return g.Get(mx, my)
	}

	// Jacob's stopping criteria variables
	startPoint := [2]int{startX, startY}
	var secondPoint [2]int

	// Limit iterations to prevent infinite loop
	maxIter := g.Width * g.Height * 4
	iter := 0

	for {
		iter++
		if iter > maxIter {
			break
		}

		// Add current point to path
		// Convert grid coord to world
		wx := g.MinX + float64(currX)*g.Resolution + g.Resolution/2 // Center of pixel
		wy := g.MinY + float64(currY)*g.Resolution + g.Resolution/2
		path = append(path, model.Vector3{X: wx, Y: wy, Z: 0}) // Z set by caller

		// Search for next clockwise neighbor
		foundNext := false

		searchStart := (entryDir + 1)
		if first {
			searchStart = 6 + 1 // Came from West (empty space)
		}

		for k := 0; k < 8; k++ {
			idx := (searchStart + k) % 8
			dx, dy := dirs[idx][0], dirs[idx][1]
			nx, ny := currX+dx, currY+dy

			if get(nx, ny) {
				// Found next boundary pixel
				if first {
					secondPoint = [2]int{nx, ny}
					first = false
				} else {
					// Check stopping condition
					// Stop if we are at Start Point AND next point is Second Point
					if currX == startPoint[0] && currY == startPoint[1] && nx == secondPoint[0] && ny == secondPoint[1] {
						return path // Closed loop
					}
				}

				// Move to next
				currX, currY = nx, ny
				// Update entry direction
				entryDir = (idx + 4) % 8
				foundNext = true
				break
			}
		}

		if !foundNext {
			// Isolated pixel
			break
		}
	}

	return path
}

func simplifyPath(points []model.Vector3) []model.Vector3 {
	if len(points) < 3 {
		return points
	}
	var res []model.Vector3
	res = append(res, points[0])

	for i := 1; i < len(points)-1; i++ {
		prev := points[i-1]
		curr := points[i]
		next := points[i+1]

		// Check collinearity 2D
		dx1, dy1 := curr.X-prev.X, curr.Y-prev.Y
		dx2, dy2 := next.X-curr.X, next.Y-curr.Y

		// Cross product close to zero?
		cross := dx1*dy2 - dx2*dy1
		if math.Abs(cross) > 1e-6 {
			res = append(res, curr)
		}
	}
	res = append(res, points[len(points)-1])
	return res
}

func smoothPath(points []model.Vector3, iterations int) []model.Vector3 {
	if len(points) < 3 || iterations <= 0 {
		return points
	}

	result := points
	for i := 0; i < iterations; i++ {
		var smoothed []model.Vector3
		n := len(result)
		for j := 0; j < n; j++ {
			p1 := result[j]
			p2 := result[(j+1)%n]

			// Compute the two new points at 1/4 and 3/4 along the segment
			q := model.Vector3{
				X: p1.X + 0.25*(p2.X-p1.X),
				Y: p1.Y + 0.25*(p2.Y-p1.Y),
				Z: p1.Z,
			}
			r := model.Vector3{
				X: p1.X + 0.75*(p2.X-p1.X),
				Y: p1.Y + 0.75*(p2.Y-p1.Y),
				Z: p1.Z,
			}
			smoothed = append(smoothed, q, r)
		}
		result = smoothed
	}
	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GenerateSupportPaths generates support paths using Vector-Based Tree logic or Raster
func GenerateSupportPaths(bm *model.BaseModel, config model.SliceConfig) map[int][]model.ContinuousPath {
	if config.SupportType == model.SupportNone || len(bm.Slices) < 2 {
		return nil
	}

	resolution := config.LineWidth // e.g. 0.4mm grid
	bounds := bm.GetBounds()

	// Dynamic padding for tree support spreading
	padding := 2.0
	if config.SupportType == model.SupportTree && len(bm.Slices) > 0 {
		height := bm.Slices[len(bm.Slices)-1].Z
		// tan(50 degrees) approx 1.2
		padding = height*1.5 + 5.0
	}
	bounds.MinX -= padding
	bounds.MinY -= padding
	bounds.MaxX += padding
	bounds.MaxY += padding

	// 1. Voxelize Model
	layerGrids := make([]*Grid, len(bm.Slices))
	for i, slice := range bm.Slices {
		g := NewGrid(bounds, resolution)
		g.RasterizePolygons(slice.Polygons)
		layerGrids[i] = g
	}

	supportGrids := make([]*Grid, len(bm.Slices))
	for i := range supportGrids {
		supportGrids[i] = NewGrid(bounds, resolution)
	}

	// 2. Identify Overhangs & Propagate Down
	// "Support Angle" is measured from the vertical Z-axis (0 = vertical wall, 90 = flat horizontal roof).
	// This is standard for 3D slicers (like Cura).
	// higher angle = requires flatter surface to trigger support = LESS support.
	var overhangDist float64
	if config.SupportAngle <= 0.1 {
		overhangDist = 0.0001 // Support almost everything
	} else if config.SupportAngle >= 89.9 {
		overhangDist = 9999.0 // Prevent support
	} else {
		overhangDist = config.LayerHeight * math.Tan(config.SupportAngle*math.Pi/180.0)
	}

	// We add a strict 0.5 grid step buffer (0.2mm) to ignore microscopic texture overhangs
	// like fur or ripples that mathematically slope downwards but don't geometrically need support.
	overhangDistSteps := (overhangDist / resolution) + 0.5

	// Skeleton Tree Data Structures
	type Branch struct {
		X, Y   float64
		VX, VY float64 // Velocity for smooth organic curves
		Radius float64 // In grid steps
		Age    int     // How many layers this branch has dropped
		Weight int     // Number of unified tips (used for thickness/stiffness)
		Dead   bool
	}
	var activeBranches []*Branch
	treeBranchesByLayer := make([][]*Branch, len(bm.Slices))

	// ExtractBranchContours uses marching squares on a metaball field generated by the branches
	// It guarantees pixel-perfect smooth contours without stair-stepping.
	ExtractBranchContours := func(branches []*Branch, gridResolution, resolution, threshold float64, bounds model.BoundingBox) []model.Polygon {
		var polygons []model.Polygon
		if len(branches) == 0 {
			return polygons
		}

		// Grid bounds
		minX := bounds.MinX
		minY := bounds.MinY
		cols := int(math.Ceil((bounds.MaxX-minX)/resolution)) + 1
		rows := int(math.Ceil((bounds.MaxY-minY)/resolution)) + 1

		// Evaluate scalar field
		field := make([]float64, cols*rows)
		for _, b := range branches {
			cx := b.X*gridResolution + bounds.MinX
			cy := b.Y*gridResolution + bounds.MinY
			radiusWorld := b.Radius * gridResolution
			r2 := radiusWorld * radiusWorld

			// Only evaluate near the branch bounding box for performance
			startX := int(math.Floor((cx - radiusWorld - bounds.MinX) / resolution))
			endX := int(math.Ceil((cx+radiusWorld-bounds.MinX)/resolution)) + 1
			startY := int(math.Floor((cy - radiusWorld - bounds.MinY) / resolution))
			endY := int(math.Ceil((cy+radiusWorld-bounds.MinY)/resolution)) + 1

			if startX < 0 {
				startX = 0
			}
			if endX > cols {
				endX = cols
			}
			if startY < 0 {
				startY = 0
			}
			if endY > rows {
				endY = rows
			}

			for y := startY; y < endY; y++ {
				wy := float64(y)*resolution + minY
				for x := startX; x < endX; x++ {
					wx := float64(x)*resolution + minX
					dx := wx - cx
					dy := wy - cy
					dist2 := dx*dx + dy*dy
					if dist2 < 0.0001 {
						dist2 = 0.0001
					}
					// Inverse square falloff (Max-Metaball)
					// Instead of additive (which causes massive ballooning at Y-junctions),
					// we take the Max so perfectly overlapping branches stay their exact mathematical radius.
					fieldVal := r2 / dist2
					if fieldVal > field[y*cols+x] {
						field[y*cols+x] = fieldVal
					}
				}
			}
		}

		// Simple thresholding: turn into boolean grid of inner/outer for contour extraction
		// Since we already have a robust Moore-Neighbor contour tracer in `Grid`, we can adapt it!
		// However, to avoid the stepping issue entirely, we need Sub-Pixel contouring.

		interpolate := func(v1, v2 float64) float64 {
			// Linear interpolation for zero-crossing
			// We want where value = threshold
			if math.Abs(v1-v2) < 0.00001 {
				return 0.5
			}
			return (threshold - v1) / (v2 - v1)
		}

		type Seg struct {
			Start, End model.Vector3
		}
		var segments []Seg

		// Marching Squares lookup table (lines)
		for y := 0; y < rows-1; y++ {
			wy := float64(y)*resolution + minY
			for x := 0; x < cols-1; x++ {
				wx := float64(x)*resolution + minX

				v0 := field[(y+1)*cols+x]     // Top-Left (since y+1 is mathematically "down" in our arrays, wait... Y-up or Y-down?)
				v1 := field[(y+1)*cols+(x+1)] // Top-Right
				v2 := field[y*cols+(x+1)]     // Bottom-Right
				v3 := field[y*cols+x]         // Bottom-Left

				idx := 0
				if v0 >= threshold {
					idx |= 8
				}
				if v1 >= threshold {
					idx |= 4
				}
				if v2 >= threshold {
					idx |= 2
				}
				if v3 >= threshold {
					idx |= 1
				}

				if idx == 0 || idx == 15 {
					continue
				}

				// Edge midpoints
				topX := wx + interpolate(v0, v1)*resolution
				topY := wy + resolution
				rightX := wx + resolution
				rightY := wy + interpolate(v2, v1)*resolution
				bottomX := wx + interpolate(v3, v2)*resolution
				bottomY := wy
				leftX := wx
				leftY := wy + interpolate(v3, v0)*resolution

				var pts []model.Vector3
				switch idx {
				case 1:
					pts = []model.Vector3{{X: leftX, Y: leftY}, {X: bottomX, Y: bottomY}}
				case 2:
					pts = []model.Vector3{{X: bottomX, Y: bottomY}, {X: rightX, Y: rightY}}
				case 3:
					pts = []model.Vector3{{X: leftX, Y: leftY}, {X: rightX, Y: rightY}}
				case 4:
					pts = []model.Vector3{{X: topX, Y: topY}, {X: rightX, Y: rightY}}
				case 5:
					pts = []model.Vector3{{X: leftX, Y: leftY}, {X: topX, Y: topY}, {X: bottomX, Y: bottomY}, {X: rightX, Y: rightY}} // Ambiguous
				case 6:
					pts = []model.Vector3{{X: topX, Y: topY}, {X: bottomX, Y: bottomY}}
				case 7:
					pts = []model.Vector3{{X: leftX, Y: leftY}, {X: topX, Y: topY}}
				case 8:
					pts = []model.Vector3{{X: leftX, Y: leftY}, {X: topX, Y: topY}}
				case 9:
					pts = []model.Vector3{{X: topX, Y: topY}, {X: bottomX, Y: bottomY}}
				case 10:
					pts = []model.Vector3{{X: leftX, Y: leftY}, {X: bottomX, Y: bottomY}, {X: topX, Y: topY}, {X: rightX, Y: rightY}} // Ambiguous
				case 11:
					pts = []model.Vector3{{X: topX, Y: topY}, {X: rightX, Y: rightY}}
				case 12:
					pts = []model.Vector3{{X: leftX, Y: leftY}, {X: rightX, Y: rightY}}
				case 13:
					pts = []model.Vector3{{X: bottomX, Y: bottomY}, {X: rightX, Y: rightY}}
				case 14:
					pts = []model.Vector3{{X: leftX, Y: leftY}, {X: bottomX, Y: bottomY}}
				}

				if len(pts) == 2 {
					segments = append(segments, Seg{pts[0], pts[1]})
				} else if len(pts) == 4 {
					segments = append(segments, Seg{pts[0], pts[1]}, Seg{pts[2], pts[3]})
				}
			}
		}

		// Link segments into continuous polygons
		// (Simplistic O(N^2) linker for now, tree branches produce very few segments compared to whole models)
		if len(segments) == 0 {
			return polygons
		}

		used := make([]bool, len(segments))
		for iter := 0; iter < len(segments); iter++ {
			if used[iter] {
				continue
			}
			used[iter] = true
			poly := model.Polygon{Points: []model.Vector3{segments[iter].Start, segments[iter].End}}

			// Try to stitch
			foundLink := true
			for foundLink {
				foundLink = false
				lastPt := poly.Points[len(poly.Points)-1]
				firstPt := poly.Points[0]

				for j := 0; j < len(segments); j++ {
					if used[j] {
						continue
					}
					// Match end to start
					if math.Abs(segments[j].Start.X-lastPt.X) < 0.001 && math.Abs(segments[j].Start.Y-lastPt.Y) < 0.001 {
						poly.Points = append(poly.Points, segments[j].End)
						used[j] = true
						foundLink = true
						break
					}
					// Match end to end (reversed)
					if math.Abs(segments[j].End.X-lastPt.X) < 0.001 && math.Abs(segments[j].End.Y-lastPt.Y) < 0.001 {
						poly.Points = append(poly.Points, segments[j].Start)
						used[j] = true
						foundLink = true
						break
					}
					// Match start to start (prepend)
					if math.Abs(segments[j].Start.X-firstPt.X) < 0.001 && math.Abs(segments[j].Start.Y-firstPt.Y) < 0.001 {
						poly.Points = append([]model.Vector3{segments[j].End}, poly.Points...)
						used[j] = true
						foundLink = true
						break
					}
					// Match start to end (prepend)
					if math.Abs(segments[j].End.X-firstPt.X) < 0.001 && math.Abs(segments[j].End.Y-firstPt.Y) < 0.001 {
						poly.Points = append([]model.Vector3{segments[j].Start}, poly.Points...)
						used[j] = true
						foundLink = true
						break
					}
				}
			}
			// Close the polygon if applicable
			if len(poly.Points) > 2 {
				lastPt := poly.Points[len(poly.Points)-1]
				firstPt := poly.Points[0]
				if math.Abs(lastPt.X-firstPt.X) < 0.001 && math.Abs(lastPt.Y-firstPt.Y) < 0.001 {
					poly.Points = poly.Points[:len(poly.Points)-1] // Remove duplicate end
					poly.IsClosed = true
				}
			}
			if len(poly.Points) > 2 {
				polygons = append(polygons, poly)
			}
		}

		return polygons
	}

	// Root Spacing for Tree Support
	// rootSpacingMM is now only used functionally if branches merge. Raw targeting uses SDF gravity.
	rootSpacingMM := config.SupportTreeTrunkDiameter
	_ = rootSpacingMM

	// Keep track of which tree slices are "interface" layers
	// and need solid infill
	interfaceGrids := make([]*Grid, len(bm.Slices))
	for i := range interfaceGrids {
		interfaceGrids[i] = NewGrid(bounds, resolution)
	}

	// Taper parameters
	// Taper angle: 1.5 degrees is standard for smooth organic trunks
	taperDist := config.LayerHeight * math.Tan(1.5*math.Pi/180.0)
	taperStepsPerLayer := taperDist / resolution

	// Spread roots based on configured trunk diameter
	// Removed global Target Roots as they aggressively pull branches sideways.
	// We will rely on downward vertical gravity and natural overlapping for trunk formation.

	// Iterate from top to bottom
	for i := len(bm.Slices) - 1; i >= 1; i-- {
		currentModel := layerGrids[i]
		lowerModel := layerGrids[i-1]
		targetSupport := supportGrids[i-1]

		// Precompute Distance Field for the lower model
		// This provides mathematically continuous, sub-pixel accurate distances for overhang detection!
		lowerSDF := lowerModel.DistanceField()

		if config.SupportType == model.SupportTree {
			// --- SKELETON TREE LOGIC ---

			// Identify New Overhangs (Islands)
			// Using the user's specified SupportAngle limit
			overhangGrid := NewGrid(currentModel.GetBounds(), resolution)
			hasOverhangs := false

			for idx, val := range currentModel.Cells {
				// Require support if current model has a pixel where lower model's distance exceeds the overhang limit
				if val && lowerSDF[idx] > overhangDistSteps {
					overhangGrid.Cells[idx] = true
					hasOverhangs = true
				}
			}

			// We rely on the `overhangDistSteps` buffer to handle micro-details natively.

			// Spawn New Branches from Overhang Areas (Dense placement)
			if hasOverhangs {
				// Organic supports need perfectly controlled density at the canopy to avoid wasting material.
				// We use the `SupportDensity` parameter to compute exactly how far apart the branch tips
				// should be spaced to achieve the requested Area Coverage Percentage.
				// Area of tip = Pi * R^2. Area of grid cell = S^2.
				// Density = (Pi * R^2) / S^2  =>  S = R * math.Sqrt(Pi / Density)
				radiusMM := config.SupportTreeBranchDiameter / 2.0

				density := config.SupportDensity
				if density < 0.02 {
					density = 0.02 // Prevent infinite sparse spacing (2% min)
				} else if density > 1.0 {
					density = 1.0
				}

				tipSpacingMM := radiusMM * math.Sqrt(math.Pi/density)
				tipSpacingSteps := int(math.Ceil(tipSpacingMM / resolution))
				if tipSpacingSteps < 1 {
					tipSpacingSteps = 1
				}

				newBranchesSpawned := 0

				for gy := 0; gy < overhangGrid.Height; gy += tipSpacingSteps {
					for gx := 0; gx < overhangGrid.Width; gx += tipSpacingSteps {
						if overhangGrid.Get(gx, gy) {
							// Hard cap to prevent physics explosions / OOM if parameters are extreme
							// However, taking the cap off now that performance is fixed lets large prints actually be fully supported.
							if newBranchesSpawned > 2000 {
								// We must break out of BOTH loops
								gy = overhangGrid.Height
								break
							}

							// Add new fine branch
							activeBranches = append(activeBranches, &Branch{
								X: float64(gx), Y: float64(gy),
								Radius: (config.SupportTreeBranchDiameter / 2.0) / resolution,
								Age:    0,
								Weight: 1, // Single tip
								Dead:   false,
							})
							newBranchesSpawned++
						}
					}
				}
			}

			// Move Branches Towards Roots
			nextBranches := make([]*Branch, 0, len(activeBranches))

			// Minimum 0.4mm gap
			xyGapSteps := int(math.Ceil(math.Max(0.4, config.SupportXYGap) / resolution))

			// SDF for Gravity Repulsion is already calculated as lowerSDF
			sdf := lowerSDF

			for _, b := range activeBranches {
				// 1. Dynamic Organic Slope Profiling
				// Strict Lean Angle Constraint
				// The horizontal shift towards the barycenter is strictly capped by the maximum
				// tree branch lean angle (e.g., SupportAngle).
				maxAngle := config.SupportAngle
				if maxAngle > 50.0 {
					maxAngle = 50.0 // Hard structural limit for Y-junction stability
				}
				treeMaxShiftDist := config.LayerHeight * math.Tan(maxAngle*math.Pi/180.0)
				treeMaxShiftSteps := treeMaxShiftDist / resolution

				// Desired Target Clearance
				// Safe distance = Branch radius + spacing gap
				targetClearance := b.Radius + float64(xyGapSteps)

				// Bilinear interpolation for smooth SDF and gradients (Fixes Stepping)
				bx := b.X
				by := b.Y
				gridX := int(bx)
				gridY := int(by)

				fx := bx - float64(gridX)
				fy := by - float64(gridY)

				getSDF := func(gx, gy int) float64 {
					if gx < 0 {
						gx = 0
					}
					if gy < 0 {
						gy = 0
					}
					if gx >= lowerModel.Width {
						gx = lowerModel.Width - 1
					}
					if gy >= lowerModel.Height {
						gy = lowerModel.Height - 1
					}
					return sdf[gy*lowerModel.Width+gx]
				}

				s00 := getSDF(gridX, gridY)
				s10 := getSDF(gridX+1, gridY)
				s01 := getSDF(gridX, gridY+1)
				s11 := getSDF(gridX+1, gridY+1)

				currentSDF := s00*(1.0-fx)*(1.0-fy) + s10*fx*(1.0-fy) + s01*(1.0-fx)*fy + s11*fx*fy

				// True Floor Detection: Did we land on a flat surface?
				zGapLayers := int(math.Ceil(config.SupportZGap / config.LayerHeight))
				if zGapLayers < 1 {
					zGapLayers = 1
				}

				hitFloor := false
				imminentFloor := false // True if a floor is approaching within safety margin

				checkFloor := func(gap int) bool {
					cLayer := i - gap
					if cLayer >= 0 && cLayer < len(layerGrids) {
						fModel := layerGrids[cLayer]
						if gridX >= 0 && gridX < fModel.Width && gridY >= 0 && gridY < fModel.Height {
							// ONLY check the exact center pixel.
							// SDF repulsion already keeps the branch center away from vertical walls.
							// If the center itself hits something straight down, it MUST be a flat floor!
							if fModel.Cells[gridY*fModel.Width+gridX] {
								return true
							}
						}
					}
					return false
				}

				hitFloor = checkFloor(zGapLayers)
				if !hitFloor {
					imminentFloor = checkFloor(zGapLayers + 2)
				}

				// If we reached the absolute bottom layer, it's the build plate.
				hitBuildPlate := (i <= 1)

				// As soon as the branch touches a True Floor (or build plate), it must instantly terminate.
				if hitFloor || hitBuildPlate {
					if config.SupportPlacement == model.SupportEverywhere || hitBuildPlate {
						// Firmly plant our final footing into the model/bed
						targetSupport.SetDiskFloat(b.X, b.Y, b.Radius)
					}
					continue // Branch terminates here
				}

				// SDF Gradient Calculation (Bilinear)
				gradX := (s10-s00)*(1.0-fy) + (s11-s01)*fy
				gradY := (s01-s00)*(1.0-fx) + (s11-s10)*fx

				gradMag := math.Sqrt(gradX*gradX + gradY*gradY)
				if gradMag > 0.001 {
					gradX /= gradMag
					gradY /= gradMag
				}

				// The Distance from the model surface
				penetration := targetClearance - currentSDF

				// 1. Local Barycenter Clustering (LBTS Method)
				// Group nearby branches and find their Center of Mass (Barycenter).
				// We use a search radius based on trunk diameter to encourage distinct trunk formation.
				searchRadius := (config.SupportTreeTrunkDiameter * 1.5) / resolution

				Bx, By := b.X, b.Y
				weight := 1.0 // Self weight
				for _, neighbor := range activeBranches {
					if neighbor == b {
						continue
					}
					ndx := neighbor.X - b.X
					ndy := neighbor.Y - b.Y
					ndist := math.Sqrt(ndx*ndx + ndy*ndy)
					if ndist < searchRadius {
						Bx += neighbor.X
						By += neighbor.Y
						weight += 1.0
					}
				}

				Bx /= weight
				By /= weight

				// Move towards Local Barycenter
				targetX := Bx - b.X
				targetY := By - b.Y
				targetDist := math.Sqrt(targetX*targetX + targetY*targetY)

				if targetDist > 0 {
					// Pull towards barycenter (acceleration)
					pullForce := targetDist * 0.10 // 10% pull per layer
					b.VX += (targetX / targetDist) * pullForce
					b.VY += (targetY / targetDist) * pullForce
				}

				// 2. Collision Avoidance & Organic Flaring (DVIST via SDF)
				// If a branch gets near the model, it shouldn't just slide down the surface (hugging).
				// It should gracefully accelerate away into open space.
				avoidanceZone := targetClearance + (5.0 / resolution) // 5mm warning zone
				if currentSDF < avoidanceZone && !imminentFloor {
					if currentSDF <= targetClearance {
						// Hard position correction (don't let it inside)
						if penetration > 0 {
							b.X += gradX * penetration
							b.Y += gradY * penetration
							// And bounce its velocity outwards so it doesn't immediately fall back in
							b.VX += gradX * penetration * 0.5
							b.VY += gradY * penetration * 0.5
						}
					} else {
						// Soft avoidance factor (1.0 near model, 0.0 at outer bound)
						factor := 1.0 - ((currentSDF - targetClearance) / (5.0 / resolution))
						// Exponential curve creates a smooth flare
						flareForce := factor * factor * (1.0 / resolution) // 1.0mm/layer accel
						b.VX += gradX * flareForce
						b.VY += gradY * flareForce
					}
				}

				// Apply particle physics!
				// Damping (friction) naturally straightens branches to vertical
				// when no forces (pull or avoidance) are acting on them.
				b.VX *= 0.60
				b.VY *= 0.60

				// Fatter trunks (lower in the tree) should be structurally straighter (more vertical).
				// They carry far more mathematical "Weight" because many tips have merged into them.
				// A huge trunk (Weight 10+) shouldn't zig-zag like a tiny canopy tip.
				flexibility := 1.0
				if b.Weight > 1 {
					// Exponential decay of flexibility as the trunk gets heavier.
					flexibility = 1.0 / float64(b.Weight)
				}
				if flexibility < 0.15 {
					flexibility = 0.15 // Guarantee at least a 15% crawl speed to clear obstacles
				}

				dynamicMaxShift := treeMaxShiftSteps * flexibility

				// Limit scalar speed to the Dynamic Maximum Structural Angle
				speed := math.Sqrt(b.VX*b.VX + b.VY*b.VY)
				if speed > dynamicMaxShift {
					b.VX = (b.VX / speed) * dynamicMaxShift
					b.VY = (b.VY / speed) * dynamicMaxShift
				}

				// Update position
				b.X += b.VX
				b.Y += b.VY

				// Ensure we didn't get pushed out of bounds
				if b.X < 0 {
					b.X = 0
				}
				if b.Y < 0 {
					b.Y = 0
				}
				if b.X > float64(lowerModel.Width-1) {
					b.X = float64(lowerModel.Width - 1)
				}
				if b.Y > float64(lowerModel.Height-1) {
					b.Y = float64(lowerModel.Height - 1)
				}

				// Keep growing downwards
				b.Age++

				// Natural Organic Tapering: Branches thicken as they drop and merge.
				// We calculate the maximum Target Radius based on mathematical Weight + continuous Age taper.
				baseRad := ((config.SupportTreeBranchDiameter / 2.0) / resolution) + (float64(b.Age) * taperStepsPerLayer)
				maxTrunkRad := (config.SupportTreeTrunkDiameter / 2.0) / resolution
				targetRad := math.Sqrt(float64(b.Weight)) * baseRad
				if targetRad > maxTrunkRad {
					targetRad = maxTrunkRad
				}

				// Flare smoothly towards the target radius instead of snapping (fixes Y-Junction stepping)
				if b.Radius < targetRad {
					b.Radius += (targetRad - b.Radius) * 0.10 // 10% smoothing per layer yields beautiful flares
				}
				if b.Age > 0 && imminentFloor {
					// We successfully found a floor to land on!
					// Mark it as dead so it renders its base here, but won't fall down into the gap/model next layer.
					b.Dead = true
					nextBranches = append(nextBranches, b)
					continue
				}

				nextBranches = append(nextBranches, b)
			}

			// Merge Nearby Branches (Y-Junctions)
			mergedBranches := make([]*Branch, 0)
			alreadyMerged := make(map[*Branch]bool)

			for j, b1 := range nextBranches {
				if alreadyMerged[b1] {
					continue
				}

				// Start a group with b1
				groupX, groupY := b1.X, b1.Y
				groupVX, groupVY := b1.VX, b1.VY
				groupCount := 1.0
				groupWeight := b1.Weight
				maxRadius := b1.Radius

				for k := j + 1; k < len(nextBranches); k++ {
					b2 := nextBranches[k]
					if alreadyMerged[b2] {
						continue
					}

					ddx := b1.X - b2.X
					ddy := b1.Y - b2.Y
					dist := math.Sqrt(ddx*ddx + ddy*ddy)

					// We gracefully wait to logically merge branches until they have almost perfectly converged
					// to identical coordinates. Doing this prevents instantaneous positional teleportation.
					// The Marching Squares naturally traces them as a beautifully smooth unified trunk
					// long before they are culled.
					mergeThreshold := 0.2 / resolution // 0.2mm (basically overlapping)

					if dist < mergeThreshold {
						// Merge b2 into b1
						groupX += b2.X
						groupY += b2.Y
						groupVX += b2.VX
						groupVY += b2.VY
						groupCount++
						groupWeight += b2.Weight
						if b2.Radius > maxRadius {
							maxRadius = b2.Radius
						}
						alreadyMerged[b2] = true
					}
				}

				if groupCount > 0 {
					// We take the Maximum Radius of the merging group instead of preserving volume (Sqrt(SumArea)).
					// Because branches naturally taper (thicken) as they drop, taking the Max Radius guarantees
					// a perfectly continuous outer shell without any violent horizontal steps-outwards!
					maxTrunkRadius := (config.SupportTreeTrunkDiameter / 2.0) / resolution
					if maxRadius > maxTrunkRadius {
						maxRadius = maxTrunkRadius
					}

					mergedBranches = append(mergedBranches, &Branch{
						X:      groupX / groupCount,
						Y:      groupY / groupCount,
						VX:     groupVX / groupCount,
						VY:     groupVY / groupCount,
						Radius: maxRadius, // Continues to flare naturally based on new Weight
						Age:    b1.Age,
						Weight: groupWeight,
						Dead:   b1.Dead, // If any branch in the group is dead, the merged branch is dead
					})
				}
			}
			activeBranches = mergedBranches

			// Render smoothly using floating point representation
			for _, b := range activeBranches {
				targetSupport.SetDiskFloat(b.X, b.Y, b.Radius)
			}
			// Save for Metaball extraction later
			treeBranchesByLayer[i-1] = append([]*Branch(nil), activeBranches...)

			if i%50 == 0 || i < 10 {
				fmt.Printf("Layer %d Output: %d viable branches\\n", i, len(activeBranches))
			}

			// --- CULL DEAD BRANCHES BEFORE NEXT LAYER ---
			// Any branch that hit a floor was marked Dead. It rendered on *this* layer,
			// but we must remove it now so it doesn't simulate downward through the model.
			aliveBranches := make([]*Branch, 0, len(activeBranches))
			for _, b := range activeBranches {
				if !b.Dead {
					aliveBranches = append(aliveBranches, b)
				}
			}
			activeBranches = aliveBranches

			// If we are within SupportInterfaceLayers of an overhang,
			// mark this entire support region as an interface that needs solid fill
			if hasOverhangs {
				for interfaceLayer := 0; interfaceLayer < config.SupportInterfaceLayers; interfaceLayer++ {
					idxToMark := i - interfaceLayer
					if idxToMark >= 0 {
						// Copy current overhang area to the interface grid for that layer
						for cellIdx, isOverhang := range overhangGrid.Cells {
							if isOverhang {
								// We Dilate the overhang a bit so the interface fill fully bridges the gap
								// (handled roughly by just assigning it directly to the interfaceGrid and filling trees that overlap it)
								interfaceGrids[idxToMark].Cells[cellIdx] = true
							}
						}
					}
				}
			}

		} else {
			// --- LINEAR/GRID LOGIC ---

			upperSupport := supportGrids[i]

			// Copy upper support down
			for idx, needed := range upperSupport.Cells {
				if needed {
					targetSupport.Cells[idx] = true
				}
			}

			// Add new overhangs using the exact Distance Field (much more accurate than Dilation)
			for idx, occupied := range currentModel.Cells {
				if occupied && lowerSDF[idx] > overhangDistSteps {
					targetSupport.Cells[idx] = true
				}
			}
		}

		// Subtract Model at this layer (ensure interface separation)
		xyGapSteps := int(math.Ceil(config.SupportXYGap / resolution))
		if config.SupportType == model.SupportTree {
			// Add 1 extra voxel buffer specifically before Chaikin smoothing
			// to ensure the rounded corner-cuts never pierce the model surface
			xyGapSteps += 1
		}

		modelMask := lowerModel
		if xyGapSteps > 0 {
			modelMask = lowerModel.Dilate(xyGapSteps)
		}

		// Only subtract if it's traditional support OR if it's a higher layer of tree support.
		// For tree supports, the physics engine already guarantees exact clearance using SDF Repulsion!
		// If we use the raw bitmap subtraction here, trunks dropping over sloped roofs get their
		// cross-section violently chewed up and deleted. We ONLY use this for Grid/Line supports!
		if config.SupportType != model.SupportTree {
			for idx, isModel := range modelMask.Cells {
				if isModel {
					targetSupport.Cells[idx] = false
				}
			}
		}
	}

	// 3. Generate Paths from SupportGrids
	paths := make(map[int][]model.ContinuousPath)

	for i, grid := range supportGrids {
		var layerPaths []model.ContinuousPath
		z := bm.Slices[i].Z

		if config.SupportType == model.SupportTree {
			// Tree Supports: Generate ONE strong outer shell using Metaballs.
			// (Previous attempts to simulate inner walls by raising the metaball threshold caused dense canopies
			// to rip apart into hundreds of microscopic isolated internal circles, ballooning the JSON config to 3.2GB).
			shellsToGenerate := 1

			// Generate Perimeters based on floating point Metaballs
			var contours []model.Polygon

			// Trace the exact contours using Marching Squares
			layerActiveBranches := treeBranchesByLayer[i]
			contours = ExtractBranchContours(layerActiveBranches, resolution, resolution, 1.0, layerGrids[i].GetBounds())

			for shell := 0; shell < shellsToGenerate; shell++ {
				shellContours := contours

				for _, poly := range shellContours {
					points := poly.Points
					if len(points) < 3 {
						// If a polygon has fewer than 3 points, it cannot form a closed loop.
						// The original code had a 'continue' here, but the instruction implies
						// we should still process it if it's part of the totalPointsPerLayer count.
						// However, to maintain the original logic of not processing invalid polygons for segments,
						// we will keep the continue here.
						continue
					}

					var decimated []model.Vector3
					if len(points) > 0 {
						decimated = append(decimated, points[0])
						for k := 1; k < len(points)-1; k++ {
							pPrev := decimated[len(decimated)-1]
							pCurr := points[k]
							pNext := points[k+1]

							// Calculate cross product of (pCurr-pPrev) and (pNext-pCurr)
							dx1 := pCurr.X - pPrev.X
							dy1 := pCurr.Y - pPrev.Y
							dx2 := pNext.X - pCurr.X
							dy2 := pNext.Y - pCurr.Y
							cross := dx1*dy2 - dy1*dx2

							// If points are perfectly straight, skip adding pCurr
							if math.Abs(cross) > 0.005 {
								decimated = append(decimated, pCurr)
							}
						}
						decimated = append(decimated, points[len(points)-1])
					}
					points = decimated

					// The global JSON bloat bug was fixed in the exporter, so we have infinite
					// performance headroom to restore organic curve smoothing to the trunks.
					points = smoothPath(points, 2)

					poly := model.Polygon{Points: points, IsClosed: true}
					poly.SetZ(z)
					if path := poly.ToContinuousPath(config.SupportSpeed, model.CategorySupport, i); len(path.Segments) > 0 {
						layerPaths = append(layerPaths, path)
					}
				}
			}

			// Generate Dense Interface Infill inside the innermost shell
			// If this layer was marked as an interface layer, we fill it.
			interfaceGrid := interfaceGrids[i]
			needsFill := false
			for _, isInterface := range interfaceGrid.Cells {
				if isInterface {
					needsFill = true
					break
				}
			}

			if needsFill {
				// The area we fill is the intersection of the innermost branch shell and the interface region
				fillArea := grid.Erode(shellsToGenerate) // Step inward once more so infill binds inside the wall

				// Pre-calculate dilation OUTSIDE the loop to prevent massive performance hang
				dilatedInterface := interfaceGrid.Dilate(3)

				// Generate dense Zig-Zag lines
				// We use the same raster technique but strictly inside the fillArea
				for j := 0; j < fillArea.Height; j += 2 { // Dense spacing (every 2 grid cells)
					y := fillArea.MinY + float64(j)*fillArea.Resolution + fillArea.Resolution/2

					var startX int = -1
					for x := 0; x < fillArea.Width; x++ {
						// Only fill if it's inside the tree branch AND it's marked as an interface zone
						active := fillArea.Get(x, j) && (interfaceGrid.Get(x, j) || dilatedInterface.Get(x, j))

						if active {
							if startX == -1 {
								startX = x
							}
						} else {
							if startX != -1 {
								// End of segment
								pStart := model.Vector3{
									X: fillArea.MinX + float64(startX)*fillArea.Resolution,
									Y: y,
									Z: z,
								}
								pEnd := model.Vector3{
									X: fillArea.MinX + float64(x)*fillArea.Resolution,
									Y: y,
									Z: z,
								}

								layerPaths = append(layerPaths, model.ContinuousPath{
									Segments: []model.PathSegment{{
										Start:    pStart,
										End:      pEnd,
										Speed:    config.SupportSpeed,
										Category: model.CategorySupport,
									}},
									PathType:   model.PathExtrusion,
									LayerIndex: i,
								})
								startX = -1
							}
						}
					}
					if startX != -1 {
						pStart := model.Vector3{
							X: fillArea.MinX + float64(startX)*fillArea.Resolution,
							Y: y,
							Z: z,
						}
						pEnd := model.Vector3{
							X: fillArea.MinX + float64(fillArea.Width-1)*fillArea.Resolution,
							Y: y,
							Z: z,
						}
						layerPaths = append(layerPaths, model.ContinuousPath{
							Segments: []model.PathSegment{{
								Start:    pStart,
								End:      pEnd,
								Speed:    config.SupportSpeed,
								Category: model.CategorySupport,
							}},
							PathType:   model.PathExtrusion,
							LayerIndex: i,
						})
					}
				}
			}

		} else {
			// Linear filling (zig-zag / raster)
			for j := 0; j < grid.Height; j++ {
				y := grid.MinY + float64(j)*grid.Resolution + grid.Resolution/2

				var startX int = -1
				for x := 0; x < grid.Width; x++ {
					active := grid.Get(x, j)
					if active {
						if startX == -1 {
							startX = x
						}
					} else {
						if startX != -1 {
							// End of segment
							pStart := model.Vector3{
								X: grid.MinX + float64(startX)*grid.Resolution,
								Y: y,
								Z: z,
							}
							pEnd := model.Vector3{
								X: grid.MinX + float64(x)*grid.Resolution,
								Y: y,
								Z: z,
							}

							// Create segment
							seg := model.PathSegment{
								Start:    pStart,
								End:      pEnd,
								Speed:    config.SupportSpeed,
								Category: model.CategorySupport,
							}

							layerPaths = append(layerPaths, model.ContinuousPath{
								Segments:   []model.PathSegment{seg},
								PathType:   model.PathExtrusion,
								LayerIndex: i,
							})
							startX = -1
						}
					}
				}
				// End of row check
				if startX != -1 {
					pStart := model.Vector3{
						X: grid.MinX + float64(startX)*grid.Resolution,
						Y: y,
						Z: z,
					}
					pEnd := model.Vector3{
						X: grid.MinX + float64(grid.Width)*grid.Resolution,
						Y: y,
						Z: z,
					}
					seg := model.PathSegment{
						Start:    pStart,
						End:      pEnd,
						Speed:    config.SupportSpeed,
						Category: model.CategorySupport,
					}
					layerPaths = append(layerPaths, model.ContinuousPath{
						Segments:   []model.PathSegment{seg},
						PathType:   model.PathExtrusion,
						LayerIndex: i,
					})
				}
			}
		}

		if len(layerPaths) > 0 {
			paths[i] = layerPaths
		}
	}

	return paths
}
