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
	Indices   []uint16
	VertCount uint16
}

func newMeshBuilder() *MeshBuilder {
	return &MeshBuilder{
		Vertices:  make([]float32, 0, 4096),
		Texcoords: make([]float32, 0, 2048),
		Colors:    make([]uint8, 0, 4096),
		Indices:   make([]uint16, 0, 6144),
	}
}

func (mb *MeshBuilder) Reset() {
	mb.Vertices = mb.Vertices[:0]
	mb.Texcoords = mb.Texcoords[:0]
	mb.Colors = mb.Colors[:0]
	mb.Indices = mb.Indices[:0]
	mb.VertCount = 0
}

func (mb *MeshBuilder) AddQuad(
	p0, p1, p2, p3 rl.Vector3,
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

// ChunkManager manages infinite dynamic chunk streaming and ultra-fast hardware GPU mesh rendering
type ChunkManager struct {
	Chunks       map[ChunkCoord]*Chunk
	World        *VoxelWorld
	Atlas        *TextureAtlas
	Material     rl.Material
	OpaqueMB     *MeshBuilder
	CutoutMB     *MeshBuilder
	WaterMB      *MeshBuilder
	IdentityMat  rl.Matrix
}

// NewChunkManager creates the infinite chunk streaming manager with GPU VBO materials
func NewChunkManager(w *VoxelWorld, atlas *TextureAtlas) *ChunkManager {
	mat := rl.LoadMaterialDefault()
	rl.SetMaterialTexture(&mat, 0, atlas.Texture) // Map diffuse texture to block texture atlas

	cm := &ChunkManager{
		Chunks:      make(map[ChunkCoord]*Chunk),
		World:       w,
		Atlas:       atlas,
		Material:    mat,
		OpaqueMB:    newMeshBuilder(),
		CutoutMB:    newMeshBuilder(),
		WaterMB:     newMeshBuilder(),
		IdentityMat: rl.MatrixIdentity(),
	}
	return cm
}

// UpdatePlayerPos streams chunks around the player as they explore the infinite world
func (cm *ChunkManager) UpdatePlayerPos(playerPos rl.Vector3) {
	pcx := int(math.Floor(float64(playerPos.X))) >> 4
	pcz := int(math.Floor(float64(playerPos.Z))) >> 4

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
			} else if chunk.IsDirty {
				cm.RebuildChunkMeshes(chunk)
				chunk.IsDirty = false
			}
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
					tCol := rl.NewColor(255, 255, 240, 255)

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
						uMin, vMin, uMax, vMax, tCol, tCol, tCol, tCol, false,
					)
					// Diagonal Quad 2
					mb.AddQuad(
						rl.Vector3{X: cx - wHalf, Y: baseY, Z: cz + wHalf},
						rl.Vector3{X: cx + wHalf, Y: baseY, Z: cz - wHalf},
						rl.Vector3{X: cx + wHalf, Y: baseY + h, Z: cz - wHalf},
						rl.Vector3{X: cx - wHalf, Y: baseY + h, Z: cz + wHalf},
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

					ao0 := CalculateVertexAO(w.IsSolid(x-1, y+1, z), w.IsSolid(x, y+1, z+1), w.IsSolid(x-1, y+1, z+1))
					ao1 := CalculateVertexAO(w.IsSolid(x+1, y+1, z), w.IsSolid(x, y+1, z+1), w.IsSolid(x+1, y+1, z+1))
					ao2 := CalculateVertexAO(w.IsSolid(x+1, y+1, z), w.IsSolid(x, y+1, z-1), w.IsSolid(x+1, y+1, z-1))
					ao3 := CalculateVertexAO(w.IsSolid(x-1, y+1, z), w.IsSolid(x, y+1, z-1), w.IsSolid(x-1, y+1, z-1))

					c0 := shadeColor(ao0 * faceMult)
					c1 := shadeColor(ao1 * faceMult)
					c2 := shadeColor(ao2 * faceMult)
					c3 := shadeColor(ao3 * faceMult)

					p0 := rl.Vector3{X: x0, Y: y1, Z: z1}
					p1 := rl.Vector3{X: x1, Y: y1, Z: z1}
					p2 := rl.Vector3{X: x1, Y: y1, Z: z0}
					p3 := rl.Vector3{X: x0, Y: y1, Z: z0}

					mb.AddQuad(p0, p1, p2, p3, uMin, vMin, uMax, vMax, c0, c1, c2, c3, ao1+ao3 > ao0+ao2)
				}

				// --- 2. BOTTOM FACE (-Y) ---
				if bottomAir {
					uMin, vMin, uMax, vMax := GetBlockTextureUVs(bType, FaceBottom)
					faceMult := FaceLightingMultipliers[FaceBottom]

					ao0 := CalculateVertexAO(w.IsSolid(x-1, y-1, z), w.IsSolid(x, y-1, z-1), w.IsSolid(x-1, y-1, z-1))
					ao1 := CalculateVertexAO(w.IsSolid(x+1, y-1, z), w.IsSolid(x, y-1, z-1), w.IsSolid(x+1, y-1, z-1))
					ao2 := CalculateVertexAO(w.IsSolid(x+1, y-1, z), w.IsSolid(x, y-1, z+1), w.IsSolid(x+1, y-1, z+1))
					ao3 := CalculateVertexAO(w.IsSolid(x-1, y-1, z), w.IsSolid(x, y-1, z+1), w.IsSolid(x-1, y-1, z+1))

					c0 := shadeColor(ao0 * faceMult)
					c1 := shadeColor(ao1 * faceMult)
					c2 := shadeColor(ao2 * faceMult)
					c3 := shadeColor(ao3 * faceMult)

					p0 := rl.Vector3{X: x0, Y: y0, Z: z0}
					p1 := rl.Vector3{X: x1, Y: y0, Z: z0}
					p2 := rl.Vector3{X: x1, Y: y0, Z: z1}
					p3 := rl.Vector3{X: x0, Y: y0, Z: z1}

					mb.AddQuad(p0, p1, p2, p3, uMin, vMin, uMax, vMax, c0, c1, c2, c3, ao1+ao3 > ao0+ao2)
				}

				// --- 3. NORTH FACE (-Z) ---
				if northAir {
					uMin, vMin, uMax, vMax := GetBlockTextureUVs(bType, FaceNorth)
					faceMult := FaceLightingMultipliers[FaceNorth]

					ao0 := CalculateVertexAO(w.IsSolid(x+1, y, z-1), w.IsSolid(x, y-1, z-1), w.IsSolid(x+1, y-1, z-1))
					ao1 := CalculateVertexAO(w.IsSolid(x-1, y, z-1), w.IsSolid(x, y-1, z-1), w.IsSolid(x-1, y-1, z-1))
					ao2 := CalculateVertexAO(w.IsSolid(x-1, y, z-1), w.IsSolid(x, y+1, z-1), w.IsSolid(x-1, y+1, z-1))
					ao3 := CalculateVertexAO(w.IsSolid(x+1, y, z-1), w.IsSolid(x, y+1, z-1), w.IsSolid(x+1, y+1, z-1))

					c0 := shadeColor(ao0 * faceMult)
					c1 := shadeColor(ao1 * faceMult)
					c2 := shadeColor(ao2 * faceMult)
					c3 := shadeColor(ao3 * faceMult)

					p0 := rl.Vector3{X: x1, Y: y0, Z: z0}
					p1 := rl.Vector3{X: x0, Y: y0, Z: z0}
					p2 := rl.Vector3{X: x0, Y: y1, Z: z0}
					p3 := rl.Vector3{X: x1, Y: y1, Z: z0}

					mb.AddQuad(p0, p1, p2, p3, uMin, vMin, uMax, vMax, c0, c1, c2, c3, ao1+ao3 > ao0+ao2)
				}

				// --- 4. SOUTH FACE (+Z) ---
				if southAir {
					uMin, vMin, uMax, vMax := GetBlockTextureUVs(bType, FaceSouth)
					faceMult := FaceLightingMultipliers[FaceSouth]

					ao0 := CalculateVertexAO(w.IsSolid(x-1, y, z+1), w.IsSolid(x, y-1, z+1), w.IsSolid(x-1, y-1, z+1))
					ao1 := CalculateVertexAO(w.IsSolid(x+1, y, z+1), w.IsSolid(x, y-1, z+1), w.IsSolid(x+1, y-1, z+1))
					ao2 := CalculateVertexAO(w.IsSolid(x+1, y, z+1), w.IsSolid(x, y+1, z+1), w.IsSolid(x+1, y+1, z+1))
					ao3 := CalculateVertexAO(w.IsSolid(x-1, y, z+1), w.IsSolid(x, y+1, z+1), w.IsSolid(x-1, y+1, z+1))

					c0 := shadeColor(ao0 * faceMult)
					c1 := shadeColor(ao1 * faceMult)
					c2 := shadeColor(ao2 * faceMult)
					c3 := shadeColor(ao3 * faceMult)

					p0 := rl.Vector3{X: x0, Y: y0, Z: z1}
					p1 := rl.Vector3{X: x1, Y: y0, Z: z1}
					p2 := rl.Vector3{X: x1, Y: y1, Z: z1}
					p3 := rl.Vector3{X: x0, Y: y1, Z: z1}

					mb.AddQuad(p0, p1, p2, p3, uMin, vMin, uMax, vMax, c0, c1, c2, c3, ao1+ao3 > ao0+ao2)
				}

				// --- 5. WEST FACE (-X) ---
				if westAir {
					uMin, vMin, uMax, vMax := GetBlockTextureUVs(bType, FaceWest)
					faceMult := FaceLightingMultipliers[FaceWest]

					ao0 := CalculateVertexAO(w.IsSolid(x-1, y, z-1), w.IsSolid(x-1, y-1, z), w.IsSolid(x-1, y-1, z-1))
					ao1 := CalculateVertexAO(w.IsSolid(x-1, y, z+1), w.IsSolid(x-1, y-1, z), w.IsSolid(x-1, y-1, z+1))
					ao2 := CalculateVertexAO(w.IsSolid(x-1, y, z+1), w.IsSolid(x-1, y+1, z), w.IsSolid(x-1, y+1, z+1))
					ao3 := CalculateVertexAO(w.IsSolid(x-1, y, z-1), w.IsSolid(x-1, y+1, z), w.IsSolid(x-1, y+1, z-1))

					c0 := shadeColor(ao0 * faceMult)
					c1 := shadeColor(ao1 * faceMult)
					c2 := shadeColor(ao2 * faceMult)
					c3 := shadeColor(ao3 * faceMult)

					p0 := rl.Vector3{X: x0, Y: y0, Z: z0}
					p1 := rl.Vector3{X: x0, Y: y0, Z: z1}
					p2 := rl.Vector3{X: x0, Y: y1, Z: z1}
					p3 := rl.Vector3{X: x0, Y: y1, Z: z0}

					mb.AddQuad(p0, p1, p2, p3, uMin, vMin, uMax, vMax, c0, c1, c2, c3, ao1+ao3 > ao0+ao2)
				}

				// --- 6. EAST FACE (+X) ---
				if eastAir {
					uMin, vMin, uMax, vMax := GetBlockTextureUVs(bType, FaceEast)
					faceMult := FaceLightingMultipliers[FaceEast]

					ao0 := CalculateVertexAO(w.IsSolid(x+1, y, z+1), w.IsSolid(x+1, y-1, z), w.IsSolid(x+1, y-1, z+1))
					ao1 := CalculateVertexAO(w.IsSolid(x+1, y, z-1), w.IsSolid(x+1, y-1, z), w.IsSolid(x+1, y-1, z-1))
					ao2 := CalculateVertexAO(w.IsSolid(x+1, y, z-1), w.IsSolid(x+1, y+1, z), w.IsSolid(x+1, y+1, z-1))
					ao3 := CalculateVertexAO(w.IsSolid(x+1, y, z+1), w.IsSolid(x+1, y+1, z), w.IsSolid(x+1, y+1, z+1))

					c0 := shadeColor(ao0 * faceMult)
					c1 := shadeColor(ao1 * faceMult)
					c2 := shadeColor(ao2 * faceMult)
					c3 := shadeColor(ao3 * faceMult)

					p0 := rl.Vector3{X: x1, Y: y0, Z: z1}
					p1 := rl.Vector3{X: x1, Y: y0, Z: z0}
					p2 := rl.Vector3{X: x1, Y: y1, Z: z0}
					p3 := rl.Vector3{X: x1, Y: y1, Z: z1}

					mb.AddQuad(p0, p1, p2, p3, uMin, vMin, uMax, vMax, c0, c1, c2, c3, ao1+ao3 > ao0+ao2)
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
			rl.DrawMesh(chunk.CutoutMesh, cm.Material, cm.IdentityMat)
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
