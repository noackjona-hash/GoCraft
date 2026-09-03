package voxel

const (
	LightMapSize   = 24 // 16x16 chunk + 4-block margin on each side (24x24 total)
	LightMapMargin = 4
)

// LightMap holds temporary simulated 3D light values for meshing a chunk.
// It covers the chunk itself (16x16) plus a 4-block border on all sides (total 24x24)
// to accurately and efficiently simulate light bleeding from neighboring chunks.
type LightMap struct {
	StartX, StartZ int
	SkyLight       [LightMapSize][WorldHeight][LightMapSize]uint8
	BlockLight     [LightMapSize][WorldHeight][LightMapSize]uint8
}

type LightNode struct {
	X, Y, Z int
	L       uint8
}

var dirs = [6][3]int{
	{1, 0, 0}, {-1, 0, 0},
	{0, 1, 0}, {0, -1, 0},
	{0, 0, 1}, {0, 0, -1},
}

func getLocalBlock24(chunks [3][3]*ChunkData, startX, startZ, lx, y, lz int) BlockType {
	if y < 0 || y >= WorldHeight || lx < 0 || lx >= LightMapSize || lz < 0 || lz >= LightMapSize {
		return BlockAir
	}
	wx := startX + lx
	wz := startZ + lz
	cIdxX := (wx >> 4) - (startX >> 4)
	cIdxZ := (wz >> 4) - (startZ >> 4)
	if cIdxX < 0 || cIdxX > 2 || cIdxZ < 0 || cIdxZ > 2 {
		return BlockAir
	}
	chunk := chunks[cIdxX][cIdxZ]
	if chunk == nil {
		return BlockAir
	}
	return chunk.Blocks[wx&15][y][wz&15]
}

// CalculateLocalLightMap performs a high-performance Breadth-First-Search (BFS) Voxel Lighting Simulation.
func CalculateLocalLightMap(w *VoxelWorld, cx, cz int) *LightMap {
	startX := cx*ChunkSize - LightMapMargin
	startZ := cz*ChunkSize - LightMapMargin

	lm := &LightMap{
		StartX: startX,
		StartZ: startZ,
	}

	// Fast 3x3 chunk cache to avoid thousands of map lookups
	var chunks [3][3]*ChunkData
	baseCX := startX >> 4
	baseCZ := startZ >> 4
	for dx := 0; dx < 3; dx++ {
		for dz := 0; dz < 3; dz++ {
			chunks[dx][dz] = w.GetChunk(baseCX+dx, baseCZ+dz)
		}
	}

	skyQueue := make([]LightNode, 0, 2048)
	blockQueue := make([]LightNode, 0, 256)

	// 1. Seed Sunlight (SkyLight)
	for lx := 0; lx < LightMapSize; lx++ {
		for lz := 0; lz < LightMapSize; lz++ {
			light := uint8(15)
			hitObstacle := false

			for y := WorldHeight - 1; y >= 0; y-- {
				b := getLocalBlock24(chunks, startX, startZ, lx, y, lz)
				opacity := GetLightOpacity(b)

				if opacity >= 15 {
					light = 0
					hitObstacle = true
				} else if opacity > 1 && light >= opacity {
					light -= opacity
					hitObstacle = true
				} else if opacity == 1 && light > 0 && b != BlockAir {
					light -= 1
					hitObstacle = true
				}

				lm.SkyLight[lx][y][lz] = light

				// Only queue nodes that actually need horizontal BFS propagation (near/under obstacles)
				if light > 0 && (hitObstacle || y == 0) {
					skyQueue = append(skyQueue, LightNode{lx, y, lz, light})
				}

				if light == 0 {
					break
				}
			}
		}
	}

	// 2. Propagate Sunlight BFS
	for i := 0; i < len(skyQueue); i++ {
		node := skyQueue[i]
		if node.L <= 1 {
			continue
		}

		for _, d := range dirs {
			nx, ny, nz := node.X+d[0], node.Y+d[1], node.Z+d[2]
			if nx >= 0 && nx < LightMapSize && ny >= 0 && ny < WorldHeight && nz >= 0 && nz < LightMapSize {
				b := getLocalBlock24(chunks, startX, startZ, nx, ny, nz)
				opacity := GetLightOpacity(b)

				if opacity < 15 {
					newL := node.L - opacity
					if newL > lm.SkyLight[nx][ny][nz] {
						lm.SkyLight[nx][ny][nz] = newL
						skyQueue = append(skyQueue, LightNode{nx, ny, nz, newL})
					}
				}
			}
		}
	}

	// 3. Seed Torchlight (BlockLight)
	for lx := 0; lx < LightMapSize; lx++ {
		for lz := 0; lz < LightMapSize; lz++ {
			for y := 0; y < WorldHeight; y++ {
				b := getLocalBlock24(chunks, startX, startZ, lx, y, lz)
				def, exists := BlockRegistry[b]
				if exists && def.IsLightSource && def.LightLevel > 0 {
					lm.BlockLight[lx][y][lz] = def.LightLevel
					blockQueue = append(blockQueue, LightNode{lx, y, lz, def.LightLevel})
				}
			}
		}
	}

	// 4. Propagate Torchlight BFS
	for i := 0; i < len(blockQueue); i++ {
		node := blockQueue[i]
		if node.L <= 1 {
			continue
		}

		for _, d := range dirs {
			nx, ny, nz := node.X+d[0], node.Y+d[1], node.Z+d[2]
			if nx >= 0 && nx < LightMapSize && ny >= 0 && ny < WorldHeight && nz >= 0 && nz < LightMapSize {
				b := getLocalBlock24(chunks, startX, startZ, nx, ny, nz)
				opacity := GetLightOpacity(b)

				if opacity < 15 {
					newL := node.L - opacity
					if newL > lm.BlockLight[nx][ny][nz] {
						lm.BlockLight[nx][ny][nz] = newL
						blockQueue = append(blockQueue, LightNode{nx, ny, nz, newL})
					}
				}
			}
		}
	}

	return lm
}

// GetLight returns the normalized (0.0 to 1.0) sky and block light from the lightmap.
func (lm *LightMap) GetLight(x, y, z int) (float32, float32) {
	lx := x - lm.StartX
	lz := z - lm.StartZ
	if lx < 0 || lx >= LightMapSize || y < 0 || y >= WorldHeight || lz < 0 || lz >= LightMapSize {
		return 1.0, 0.0 // Out of bounds fallback
	}

	sLight := float32(lm.SkyLight[lx][y][lz]) / 15.0
	bLight := float32(lm.BlockLight[lx][y][lz]) / 15.0
	return sLight, bLight
}
