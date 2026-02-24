// --- Globale Variablen ---
const canvas = document.getElementById("stl-canvas");
const ctx = canvas.getContext("webgpu");

window.setModelData = setModelData;
window.setModelPath = setModelPath;
window.setSliceConfig = setSliceConfig;

let device,
  pipeline,
  gridPipeline,
  axisPipeline,
  printPathPipeline,
  vertexBuffer,
  normalBuffer,
  gridBuffer,
  axisBuffer,
  axisColorBuffer,
  printPathBuffer,
  printPathNormalBuffer,
  printPathColorBuffer,
  uniformBuffer,
  gridUniformBuffer,
  axisUniformBuffer,
  printPathUniformBuffer,
  bindGroup,
  gridBindGroup,
  axisBindGroup,
  printPathBindGroup,
  depthTextureView;

let webgpuReady = false;
let pendingModelData = null;
let gridVertexCount = 0;
let axisVertexCount = 0;
let printPathVertexCount = 0;

let vertices = new Float32Array(0);
let normals = new Float32Array(0);
let vertexCount = 0;
let center = [0, 0, 0];
let bounds = { min: [0, 0, 0], max: [0, 0, 0] };
let modelSize = 1;

let camera = {
  distance: 2,
  rotationX: -Math.PI / 4,
  rotationY: Math.PI / 4,
  isDragging: false,
  lastMouse: { x: 0, y: 0 },
};

let currentClipLevel = 1.0; // 1.0 = show full model, 0.0 = show nothing
let viewMode = "model"; // 'model' or 'print'
let printPathData = null; // Store print path data
let sliceConfig = null; // Store slice configuration


// --- Interaction ---
canvas.addEventListener("mousedown", (e) => {
  camera.isDragging = true;
  camera.lastMouse = { x: e.clientX, y: e.clientY };
});
window.addEventListener("mouseup", () => (camera.isDragging = false));
window.addEventListener("mousemove", (e) => {
  if (!camera.isDragging) return;
  camera.rotationY += (e.clientX - camera.lastMouse.x) * 0.01;
  camera.rotationX += (e.clientY - camera.lastMouse.y) * 0.01;
  camera.lastMouse = { x: e.clientX, y: e.clientY };
  render();
});
canvas.addEventListener(
  "wheel",
  (e) => {
    e.preventDefault();
    camera.distance += e.deltaY * (modelSize * 0.002);
    camera.distance = Math.max(modelSize * 0.1, camera.distance);
    render();
  },
  { passive: false },
);

function resize() {
  canvas.width = canvas.clientWidth;
  canvas.height = canvas.clientHeight;
  if (device) {
    const depthTexture = device.createTexture({
      size: [canvas.width, canvas.height],
      format: "depth24plus",
      usage: GPUTextureUsage.RENDER_ATTACHMENT,
    });
    depthTextureView = depthTexture.createView();
  }
  render();
}
window.addEventListener("resize", resize);

initWebGPU().then((ok) => {
  if (ok) {
    webgpuReady = true;
    if (pendingModelData) setModelData(pendingModelData);
  }
});

// --- WebGPU Initialization ---
async function initWebGPU() {
  if (!navigator.gpu) return (alert("WebGPU not supported"), false);

  const adapter = await navigator.gpu.requestAdapter();
  device = await adapter.requestDevice();

  const format = navigator.gpu.getPreferredCanvasFormat();
  ctx.configure({ device, format, alphaMode: "opaque" });

  const shaderModule = device.createShaderModule({
    label: "STL Local Clipping Shader",
    code: `
        struct Uniforms {
            viewProj: mat4x4<f32>,
            model: mat4x4<f32>,
            lightDir: vec4<f32>,
            clipHeight: f32, // Dies ist nun ein Wert zwischen bounds.min.y und bounds.max.y
        }
        @group(0) @binding(0) var<uniform> uniforms: Uniforms;

        struct Out {
            @builtin(position) pos: vec4<f32>,
            @location(0) normal: vec3<f32>,
            @location(1) localY: f32, // Wir geben die lokale Y-Höhe an den Fragment Shader weiter
        }

        @vertex
        fn vertexMain(@location(0) pos: vec3<f32>, @location(1) normal: vec3<f32>) -> Out {
            var o: Out;
            // WICHTIG: Wir nutzen 'pos' (lokal), nicht worldPos für das Clipping
            o.localY = pos.z; 
            
            let worldPos = uniforms.model * vec4<f32>(pos, 1.0);
            o.pos = uniforms.viewProj * worldPos;
            o.normal = (uniforms.model * vec4<f32>(normal, 0.0)).xyz;
            return o;
        }

        @fragment
        fn fragmentMain(input: Out, @builtin(front_facing) isFront: bool) -> @location(0) vec4<f32> {
            // LOKALER CUT: Wir prüfen gegen die lokale Höhe des Modells
            if (input.localY > uniforms.clipHeight) {
                discard;
            }

            let n = normalize(input.normal);
            let l = normalize(uniforms.lightDir.xyz);
            var diff = abs(dot(n, l)) * 0.7 + 0.3;
            
            var color = vec3(0.5, 0.8, 1.0);
            if (!isFront) { color = vec3(0.2, 0.4, 0.6); }

            return vec4(color * diff, 1.0);
        }
    `,
  });

  pipeline = device.createRenderPipeline({
    layout: "auto",
    vertex: {
      module: shaderModule,
      entryPoint: "vertexMain",
      buffers: [
        {
          arrayStride: 12,
          attributes: [{ shaderLocation: 0, offset: 0, format: "float32x3" }],
        },
        {
          arrayStride: 12,
          attributes: [{ shaderLocation: 1, offset: 0, format: "float32x3" }],
        },
      ],
    },
    fragment: {
      module: shaderModule,
      entryPoint: "fragmentMain",
      targets: [{ format }],
    },
    primitive: { topology: "triangle-list", cullMode: "none" },
    depthStencil: {
      depthWriteEnabled: true,
      depthCompare: "less",
      format: "depth24plus",
    },
  });

  resize();
  // Create grid pipeline
  const gridShaderModule = device.createShaderModule({
    label: "Grid Shader",
    code: `
      struct Uniforms {
        viewProj: mat4x4<f32>,
        model: mat4x4<f32>,
      }
      @group(0) @binding(0) var<uniform> uniforms: Uniforms;

      struct Out {
        @builtin(position) pos: vec4<f32>,
      }

      @vertex
      fn vertexMain(@location(0) pos: vec3<f32>) -> Out {
        var o: Out;
        let worldPos = uniforms.model * vec4<f32>(pos, 1.0);
        o.pos = uniforms.viewProj * worldPos;
        return o;
      }

      @fragment
      fn fragmentMain() -> @location(0) vec4<f32> {
        return vec4<f32>(0.4, 0.4, 0.4, 1.0); // Dark grey grid lines
      }
    `,
  });

  gridPipeline = device.createRenderPipeline({
    label: "Grid Pipeline",
    layout: "auto",
    vertex: {
      module: gridShaderModule,
      entryPoint: "vertexMain",
      buffers: [
        {
          arrayStride: 12,
          attributes: [{ shaderLocation: 0, offset: 0, format: "float32x3" }],
        },
      ],
    },
    fragment: {
      module: gridShaderModule,
      entryPoint: "fragmentMain",
      targets: [{ format }],
    },
    primitive: { topology: "line-list", cullMode: "none" },
    depthStencil: {
      depthWriteEnabled: true,
      depthCompare: "less",
      format: "depth24plus",
    },
  });

  // Create axis pipeline with color support
  const axisShaderModule = device.createShaderModule({
    label: "Axis Shader",
    code: `
      struct Uniforms {
        viewProj: mat4x4<f32>,
        model: mat4x4<f32>,
      }
      @group(0) @binding(0) var<uniform> uniforms: Uniforms;

      struct Out {
        @builtin(position) pos: vec4<f32>,
        @location(0) color: vec3<f32>,
      }

      @vertex
      fn vertexMain(@location(0) pos: vec3<f32>, @location(1) color: vec3<f32>) -> Out {
        var o: Out;
        let worldPos = uniforms.model * vec4<f32>(pos, 1.0);
        o.pos = uniforms.viewProj * worldPos;
        o.color = color;
        return o;
      }

      @fragment
      fn fragmentMain(input: Out) -> @location(0) vec4<f32> {
        return vec4<f32>(input.color, 1.0);
      }
    `,
  });

  axisPipeline = device.createRenderPipeline({
    label: "Axis Pipeline",
    layout: "auto",
    vertex: {
      module: axisShaderModule,
      entryPoint: "vertexMain",
      buffers: [
        {
          arrayStride: 12,
          attributes: [{ shaderLocation: 0, offset: 0, format: "float32x3" }],
        },
        {
          arrayStride: 12,
          attributes: [{ shaderLocation: 1, offset: 0, format: "float32x3" }],
        },
      ],
    },
    fragment: {
      module: axisShaderModule,
      entryPoint: "fragmentMain",
      targets: [{ format }],
    },
    primitive: { topology: "line-list", cullMode: "none" },
    depthStencil: {
      depthWriteEnabled: true,
      depthCompare: "less",
      format: "depth24plus",
    },
  });

  // Create print path pipeline with screen-space thick lines
  const printPathShaderModule = device.createShaderModule({
    label: "Print Path Shader",
    code: `
      struct Uniforms {
        viewProj: mat4x4<f32>,
        model: mat4x4<f32>,
        clipHeight: f32,
        thickness: f32,
        aspect: f32,
      }
      @group(0) @binding(0) var<uniform> uniforms: Uniforms;

      struct Out {
        @builtin(position) pos: vec4<f32>,
        @location(0) color: vec3<f32>,
        @location(1) localZ: f32,
        @location(2) uv: f32,
        @location(3) lineType: f32,
      }

      @vertex
      fn vertexMain(
        @location(0) pos: vec3<f32>,
        @location(1) prev: vec3<f32>,
        @location(2) next: vec3<f32>,
        @location(3) orientation: f32,
        @location(4) color: vec3<f32>,
        @location(5) lineType: f32
      ) -> Out {
        var o: Out;
        o.localZ = pos.z;
        o.color = color;
        o.uv = orientation; // -1 or 1, represents distance from center
        o.lineType = lineType;
        
        // Transform current, previous, and next to clip space
        let worldPos = uniforms.model * vec4<f32>(pos, 1.0);
        let clipPos = uniforms.viewProj * worldPos;
        
        let worldPrev = uniforms.model * vec4<f32>(prev, 1.0);
        let clipPrev = uniforms.viewProj * worldPrev;
        
        let worldNext = uniforms.model * vec4<f32>(next, 1.0);
        let clipNext = uniforms.viewProj * worldNext;
        
        // Convert to NDC (normalized device coordinates)
        var ndcPos = clipPos.xy / clipPos.w;
        var ndcPrev = clipPrev.xy / clipPrev.w;
        var ndcNext = clipNext.xy / clipNext.w;
        
        // Correct for aspect ratio
        ndcPos.x *= uniforms.aspect;
        ndcPrev.x *= uniforms.aspect;
        ndcNext.x *= uniforms.aspect;
        
        // Compute normals and handle joins properly
        var offset: vec2<f32> = vec2<f32>(0.0);
        let actualThickness = uniforms.thickness * mix(0.15, 1.0, lineType);
        
        let delta1 = ndcPos - ndcPrev;
        let delta2 = ndcNext - ndcPos;
        let len1 = length(delta1);
        let len2 = length(delta2);
        let epsilon = 0.000001;
        
        if (len1 < epsilon && len2 < epsilon) {
          offset = vec2<f32>(0.0);
        } else if (len1 < epsilon) {
          // Start of line
          let tangent = delta2 / len2;
          let normal = vec2<f32>(-tangent.y, tangent.x);
          offset = normal * actualThickness * orientation;
        } else if (len2 < epsilon) {
          // End of line
          let tangent = delta1 / len1;
          let normal = vec2<f32>(-tangent.y, tangent.x);
          offset = normal * actualThickness * orientation;
        } else {
          // Middle of line - compute proper miter with limit
          let tangent1 = delta1 / len1;
          let tangent2 = delta2 / len2;
          let normal1 = vec2<f32>(-tangent1.y, tangent1.x);
          let normal2 = vec2<f32>(-tangent2.y, tangent2.x);
          
          // Miter is the average of the two normals
          var miter = normal1 + normal2;
          let miterLenSq = dot(miter, miter);
          
          if (miterLenSq < epsilon) {
            // Nearly 180 degree turn
            offset = normal1 * actualThickness * orientation;
          } else {
            miter = normalize(miter);
            let miterDot = dot(miter, normal1);
            let miterLimit = 2.0; // Slightly more generous limit
            let miterLength = clamp(1.0 / max(abs(miterDot), 0.1), 1.0, miterLimit);
            offset = miter * actualThickness * orientation * miterLength;
          }
        }
        
        // Add miter offset to position
        ndcPos += offset;
        
        // Convert back from square space to real NDC x
        ndcPos.x /= uniforms.aspect;
        
        // Convert back to clip space
        o.pos = vec4<f32>(ndcPos * clipPos.w, clipPos.z, clipPos.w);
        
        return o;
      }

      @fragment
      fn fragmentMain(input: Out) -> @location(0) vec4<f32> {
        if (input.localZ > uniforms.clipHeight) {
          discard;
        }
        
        // Create round appearance for extrusion lines
        if (input.lineType > 0.5) {
          // For extrusion lines, apply circular cross-section
          let dist = abs(input.uv);
          
          // Discard pixels outside the circle
          if (dist > 1.0) {
            discard;
          }
          
          // Create lighting effect - simulate round surface
          // dist goes from 0 (center) to 1 (edge)
          let normalZ = sqrt(1.0 - dist * dist); // simulate circular normal
          let lighting = normalZ * 0.6 + 0.4; // ambient + diffuse
          
          return vec4<f32>(input.color * lighting, 1.0);
        } else {
          // Travel lines - flat appearance
          return vec4<f32>(input.color, 1.0);
        }
      }
    `,
  });

  printPathPipeline = device.createRenderPipeline({
    label: "Print Path Pipeline",
    layout: "auto",
    vertex: {
      module: printPathShaderModule,
      entryPoint: "vertexMain",
      buffers: [
        {
          arrayStride: 12,
          attributes: [{ shaderLocation: 0, offset: 0, format: "float32x3" }],
        },
        {
          arrayStride: 12,
          attributes: [{ shaderLocation: 1, offset: 0, format: "float32x3" }],
        },
        {
          arrayStride: 12,
          attributes: [{ shaderLocation: 2, offset: 0, format: "float32x3" }],
        },
        {
          arrayStride: 4,
          attributes: [{ shaderLocation: 3, offset: 0, format: "float32" }],
        },
        {
          arrayStride: 12,
          attributes: [{ shaderLocation: 4, offset: 0, format: "float32x3" }],
        },
        {
          arrayStride: 4,
          attributes: [{ shaderLocation: 5, offset: 0, format: "float32" }],
        },
      ],
    },
    fragment: {
      module: printPathShaderModule,
      entryPoint: "fragmentMain",
      targets: [{ format }],
    },
    primitive: { topology: "triangle-strip", cullMode: "none" },
    depthStencil: {
      depthWriteEnabled: true,
      depthCompare: "less",
      format: "depth24plus",
    },
  });

  resize();
  return true;
}

// --- Daten-Handling ---
function setModelData(data) {
  // Parse JSON if it's a string
  if (typeof data === "string") {
    data = JSON.parse(data);
  }

  if (!data.triangles) {
    console.error("No triangles in data!");
    return;
  }

  // Convert triangles to vertices and normals
  const triangleCount = data.triangles.length;
  vertices = new Float32Array(triangleCount * 9); // 3 vertices * 3 components
  normals = new Float32Array(triangleCount * 9);

  for (let i = 0; i < triangleCount; i++) {
    const tri = data.triangles[i];
    const offset = i * 9;

    // Vertex 1
    vertices[offset] = tri.v1.x;
    vertices[offset + 1] = tri.v1.y;
    vertices[offset + 2] = tri.v1.z;
    // Vertex 2
    vertices[offset + 3] = tri.v2.x;
    vertices[offset + 4] = tri.v2.y;
    vertices[offset + 5] = tri.v2.z;
    // Vertex 3
    vertices[offset + 6] = tri.v3.x;
    vertices[offset + 7] = tri.v3.y;
    vertices[offset + 8] = tri.v3.z;

    // Normals (same for all 3 vertices of a triangle)
    for (let j = 0; j < 3; j++) {
      normals[offset + j * 3] = tri.normal.x;
      normals[offset + j * 3 + 1] = tri.normal.y;
      normals[offset + j * 3 + 2] = tri.normal.z;
    }
  }

  vertexCount = vertices.length / 3;

  // Convert bounds from STL JSON format
  bounds = {
    min: [data.bounds.min_x, data.bounds.min_y, data.bounds.min_z],
    max: [data.bounds.max_x, data.bounds.max_y, data.bounds.max_z],
  };

  modelSize = Math.max(
    bounds.max[0] - bounds.min[0],
    bounds.max[1] - bounds.min[1],
    bounds.max[2] - bounds.min[2],
  );

  // Ensure modelSize is never 0
  if (modelSize === 0 || !isFinite(modelSize)) {
    modelSize = 100;
  }

  center = [
    (bounds.max[0] + bounds.min[0]) / 2,
    (bounds.max[1] + bounds.min[1]) / 2,
    (bounds.max[2] + bounds.min[2]) / 2,
  ];
  camera.distance = modelSize * 1.5;

  if (webgpuReady) {
    updateBuffers();
    createGrid();
    createAxes();
    if (printPathData) {
      createPrintPaths();
    }
    render();
  } else {
    pendingModelData = data;
  }
}

// Set print path data separately
function setModelPath(paths) {
  // Parse JSON if it's a string
  if (typeof paths === "string") {
    try {
      paths = JSON.parse(paths);
    } catch (e) {
      console.error("Failed to parse paths:", e);
      paths = null;
    }
  }

  // OPTIMIZATION: Store data directly without re-mapping entire structure
  if (paths && Array.isArray(paths) && paths.length > 0) {
    printPathData = {
      layers: [{ paths: paths }], // Wrap in layers structure to maintain compatibility
    };
  } else {
    printPathData = null;
  }

  if (webgpuReady && printPathData) {
    createPrintPaths();
    render();
  }
}

function setSliceConfig(config) {
  // Parse JSON if it's a string
  if (typeof config === "string") {
    try {
      config = JSON.parse(config);
    } catch (e) {
      console.error("Failed to parse config:", e);
      config = null;
    }
  }

  sliceConfig = config;
  console.log("Slice config set:", sliceConfig);

  // Recreate print paths if they exist to apply the new line width
  if (webgpuReady && printPathData) {
    createPrintPaths();
    render();
  }
}

// Function to create print path buffers from slice data
function createPrintPaths() {
  if (!printPathData || !printPathData.layers) {
    printPathVertexCount = 0;
    return;
  }

  const lineWidth =
    sliceConfig && sliceConfig.line_width ? sliceConfig.line_width : 0.4;
  console.log("Creating print paths, line width:", lineWidth);
  const startTime = performance.now();

  // --- PASS 1: Calculate total vertices needed ---
  let vertexCount = 0;

  for (const layer of printPathData.layers) {
    if (!layer.paths) continue;
    for (const path of layer.paths) {
      if (!path.segments || path.segments.length === 0) continue;

      const segments = path.segments;
      const numSegs = segments.length;
      if (numSegs === 0) continue;

      let currentLineType = segments[0].is_travel ? 0.0 : 1.0;
      let currentRunLen = 0;

      for (let i = 0; i < numSegs; i++) {
        const seg = segments[i];
        const lineType = seg.is_travel ? 0.0 : 1.0;

        if (lineType !== currentLineType) {
          // Vertices = 2 * (segments in run + 1)
          // Degenerate connection = 4 vertices
          vertexCount += 2 * (currentRunLen + 1) + 4;

          currentLineType = lineType;
          currentRunLen = 0;
        }
        currentRunLen++;
      }
      // Final run
      vertexCount += 2 * (currentRunLen + 1);

      // Degenerate triangles connecting to next PATH (if not last path)
      const isLastPath = (path === layer.paths[layer.paths.length - 1]);
      if (!isLastPath) {
        vertexCount += 4;
      }
    }
  }

  if (vertexCount === 0) {
    printPathVertexCount = 0;
    return;
  }

  console.log(`Allocating buffers for ${vertexCount} vertices`);

  // --- PASS 2: Fill Buffers ---
  const positions = new Float32Array(vertexCount * 3);
  const prevs = new Float32Array(vertexCount * 3);
  const nexts = new Float32Array(vertexCount * 3);
  const orientations = new Float32Array(vertexCount);
  const colors = new Float32Array(vertexCount * 3);
  const lineTypes = new Float32Array(vertexCount);

  let vIdx = 0; // Vertex index

  // Helper to add vertex
  const addVert = (p, prev, next, orient, color, type) => {
    const i = vIdx;
    // Pos
    positions[i * 3] = p.x; positions[i * 3 + 1] = p.y; positions[i * 3 + 2] = p.z;
    // Prev
    prevs[i * 3] = prev.x; prevs[i * 3 + 1] = prev.y; prevs[i * 3 + 2] = prev.z;
    // Next
    nexts[i * 3] = next.x; nexts[i * 3 + 1] = next.y; nexts[i * 3 + 2] = next.z;

    orientations[i] = orient;

    colors[i * 3] = color[0]; colors[i * 3 + 1] = color[1]; colors[i * 3 + 2] = color[2];
    lineTypes[i] = type;

    vIdx++;
  };

  const travelColor = [0.3, 0.6, 1.0];
  const extrudeColor = [1.0, 0.4, 0.1];

  for (const layer of printPathData.layers) {
    if (!layer.paths) continue;
    const pathCount = layer.paths.length;

    for (let pIdx = 0; pIdx < pathCount; pIdx++) {
      const path = layer.paths[pIdx];
      if (!path.segments || path.segments.length === 0) continue;

      const segments = path.segments;
      const numSegs = segments.length;

      // Iterate to find runs
      let runStartIdx = 0;
      let currentType = segments[0].is_travel ? 0.0 : 1.0;

      for (let i = 0; i <= numSegs; i++) {
        // Check if run ends (type change or end of segments)
        let typeChanged = false;

        if (i < numSegs) {
          const lineType = segments[i].is_travel ? 0.0 : 1.0;
          if (lineType !== currentType) typeChanged = true;
        } else {
          typeChanged = true; // Force end loop
        }

        if (typeChanged) {
          // Process run from runStartIdx to i-1
          const runColor = currentType === 0.0 ? travelColor : extrudeColor;
          const runLen = i - runStartIdx; // Number of segments
          const pointCount = runLen + 1; // Number of points

          // Generate vertices for this run
          for (let j = 0; j < pointCount; j++) {
            let curr, prev, next;

            // Get Current Point
            if (j === 0) {
              curr = segments[runStartIdx].start;
            } else {
              curr = segments[runStartIdx + j - 1].end;
            }

            // Get Prev Point
            if (j === 0) {
              prev = curr;
            } else {
              if (j === 1) prev = segments[runStartIdx].start;
              else prev = segments[runStartIdx + j - 2].end;
            }

            // Get Next Point
            if (j === pointCount - 1) {
              next = curr;
            } else {
              next = segments[runStartIdx + j].end;
            }

            // Add two vertices (left/right)
            addVert(curr, prev, next, -1.0, runColor, currentType);
            addVert(curr, prev, next, 1.0, runColor, currentType);
          }

          // Add degenerate triangles if needed
          const isLastRunOfPath = (i === numSegs);
          const isLastPath = (pIdx === pathCount - 1);

          if (!isLastRunOfPath || !isLastPath) {
            const lastP = segments[i - 1].end;
            addVert(lastP, lastP, lastP, 1.0, runColor, currentType);
            addVert(lastP, lastP, lastP, 1.0, runColor, currentType);
          }

          // Update for next run
          currentType = i < numSegs ? (segments[i].is_travel ? 0.0 : 1.0) : currentType;
          runStartIdx = i;
        }
      }
    }
  }

  console.log(`Optimized generation: ${Math.round(performance.now() - startTime)}ms`);

  const printPathPositionsArray = positions;
  const printPathPrevArray = prevs;
  const printPathNextArray = nexts;
  const printPathOrientationsArray = orientations;
  const printPathColorsArray = colors;
  const printPathLineTypesArray = lineTypes;
  printPathVertexCount = vIdx;

  if (printPathVertexCount === 0) return;

  console.log("Created", printPathVertexCount, "vertices for print paths");

  printPathBuffer = device.createBuffer({
    size: printPathPositionsArray.byteLength,
    usage: GPUBufferUsage.VERTEX | GPUBufferUsage.COPY_DST,
  });
  device.queue.writeBuffer(printPathBuffer, 0, printPathPositionsArray);

  const printPathPrevBuffer = device.createBuffer({
    size: printPathPrevArray.byteLength,
    usage: GPUBufferUsage.VERTEX | GPUBufferUsage.COPY_DST,
  });
  device.queue.writeBuffer(printPathPrevBuffer, 0, printPathPrevArray);

  const printPathNextBuffer = device.createBuffer({
    size: printPathNextArray.byteLength,
    usage: GPUBufferUsage.VERTEX | GPUBufferUsage.COPY_DST,
  });
  device.queue.writeBuffer(printPathNextBuffer, 0, printPathNextArray);

  const printPathOrientationBuffer = device.createBuffer({
    size: printPathOrientationsArray.byteLength,
    usage: GPUBufferUsage.VERTEX | GPUBufferUsage.COPY_DST,
  });
  device.queue.writeBuffer(
    printPathOrientationBuffer,
    0,
    printPathOrientationsArray,
  );

  printPathColorBuffer = device.createBuffer({
    size: printPathColorsArray.byteLength,
    usage: GPUBufferUsage.VERTEX | GPUBufferUsage.COPY_DST,
  });
  device.queue.writeBuffer(printPathColorBuffer, 0, printPathColorsArray);

  const printPathLineTypeBuffer = device.createBuffer({
    size: printPathLineTypesArray.byteLength,
    usage: GPUBufferUsage.VERTEX | GPUBufferUsage.COPY_DST,
  });
  device.queue.writeBuffer(printPathLineTypeBuffer, 0, printPathLineTypesArray);

  printPathUniformBuffer = device.createBuffer({
    size: 256, // 64 (viewProj) + 64 (model) + 4 (clipHeight) + 4 (thickness) + 4 (aspect) + padding
    usage: GPUBufferUsage.UNIFORM | GPUBufferUsage.COPY_DST,
  });

  printPathBindGroup = device.createBindGroup({
    layout: printPathPipeline.getBindGroupLayout(0),
    entries: [{ binding: 0, resource: { buffer: printPathUniformBuffer } }],
  });

  // Store these buffers so we can use them during rendering
  window.printPathPrevBuffer = printPathPrevBuffer;
  window.printPathNextBuffer = printPathNextBuffer;
  window.printPathOrientationBuffer = printPathOrientationBuffer;
  window.printPathLineTypeBuffer = printPathLineTypeBuffer;
}

// Function to switch between model and print path view
window.setViewMode = function (mode) {
  viewMode = mode;

  // Update button styles
  const modelBtn = document.getElementById("view-model-btn");
  const printBtn = document.getElementById("view-print-btn");

  if (mode === "model") {
    modelBtn.style.background = "#4CAF50";
    modelBtn.style.color = "white";
    modelBtn.style.borderColor = "#4CAF50";

    printBtn.style.background = "white";
    printBtn.style.color = "#666";
    printBtn.style.borderColor = "#ddd";
  } else {
    modelBtn.style.background = "white";
    modelBtn.style.color = "#666";
    modelBtn.style.borderColor = "#ddd";

    printBtn.style.background = "#FF5722";
    printBtn.style.color = "white";
    printBtn.style.borderColor = "#FF5722";
  }

  render();
};

function createGrid() {
  // Grid size: 2x the model dimensions in X and Y
  const gridWidth = (bounds.max[0] - bounds.min[0]) * 2;
  const gridLength = (bounds.max[1] - bounds.min[1]) * 2;
  const gridCenterX = (bounds.max[0] + bounds.min[0]) / 2;
  const gridCenterY = (bounds.max[1] + bounds.min[1]) / 2;
  const gridZ = 0; // Grid at z=0

  // Determine grid spacing (aim for ~20 lines per dimension)
  const spacing = Math.max(gridWidth, gridLength) / 20;

  const gridLines = [];

  // Lines parallel to X axis
  const numLinesY = Math.ceil(gridLength / spacing) + 1;
  for (let i = 0; i < numLinesY; i++) {
    const y = gridCenterY - gridLength / 2 + i * spacing;
    gridLines.push(
      gridCenterX - gridWidth / 2,
      y,
      gridZ,
      gridCenterX + gridWidth / 2,
      y,
      gridZ,
    );
  }

  // Lines parallel to Y axis
  const numLinesX = Math.ceil(gridWidth / spacing) + 1;
  for (let i = 0; i < numLinesX; i++) {
    const x = gridCenterX - gridWidth / 2 + i * spacing;
    gridLines.push(
      x,
      gridCenterY - gridLength / 2,
      gridZ,
      x,
      gridCenterY + gridLength / 2,
      gridZ,
    );
  }

  const gridVertices = new Float32Array(gridLines);
  gridVertexCount = gridVertices.length / 3;

  gridBuffer = device.createBuffer({
    size: gridVertices.byteLength,
    usage: GPUBufferUsage.VERTEX | GPUBufferUsage.COPY_DST,
  });
  device.queue.writeBuffer(gridBuffer, 0, gridVertices);

  gridUniformBuffer = device.createBuffer({
    size: 128,
    usage: GPUBufferUsage.UNIFORM | GPUBufferUsage.COPY_DST,
  });

  gridBindGroup = device.createBindGroup({
    layout: gridPipeline.getBindGroupLayout(0),
    entries: [{ binding: 0, resource: { buffer: gridUniformBuffer } }],
  });
}

function createAxes() {
  // Axis length: 30% of model size
  const axisLength = modelSize * 1;
  const origin = [0, 0, 0];

  // X axis (red), Y axis (blue), Z axis (green)
  const axisVertices = new Float32Array([
    // X axis - Red
    origin[0],
    origin[1],
    origin[2],
    origin[0] + axisLength,
    origin[1],
    origin[2],
    // Y axis - Blue
    origin[0],
    origin[1],
    origin[2],
    origin[0],
    origin[1] + axisLength,
    origin[2],
    // Z axis - Green
    origin[0],
    origin[1],
    origin[2],
    origin[0],
    origin[1],
    origin[2] + axisLength,
  ]);

  const axisColors = new Float32Array([
    // X axis - Red
    1.0, 0.0, 0.0, 1.0, 0.0, 0.0,
    // Y axis - Blue
    0.0, 0.0, 1.0, 0.0, 0.0, 1.0,
    // Z axis - Green
    0.0, 1.0, 0.0, 0.0, 1.0, 0.0,
  ]);

  axisVertexCount = axisVertices.length / 3;

  axisBuffer = device.createBuffer({
    size: axisVertices.byteLength,
    usage: GPUBufferUsage.VERTEX | GPUBufferUsage.COPY_DST,
  });
  device.queue.writeBuffer(axisBuffer, 0, axisVertices);

  axisColorBuffer = device.createBuffer({
    size: axisColors.byteLength,
    usage: GPUBufferUsage.VERTEX | GPUBufferUsage.COPY_DST,
  });
  device.queue.writeBuffer(axisColorBuffer, 0, axisColors);

  axisUniformBuffer = device.createBuffer({
    size: 128,
    usage: GPUBufferUsage.UNIFORM | GPUBufferUsage.COPY_DST,
  });

  axisBindGroup = device.createBindGroup({
    layout: axisPipeline.getBindGroupLayout(0),
    entries: [{ binding: 0, resource: { buffer: axisUniformBuffer } }],
  });
}

function updateBuffers() {
  vertexBuffer = device.createBuffer({
    size: vertices.byteLength,
    usage: GPUBufferUsage.VERTEX | GPUBufferUsage.COPY_DST,
  });
  device.queue.writeBuffer(vertexBuffer, 0, vertices);

  normalBuffer = device.createBuffer({
    size: normals.byteLength,
    usage: GPUBufferUsage.VERTEX | GPUBufferUsage.COPY_DST,
  });
  device.queue.writeBuffer(normalBuffer, 0, normals);

  uniformBuffer = device.createBuffer({
    size: 256,
    usage: GPUBufferUsage.UNIFORM | GPUBufferUsage.COPY_DST,
  });

  bindGroup = device.createBindGroup({
    layout: pipeline.getBindGroupLayout(0),
    entries: [{ binding: 0, resource: { buffer: uniformBuffer } }],
  });
}

// --- Mathe Helfer ---
function mat4Identity() {
  return new Float32Array([1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1]);
}
function mat4Multiply(a, b) {
  const out = new Float32Array(16);
  for (let i = 0; i < 4; i++) {
    for (let j = 0; j < 4; j++) {
      let sum = 0;
      for (let k = 0; k < 4; k++) sum += a[k * 4 + j] * b[i * 4 + k];
      out[i * 4 + j] = sum;
    }
  }
  return out;
}
function mat4Translate(x, y, z) {
  const m = mat4Identity();
  m[12] = x;
  m[13] = y;
  m[14] = z;
  return m;
}
function mat4RotateX(a) {
  const c = Math.cos(a),
    s = Math.sin(a),
    m = mat4Identity();
  m[5] = c;
  m[6] = s;
  m[9] = -s;
  m[10] = c;
  return m;
}
function mat4RotateY(a) {
  const c = Math.cos(a),
    s = Math.sin(a),
    m = mat4Identity();
  m[0] = c;
  m[2] = -s;
  m[8] = s;
  m[10] = c;
  return m;
}

// --- Schnitttiefe-Funktion ---
window.updateCutDepth = function (level) {
  currentClipLevel = Math.max(0, Math.min(1, level));
  render();
};

// --- Rendering ---
function render() {
  if (!webgpuReady || vertexCount === 0) {
    return;
  }
  if (canvas.width === 0 || canvas.height === 0) {
    return;
  }

  const aspect = canvas.width / canvas.height;
  const s = camera.distance;
  const near = -2000,
    far = 2000;
  const projection = new Float32Array([
    2 / (s * aspect),
    0,
    0,
    0,
    0,
    2 / s,
    0,
    0,
    0,
    0,
    1 / (near - far),
    0,
    0,
    0,
    near / (near - far),
    1,
  ]);

  let model = mat4RotateX(camera.rotationX);
  model = mat4Multiply(mat4RotateY(camera.rotationY), model);
  model = mat4Multiply(
    model,
    mat4Translate(-center[0], -center[1], -center[2]),
  );

  // Clip-Calculation
  const minH = bounds.min[2];
  const maxH = bounds.max[2];
  const clipH = minH + (maxH - minH) * currentClipLevel;

  const uniformData = new Float32Array(40);
  uniformData.set(projection, 0);
  uniformData.set(model, 16);
  uniformData.set([0.5, 0.5, 1.0, 0.0], 32);
  uniformData[36] = clipH;

  device.queue.writeBuffer(uniformBuffer, 0, uniformData);

  const encoder = device.createCommandEncoder();
  const pass = encoder.beginRenderPass({
    colorAttachments: [
      {
        view: ctx.getCurrentTexture().createView(),
        clearValue: { r: 0.85, g: 0.85, b: 0.85, a: 1.0 },
        loadOp: "clear",
        storeOp: "store",
      },
    ],
    depthStencilAttachment: {
      view: depthTextureView,
      depthClearValue: 1.0,
      depthLoadOp: "clear",
      depthStoreOp: "store",
    },
  });

  // Render grid first
  if (gridVertexCount > 0) {
    const gridUniformData = new Float32Array(32);
    gridUniformData.set(projection, 0);
    gridUniformData.set(model, 16);
    device.queue.writeBuffer(gridUniformBuffer, 0, gridUniformData);

    pass.setPipeline(gridPipeline);
    pass.setBindGroup(0, gridBindGroup);
    pass.setVertexBuffer(0, gridBuffer);
    pass.draw(gridVertexCount);
  }

  // Render axes
  if (axisVertexCount > 0) {
    const axisUniformData = new Float32Array(32);
    axisUniformData.set(projection, 0);
    axisUniformData.set(model, 16);
    device.queue.writeBuffer(axisUniformBuffer, 0, axisUniformData);

    pass.setPipeline(axisPipeline);
    pass.setBindGroup(0, axisBindGroup);
    pass.setVertexBuffer(0, axisBuffer);
    pass.setVertexBuffer(1, axisColorBuffer);
    pass.draw(axisVertexCount);
  }

  // Render based on view mode
  if (viewMode === "model") {
    // Render model
    pass.setPipeline(pipeline);
    pass.setBindGroup(0, bindGroup);
    pass.setVertexBuffer(0, vertexBuffer);
    pass.setVertexBuffer(1, normalBuffer);
    pass.draw(vertexCount);
  } else if (viewMode === "print" && printPathVertexCount > 0) {
    // Render print paths
    const lineWidth =
      sliceConfig && sliceConfig.line_width ? sliceConfig.line_width : 0.4;
    // Convert line width to screen space thickness (adjust this factor as needed)
    const screenThickness = (lineWidth / modelSize) * 0.5;

    const printPathUniformData = new Float32Array(38);
    printPathUniformData.set(projection, 0);
    printPathUniformData.set(model, 16);
    printPathUniformData[32] = clipH; // Clip height
    printPathUniformData[33] = screenThickness; // Thickness
    printPathUniformData[34] = aspect; // Aspect ratio
    device.queue.writeBuffer(printPathUniformBuffer, 0, printPathUniformData);

    pass.setPipeline(printPathPipeline);
    pass.setBindGroup(0, printPathBindGroup);
    pass.setVertexBuffer(0, printPathBuffer);
    pass.setVertexBuffer(1, window.printPathPrevBuffer);
    pass.setVertexBuffer(2, window.printPathNextBuffer);
    pass.setVertexBuffer(3, window.printPathOrientationBuffer);
    pass.setVertexBuffer(4, printPathColorBuffer);
    pass.setVertexBuffer(5, window.printPathLineTypeBuffer);
    pass.draw(printPathVertexCount);
  }

  pass.end();
  device.queue.submit([encoder.finish()]);
}
