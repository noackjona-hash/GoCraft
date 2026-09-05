package voxel

import (
	"math"
	"sort"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	DefaultChunkRenderRadius = 5 // Smooth 11x11 chunk grid (176x176 blocks) on GPU
)

// MeshBuilder accumulates vertex buffer data to upload into GPU VRAM VBOs
type MeshBuilder struct {
	Vertices  []float32
	Texcoords []float32
	Colors    []uint8
	Normals   []float32
	Indices   []uint16
	VertCount uint16
}

func newMeshBuilder() *MeshBuilder {
	return &MeshBuilder{
		Vertices:  make([]float32, 0, 32768),
		Texcoords: make([]float32, 0, 16384),
		Colors:    make([]uint8, 0, 32768),
		Normals:   make([]float32, 0, 32768),
		Indices:   make([]uint16, 0, 49152),
	}
}

func (mb *MeshBuilder) Reset() {
	mb.Vertices = mb.Vertices[:0]
	mb.Texcoords = mb.Texcoords[:0]
	mb.Colors = mb.Colors[:0]
	mb.Normals = mb.Normals[:0]
	mb.Indices = mb.Indices[:0]
	mb.VertCount = 0
}

func (mb *MeshBuilder) AddQuad(
	p0, p1, p2, p3 rl.Vector3,
	norm rl.Vector3,
	uMin, vMin, uMax, vMax float32,
	c0, c1, c2, c3 rl.Color,
	flipDiag bool,
) {
	baseIdx := mb.VertCount

	// 4 Vertices (X, Y, Z)
	mb.Vertices = append(mb.Vertices,
		p0.X, p0.Y, p0.Z,
		p1.X, p1.Y, p1.Z,
		p2.X, p2.Y, p2.Z,
		p3.X, p3.Y, p3.Z,
	)

	// 4 Normals (NX, NY, NZ)
	mb.Normals = append(mb.Normals,
		norm.X, norm.Y, norm.Z,
		norm.X, norm.Y, norm.Z,
		norm.X, norm.Y, norm.Z,
		norm.X, norm.Y, norm.Z,
	)

	// 4 Texcoords (UV matching p0=Bottom-Left, p1=Bottom-Right, p2=Top-Right, p3=Top-Left)
	mb.Texcoords = append(mb.Texcoords,
		uMin, vMax,
		uMax, vMax,
		uMax, vMin,
		uMin, vMin,
	)

	// 4 Colors (RGBA)
	mb.Colors = append(mb.Colors,
		c0.R, c0.G, c0.B, c0.A,
		c1.R, c1.G, c1.B, c1.A,
		c2.R, c2.G, c2.B, c2.A,
		c3.R, c3.G, c3.B, c3.A,
	)

	// 6 Indices (2 Triangles per quad)
	if flipDiag {
		mb.Indices = append(mb.Indices,
			baseIdx+1, baseIdx+2, baseIdx+3,
			baseIdx+1, baseIdx+3, baseIdx+0,
		)
	} else {
		mb.Indices = append(mb.Indices,
			baseIdx+0, baseIdx+1, baseIdx+2,
			baseIdx+0, baseIdx+2, baseIdx+3,
		)
	}

	mb.VertCount += 4
}

func (mb *MeshBuilder) BuildGPUMesh() (rl.Mesh, bool) {
	if mb.VertCount == 0 || len(mb.Indices) == 0 {
		return rl.Mesh{}, false
	}

	mesh := rl.Mesh{
		VertexCount:   int32(mb.VertCount),
		TriangleCount: int32(len(mb.Indices) / 3),
		Vertices:      &mb.Vertices[0],
		Texcoords:     &mb.Texcoords[0],
		Colors:        &mb.Colors[0],
		Normals:       &mb.Normals[0],
		Indices:       &mb.Indices[0],
	}

	rl.UploadMesh(&mesh, false) // Uploads to GPU VRAM (OpenGL VAO/VBOs)
	return mesh, true
}

// Chunk stores the GPU VAO meshes for solid, cutout, and transparent water geometry
type Chunk struct {
	Coord      ChunkCoord
	OpaqueMesh rl.Mesh
	HasOpaque  bool
	CutoutMesh rl.Mesh
	HasCutout  bool
	WaterMesh  rl.Mesh
	HasWater   bool
	IsDirty    bool
	CenterX    float32
	CenterZ    float32
}

const minecraftVertShader = `#version 330
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec4 vertexColor;
in vec3 vertexNormal;

out vec2 fragTexCoord;
out vec3 fragLight;     // r=skyLight, g=blockLight, b=ao*facet
out vec3 fragNormal;
out vec3 fragWorldPos;
out float fragAlpha;
out float vertexDistance;

uniform mat4 mvp;
uniform mat4 matModel;
uniform vec3 viewPos;

void main() {
    fragTexCoord = vertexTexCoord;
    vec4 worldPos = matModel * vec4(vertexPosition, 1.0);
    fragWorldPos = worldPos.xyz;
    fragNormal = normalize(mat3(matModel) * vertexNormal);
    vertexDistance = length(worldPos.xyz - viewPos);

    fragLight = vertexColor.rgb;
    fragAlpha = vertexColor.a;

    gl_Position = mvp * vec4(vertexPosition, 1.0);
}
`

const minecraftFragShader = `#version 330
in vec2 fragTexCoord;
in vec3 fragLight;
in vec3 fragNormal;
in vec3 fragWorldPos;
in float fragAlpha;
in float vertexDistance;

out vec4 finalColor;

uniform sampler2D texture0;
uniform vec4 colDiffuse;
uniform vec3 lightDir0;
uniform float sunIntensity;
uniform vec3 sunColor;
uniform vec3 torchColor;
uniform float time;
uniform vec3 heldLightPos;
uniform float heldLightLevel;
uniform float FogStart;
uniform float FogEnd;
uniform vec4 FogColor;
uniform float isCutout;

// ACES Filmic Tone Mapping for authentic cinematic depth
vec3 ACESFilm(vec3 x) {
    float a = 2.51;
    float b = 0.03;
    float c = 2.43;
    float d = 0.59;
    float e = 0.14;
    return clamp((x*(a*x+b))/(x*(c*x+d)+e), 0.0, 1.0);
}

// Fog implementation from assets/shaders/include/fog.glsl
vec4 linear_fog(vec4 inColor, float dist, float fogStart, float fogEnd, vec4 fogCol) {
    if (dist <= fogStart) {
        return inColor;
    }
    float fogValue = dist < fogEnd ? smoothstep(fogStart, fogEnd, dist) : 1.0;
    return vec4(mix(inColor.rgb, fogCol.rgb, fogValue * fogCol.a), inColor.a);
}

void main() {
    vec4 texelColor = texture(texture0, fragTexCoord);
    if (isCutout > 0.5 && texelColor.a < 0.1) discard;

    float skyLight = fragLight.r;
    float blockLight = fragLight.g;
    float aoAndFacet = fragLight.b;

    // Organic flame flicker
    float flicker = 1.0 + sin(time * 7.5 + fragWorldPos.x * 2.3) * 0.025 + cos(time * 14.0 + fragWorldPos.z * 1.7) * 0.018;

    // Dynamic handheld torch lighting (OptiFine style!)
    float heldDist = length(fragWorldPos - heldLightPos);
    if (heldLightLevel > 0.0 && heldDist < 13.0) {
        float hLight = clamp((13.0 - heldDist) / 13.0, 0.0, 1.0) * heldLightLevel;
        hLight = pow(hLight, 1.4);
        blockLight = max(blockLight, hLight);
    }

    // Authentic non-linear Minecraft light curves
    float skyCurve = pow(skyLight, 1.4) * sunIntensity;
    vec3 skyContrib = sunColor * skyCurve;

    float blockCurve = pow(blockLight, 1.45) * flicker;
    vec3 torchContrib = torchColor * blockCurve;

    // Subtle cave ambient (pitch black in deep caves, no grey washout!)
    vec3 caveAmbient = vec3(0.028, 0.032, 0.042);

    // Combine light contributions
    vec3 totalLight = (max(skyContrib, torchContrib) + caveAmbient + skyContrib * 0.10 + torchContrib * 0.10) * aoAndFacet;

    // Directional sunlight bounce on top/sloped faces
    vec3 l0 = normalize(lightDir0);
    float sunFacing = max(0.0, dot(fragNormal, l0));
    totalLight += sunColor * sunFacing * sunIntensity * 0.18 * skyLight;

    vec3 tonemapped = ACESFilm(totalLight * 1.18);
    vec4 litColor = texelColor * vec4(tonemapped, fragAlpha) * colDiffuse;

    finalColor = linear_fog(litColor, vertexDistance, FogStart, FogEnd, FogColor);
}
`

type ChunkManager struct {
	Chunks         map[ChunkCoord]*Chunk
	World          *VoxelWorld
	Atlas          *TextureAtlas
	Material       rl.Material
	CutoutMaterial rl.Material
	OpaqueMB       *MeshBuilder
	CutoutMB       *MeshBuilder
	WaterMB        *MeshBuilder
	IdentityMat    rl.Matrix
	RenderRadius   int

	// Shader Uniform Locations
	Shader                 rl.Shader
	CutoutShader           rl.Shader
	ViewPosLoc             int32
	LightDirLoc            int32
	SunIntensityLoc        int32
	SunColorLoc            int32
	TorchColorLoc          int32
	TimeLoc                int32
	HeldLightPosLoc        int32
	HeldLightLevelLoc      int32
	FogStartLoc            int32
	FogEndLoc              int32
	FogColorLoc            int32
	CutoutViewPosLoc       int32
	CutoutLightDirLoc      int32
	CutoutSunIntLoc        int32
	CutoutSunColorLoc      int32
	CutoutTorchColorLoc    int32
	CutoutTimeLoc          int32
	CutoutHeldLightPosLoc  int32
	CutoutHeldLightLevelLoc int32
	CutoutFogStartLoc      int32
	CutoutFogEndLoc        int32
	CutoutFogColorLoc      int32
	CutoutFlagLoc          int32
}

// NewChunkManager creates the infinite chunk streaming manager with authentic Minecraft GPU shaders
func NewChunkManager(w *VoxelWorld, atlas *TextureAtlas) *ChunkManager {
	// 1. Opaque & Water Material with Minecraft Shader
	opaqueShader := rl.LoadShaderFromMemory(minecraftVertShader, minecraftFragShader)
	viewPosLoc := rl.GetShaderLocation(opaqueShader, "viewPos")
	lightDirLoc := rl.GetShaderLocation(opaqueShader, "lightDir0")
	sunIntLoc := rl.GetShaderLocation(opaqueShader, "sunIntensity")
	sunColorLoc := rl.GetShaderLocation(opaqueShader, "sunColor")
	torchColorLoc := rl.GetShaderLocation(opaqueShader, "torchColor")
	timeLoc := rl.GetShaderLocation(opaqueShader, "time")
	heldLightPosLoc := rl.GetShaderLocation(opaqueShader, "heldLightPos")
	heldLightLevelLoc := rl.GetShaderLocation(opaqueShader, "heldLightLevel")
	fogStartLoc := rl.GetShaderLocation(opaqueShader, "FogStart")
	fogEndLoc := rl.GetShaderLocation(opaqueShader, "FogEnd")
	fogColorLoc := rl.GetShaderLocation(opaqueShader, "FogColor")

	mat := rl.LoadMaterialDefault()
	mat.Shader = opaqueShader
	rl.SetMaterialTexture(&mat, 0, atlas.Texture)

	// 2. Cutout Material (Leaves, Glass, Torches) with Alpha Discard + Minecraft Shader
	cutoutShader := rl.LoadShaderFromMemory(minecraftVertShader, minecraftFragShader)
	cViewPosLoc := rl.GetShaderLocation(cutoutShader, "viewPos")
	cLightDirLoc := rl.GetShaderLocation(cutoutShader, "lightDir0")
	cSunIntLoc := rl.GetShaderLocation(cutoutShader, "sunIntensity")
	cSunColorLoc := rl.GetShaderLocation(cutoutShader, "sunColor")
	cTorchColorLoc := rl.GetShaderLocation(cutoutShader, "torchColor")
	cTimeLoc := rl.GetShaderLocation(cutoutShader, "time")
	cHeldLightPosLoc := rl.GetShaderLocation(cutoutShader, "heldLightPos")
	cHeldLightLevelLoc := rl.GetShaderLocation(cutoutShader, "heldLightLevel")
	cFogStartLoc := rl.GetShaderLocation(cutoutShader, "FogStart")
	cFogEndLoc := rl.GetShaderLocation(cutoutShader, "FogEnd")
	cFogColorLoc := rl.GetShaderLocation(cutoutShader, "FogColor")
	cCutoutFlagLoc := rl.GetShaderLocation(cutoutShader, "isCutout")

	// Set isCutout = 1.0 on cutout shader
	rl.SetShaderValue(cutoutShader, cCutoutFlagLoc, []float32{1.0}, rl.ShaderUniformFloat)

	cutoutMat := rl.LoadMaterialDefault()
	cutoutMat.Shader = cutoutShader
	rl.SetMaterialTexture(&cutoutMat, 0, atlas.Texture)

	cm := &ChunkManager{
		Chunks:         make(map[ChunkCoord]*Chunk),
		World:          w,
		Atlas:          atlas,
		Material:       mat,
		CutoutMaterial: cutoutMat,
		OpaqueMB:       newMeshBuilder(),
		CutoutMB:       newMeshBuilder(),
		WaterMB:        newMeshBuilder(),
		IdentityMat:    rl.MatrixIdentity(),

		Shader:                 opaqueShader,
		CutoutShader:           cutoutShader,
		ViewPosLoc:             viewPosLoc,
		LightDirLoc:            lightDirLoc,
		SunIntensityLoc:        sunIntLoc,
		SunColorLoc:            sunColorLoc,
		TorchColorLoc:          torchColorLoc,
		TimeLoc:                timeLoc,
		HeldLightPosLoc:        heldLightPosLoc,
		HeldLightLevelLoc:      heldLightLevelLoc,
		FogStartLoc:            fogStartLoc,
		FogEndLoc:              fogEndLoc,
		FogColorLoc:            fogColorLoc,
		CutoutViewPosLoc:       cViewPosLoc,
		CutoutLightDirLoc:      cLightDirLoc,
		CutoutSunIntLoc:        cSunIntLoc,
		CutoutSunColorLoc:      cSunColorLoc,
		CutoutTorchColorLoc:    cTorchColorLoc,
		CutoutTimeLoc:          cTimeLoc,
		CutoutHeldLightPosLoc:  cHeldLightPosLoc,
		CutoutHeldLightLevelLoc: cHeldLightLevelLoc,
		CutoutFogStartLoc:      cFogStartLoc,
		CutoutFogEndLoc:        cFogEndLoc,
		CutoutFogColorLoc:      cFogColorLoc,
		CutoutFlagLoc:          cCutoutFlagLoc,
		RenderRadius:           DefaultChunkRenderRadius,
	}

	// Set initial fog & lighting parameters
	cm.UpdateFogAndSky(rl.NewColor(135, 206, 235, 255), false, rl.Vector3{}, 0.2, 0.8, 0, rl.Vector3{}, 0)

	return cm
}

// CycleRenderRadius cycles render distance between 3, 4, 5, 6 chunks
func (cm *ChunkManager) CycleRenderRadius() int {
	cm.RenderRadius++
	if cm.RenderRadius > 6 {
		cm.RenderRadius = 3
	}
	return cm.RenderRadius
}

// UpdateFogAndSky passes dynamic sky color, sunlight direction, torch light, time, and handheld light into GPU chunk shaders
func (cm *ChunkManager) UpdateFogAndSky(skyCol rl.Color, isUnderwater bool, cameraPos rl.Vector3, sunAngle, sunHeight float32, time float32, heldLightPos rl.Vector3, heldLightLevel float32) {
	fogR := float32(skyCol.R) / 255.0
	fogG := float32(skyCol.G) / 255.0
	fogB := float32(skyCol.B) / 255.0

	radius := cm.RenderRadius
	if radius <= 0 {
		radius = DefaultChunkRenderRadius
	}

	near := float32(radius*ChunkSize) * 0.55
	far := float32(radius*ChunkSize) * 0.95
	if isUnderwater {
		near = 2.0
		far = 28.0
	}

	// Realistic dynamic sunlight intensity: bright day (1.0) -> sunset (0.45) -> night (0.04)
	sunIntensity := float32(math.Max(0.04, float64((sunHeight+0.25)/1.25)))
	sunDirX := float32(math.Cos(float64(sunAngle)))
	sunDirY := sunHeight
	sunDirZ := float32(0.0)

	// Dynamic sunlight color calculation
	var sunColR, sunColG, sunColB float32
	if sunHeight > 0.25 {
		// Daylight: Crisp warm sun
		sunColR = 1.00
		sunColG = 0.98
		sunColB = 0.90
	} else if sunHeight > -0.10 {
		// Sunset / Sunrise: Dramatic golden-orange hour
		t := (sunHeight + 0.10) / 0.35 // 0 to 1
		sunColR = 1.00
		sunColG = 0.35 + t*0.63
		sunColB = 0.12 + t*0.78
	} else {
		// Night: Pale moonlight
		sunColR = 0.12
		sunColG = 0.18
		sunColB = 0.32
	}

	// Warm amber torchlight
	torchCol := []float32{1.00, 0.65, 0.25}
	sunCol := []float32{sunColR, sunColG, sunColB}
	timeVal := []float32{time}
	heldPos := []float32{heldLightPos.X, heldLightPos.Y, heldLightPos.Z}
	heldLevel := []float32{heldLightLevel}

	fogCol := []float32{fogR, fogG, fogB, 1.0}
	nearVal := []float32{near}
	farVal := []float32{far}
	viewPos := []float32{cameraPos.X, cameraPos.Y, cameraPos.Z}
	lightDir := []float32{sunDirX, sunDirY, sunDirZ}
	sunInt := []float32{sunIntensity}

	// Update Opaque Shader
	rl.SetShaderValue(cm.Shader, cm.ViewPosLoc, viewPos, rl.ShaderUniformVec3)
	rl.SetShaderValue(cm.Shader, cm.LightDirLoc, lightDir, rl.ShaderUniformVec3)
	rl.SetShaderValue(cm.Shader, cm.SunIntensityLoc, sunInt, rl.ShaderUniformFloat)
	rl.SetShaderValue(cm.Shader, cm.SunColorLoc, sunCol, rl.ShaderUniformVec3)
	rl.SetShaderValue(cm.Shader, cm.TorchColorLoc, torchCol, rl.ShaderUniformVec3)
	rl.SetShaderValue(cm.Shader, cm.TimeLoc, timeVal, rl.ShaderUniformFloat)
	rl.SetShaderValue(cm.Shader, cm.HeldLightPosLoc, heldPos, rl.ShaderUniformVec3)
	rl.SetShaderValue(cm.Shader, cm.HeldLightLevelLoc, heldLevel, rl.ShaderUniformFloat)
	rl.SetShaderValue(cm.Shader, cm.FogStartLoc, nearVal, rl.ShaderUniformFloat)
	rl.SetShaderValue(cm.Shader, cm.FogEndLoc, farVal, rl.ShaderUniformFloat)
	rl.SetShaderValue(cm.Shader, cm.FogColorLoc, fogCol, rl.ShaderUniformVec4)

	// Update Cutout Shader
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutViewPosLoc, viewPos, rl.ShaderUniformVec3)
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutLightDirLoc, lightDir, rl.ShaderUniformVec3)
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutSunIntLoc, sunInt, rl.ShaderUniformFloat)
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutSunColorLoc, sunCol, rl.ShaderUniformVec3)
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutTorchColorLoc, torchCol, rl.ShaderUniformVec3)
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutTimeLoc, timeVal, rl.ShaderUniformFloat)
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutHeldLightPosLoc, heldPos, rl.ShaderUniformVec3)
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutHeldLightLevelLoc, heldLevel, rl.ShaderUniformFloat)
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutFogStartLoc, nearVal, rl.ShaderUniformFloat)
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutFogEndLoc, farVal, rl.ShaderUniformFloat)
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutFogColorLoc, fogCol, rl.ShaderUniformVec4)
}

// UpdatePlayerPos streams chunks around the player as they explore the infinite world
func (cm *ChunkManager) UpdatePlayerPos(playerPos rl.Vector3) {
	pcx := int(math.Floor(float64(playerPos.X))) >> 4
	pcz := int(math.Floor(float64(playerPos.Z))) >> 4

	radius := cm.RenderRadius
	if radius <= 0 {
		radius = DefaultChunkRenderRadius
	}

	// 1. Discover all needed chunks and queue dirty ones
	type chunkDist struct {
		chunk  *Chunk
		distSq int
	}
	var dirtyList []chunkDist

	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			coord := ChunkCoord{X: pcx + dx, Z: pcz + dz}
			chunk, exists := cm.Chunks[coord]
			if !exists {
				chunk = &Chunk{
					Coord:   coord,
					IsDirty: true,
					CenterX: float32(coord.X*ChunkSize + ChunkSize/2),
					CenterZ: float32(coord.Z*ChunkSize + ChunkSize/2),
				}
				cm.Chunks[coord] = chunk
			}

			if chunk.IsDirty {
				distSq := dx*dx + dz*dz
				dirtyList = append(dirtyList, chunkDist{chunk: chunk, distSq: distSq})
			}
		}
	}

	// 2. Sort dirty chunks by distance to player (closest chunks rebuilt first!)
	sort.Slice(dirtyList, func(i, j int) bool {
		return dirtyList[i].distSq < dirtyList[j].distSq
	})

	// 3. Rebuild at most 2 chunks per frame to guarantee perfectly smooth 60-144+ FPS
	rebuilt := 0
	maxRebuilds := 2
	for _, item := range dirtyList {
		if rebuilt >= maxRebuilds {
			break
		}
		cm.RebuildChunkMeshes(item.chunk)
		item.chunk.IsDirty = false
		rebuilt++
	}

	// 4. Unload chunks far outside render radius to save GPU memory
	unloadRadius := radius + 2
	for coord, chunk := range cm.Chunks {
		dx := coord.X - pcx
		dz := coord.Z - pcz
		if dx < -unloadRadius || dx > unloadRadius || dz < -unloadRadius || dz > unloadRadius {
			unloadChunkGPUMeshes(chunk)
			delete(cm.Chunks, coord)
		}
	}
}

// MarkBlockDirty flags the chunk at (x, z) for rebuild (and neighbors only if on border)
func (cm *ChunkManager) MarkBlockDirty(x, z int) {
	cx := x >> 4
	cz := z >> 4
	lx := x & 15
	lz := z & 15

	if chunk, exists := cm.Chunks[ChunkCoord{X: cx, Z: cz}]; exists {
		chunk.IsDirty = true
	}

	// Only mark neighbors if the block is on the outer boundary
	if lx == 0 {
		if chunk, exists := cm.Chunks[ChunkCoord{X: cx - 1, Z: cz}]; exists {
			chunk.IsDirty = true
		}
	} else if lx == 15 {
		if chunk, exists := cm.Chunks[ChunkCoord{X: cx + 1, Z: cz}]; exists {
			chunk.IsDirty = true
		}
	}

	if lz == 0 {
		if chunk, exists := cm.Chunks[ChunkCoord{X: cx, Z: cz - 1}]; exists {
			chunk.IsDirty = true
		}
	} else if lz == 15 {
		if chunk, exists := cm.Chunks[ChunkCoord{X: cx, Z: cz + 1}]; exists {
			chunk.IsDirty = true
		}
	}
}

// unloadChunkGPUMeshes frees old OpenGL VAO buffers from the GPU
func unloadChunkGPUMeshes(c *Chunk) {
	if c.HasOpaque && c.OpaqueMesh.VaoID > 0 {
		rl.UnloadVertexArray(c.OpaqueMesh.VaoID)
		c.HasOpaque = false
		c.OpaqueMesh.VaoID = 0
	}
	if c.HasCutout && c.CutoutMesh.VaoID > 0 {
		rl.UnloadVertexArray(c.CutoutMesh.VaoID)
		c.HasCutout = false
		c.CutoutMesh.VaoID = 0
	}
	if c.HasWater && c.WaterMesh.VaoID > 0 {
		rl.UnloadVertexArray(c.WaterMesh.VaoID)
		c.HasWater = false
		c.WaterMesh.VaoID = 0
	}
}

// calculateMinecraftVertexColor encodes SkyLight, BlockLight, AO, and Face Shading into vertex attributes
func calculateMinecraftVertexColor(skyLight, torchLight, ao, faceMult float32, alpha uint8) rl.Color {
	r := uint8(skyLight * 255.0)
	g := uint8(torchLight * 255.0)
	b := uint8(ao * faceMult * 255.0)
	return rl.NewColor(r, g, b, alpha)
}

// RebuildChunkMeshes compiles chunk voxels into optimized GPU VBOs
func (cm *ChunkManager) RebuildChunkMeshes(c *Chunk) {
	unloadChunkGPUMeshes(c)

	cm.OpaqueMB.Reset()
	cm.CutoutMB.Reset()
	cm.WaterMB.Reset()

	startX := c.Coord.X * ChunkSize
	endX := startX + ChunkSize
	startZ := c.Coord.Z * ChunkSize
	endZ := startZ + ChunkSize

	w := cm.World

	// --- LIGHTING COMPUTATION ---
	lightMap := CalculateLocalLightMap(w, c.Coord.X, c.Coord.Z)

	for x := startX; x < endX; x++ {
		for z := startZ; z < endZ; z++ {
			for y := 0; y < WorldHeight; y++ {
				bType := w.GetBlock(x, y, z)
				if bType == BlockAir {
					continue
				}

				x0 := float32(x)
				x1 := float32(x + 1)
				y0 := float32(y)
				y1 := float32(y + 1)
				z0 := float32(z)
				z1 := float32(z + 1)

				isWaterBlock := IsWater(bType)
				blockAlpha := uint8(255)
				if isWaterBlock {
					blockAlpha = 205
				}

				// Determine destination MeshBuilder
				var mb *MeshBuilder
				if isWaterBlock {
					mb = cm.WaterMB
				} else if IsLeaf(bType) || bType == BlockGlass || bType == BlockTorch || IsPlant(bType) {
					mb = cm.CutoutMB
				} else {
					mb = cm.OpaqueMB
				}

				// --- AUTHENTIC MINECRAFT CROSSED-QUADS TORCH RENDERING ---
				if bType == BlockTorch {
					uMin, vMin, uMax, vMax := GetBlockTextureUVs(BlockTorch, FaceNorth)
					tCol := rl.NewColor(255, 255, 230, 255)
					torchNorm := rl.Vector3{X: 0, Y: 1, Z: 0}

					// Check wall attachment
					isWall := false
					wallOffsetX := float32(0)
					wallOffsetZ := float32(0)

					if w.GetBlock(x-1, y, z) != BlockAir && w.GetBlock(x-1, y, z) != BlockTorch && w.GetBlock(x-1, y, z) != BlockWater {
						isWall = true
						wallOffsetX = -0.20
					} else if w.GetBlock(x+1, y, z) != BlockAir && w.GetBlock(x+1, y, z) != BlockTorch && w.GetBlock(x+1, y, z) != BlockWater {
						isWall = true
						wallOffsetX = 0.20
					} else if w.GetBlock(x, y, z-1) != BlockAir && w.GetBlock(x, y, z-1) != BlockTorch && w.GetBlock(x, y, z-1) != BlockWater {
						isWall = true
						wallOffsetZ = -0.20
					} else if w.GetBlock(x, y, z+1) != BlockAir && w.GetBlock(x, y, z+1) != BlockTorch && w.GetBlock(x, y, z+1) != BlockWater {
						isWall = true
						wallOffsetZ = 0.20
					}

					baseY := y0
					if isWall {
						baseY = y0 + 0.12
					}

					cx := x0 + 0.5 + wallOffsetX
					cz := z0 + 0.5 + wallOffsetZ
					h := float32(0.85)
					wHalf := float32(0.35)

					// Diagonal Quad 1
					mb.AddQuad(
						rl.Vector3{X: cx - wHalf, Y: baseY, Z: cz - wHalf},
						rl.Vector3{X: cx + wHalf, Y: baseY, Z: cz + wHalf},
						rl.Vector3{X: cx + wHalf, Y: baseY + h, Z: cz + wHalf},
						rl.Vector3{X: cx - wHalf, Y: baseY + h, Z: cz - wHalf},
						torchNorm,
						uMin, vMin, uMax, vMax, tCol, tCol, tCol, tCol, false,
					)
					// Diagonal Quad 2
					mb.AddQuad(
						rl.Vector3{X: cx - wHalf, Y: baseY, Z: cz + wHalf},
						rl.Vector3{X: cx + wHalf, Y: baseY, Z: cz - wHalf},
						rl.Vector3{X: cx + wHalf, Y: baseY + h, Z: cz - wHalf},
						rl.Vector3{X: cx - wHalf, Y: baseY + h, Z: cz + wHalf},
						torchNorm,
						uMin, vMin, uMax, vMax, tCol, tCol, tCol, tCol, false,
					)
					continue
				}

				// --- AUTHENTIC MINECRAFT CROSSED-QUADS PLANT RENDERING ---
				if IsPlant(bType) {
					uMin, vMin, uMax, vMax := GetBlockTextureUVs(bType, FaceNorth)
					skyLight, torchLight := lightMap.GetLight(x, y, z)
					pCol := calculateMinecraftVertexColor(skyLight, torchLight, 1.0, 0.92, 255)
					plantNorm := rl.Vector3{X: 0, Y: 1, Z: 0}

					cx := x0 + 0.5
					cz := z0 + 0.5
					h := float32(1.0)
					if bType == BlockRedMushroom || bType == BlockBrownMushroom {
						h = 0.65
					} else if bType == BlockDandelion || bType == BlockPoppy || bType == BlockCornflower || bType == BlockAllium {
						h = 0.85
					}
					wHalf := float32(0.45)

					// Diagonal Quad 1
					mb.AddQuad(
						rl.Vector3{X: cx - wHalf, Y: y0, Z: cz - wHalf},
						rl.Vector3{X: cx + wHalf, Y: y0, Z: cz + wHalf},
						rl.Vector3{X: cx + wHalf, Y: y0 + h, Z: cz + wHalf},
						rl.Vector3{X: cx - wHalf, Y: y0 + h, Z: cz - wHalf},
						plantNorm,
						uMin, vMin, uMax, vMax, pCol, pCol, pCol, pCol, false,
					)
					// Diagonal Quad 2
					mb.AddQuad(
						rl.Vector3{X: cx - wHalf, Y: y0, Z: cz + wHalf},
						rl.Vector3{X: cx + wHalf, Y: y0, Z: cz - wHalf},
						rl.Vector3{X: cx + wHalf, Y: y0 + h, Z: cz - wHalf},
						rl.Vector3{X: cx - wHalf, Y: y0 + h, Z: cz + wHalf},
						plantNorm,
						uMin, vMin, uMax, vMax, pCol, pCol, pCol, pCol, false,
					)
					continue
				}

				// --- NEIGHBOR OCCLUSION & WATER CULLING ---
				topBlock := BlockAir
				if y < WorldHeight-1 {
					topBlock = w.GetBlock(x, y+1, z)
				}
				bottomBlock := BlockAir
				if y > 0 {
					bottomBlock = w.GetBlock(x, y-1, z)
				}
				northBlock := w.GetBlock(x, y, z-1)
				southBlock := w.GetBlock(x, y, z+1)
				westBlock := w.GetBlock(x-1, y, z)
				eastBlock := w.GetBlock(x+1, y, z)

				yCorner0 := y1
				yCorner1 := y1
				yCorner2 := y1
				yCorner3 := y1

				if isWaterBlock {
					// Authentic Minecraft fluid levels & smooth corner height averaging
					yCorner0 = y0 + w.GetWaterCornerHeight(x, y, z, 0)
					yCorner1 = y0 + w.GetWaterCornerHeight(x, y, z, 1)
					yCorner2 = y0 + w.GetWaterCornerHeight(x, y, z, 2)
					yCorner3 = y0 + w.GetWaterCornerHeight(x, y, z, 3)
				}

				var topAir, bottomAir, northAir, southAir, westAir, eastAir bool

				if isWaterBlock {
					// Water faces only rendered against AIR or non-solid blocks
					// (NEVER against solid blocks like sand, dirt, clay, stone - this eliminates 100% of Z-fighting!)
					topAir = !IsWater(topBlock)
					bottomAir = y > 0 && !IsWater(bottomBlock) && !w.IsSolid(x, y-1, z)
					northAir = !IsWater(northBlock) && !w.IsSolid(x, y, z-1)
					southAir = !IsWater(southBlock) && !w.IsSolid(x, y, z+1)
					westAir = !IsWater(westBlock) && !w.IsSolid(x-1, y, z)
					eastAir = !IsWater(eastBlock) && !w.IsSolid(x+1, y, z)
				} else if bType == BlockGlass {
					topAir = topBlock != BlockGlass && (y == WorldHeight-1 || BlockRegistry[topBlock].IsTransparent)
					bottomAir = bottomBlock != BlockGlass && (y > 0 && BlockRegistry[bottomBlock].IsTransparent)
					northAir = northBlock != BlockGlass && BlockRegistry[northBlock].IsTransparent
					southAir = southBlock != BlockGlass && BlockRegistry[southBlock].IsTransparent
					westAir = westBlock != BlockGlass && BlockRegistry[westBlock].IsTransparent
					eastAir = eastBlock != BlockGlass && BlockRegistry[eastBlock].IsTransparent
				} else if IsLeaf(bType) {
					// Intelligent leaf culling: cull interior faces between same leaves or opaque blocks
					topAir = y == WorldHeight-1 || (topBlock != bType && (BlockRegistry[topBlock].IsTransparent || !BlockRegistry[topBlock].IsSolid))
					bottomAir = y > 0 && (bottomBlock != bType && (BlockRegistry[bottomBlock].IsTransparent || !BlockRegistry[bottomBlock].IsSolid))
					northAir = northBlock != bType && (BlockRegistry[northBlock].IsTransparent || !BlockRegistry[northBlock].IsSolid)
					southAir = southBlock != bType && (BlockRegistry[southBlock].IsTransparent || !BlockRegistry[southBlock].IsSolid)
					westAir = westBlock != bType && (BlockRegistry[westBlock].IsTransparent || !BlockRegistry[westBlock].IsSolid)
					eastAir = eastBlock != bType && (BlockRegistry[eastBlock].IsTransparent || !BlockRegistry[eastBlock].IsSolid)
				} else {
					topAir = y == WorldHeight-1 || BlockRegistry[topBlock].IsTransparent
					bottomAir = y > 0 && BlockRegistry[bottomBlock].IsTransparent
					northAir = BlockRegistry[northBlock].IsTransparent
					southAir = BlockRegistry[southBlock].IsTransparent
					westAir = BlockRegistry[westBlock].IsTransparent
					eastAir = BlockRegistry[eastBlock].IsTransparent
				}

				if !topAir && !bottomAir && !northAir && !southAir && !westAir && !eastAir {
					continue
				}

				// --- 1. TOP FACE (+Y) ---
				if topAir {
					uMin, vMin, uMax, vMax := GetBlockTextureUVs(bType, FaceTop)
					faceMult := FaceLightingMultipliers[FaceTop]
					skyLight, torchLight := lightMap.GetLight(x, y+1, z)

					ao0 := CalculateVertexAO(w.IsSolid(x-1, y+1, z), w.IsSolid(x, y+1, z+1), w.IsSolid(x-1, y+1, z+1))
					ao1 := CalculateVertexAO(w.IsSolid(x+1, y+1, z), w.IsSolid(x, y+1, z+1), w.IsSolid(x+1, y+1, z+1))
					ao2 := CalculateVertexAO(w.IsSolid(x+1, y+1, z), w.IsSolid(x, y+1, z-1), w.IsSolid(x+1, y+1, z-1))
					ao3 := CalculateVertexAO(w.IsSolid(x-1, y+1, z), w.IsSolid(x, y+1, z-1), w.IsSolid(x-1, y+1, z-1))

					c0 := calculateMinecraftVertexColor(skyLight, torchLight, ao0, faceMult, blockAlpha)
					c1 := calculateMinecraftVertexColor(skyLight, torchLight, ao1, faceMult, blockAlpha)
					c2 := calculateMinecraftVertexColor(skyLight, torchLight, ao2, faceMult, blockAlpha)
					c3 := calculateMinecraftVertexColor(skyLight, torchLight, ao3, faceMult, blockAlpha)

					p0 := rl.Vector3{X: x0, Y: yCorner0, Z: z1}
					p1 := rl.Vector3{X: x1, Y: yCorner1, Z: z1}
					p2 := rl.Vector3{X: x1, Y: yCorner2, Z: z0}
					p3 := rl.Vector3{X: x0, Y: yCorner3, Z: z0}
					norm := rl.Vector3{X: 0, Y: 1, Z: 0}

					mb.AddQuad(p0, p1, p2, p3, norm, uMin, vMin, uMax, vMax, c0, c1, c2, c3, ao1+ao3 > ao0+ao2)
				}

				// --- 2. BOTTOM FACE (-Y) ---
				if bottomAir {
					uMin, vMin, uMax, vMax := GetBlockTextureUVs(bType, FaceBottom)
					faceMult := FaceLightingMultipliers[FaceBottom]
					skyLight, torchLight := lightMap.GetLight(x, y-1, z)

					ao0 := CalculateVertexAO(w.IsSolid(x-1, y-1, z), w.IsSolid(x, y-1, z-1), w.IsSolid(x-1, y-1, z-1))
					ao1 := CalculateVertexAO(w.IsSolid(x+1, y-1, z), w.IsSolid(x, y-1, z-1), w.IsSolid(x+1, y-1, z-1))
					ao2 := CalculateVertexAO(w.IsSolid(x+1, y-1, z), w.IsSolid(x, y-1, z+1), w.IsSolid(x+1, y-1, z+1))
					ao3 := CalculateVertexAO(w.IsSolid(x-1, y-1, z), w.IsSolid(x, y-1, z+1), w.IsSolid(x-1, y-1, z+1))

					c0 := calculateMinecraftVertexColor(skyLight, torchLight, ao0, faceMult, blockAlpha)
					c1 := calculateMinecraftVertexColor(skyLight, torchLight, ao1, faceMult, blockAlpha)
					c2 := calculateMinecraftVertexColor(skyLight, torchLight, ao2, faceMult, blockAlpha)
					c3 := calculateMinecraftVertexColor(skyLight, torchLight, ao3, faceMult, blockAlpha)

					p0 := rl.Vector3{X: x0, Y: y0, Z: z0}
					p1 := rl.Vector3{X: x1, Y: y0, Z: z0}
					p2 := rl.Vector3{X: x1, Y: y0, Z: z1}
					p3 := rl.Vector3{X: x0, Y: y0, Z: z1}
					norm := rl.Vector3{X: 0, Y: -1, Z: 0}

					mb.AddQuad(p0, p1, p2, p3, norm, uMin, vMin, uMax, vMax, c0, c1, c2, c3, ao1+ao3 > ao0+ao2)
				}

				// --- 3. NORTH FACE (-Z) ---
				if northAir {
					uMin, vMin, uMax, vMax := GetBlockTextureUVs(bType, FaceNorth)
					faceMult := FaceLightingMultipliers[FaceNorth]
					skyLight, torchLight := lightMap.GetLight(x, y, z-1)

					ao0 := CalculateVertexAO(w.IsSolid(x+1, y, z-1), w.IsSolid(x, y-1, z-1), w.IsSolid(x+1, y-1, z-1))
					ao1 := CalculateVertexAO(w.IsSolid(x-1, y, z-1), w.IsSolid(x, y-1, z-1), w.IsSolid(x-1, y-1, z-1))
					ao2 := CalculateVertexAO(w.IsSolid(x-1, y, z-1), w.IsSolid(x, y+1, z-1), w.IsSolid(x-1, y+1, z-1))
					ao3 := CalculateVertexAO(w.IsSolid(x+1, y, z-1), w.IsSolid(x, y+1, z-1), w.IsSolid(x+1, y+1, z-1))

					c0 := calculateMinecraftVertexColor(skyLight, torchLight, ao0, faceMult, blockAlpha)
					c1 := calculateMinecraftVertexColor(skyLight, torchLight, ao1, faceMult, blockAlpha)
					c2 := calculateMinecraftVertexColor(skyLight, torchLight, ao2, faceMult, blockAlpha)
					c3 := calculateMinecraftVertexColor(skyLight, torchLight, ao3, faceMult, blockAlpha)

					p0 := rl.Vector3{X: x1, Y: y0, Z: z0}
					p1 := rl.Vector3{X: x0, Y: y0, Z: z0}
					p2 := rl.Vector3{X: x0, Y: yCorner3, Z: z0}
					p3 := rl.Vector3{X: x1, Y: yCorner2, Z: z0}
					norm := rl.Vector3{X: 0, Y: 0, Z: -1}

					mb.AddQuad(p0, p1, p2, p3, norm, uMin, vMin, uMax, vMax, c0, c1, c2, c3, ao1+ao3 > ao0+ao2)
				}

				// --- 4. SOUTH FACE (+Z) ---
				if southAir {
					uMin, vMin, uMax, vMax := GetBlockTextureUVs(bType, FaceSouth)
					faceMult := FaceLightingMultipliers[FaceSouth]
					skyLight, torchLight := lightMap.GetLight(x, y, z+1)

					ao0 := CalculateVertexAO(w.IsSolid(x-1, y, z+1), w.IsSolid(x, y-1, z+1), w.IsSolid(x-1, y-1, z+1))
					ao1 := CalculateVertexAO(w.IsSolid(x+1, y, z+1), w.IsSolid(x, y-1, z+1), w.IsSolid(x+1, y-1, z+1))
					ao2 := CalculateVertexAO(w.IsSolid(x+1, y, z+1), w.IsSolid(x, y+1, z+1), w.IsSolid(x+1, y+1, z+1))
					ao3 := CalculateVertexAO(w.IsSolid(x-1, y, z+1), w.IsSolid(x, y+1, z+1), w.IsSolid(x-1, y+1, z+1))

					c0 := calculateMinecraftVertexColor(skyLight, torchLight, ao0, faceMult, blockAlpha)
					c1 := calculateMinecraftVertexColor(skyLight, torchLight, ao1, faceMult, blockAlpha)
					c2 := calculateMinecraftVertexColor(skyLight, torchLight, ao2, faceMult, blockAlpha)
					c3 := calculateMinecraftVertexColor(skyLight, torchLight, ao3, faceMult, blockAlpha)

					p0 := rl.Vector3{X: x0, Y: y0, Z: z1}
					p1 := rl.Vector3{X: x1, Y: y0, Z: z1}
					p2 := rl.Vector3{X: x1, Y: yCorner1, Z: z1}
					p3 := rl.Vector3{X: x0, Y: yCorner0, Z: z1}
					norm := rl.Vector3{X: 0, Y: 0, Z: 1}

					mb.AddQuad(p0, p1, p2, p3, norm, uMin, vMin, uMax, vMax, c0, c1, c2, c3, ao1+ao3 > ao0+ao2)
				}

				// --- 5. WEST FACE (-X) ---
				if westAir {
					uMin, vMin, uMax, vMax := GetBlockTextureUVs(bType, FaceWest)
					faceMult := FaceLightingMultipliers[FaceWest]
					skyLight, torchLight := lightMap.GetLight(x-1, y, z)

					ao0 := CalculateVertexAO(w.IsSolid(x-1, y, z-1), w.IsSolid(x-1, y-1, z), w.IsSolid(x-1, y-1, z-1))
					ao1 := CalculateVertexAO(w.IsSolid(x-1, y, z+1), w.IsSolid(x-1, y-1, z), w.IsSolid(x-1, y-1, z+1))
					ao2 := CalculateVertexAO(w.IsSolid(x-1, y, z+1), w.IsSolid(x-1, y+1, z), w.IsSolid(x-1, y+1, z+1))
					ao3 := CalculateVertexAO(w.IsSolid(x-1, y, z-1), w.IsSolid(x-1, y+1, z), w.IsSolid(x-1, y+1, z-1))

					c0 := calculateMinecraftVertexColor(skyLight, torchLight, ao0, faceMult, blockAlpha)
					c1 := calculateMinecraftVertexColor(skyLight, torchLight, ao1, faceMult, blockAlpha)
					c2 := calculateMinecraftVertexColor(skyLight, torchLight, ao2, faceMult, blockAlpha)
					c3 := calculateMinecraftVertexColor(skyLight, torchLight, ao3, faceMult, blockAlpha)

					p0 := rl.Vector3{X: x0, Y: y0, Z: z0}
					p1 := rl.Vector3{X: x0, Y: y0, Z: z1}
					p2 := rl.Vector3{X: x0, Y: yCorner0, Z: z1}
					p3 := rl.Vector3{X: x0, Y: yCorner3, Z: z0}
					norm := rl.Vector3{X: -1, Y: 0, Z: 0}

					mb.AddQuad(p0, p1, p2, p3, norm, uMin, vMin, uMax, vMax, c0, c1, c2, c3, ao1+ao3 > ao0+ao2)
				}

				// --- 6. EAST FACE (+X) ---
				if eastAir {
					uMin, vMin, uMax, vMax := GetBlockTextureUVs(bType, FaceEast)
					faceMult := FaceLightingMultipliers[FaceEast]
					skyLight, torchLight := lightMap.GetLight(x+1, y, z)

					ao0 := CalculateVertexAO(w.IsSolid(x+1, y, z+1), w.IsSolid(x+1, y-1, z), w.IsSolid(x+1, y-1, z+1))
					ao1 := CalculateVertexAO(w.IsSolid(x+1, y, z-1), w.IsSolid(x+1, y-1, z), w.IsSolid(x+1, y-1, z-1))
					ao2 := CalculateVertexAO(w.IsSolid(x+1, y, z-1), w.IsSolid(x+1, y+1, z), w.IsSolid(x+1, y+1, z-1))
					ao3 := CalculateVertexAO(w.IsSolid(x+1, y, z+1), w.IsSolid(x+1, y+1, z), w.IsSolid(x+1, y+1, z+1))

					c0 := calculateMinecraftVertexColor(skyLight, torchLight, ao0, faceMult, blockAlpha)
					c1 := calculateMinecraftVertexColor(skyLight, torchLight, ao1, faceMult, blockAlpha)
					c2 := calculateMinecraftVertexColor(skyLight, torchLight, ao2, faceMult, blockAlpha)
					c3 := calculateMinecraftVertexColor(skyLight, torchLight, ao3, faceMult, blockAlpha)

					p0 := rl.Vector3{X: x1, Y: y0, Z: z1}
					p1 := rl.Vector3{X: x1, Y: y0, Z: z0}
					p2 := rl.Vector3{X: x1, Y: yCorner2, Z: z0}
					p3 := rl.Vector3{X: x1, Y: yCorner1, Z: z1}
					norm := rl.Vector3{X: 1, Y: 0, Z: 0}

					mb.AddQuad(p0, p1, p2, p3, norm, uMin, vMin, uMax, vMax, c0, c1, c2, c3, ao1+ao3 > ao0+ao2)
				}
			}
		}
	}

	// Upload built mesh data directly into GPU VRAM VBOs
	if mesh, ok := cm.OpaqueMB.BuildGPUMesh(); ok {
		c.OpaqueMesh = mesh
		c.HasOpaque = true
	}
	if mesh, ok := cm.CutoutMB.BuildGPUMesh(); ok {
		c.CutoutMesh = mesh
		c.HasCutout = true
	}
	if mesh, ok := cm.WaterMB.BuildGPUMesh(); ok {
		c.WaterMesh = mesh
		c.HasWater = true
	}
}

// Render3D draws all chunks directly on the GPU using hardware VBO DrawMesh calls with Frustum Culling
func (cm *ChunkManager) Render3D(cameraPos rl.Vector3, forwardDir rl.Vector3) {
	pcx := int(math.Floor(float64(cameraPos.X))) >> 4
	pcz := int(math.Floor(float64(cameraPos.Z))) >> 4

	radius := cm.RenderRadius
	if radius <= 0 {
		radius = DefaultChunkRenderRadius
	}

	visibleChunks := make([]*Chunk, 0, (radius*2+1)*(radius*2+1))

	// Frustum & Distance Culling
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			coord := ChunkCoord{X: pcx + dx, Z: pcz + dz}
			chunk, exists := cm.Chunks[coord]
			if !exists {
				continue
			}

			// Skip completely empty chunks
			if !chunk.HasOpaque && !chunk.HasCutout && !chunk.HasWater {
				continue
			}

			// If chunk is further than 1 chunk away, check view direction
			if math.Abs(float64(dx)) > 1 || math.Abs(float64(dz)) > 1 {
				toChunkX := chunk.CenterX - cameraPos.X
				toChunkZ := chunk.CenterZ - cameraPos.Z
				dist := float32(math.Sqrt(float64(toChunkX*toChunkX + toChunkZ*toChunkZ)))
				if dist > 0.001 {
					dirX := toChunkX / dist
					dirZ := toChunkZ / dist
					dot := dirX*forwardDir.X + dirZ*forwardDir.Z
					if dot < -0.15 { // Behind or outside player FOV cone
						continue
					}
				}
			}

			visibleChunks = append(visibleChunks, chunk)
		}
	}

	// --- PASS 1: SOLID OPAQUE GEOMETRY ON GPU (WITH HARDWARE BACKFACE OCCLUSION CULLING) ---
	rl.EnableBackfaceCulling()
	for _, chunk := range visibleChunks {
		if chunk.HasOpaque {
			rl.DrawMesh(chunk.OpaqueMesh, cm.Material, cm.IdentityMat)
		}
	}

	// --- PASS 2: ALPHA CUTOUT (LEAVES, GLASS, TORCHES) ---
	rl.DisableBackfaceCulling()
	for _, chunk := range visibleChunks {
		if chunk.HasCutout {
			rl.DrawMesh(chunk.CutoutMesh, cm.CutoutMaterial, cm.IdentityMat)
		}
	}

	// --- PASS 3: TRANSLUCENT WATER ON GPU WITH ALPHA BLENDING ---
	rl.DisableBackfaceCulling() // Double-sided so water surface is visible both from above and from underwater
	rl.BeginBlendMode(rl.BlendAlpha)
	for _, chunk := range visibleChunks {
		if chunk.HasWater {
			rl.DrawMesh(chunk.WaterMesh, cm.Material, cm.IdentityMat)
		}
	}
	rl.EndBlendMode()
}
