package voxel

// LightMap holds temporary simulated 3D light values for meshing a chunk.
// It covers the chunk itself (16x16) plus a 16-block border on all sides (total 48x48)
// to accurately simulate light bleeding in from neighboring chunks.
type LightMap struct {
	StartX, StartZ int
	SkyLight       [48][WorldHeight][48]uint8
	BlockLight     [48][WorldHeight][48]uint8
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

func getLocalBlock(chunks [3][3]*ChunkData, lx, y, lz int) BlockType {
	if y < 0 || y >= WorldHeight || lx < 0 || lx >= 48 || lz < 0 || lz >= 48 {
		return BlockAir
	}
	cIdxX, cIdxZ := lx>>4, lz>>4
	chunk := chunks[cIdxX][cIdxZ]
	if chunk == nil {
		return BlockAir
	}
	return chunk.Blocks[lx&15][y][lz&15]
}

// CalculateLocalLightMap performs a true Breadth-First-Search (BFS) Voxel Lighting Simulation.
func CalculateLocalLightMap(w *VoxelWorld, cx, cz int) *LightMap {
	lm := &LightMap{
		StartX: cx*ChunkSize - 16,
		StartZ: cz*ChunkSize - 16,
	}

	// Fast 3x3 chunk cache to avoid 589,824 map lookups per mesh build!
	var chunks [3][3]*ChunkData
	for dx := -1; dx <= 1; dx++ {
		for dz := -1; dz <= 1; dz++ {
			chunks[dx+1][dz+1] = w.GetChunk(cx+dx, cz+dz)
		}
	}

	skyQueue := make([]LightNode, 0, 48*48*WorldHeight/10)
	blockQueue := make([]LightNode, 0, 48*48)

	// 1. Seed Sunlight (SkyLight)
	for lx := 0; lx < 48; lx++ {
		for lz := 0; lz < 48; lz++ {
			// Sunlight shoots straight down at strength 15 until it hits something
			light := uint8(15)
			for y := WorldHeight - 1; y >= 0; y-- {
				b := getLocalBlock(chunks, lx, y, lz)
				opacity := GetLightOpacity(b)
				
				if opacity >= 15 {
					light = 0 // Blocked completely
				} else if opacity > 1 && light >= opacity {
					light -= opacity // e.g. Leaves take 2, Water takes 3
				} else if opacity == 1 && light > 0 && b != BlockAir {
					// Glass takes 1, but we don't subtract 1 for pure vertical air (sun shafts)
					light -= 1
				}
				
				lm.SkyLight[lx][y][lz] = light
				if light > 0 {
					skyQueue = append(skyQueue, LightNode{lx, y, lz, light})
				}
				
				if light == 0 {
					break // Stop dropping sunlight here, BFS will handle bleeding
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
			if nx >= 0 && nx < 48 && ny >= 0 && ny < WorldHeight && nz >= 0 && nz < 48 {
				b := getLocalBlock(chunks, nx, ny, nz)
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
	for lx := 0; lx < 48; lx++ {
		for lz := 0; lz < 48; lz++ {
			for y := 0; y < WorldHeight; y++ {
				b := getLocalBlock(chunks, lx, y, lz)
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
			if nx >= 0 && nx < 48 && ny >= 0 && ny < WorldHeight && nz >= 0 && nz < 48 {
				b := getLocalBlock(chunks, nx, ny, nz)
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
	if lx < 0 || lx >= 48 || y < 0 || y >= WorldHeight || lz < 0 || lz >= 48 {
		return 1.0, 0.0 // Out of bounds fallback
	}
	
	sLight := float32(lm.SkyLight[lx][y][lz]) / 15.0
	bLight := float32(lm.BlockLight[lx][y][lz]) / 15.0
	return sLight, bLight
}
