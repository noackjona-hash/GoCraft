package voxel

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	ChunkRenderRadius = 6 // 13x13 chunk grid (208x208 blocks) rendered entirely on GPU
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
		Vertices:  make([]float32, 0, 4096),
		Texcoords: make([]float32, 0, 2048),
		Colors:    make([]uint8, 0, 4096),
		Normals:   make([]float32, 0, 4096),
		Indices:   make([]uint16, 0, 6144),
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
out vec4 fragColor;
out float vertexDistance;

uniform mat4 mvp;
uniform mat4 matModel;
uniform vec3 viewPos;
uniform vec3 lightDir0;
uniform float sunIntensity;

// Minecraft light mixing from assets/shaders/include/light.glsl
vec4 minecraft_mix_light(vec3 lightDir, vec3 normal, vec4 color) {
    vec3 l0 = normalize(lightDir);
    vec3 n = normalize(normal);
    float light0 = max(0.0, dot(l0, n));
    float sunFacet = min(1.0, light0 * sunIntensity * 0.15 + 0.85);
    return vec4(color.rgb * sunFacet, color.a);
}

// ACES Filmic Tone Mapping for "Raytracing" aesthetics
vec3 ACESFilm(vec3 x) {
    float a = 2.51;
    float b = 0.03;
    float c = 2.43;
    float d = 0.59;
    float e = 0.14;
    return clamp((x*(a*x+b))/(x*(c*x+d)+e), 0.0, 1.0);
}

void main() {
    fragTexCoord = vertexTexCoord;
    vec4 worldPos = matModel * vec4(vertexPosition, 1.0);
    vertexDistance = length(worldPos.xyz - viewPos);

    vec4 mixedColor = minecraft_mix_light(lightDir0, vertexNormal, vertexColor);
    mixedColor.rgb = ACESFilm(mixedColor.rgb * 1.2);
    fragColor = mixedColor;
    
    gl_Position = mvp * vec4(vertexPosition, 1.0);
}
`

const minecraftFragShader = `#version 330
in vec2 fragTexCoord;
in vec4 fragColor;
in float vertexDistance;

out vec4 finalColor;

uniform sampler2D texture0;
uniform vec4 colDiffuse;
uniform float FogStart;
uniform float FogEnd;
uniform vec4 FogColor;
uniform float isCutout;

// Direct implementation of assets/shaders/include/fog.glsl
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

    vec4 color = texelColor * fragColor * colDiffuse;
    finalColor = linear_fog(color, vertexDistance, FogStart, FogEnd, FogColor);
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

	// Shader Uniform Locations
	Shader             rl.Shader
	CutoutShader       rl.Shader
	ViewPosLoc         int32
	LightDirLoc        int32
	SunIntensityLoc    int32
	FogStartLoc        int32
	FogEndLoc          int32
	FogColorLoc        int32
	CutoutViewPosLoc   int32
	CutoutLightDirLoc  int32
	CutoutSunIntLoc    int32
	CutoutFogStartLoc  int32
	CutoutFogEndLoc    int32
	CutoutFogColorLoc  int32
	CutoutFlagLoc      int32
}

// NewChunkManager creates the infinite chunk streaming manager with authentic Minecraft GPU shaders
func NewChunkManager(w *VoxelWorld, atlas *TextureAtlas) *ChunkManager {
	// 1. Opaque & Water Material with Minecraft Shader
	opaqueShader := rl.LoadShaderFromMemory(minecraftVertShader, minecraftFragShader)
	viewPosLoc := rl.GetShaderLocation(opaqueShader, "viewPos")
	lightDirLoc := rl.GetShaderLocation(opaqueShader, "lightDir0")
	sunIntLoc := rl.GetShaderLocation(opaqueShader, "sunIntensity")
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

		Shader:            opaqueShader,
		CutoutShader:      cutoutShader,
		ViewPosLoc:        viewPosLoc,
		LightDirLoc:       lightDirLoc,
		SunIntensityLoc:   sunIntLoc,
		FogStartLoc:       fogStartLoc,
		FogEndLoc:         fogEndLoc,
		FogColorLoc:       fogColorLoc,
		CutoutViewPosLoc:  cViewPosLoc,
		CutoutLightDirLoc: cLightDirLoc,
		CutoutSunIntLoc:   cSunIntLoc,
		CutoutFogStartLoc: cFogStartLoc,
		CutoutFogEndLoc:   cFogEndLoc,
		CutoutFogColorLoc: cFogColorLoc,
		CutoutFlagLoc:     cCutoutFlagLoc,
	}

	// Set initial fog parameters
	cm.UpdateFogAndSky(rl.NewColor(135, 206, 235, 255), false, rl.Vector3{}, 0.2, 0.8)

	return cm
}

// UpdateFogAndSky passes dynamic sky color, camera viewPos, sunlight direction and fog distances into the GPU chunk shaders
func (cm *ChunkManager) UpdateFogAndSky(skyCol rl.Color, isUnderwater bool, cameraPos rl.Vector3, sunAngle, sunHeight float32) {
	fogR := float32(skyCol.R) / 255.0
	fogG := float32(skyCol.G) / 255.0
	fogB := float32(skyCol.B) / 255.0

	near := float32(ChunkRenderRadius*ChunkSize) * 0.55
	far := float32(ChunkRenderRadius*ChunkSize) * 0.95
	if isUnderwater {
		near = 2.0
		far = 28.0
	}

	sunIntensity := float32(math.Max(0.08, float64((sunHeight+0.3)/1.3)))
	sunDirX := float32(math.Cos(float64(sunAngle)))
	sunDirY := sunHeight
	sunDirZ := float32(0.0)

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
	rl.SetShaderValue(cm.Shader, cm.FogStartLoc, nearVal, rl.ShaderUniformFloat)
	rl.SetShaderValue(cm.Shader, cm.FogEndLoc, farVal, rl.ShaderUniformFloat)
	rl.SetShaderValue(cm.Shader, cm.FogColorLoc, fogCol, rl.ShaderUniformVec4)

	// Update Cutout Shader
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutViewPosLoc, viewPos, rl.ShaderUniformVec3)
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutLightDirLoc, lightDir, rl.ShaderUniformVec3)
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutSunIntLoc, sunInt, rl.ShaderUniformFloat)
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutFogStartLoc, nearVal, rl.ShaderUniformFloat)
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutFogEndLoc, farVal, rl.ShaderUniformFloat)
	rl.SetShaderValue(cm.CutoutShader, cm.CutoutFogColorLoc, fogCol, rl.ShaderUniformVec4)
}

// UpdatePlayerPos streams chunks around the player as they explore the infinite world
func (cm *ChunkManager) UpdatePlayerPos(playerPos rl.Vector3) {
	pcx := int(math.Floor(float64(playerPos.X))) >> 4
	pcz := int(math.Floor(float64(playerPos.Z))) >> 4

	// Rate-limit dirty chunk rebuilds to avoid lag spikes when moving
	rebuildsThisFrame := 0
	const maxRebuildsPerFrame = 2

	for dx := -ChunkRenderRadius; dx <= ChunkRenderRadius; dx++ {
		for dz := -ChunkRenderRadius; dz <= ChunkRenderRadius; dz++ {
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
				cm.RebuildChunkMeshes(chunk)
				rebuildsThisFrame++
			} else if chunk.IsDirty {
				if rebuildsThisFrame < maxRebuildsPerFrame {
					cm.RebuildChunkMeshes(chunk)
					chunk.IsDirty = false
					rebuildsThisFrame++
				}
			}
		}
	}

	// Unload chunks far outside render radius to save memory
	unloadRadius := ChunkRenderRadius + 3
	for coord, chunk := range cm.Chunks {
		dx := coord.X - pcx
		dz := coord.Z - pcz
		if dx < -unloadRadius || dx > unloadRadius || dz < -unloadRadius || dz > unloadRadius {
			unloadChunkGPUMeshes(chunk)
			delete(cm.Chunks, coord)
		}
	}
}

// MarkBlockDirty flags the chunk at (x, z) for rebuild
func (cm *ChunkManager) MarkBlockDirty(x, z int) {
	cx := x >> 4
	cz := z >> 4

	for dx := -1; dx <= 1; dx++ {
		for dz := -1; dz <= 1; dz++ {
			coord := ChunkCoord{X: cx + dx, Z: cz + dz}
			if chunk, exists := cm.Chunks[coord]; exists {
				chunk.IsDirty = true
			}
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

// calculateMinecraftVertexColor computes authentic Minecraft vertex colors combining SkyLight, TorchLight, AO, and face shading
func calculateMinecraftVertexColor(skyLight, torchLight, ao, faceMult float32) rl.Color {
	// Base ambient in deep caves
	ambient := float32(0.12)

	// Sky light color: Neutral cool daylight
	skyR := skyLight * 0.96
	skyG := skyLight * 0.98
	skyB := skyLight * 1.00

	// Torch light color: Warm golden orange
	torchR := torchLight * 1.25
	torchG := torchLight * 0.95
	torchB := torchLight * 0.55

	// Smooth gentle ambient occlusion curve
	softAO := 0.75 + ao*0.25

	// Soft face shading curve
	softFace := 0.70 + faceMult*0.30

	r := (skyR + torchR + ambient) * softFace * softAO
	g := (skyG + torchG + ambient) * softFace * softAO
	b := (skyB + torchB + ambient) * softFace * softAO

	if r > 1.0 {
		r = 1.0
	}
	if g > 1.0 {
		g = 1.0
	}
	if b > 1.0 {
		b = 1.0
	}

	return rl.NewColor(uint8(r*255.0), uint8(g*255.0), uint8(b*255.0), 255)
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

				// Determine destination MeshBuilder
				var mb *MeshBuilder
				if bType == BlockWater {
					mb = cm.WaterMB
				} else if bType == BlockOakLeaves || bType == BlockGlass || bType == BlockTorch {
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

				var topAir, bottomAir, northAir, southAir, westAir, eastAir bool

				if bType == BlockWater {
					// Water faces only rendered if neighbor is not water
					topAir = topBlock != BlockWater
					bottomAir = y > 0 && bottomBlock != BlockWater
					northAir = northBlock != BlockWater
					southAir = southBlock != BlockWater
					westAir = westBlock != BlockWater
					eastAir = eastBlock != BlockWater
				} else if bType == BlockGlass {
					topAir = topBlock != BlockGlass && (y == WorldHeight-1 || BlockRegistry[topBlock].IsTransparent)
					bottomAir = bottomBlock != BlockGlass && (y > 0 && BlockRegistry[bottomBlock].IsTransparent)
					northAir = northBlock != BlockGlass && BlockRegistry[northBlock].IsTransparent
					southAir = southBlock != BlockGlass && BlockRegistry[southBlock].IsTransparent
					westAir = westBlock != BlockGlass && BlockRegistry[westBlock].IsTransparent
					eastAir = eastBlock != BlockGlass && BlockRegistry[eastBlock].IsTransparent
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

					c0 := calculateMinecraftVertexColor(skyLight, torchLight, ao0, faceMult)
					c1 := calculateMinecraftVertexColor(skyLight, torchLight, ao1, faceMult)
					c2 := calculateMinecraftVertexColor(skyLight, torchLight, ao2, faceMult)
					c3 := calculateMinecraftVertexColor(skyLight, torchLight, ao3, faceMult)

					p0 := rl.Vector3{X: x0, Y: y1, Z: z1}
					p1 := rl.Vector3{X: x1, Y: y1, Z: z1}
					p2 := rl.Vector3{X: x1, Y: y1, Z: z0}
					p3 := rl.Vector3{X: x0, Y: y1, Z: z0}
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

					c0 := calculateMinecraftVertexColor(skyLight, torchLight, ao0, faceMult)
					c1 := calculateMinecraftVertexColor(skyLight, torchLight, ao1, faceMult)
					c2 := calculateMinecraftVertexColor(skyLight, torchLight, ao2, faceMult)
					c3 := calculateMinecraftVertexColor(skyLight, torchLight, ao3, faceMult)

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

					c0 := calculateMinecraftVertexColor(skyLight, torchLight, ao0, faceMult)
					c1 := calculateMinecraftVertexColor(skyLight, torchLight, ao1, faceMult)
					c2 := calculateMinecraftVertexColor(skyLight, torchLight, ao2, faceMult)
					c3 := calculateMinecraftVertexColor(skyLight, torchLight, ao3, faceMult)

					p0 := rl.Vector3{X: x1, Y: y0, Z: z0}
					p1 := rl.Vector3{X: x0, Y: y0, Z: z0}
					p2 := rl.Vector3{X: x0, Y: y1, Z: z0}
					p3 := rl.Vector3{X: x1, Y: y1, Z: z0}
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

					c0 := calculateMinecraftVertexColor(skyLight, torchLight, ao0, faceMult)
					c1 := calculateMinecraftVertexColor(skyLight, torchLight, ao1, faceMult)
					c2 := calculateMinecraftVertexColor(skyLight, torchLight, ao2, faceMult)
					c3 := calculateMinecraftVertexColor(skyLight, torchLight, ao3, faceMult)

					p0 := rl.Vector3{X: x0, Y: y0, Z: z1}
					p1 := rl.Vector3{X: x1, Y: y0, Z: z1}
					p2 := rl.Vector3{X: x1, Y: y1, Z: z1}
					p3 := rl.Vector3{X: x0, Y: y1, Z: z1}
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

					c0 := calculateMinecraftVertexColor(skyLight, torchLight, ao0, faceMult)
					c1 := calculateMinecraftVertexColor(skyLight, torchLight, ao1, faceMult)
					c2 := calculateMinecraftVertexColor(skyLight, torchLight, ao2, faceMult)
					c3 := calculateMinecraftVertexColor(skyLight, torchLight, ao3, faceMult)

					p0 := rl.Vector3{X: x0, Y: y0, Z: z0}
					p1 := rl.Vector3{X: x0, Y: y0, Z: z1}
					p2 := rl.Vector3{X: x0, Y: y1, Z: z1}
					p3 := rl.Vector3{X: x0, Y: y1, Z: z0}
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

					c0 := calculateMinecraftVertexColor(skyLight, torchLight, ao0, faceMult)
					c1 := calculateMinecraftVertexColor(skyLight, torchLight, ao1, faceMult)
					c2 := calculateMinecraftVertexColor(skyLight, torchLight, ao2, faceMult)
					c3 := calculateMinecraftVertexColor(skyLight, torchLight, ao3, faceMult)

					p0 := rl.Vector3{X: x1, Y: y0, Z: z1}
					p1 := rl.Vector3{X: x1, Y: y0, Z: z0}
					p2 := rl.Vector3{X: x1, Y: y1, Z: z0}
					p3 := rl.Vector3{X: x1, Y: y1, Z: z1}
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

	visibleChunks := make([]*Chunk, 0, (ChunkRenderRadius*2+1)*(ChunkRenderRadius*2+1))

	// Frustum & Distance Culling
	for dx := -ChunkRenderRadius; dx <= ChunkRenderRadius; dx++ {
		for dz := -ChunkRenderRadius; dz <= ChunkRenderRadius; dz++ {
			coord := ChunkCoord{X: pcx + dx, Z: pcz + dz}
			chunk, exists := cm.Chunks[coord]
			if !exists {
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
					if dot < -0.45 { // Behind player camera
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
	rl.EnableBackfaceCulling()
	rl.BeginBlendMode(rl.BlendAlpha)
	for _, chunk := range visibleChunks {
		if chunk.HasWater {
			rl.DrawMesh(chunk.WaterMesh, cm.Material, cm.IdentityMat)
		}
	}
	rl.EndBlendMode()
	rl.DisableBackfaceCulling()
}
