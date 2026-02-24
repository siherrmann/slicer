# Clean Code & Performance Guidelines

## General Go Best Practices

### Naming
- Use short, descriptive names: `pts` not `pointsSlice`, `cfg` not `configuration`
- Exported types/functions: `PascalCase`. Unexported: `camelCase`
- Receiver names: single letter matching type (`s *Slicer`, `p *Polygon`)

### Error Handling
- Always wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
- Never ignore errors — if intentionally unused, add a comment explaining why
- Return early on errors (guard clauses) rather than nesting

### Code Organization
- One responsibility per file. If a file exceeds ~300 lines, consider splitting
- Keep package-level functions (no receiver) to a minimum — prefer methods
- Group imports: stdlib, external deps, internal packages

---

## Project-Specific Guidelines

### Geometry & Math
- **Use `1e-6` epsilon** for floating-point comparisons, not `== 0`
- **Clamp `asin`/`acos` inputs** to `[-1, 1]` — floating-point drift causes NaN
- **Avoid redundant allocations** in tight loops — pre-allocate slices with `make([]T, 0, expectedCap)`
- **Cache computed values**: extract `polygon.GetLines()` and `polygon.GetBounds()` outside loops instead of recomputing per iteration

### Infill Generation
- Infill is generated in **model-space coordinates** — no rotation is applied
- `cutInfill` clips to the inner wall using intersection + midpoint testing
- New infill patterns should follow the signature: `func GenerateXInfill(bounds model.BoundingBox, params model.SliceConfig, layerIndex int, z float64) model.ContinuousPath`
- Always register new patterns in both `GenerateInfill()` switch and `model.InfillType` constants

### Concurrency
- Layer processing is parallelized in `core/full.go` — ensure all per-layer functions are **goroutine-safe** (no shared mutable state)
- Use `sync.Mutex` only for collecting results into the shared output slice
- Avoid goroutine leaks — always pair `wg.Add(1)` with `defer wg.Done()`

### WebGPU Viewer (`viewer.js`)
- **Triangle strip topology**: always insert degenerate triangles (repeat last vertex 2×) when breaking the strip
- **Split strips at type boundaries**: travel and extrusion segments must not share a continuous strip
- **Buffer management**: store auxiliary buffers on `window.*` for rendering access
- Keep shader code inline in `createShaderModule` — no external WGSL files

### HTMX Integration
- POST `/slice` returns **only the sidebar partial** (`view.SlicerSidebar`)
- Target is `#sidebar-container` with `hx-swap="innerHTML"`
- **Never** return full-page HTML from `/slice` — it destroys the WebGPU context
- Clear `Model.Slices = nil` before re-slicing to force fresh layer generation

---

## Performance Hotspots

| Area | Concern | Mitigation |
|------|---------|------------|
| `cutInfill` | Called per-shell per-layer × segments | Cache shell lines, pre-compute bounds |
| `ContainsPoint` | Ray-casting per call | Called at midpoints; keep polygons simple |
| `GenerateGyroidInfill` | Adaptive subdivision loop | Bounded by tolerance; insertion sort for nearly-sorted data |
| `polygon.OffsetPolygon` | Complex polygon ops | Result is cached implicitly (assigned to `infillArea`) |
| Viewer `createPrintPaths` | Rebuilds all GPU buffers on each slice | Acceptable for now; optimize with incremental updates if latency becomes an issue |

---

## Code Smells to Watch For

- **Double rotation**: do NOT rotate coordinates in both the pattern generator AND the clipper
- **Unused struct fields**: run `go vet ./...` regularly
- **Magic numbers**: define constants (e.g., `const epsilon = 1e-6`) instead of inline values
- **Unbounded slice growth**: use `make([]T, 0, n)` when size is predictable
- **Missing Z coordinates**: toolpath segments must always have correct Z from the layer height
