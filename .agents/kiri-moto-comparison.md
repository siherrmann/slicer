# Kiri:Moto vs Our Slicer — Configuration Comparison

## Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Supported |
| ⚙️ | Partially supported / limited |
| ❌ | Not supported |

---

## 1. Layers / Paths

| Setting | Kiri:Moto | Ours | Notes |
|---------|-----------|------|-------|
| Layer Height | ✅ | ✅ | `LayerHeight` |
| First Layer Height | ✅ | ✅ | `FirstLayer` |
| Top Solid Layers | ✅ | ✅ | `TopLayers` |
| Bottom Solid Layers | ✅ | ✅ | `BottomLayers` |
| Adaptive Layer Height | ✅ | ❌ | Variable height per layer |
| Start Point (nearest/random) | ✅ | ✅ | `StartPoint` |
| Force Retract | ✅ | ❌ | |
| Alternating Z / Interleave | ✅ | ❌ | |

---

## 2. Shells / Walls

| Setting | Kiri:Moto | Ours | Notes |
|---------|-----------|------|-------|
| Shell Count | ✅ | ✅ | `ShellCount` |
| Line Width | ✅ | ✅ | `LineWidth` |
| Wall Thickness | ✅ (computed) | ✅ | `WallThickness` |
| Fill Overlap | ✅ | ✅ | `Overlap` |
| Shell Order (in→out / out→in) | ✅ | ✅ | `ShellOrder` (configurable) |
| Thin Walls (off/basic/adaptive) | ✅ | ❌ | |

---

## 3. Infill

| Setting | Kiri:Moto | Ours | Notes |
|---------|-----------|------|-------|
| Fill Type | ✅ Hex, Grid, Linear, Triangle, Gyroid, Vase | ✅ Line, Grid, TriHexagon, Cross, Honeycomb, Gyroid | Different pattern sets |
| Fill Density | ✅ | ✅ | `InfillDensity` |
| Fill Angle | ✅ (start angle) | ✅ | `InfillAngle` — rotates per layer |
| Fill Repeat | ✅ | ❌ | |
| Solid Fill Expansion | ✅ | ❌ | |
| Solid Fill Min Area | ✅ | ❌ | |
| Variable Infill Density | ✅ (ranges) | ❌ | |
| Vase Mode | ✅ | ✅ | `VaseMode` — spiral single-wall |
| Top Layer Type | ❌ | ✅ | `TopLayerType` — separate infill for top |
| Bottom Layer Type | ❌ | ✅ | `BottomLayerType` |

---

## 4. Temperature / Heating

| Setting | Kiri:Moto | Ours | Notes |
|---------|-----------|------|-------|
| Nozzle Temperature | ✅ | ✅ | `NozzleTemp` |
| Bed Temperature | ✅ | ✅ | `BedTemp` |
| Draft Shield | ✅ | ❌ | Future extensibility |

---

## 5. Cooling

| Setting | Kiri:Moto | Ours | Notes |
|---------|-----------|------|-------|
| Fan Speed | ✅ | ✅ | `FanSpeed` (0-255) |
| Fan On Layer | ✅ | ✅ | `FanOnLayer` |
| Min Layer Time | ✅ | ✅ | `MinLayerTime` — slows to `MinSpeed` |

---

## 6. Speed / Output

| Setting | Kiri:Moto | Ours | Notes |
|---------|-----------|------|-------|
| Print Speed (general) | ✅ | ⚙️ | Separate wall/infill/travel + first layer speeds |
| Infill Speed | ✅ | ✅ | `InfillSpeed` |
| Wall/Shell Speed | ✅ | ✅ | `WallSpeed` |
| Travel Speed | ✅ | ✅ | `TravelSpeed` |
| Finish/Outer Shell Speed | ✅ | ✅ | `OuterShellSpeed` |
| First Layer Speed | ✅ | ✅ | `FirstLayerSpeed` |
| Minimum Speed | ✅ | ✅ | `MinSpeed` |
| Flow Factor / Multiplier | ✅ | ✅ | `FlowMultiplier` |

---

## 7. Retraction

| Setting | Kiri:Moto | Ours | Notes |
|---------|-----------|------|-------|
| Retraction Distance | ✅ | ✅ | `RetractionDist` |
| Retraction Speed | ✅ | ✅ | `RetractionSpeed` |

---

## 8. Support

| Setting | Kiri:Moto | Ours | Notes |
|---------|-----------|------|-------|
| Support Type (auto/manual) | ✅ | ✅ | `SupportType`: None, Auto (Linear), Tree (Stump/Organic) |
| Support Density | ✅ | ✅ | `SupportDensity` |
| Support Base Layers | ✅ | ⚙️ | Support extends to bed or model surface (grid logic) |
| Detect Tool (preview) | ✅ | ❌ | |

---

## 9. Skirt / Brim / Raft

| Setting | Kiri:Moto | Ours | Notes |
|---------|-----------|------|-------|
| Skirt Count | ✅ | ✅ | `SkirtCount` |
| Skirt Offset | ✅ | ✅ | `SkirtOffset` |
| Brim Count | ✅ | ✅ | `BrimCount` |
| Raft | ✅ | ✅ | `RaftLayers` + `RaftOffset` |

---

## 10. Machine / Nozzle

| Setting | Kiri:Moto | Ours | Notes |
|---------|-----------|------|-------|
| Nozzle Diameter | ✅ | ✅ | `NozzleDiameter` |
| Build Volume | ✅ | ✅ | `BuildVolumeX/Y/Z` |
| Multiple Extruders | ✅ | ❌ | |
| Device Profiles (save/load) | ✅ | ❌ | |

---

## 11. Expert / Advanced

| Setting | Kiri:Moto | Ours | Notes |
|---------|-----------|------|-------|
| Slice Angle | ✅ | ❌ | Non-planar slicing |
| Anti-Backlash | ✅ | ❌ | |
| G-code Flavour | ✅ | ✅ | `GCodeFlavor` (Marlin/RepRap/Klipper) |
| Custom G-code (start/end) | ✅ | ✅ | `StartGCode` / `EndGCode` |
| Tolerance | ❌ | ✅ | `Tolerance` for segment merging |

---

## Summary

| Category | Kiri:Moto | Our Slicer |
|----------|-----------|------------|
| Total unique settings | ~50 | ~55 |
| Infill patterns | 6 | 9 (including solid variants) |
| Speed controls | 5+ | 7 (infill, wall, outer, travel, first layer, min, flow) |
| Support | Full | Auto (Linear + Tree/Stump modes) |
| Profiles/presets | Yes | No |
| G-code export | Yes | ✅ Yes |

### Remaining gaps to close
1. **Device profiles** — save/load printer presets
2. **Adaptive layer height** — variable height per layer
3. **Support preview** — detect tool for visual preview
