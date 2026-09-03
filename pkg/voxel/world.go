package voxel

import (
	"fmt"
	"math"
	"math/rand"
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	ChunkSize   = 16
	WorldHeight = 48
	WaterLevel  = 14
)

// BlockPos represents an integer 3D block coordinate
type BlockPos struct {
	X int
	Y int
	Z int
}

// ChunkCoord represents a 2D chunk grid position (can be negative or positive infinite)
type ChunkCoord struct {
	X int
	Z int
}

// ChunkData holds the 16x48x16 voxels for an infinite world chunk
type ChunkData struct {
	Coord      ChunkCoord
	Blocks     [ChunkSize][WorldHeight][ChunkSize]BlockType
	SkyLight   [ChunkSize][WorldHeight][ChunkSize]uint8
	BlockLight [ChunkSize][WorldHeight][ChunkSize]uint8
	Modified   bool
}

// VoxelWorld manages infinite chunk generation, block queries, and chunk persistence
type VoxelWorld struct {
	Chunks           map[ChunkCoord]*ChunkData
	ScheduledUpdates []ScheduledUpdate
	Torches          map[BlockPos]uint8
	SaveDir          string
}

// NewVoxelWorld initializes the infinite procedural voxel world
func NewVoxelWorld() *VoxelWorld {
	return &VoxelWorld{
		Chunks:           make(map[ChunkCoord]*ChunkData),
		ScheduledUpdates: make([]ScheduledUpdate, 0),
		Torches:          make(map[BlockPos]uint8),
	}
}

// GetChunk returns chunk at (cx, cz), loading it from disk or generating it if needed
func (w *VoxelWorld) GetChunk(cx, cz int) *ChunkData {
	coord := ChunkCoord{X: cx, Z: cz}
	chunk, exists := w.Chunks[coord]
	if !exists {
		if w.SaveDir != "" {
			chunkPath := filepath.Join(w.SaveDir, "chunks", fmt.Sprintf("c.%d.%d.bin", cx, cz))
			if loaded, err := LoadChunkGzip(chunkPath, cx, cz); err == nil && loaded != nil {
				chunk = loaded
			}
		}
		if chunk == nil {
			chunk = w.generateChunk(cx, cz)
		}
		w.Chunks[coord] = chunk
	}
	return chunk
}

// SaveAllModifiedChunks writes all dirty chunks to saves/<world>/chunks/
func (w *VoxelWorld) SaveAllModifiedChunks() int {
	if w.SaveDir == "" {
		return 0
	}
	count := 0
	for coord, chunk := range w.Chunks {
		if chunk.Modified {
			filePath := filepath.Join(w.SaveDir, "chunks", fmt.Sprintf("c.%d.%d.bin", coord.X, coord.Z))
			if err := SaveChunkGzip(filePath, chunk); err == nil {
				chunk.Modified = false
				count++
			}
		}
	}
	return count
}

type BiomeType int

const (
	BiomeOcean BiomeType = iota
	BiomePlains
	BiomeForest
	BiomeBirchForest
	BiomeTaiga
	BiomeMountains
	BiomeDesert
)

type TreeType int

const (
	TreeSmallOak TreeType = iota
	TreeLargeOak
	TreeBirch
	TreeSpruce
)

type TreeVoxel struct {
	X, Y, Z int
	Block   BlockType
}

// GetBiome returns the deterministic biome at world coordinate (wx, wz)
func GetBiome(wx, wz float64) BiomeType {
	continental := FractalNoise2D(wx*0.0035, wz*0.0035, 3, 0.5, 2.0)
	if continental < -0.30 {
		return BiomeOcean
	}
	moisture := FractalNoise2D((wx+600.0)*0.005, (wz+600.0)*0.005, 3, 0.5, 2.0)
	temperature := FractalNoise2D((wx+1200.0)*0.004, (wz+1200.0)*0.004, 3, 0.5, 2.0)
	erosion := FractalNoise2D((wx-400.0)*0.008, (wz-400.0)*0.008, 3, 0.5, 2.0)

	if temperature > 0.28 && moisture < -0.15 {
		return BiomeDesert
	}
	if continental > 0.18 && erosion > 0.20 {
		return BiomeMountains
	}
	if temperature < -0.15 {
		return BiomeTaiga
	}
	if moisture > 0.22 {
		return BiomeBirchForest
	}
	if moisture > -0.05 {
		return BiomeForest
	}
	return BiomePlains
}

// GetTerrainHeight computes the elevation and biome at world coordinate (wx, wz) deterministically
func GetTerrainHeight(wx, wz float64) (int, BiomeType) {
	biome := GetBiome(wx, wz)
	continental := FractalNoise2D(wx*0.0035, wz*0.0035, 3, 0.5, 2.0)
	erosion := FractalNoise2D((wx-400.0)*0.008, (wz-400.0)*0.008, 3, 0.5, 2.0)
	detail := Perlin2D(wx*0.035, wz*0.035)*2.2 + Perlin2D(wx*0.08, wz*0.08)*0.8

	var baseH float64
	switch biome {
	case BiomeOcean:
		baseH = 8.0 + (continental+0.5)*12.0 + detail*0.5
	case BiomeMountains:
		peakBoost := (erosion - 0.15)*24.0 + (continental - 0.15)*16.0
		if peakBoost < 0 {
			peakBoost = 0
		}
		baseH = 24.0 + peakBoost + detail*3.0
	case BiomeDesert:
		baseH = 16.0 + continental*3.5 + erosion*3.0 + detail*1.2
	case BiomeTaiga:
		baseH = 20.0 + continental*5.5 + erosion*4.5 + detail*1.8
	case BiomeBirchForest:
		baseH = 18.0 + continental*5.0 + erosion*3.8 + detail*1.6
	case BiomeForest:
		baseH = 18.0 + continental*5.0 + erosion*4.0 + detail*1.6
	default: // BiomePlains
		baseH = 17.0 + continental*4.0 + erosion*3.0 + detail*1.5
	}

	h := int(math.Round(baseH))
	if h < 2 {
		h = 2
	}
	if h >= WorldHeight-3 {
		h = WorldHeight - 3
	}
	return h, biome
}

// generateTreeVoxels creates authentic Minecraft tree logs and leaf canopies centered at root (gx, gy, gz)
func generateTreeVoxels(gx, gy, gz int, treeType TreeType, seed int) []TreeVoxel {
	var voxels []TreeVoxel

	switch treeType {
	case TreeBirch:
		height := 6 + (seed % 3) // 6 to 8 blocks tall
		logBlock := BlockBirchLog
		leafBlock := BlockBirchLeaves

		// Trunk
		for h := 1; h < height; h++ {
			voxels = append(voxels, TreeVoxel{gx, gy + h, gz, logBlock})
		}

		topY := gy + height
		// Top Layer (3x3 cross)
		cross := [][2]int{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}}
		for _, c := range cross {
			voxels = append(voxels, TreeVoxel{gx + c[0], topY, gz + c[1], leafBlock})
		}

		// Layer topY - 1: 3x3 square + cross tips
		for dx := -1; dx <= 1; dx++ {
			for dz := -1; dz <= 1; dz++ {
				voxels = append(voxels, TreeVoxel{gx + dx, topY - 1, gz + dz, leafBlock})
			}
		}
		for _, c := range [][2]int{{2, 0}, {-2, 0}, {0, 2}, {0, -2}} {
			voxels = append(voxels, TreeVoxel{gx + c[0], topY - 1, gz + c[1], leafBlock})
		}

		// Layers topY - 2 and topY - 3: 5x5 square with 4 corners omitted
		for yOffset := 2; yOffset <= 3; yOffset++ {
			curY := topY - yOffset
			for dx := -2; dx <= 2; dx++ {
				for dz := -2; dz <= 2; dz++ {
					if (dx == -2 || dx == 2) && (dz == -2 || dz == 2) {
						continue
					}
					voxels = append(voxels, TreeVoxel{gx + dx, curY, gz + dz, leafBlock})
				}
			}
		}

		// Bottom leaf layer (topY - 4): 3x3 square
		for dx := -1; dx <= 1; dx++ {
			for dz := -1; dz <= 1; dz++ {
				voxels = append(voxels, TreeVoxel{gx + dx, topY - 4, gz + dz, leafBlock})
			}
		}

	case TreeSpruce:
		height := 8 + (seed % 4) // 8 to 11 blocks tall
		logBlock := BlockSpruceLog
		leafBlock := BlockSpruceLeaves

		// Trunk
		for h := 1; h < height; h++ {
			voxels = append(voxels, TreeVoxel{gx, gy + h, gz, logBlock})
		}

		// Conical evergreen canopy
		topY := gy + height
		voxels = append(voxels, TreeVoxel{gx, topY, gz, leafBlock})

		cross1 := [][2]int{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}}
		for _, c := range cross1 {
			voxels = append(voxels, TreeVoxel{gx + c[0], topY - 1, gz + c[1], leafBlock})
		}

		for yOffset := 2; yOffset < height-1; yOffset++ {
			curY := topY - yOffset
			isWide := (yOffset % 2) == 1
			if isWide {
				// Radius 2 cross (5x5 with corners missing)
				for dx := -2; dx <= 2; dx++ {
					for dz := -2; dz <= 2; dz++ {
						if (dx == -2 || dx == 2) && (dz == -2 || dz == 2) {
							continue
						}
						voxels = append(voxels, TreeVoxel{gx + dx, curY, gz + dz, leafBlock})
					}
				}
			} else {
				// Radius 1: 3x3 square
				for dx := -1; dx <= 1; dx++ {
					for dz := -1; dz <= 1; dz++ {
						voxels = append(voxels, TreeVoxel{gx + dx, curY, gz + dz, leafBlock})
					}
				}
			}
		}

	case TreeLargeOak:
		height := 8 + (seed % 3) // 8 to 10 blocks tall
		logBlock := BlockOakLog
		leafBlock := BlockOakLeaves

		for h := 1; h < height; h++ {
			voxels = append(voxels, TreeVoxel{gx, gy + h, gz, logBlock})
		}

		topY := gy + height
		cross := [][2]int{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}}
		for _, c := range cross {
			voxels = append(voxels, TreeVoxel{gx + c[0], topY, gz + c[1], leafBlock})
		}
		for dx := -1; dx <= 1; dx++ {
			for dz := -1; dz <= 1; dz++ {
				voxels = append(voxels, TreeVoxel{gx + dx, topY - 1, gz + dz, leafBlock})
			}
		}
		for dx := -2; dx <= 2; dx++ {
			for dz := -2; dz <= 2; dz++ {
				if (dx == -2 || dx == 2) && (dz == -2 || dz == 2) {
					continue
				}
				voxels = append(voxels, TreeVoxel{gx + dx, topY - 2, gz + dz, leafBlock})
			}
		}

		branchDirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, -1}}
		b1 := branchDirs[seed%len(branchDirs)]
		b2 := branchDirs[(seed+3)%len(branchDirs)]
		for _, bd := range [][2]int{b1, b2} {
			bx := gx + bd[0]
			by := gy + height - 3
			bz := gz + bd[1]
			voxels = append(voxels, TreeVoxel{bx, by, bz, logBlock})
			for lx := -1; lx <= 1; lx++ {
				for lz := -1; lz <= 1; lz++ {
					for ly := -1; ly <= 1; ly++ {
						if lx*lx+ly*ly+lz*lz <= 2 {
							voxels = append(voxels, TreeVoxel{bx + lx, by + ly, bz + lz, leafBlock})
						}
					}
				}
			}
		}

	default: // TreeSmallOak (Classic Minecraft Oak)
		height := 5 + (seed % 2) // 5 or 6 blocks tall
		logBlock := BlockOakLog
		leafBlock := BlockOakLeaves

		for h := 1; h < height; h++ {
			voxels = append(voxels, TreeVoxel{gx, gy + h, gz, logBlock})
		}

		topY := gy + height
		cross := [][2]int{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}}
		for _, c := range cross {
			voxels = append(voxels, TreeVoxel{gx + c[0], topY, gz + c[1], leafBlock})
		}

		for dx := -1; dx <= 1; dx++ {
			for dz := -1; dz <= 1; dz++ {
				voxels = append(voxels, TreeVoxel{gx + dx, topY - 1, gz + dz, leafBlock})
			}
		}
		for _, c := range [][2]int{{2, 0}, {-2, 0}, {0, 2}, {0, -2}} {
			voxels = append(voxels, TreeVoxel{gx + c[0], topY - 1, gz + c[1], leafBlock})
		}

		for dx := -2; dx <= 2; dx++ {
			for dz := -2; dz <= 2; dz++ {
				if (dx == -2 || dx == 2) && (dz == -2 || dz == 2) {
					continue
				}
				voxels = append(voxels, TreeVoxel{gx + dx, topY - 2, gz + dz, leafBlock})
			}
		}

		for dx := -2; dx <= 2; dx++ {
			for dz := -2; dz <= 2; dz++ {
				isCorner := (dx == -2 || dx == 2) && (dz == -2 || dz == 2)
				if isCorner && ((seed+dx*3+dz*7)%4 != 0) {
					continue
				}
				voxels = append(voxels, TreeVoxel{gx + dx, topY - 3, gz + dz, leafBlock})
			}
		}
	}

	return voxels
}

// generateChunk deterministically generates rich biomes, mountains, valleys, 3D caves, balanced ores, and seamless trees
func (w *VoxelWorld) generateChunk(cx, cz int) *ChunkData {
	chunk := &ChunkData{
		Coord: ChunkCoord{X: cx, Z: cz},
	}

	startX := cx * ChunkSize
	startZ := cz * ChunkSize

	// 1. Terrain Shape, 3D Caves, and Balanced Ore Stratification
	for lx := 0; lx < ChunkSize; lx++ {
		for lz := 0; lz < ChunkSize; lz++ {
			wx := float64(startX + lx)
			wz := float64(startZ + lz)

			height, biome := GetTerrainHeight(wx, wz)

			// Bedrock at y = 0
			chunk.Blocks[lx][0][lz] = BlockBedrock

			// Underground carving & ores
			for y := 1; y < height; y++ {
				// 3D Tubular Cave Carving (Interconnecting tunnels)
				n1 := Perlin3D(wx*0.038, float64(y)*0.07, wz*0.038)
				n2 := Perlin3D((wx+311.0)*0.038, float64(y)*0.07, (wz+311.0)*0.038)
				caveSample := n1*n1 + n2*n2
				if caveSample < 0.032 && y > 2 && (y < height-1 || (height > 24 && y < height && caveSample < 0.018)) {
					chunk.Blocks[lx][y][lz] = BlockAir
					continue
				}

				// Subsurface top layers (dirt, sand, sandstone)
				if y >= height-3 {
					if height <= WaterLevel+1 || biome == BiomeDesert {
						if y == height-3 && biome != BiomeDesert {
							chunk.Blocks[lx][y][lz] = BlockSandstone
						} else {
							chunk.Blocks[lx][y][lz] = BlockSand
						}
					} else if biome == BiomeMountains && height > 30 {
						chunk.Blocks[lx][y][lz] = BlockStone
					} else {
						chunk.Blocks[lx][y][lz] = BlockDirt
					}
					continue
				}

				// Balanced Authentic Ore Veins
				oreHash := int(math.Abs(math.Sin(wx*12.9898+wz*78.233+float64(y)*37.719)*43758.5453)) % 1000

				if y <= 8 && oreHash == 24 {
					chunk.Blocks[lx][y][lz] = BlockDiamondOre // Deep rare diamond
				} else if biome == BiomeMountains && y >= 16 && y <= 40 && oreHash == 25 {
					chunk.Blocks[lx][y][lz] = BlockEmeraldOre // High mountain emeralds
				} else if y <= 11 && oreHash >= 20 && oreHash < 23 {
					chunk.Blocks[lx][y][lz] = BlockRedstoneOre // Deep redstone
				} else if y >= 4 && y <= 16 && oreHash >= 18 && oreHash < 20 {
					chunk.Blocks[lx][y][lz] = BlockLapisOre // Lapis lazuli
				} else if y <= 14 && oreHash >= 15 && oreHash < 18 {
					chunk.Blocks[lx][y][lz] = BlockGoldOre // Deep gold
				} else if y <= 32 && oreHash >= 8 && oreHash < 15 {
					chunk.Blocks[lx][y][lz] = BlockIronOre // Common underground iron
				} else if y <= 42 && oreHash < 8 {
					chunk.Blocks[lx][y][lz] = BlockCoalOre // Coal throughout mountains & stone
				} else if oreHash >= 50 && oreHash <= 52 {
					chunk.Blocks[lx][y][lz] = BlockGravel // Gravel pockets
				} else {
					chunk.Blocks[lx][y][lz] = BlockStone
				}
			}

			// Surface Block
			if height <= WaterLevel+1 {
				if biome == BiomeTaiga || biome == BiomeMountains {
					chunk.Blocks[lx][height][lz] = BlockGravel
				} else {
					chunk.Blocks[lx][height][lz] = BlockSand
				}
			} else if biome == BiomeDesert {
				chunk.Blocks[lx][height][lz] = BlockSand
			} else if biome == BiomeMountains {
				if height >= 35 {
					chunk.Blocks[lx][height][lz] = BlockSnow
				} else if height > 28 {
					if (int(wx+wz)%2) == 0 && height > 32 {
						chunk.Blocks[lx][height][lz] = BlockCobblestone
					} else {
						chunk.Blocks[lx][height][lz] = BlockStone
					}
				} else {
					chunk.Blocks[lx][height][lz] = BlockGrass
				}
			} else {
				chunk.Blocks[lx][height][lz] = BlockGrass
			}

			// Water Filling
			for y := height + 1; y <= WaterLevel; y++ {
				chunk.Blocks[lx][y][lz] = BlockWater
				// Clay deposits in shallow riverbed/ocean
				if y == height+1 && height >= WaterLevel-3 && (int(wx*7+wz*13)%5 == 0) {
					chunk.Blocks[lx][height][lz] = BlockClay
				}
			}
		}
	}

	// 2. Multi-Chunk Seamless Tree Placement (3x3 neighborhood query prevents border clipping)
	for ndx := -1; ndx <= 1; ndx++ {
		for ndz := -1; ndz <= 1; ndz++ {
			ncx := cx + ndx
			ncz := cz + ndz

			chunkCenterX := float64(ncx*ChunkSize + 8)
			chunkCenterZ := float64(ncz*ChunkSize + 8)
			_, cBiome := GetTerrainHeight(chunkCenterX, chunkCenterZ)

			numTrees := 0
			switch cBiome {
			case BiomeForest:
				numTrees = 2 + int(math.Abs(math.Sin(float64(ncx*17+ncz*31))*2.0))
			case BiomeBirchForest:
				numTrees = 2 + int(math.Abs(math.Sin(float64(ncx*19+ncz*29))*3.0))
			case BiomeTaiga:
				numTrees = 2 + int(math.Abs(math.Sin(float64(ncx*23+ncz*37))*2.0))
			case BiomePlains:
				treeChance := int(math.Abs(math.Sin(float64(ncx*13+ncz*47))*1000.0)) % 100
				if treeChance < 25 {
					numTrees = 1
				}
			case BiomeMountains:
				treeChance := int(math.Abs(math.Sin(float64(ncx*29+ncz*53))*1000.0)) % 100
				if treeChance < 20 {
					numTrees = 1
				}
			}

			for t := 0; t < numTrees; t++ {
				seed := int(math.Abs(math.Sin(float64(ncx*101+ncz*73+t*37))*10000.0))
				tx := ncx*ChunkSize + 3 + (seed % 10)
				tz := ncz*ChunkSize + 3 + ((seed / 10) % 10)
				ty, tBiome := GetTerrainHeight(float64(tx), float64(tz))

				if ty > WaterLevel && tBiome != BiomeDesert && tBiome != BiomeOcean {
					treeType := TreeSmallOak
					switch tBiome {
					case BiomeTaiga:
						treeType = TreeSpruce
					case BiomeBirchForest:
						treeType = TreeBirch
					case BiomeForest:
						roll := seed % 100
						if roll < 15 {
							treeType = TreeLargeOak
						} else if roll < 45 {
							treeType = TreeBirch
						} else {
							treeType = TreeSmallOak
						}
					case BiomeMountains:
						if (seed % 2) == 0 {
							treeType = TreeSpruce
						} else {
							treeType = TreeSmallOak
						}
					}

					treeVoxels := generateTreeVoxels(tx, ty, tz, treeType, seed)
					for _, tv := range treeVoxels {
						if tv.X >= startX && tv.X < startX+ChunkSize && tv.Z >= startZ && tv.Z < startZ+ChunkSize {
							lx := tv.X - startX
							lz := tv.Z - startZ
							if tv.Y > 0 && tv.Y < WorldHeight {
								if IsLog(tv.Block) {
									chunk.Blocks[lx][tv.Y][lz] = tv.Block
								} else if IsLeaf(tv.Block) {
									if chunk.Blocks[lx][tv.Y][lz] == BlockAir || chunk.Blocks[lx][tv.Y][lz] == BlockSnow {
										chunk.Blocks[lx][tv.Y][lz] = tv.Block
									}
								}
							}
						}
					}
				}
			}

			// Fallen Logs in Woodlands
			if cBiome == BiomeForest || cBiome == BiomeBirchForest || cBiome == BiomeTaiga {
				logChance := int(math.Abs(math.Sin(float64(ncx*43+ncz*61))*1000.0)) % 100
				if logChance < 18 {
					fx := ncx*ChunkSize + 4 + (logChance % 8)
					fz := ncz*ChunkSize + 4 + ((logChance * 3) % 8)
					fy, _ := GetTerrainHeight(float64(fx), float64(fz))
					if fy > WaterLevel {
						logType := BlockOakLog
						if cBiome == BiomeBirchForest {
							logType = BlockBirchLog
						} else if cBiome == BiomeTaiga {
							logType = BlockSpruceLog
						}
						length := 3 + (logChance % 2)
						for l := 0; l < length; l++ {
							gx := fx + l
							gz := fz
							if gx >= startX && gx < startX+ChunkSize && gz >= startZ && gz < startZ+ChunkSize {
								lx := gx - startX
								lz := gz - startZ
								if fy+1 < WorldHeight && chunk.Blocks[lx][fy+1][lz] == BlockAir {
									chunk.Blocks[lx][fy+1][lz] = logType
									if l == 1 && (logChance%2 == 0) && fy+2 < WorldHeight {
										chunk.Blocks[lx][fy+2][lz] = BlockBrownMushroom
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// 3. Surface Foliage & Decoration Pass
	for lx := 0; lx < ChunkSize; lx++ {
		for lz := 0; lz < ChunkSize; lz++ {
			wx := float64(startX + lx)
			wz := float64(startZ + lz)
			y := chunk.GetHighestBlock(lx, lz)
			if y <= 0 || y >= WorldHeight-3 {
				continue
			}

			surf := chunk.Blocks[lx][y][lz]
			above := chunk.Blocks[lx][y+1][lz]
			if above != BlockAir {
				continue
			}

			foliageSeed := int(math.Abs(math.Sin(wx*12.9898+wz*78.233)*43758.5453)) % 1000
			_, biome := GetTerrainHeight(wx, wz)

			if surf == BlockGrass && y > WaterLevel {
				switch biome {
				case BiomePlains:
					if foliageSeed < 160 {
						chunk.Blocks[lx][y+1][lz] = BlockTallGrass
					} else if foliageSeed < 195 {
						chunk.Blocks[lx][y+1][lz] = BlockDandelion
					} else if foliageSeed < 230 {
						chunk.Blocks[lx][y+1][lz] = BlockPoppy
					} else if foliageSeed < 255 {
						chunk.Blocks[lx][y+1][lz] = BlockCornflower
					} else if foliageSeed < 275 {
						chunk.Blocks[lx][y+1][lz] = BlockAllium
					} else if foliageSeed == 280 {
						chunk.Blocks[lx][y+1][lz] = BlockPumpkin
					}
				case BiomeForest:
					if foliageSeed < 110 {
						chunk.Blocks[lx][y+1][lz] = BlockTallGrass
					} else if foliageSeed < 135 {
						chunk.Blocks[lx][y+1][lz] = BlockPoppy
					} else if foliageSeed < 155 {
						chunk.Blocks[lx][y+1][lz] = BlockDandelion
					} else if foliageSeed < 175 {
						chunk.Blocks[lx][y+1][lz] = BlockRedMushroom
					} else if foliageSeed < 195 {
						chunk.Blocks[lx][y+1][lz] = BlockBrownMushroom
					}
				case BiomeBirchForest:
					if foliageSeed < 130 {
						chunk.Blocks[lx][y+1][lz] = BlockTallGrass
					} else if foliageSeed < 165 {
						chunk.Blocks[lx][y+1][lz] = BlockAllium
					} else if foliageSeed < 190 {
						chunk.Blocks[lx][y+1][lz] = BlockDandelion
					}
				case BiomeTaiga:
					if foliageSeed < 130 {
						chunk.Blocks[lx][y+1][lz] = BlockTallGrass
					} else if foliageSeed < 150 {
						chunk.Blocks[lx][y+1][lz] = BlockBrownMushroom
					}
				}
			} else if surf == BlockSand {
				if biome == BiomeDesert {
					if foliageSeed < 22 {
						chunk.Blocks[lx][y+1][lz] = BlockDeadBush
					} else if foliageSeed >= 30 && foliageSeed < 55 {
						cactusH := 1 + (foliageSeed % 3)
						for ch := 1; ch <= cactusH && y+ch < WorldHeight-1; ch++ {
							chunk.Blocks[lx][y+ch][lz] = BlockCactus
						}
					}
				} else if y == WaterLevel || y == WaterLevel+1 {
					hasAdjacentWater := (lx > 0 && chunk.Blocks[lx-1][WaterLevel][lz] == BlockWater) ||
						(lx < ChunkSize-1 && chunk.Blocks[lx+1][WaterLevel][lz] == BlockWater) ||
						(lz > 0 && chunk.Blocks[lx][WaterLevel][lz-1] == BlockWater) ||
						(lz < ChunkSize-1 && chunk.Blocks[lx][WaterLevel][lz+1] == BlockWater)
					if hasAdjacentWater && foliageSeed < 60 {
						caneH := 2 + (foliageSeed % 2)
						for ch := 1; ch <= caneH && y+ch < WorldHeight-1; ch++ {
							chunk.Blocks[lx][y+ch][lz] = BlockSugarCane
						}
					}
				}
			}
		}
	}

	return chunk
}

// GetHighestBlock in local chunk coordinates
func (c *ChunkData) GetHighestBlock(lx, lz int) int {
	for y := WorldHeight - 1; y >= 0; y-- {
		if c.Blocks[lx][y][lz] != BlockAir && c.Blocks[lx][y][lz] != BlockWater {
			return y
		}
	}
	return 0
}

// GetBlock returns block type at ANY global infinite coordinate (x, y, z)
func (w *VoxelWorld) GetBlock(x, y, z int) BlockType {
	if y < 0 || y >= WorldHeight {
		return BlockAir
	}

	cx := x >> 4
	cz := z >> 4
	lx := x & 15
	lz := z & 15

	chunk := w.GetChunk(cx, cz)
	return chunk.Blocks[lx][y][lz]
}

// SetBlock changes block type at ANY global infinite coordinate (x, y, z)
func (w *VoxelWorld) SetBlock(x, y, z int, b BlockType) {
	if y < 0 || y >= WorldHeight {
		return
	}

	cx := x >> 4
	cz := z >> 4
	lx := x & 15
	lz := z & 15

	chunk := w.GetChunk(cx, cz)
	oldB := chunk.Blocks[lx][y][lz]
	if oldB == b {
		return
	}
	chunk.Blocks[lx][y][lz] = b
	chunk.Modified = true

	pos := BlockPos{X: x, Y: y, Z: z}
	
	newDef := BlockRegistry[b]
	if newDef.IsLightSource {
		w.Torches[pos] = newDef.LightLevel
	} else {
		oldDef := BlockRegistry[oldB]
		if oldDef.IsLightSource {
			delete(w.Torches, pos)
		}
	}
}

// GetHighestOpaqueBlock finds the top solid non-transparent block (ignoring leaves, glass, torches, water, plants)
func (w *VoxelWorld) GetHighestOpaqueBlock(x, z int) int {
	for y := WorldHeight - 1; y >= 0; y-- {
		b := w.GetBlock(x, y, z)
		if b == BlockAir || b == BlockWater || IsLeaf(b) || b == BlockGlass || b == BlockTorch || IsPlant(b) {
			continue
		}
		if BlockRegistry[b].IsSolid {
			return y
		}
	}
	return 0
}

// CheckLineOfSight performs a 3D Bresenham line algorithm to check for occlusion.
// Returns true if there is a clear line of sight (no solid blocks) between start and end.
func (w *VoxelWorld) CheckLineOfSight(x0, y0, z0, x1, y1, z1 int) bool {
	dx := math.Abs(float64(x1 - x0))
	dy := math.Abs(float64(y1 - y0))
	dz := math.Abs(float64(z1 - z0))

	var sx, sy, sz int
	if x0 < x1 { sx = 1 } else { sx = -1 }
	if y0 < y1 { sy = 1 } else { sy = -1 }
	if z0 < z1 { sz = 1 } else { sz = -1 }

	if dx >= dy && dx >= dz {
		err1 := 2*dy - dx
		err2 := 2*dz - dx
		for i := 0; i < int(dx); i++ {
			if w.IsSolid(x0, y0, z0) { return false }
			if err1 > 0 {
				y0 += sy
				err1 -= 2 * dx
			}
			if err2 > 0 {
				z0 += sz
				err2 -= 2 * dx
			}
			err1 += 2 * dy
			err2 += 2 * dz
			x0 += sx
		}
	} else if dy >= dx && dy >= dz {
		err1 := 2*dx - dy
		err2 := 2*dz - dy
		for i := 0; i < int(dy); i++ {
			if w.IsSolid(x0, y0, z0) { return false }
			if err1 > 0 {
				x0 += sx
				err1 -= 2 * dy
			}
			if err2 > 0 {
				z0 += sz
				err2 -= 2 * dy
			}
			err1 += 2 * dx
			err2 += 2 * dz
			y0 += sy
		}
	} else {
		err1 := 2*dy - dz
		err2 := 2*dx - dz
		for i := 0; i < int(dz); i++ {
			if w.IsSolid(x0, y0, z0) { return false }
			if err1 > 0 {
				y0 += sy
				err1 -= 2 * dz
			}
			if err2 > 0 {
				x0 += sx
				err2 -= 2 * dz
			}
			err1 += 2 * dy
			err2 += 2 * dx
			z0 += sz
		}
	}
	return !w.IsSolid(x0, y0, z0)
}

// GetLightLevel returns authentic Minecraft SkyLight (0.0..1.0) and TorchLight (0.0..1.0)
func (w *VoxelWorld) GetLightLevel(x, y, z int) (float32, float32) {
	if y >= WorldHeight-1 {
		return 1.0, 0.0
	}

	topOpaque := w.GetHighestOpaqueBlock(x, z)
	var skyLight float32 = 1.0

	if y < topOpaque {
		// Inside solid cave or under solid overhang
		skyLight = 0.12 // Base cave darkness
		
		// Look up and diagonally for skylight, but only if we have line of sight!
		for ox := -2; ox <= 2; ox++ {
			for oz := -2; oz <= 2; oz++ {
				if ox == 0 && oz == 0 {
					continue
				}
				dist := float32(math.Sqrt(float64(ox*ox + oz*oz)))
				neighborOpaque := w.GetHighestOpaqueBlock(x+ox, z+oz)
				
				// If the neighbor column has open sky, and we can SEE it without blocks in the way
				if y >= neighborOpaque {
					// Add 1 to y to raycast towards the sky hole
					if w.CheckLineOfSight(x, y, z, x+ox, neighborOpaque, z+oz) {
						diffused := 1.0 - dist*0.14
						if diffused > skyLight {
							skyLight = diffused
						}
					}
				}
			}
		}
	} else {
		// Above all opaque terrain: check for soft tree leaf shade
		leafCount := 0
		for checkY := y + 1; checkY < WorldHeight; checkY++ {
			if IsLeaf(w.GetBlock(x, checkY, z)) {
				leafCount++
			}
		}
		if leafCount > 0 {
			skyLight = 1.0 - float32(leafCount)*0.04
			if skyLight < 0.82 {
				skyLight = 0.82
			}
		}
	}

	// 2. Torch / Block Light Level
	var maxTorchLight float32 = 0.0
	for pos, strength := range w.Torches {
		dx := math.Abs(float64(x - pos.X))
		dy := math.Abs(float64(y - pos.Y))
		dz := math.Abs(float64(z - pos.Z))
		manhattan := dx + dy + dz
		if manhattan <= float64(strength) {
			// CRITICAL FIX: Only illuminate if there is a clear line of sight to the torch!
			if w.CheckLineOfSight(x, y, z, pos.X, pos.Y, pos.Z) {
				tLight := float32(float64(strength)-manhattan) / 15.0 // Max light level is 15
				if tLight > maxTorchLight {
					maxTorchLight = tLight
				}
			}
		}
	}

	return skyLight, maxTorchLight
}

// GetHighestBlock at ANY global (x, z) coordinate
func (w *VoxelWorld) GetHighestBlock(x, z int) int {
	for y := WorldHeight - 1; y >= 0; y-- {
		b := w.GetBlock(x, y, z)
		if b != BlockAir && b != BlockWater {
			return y
		}
	}
	return 0
}

// IsSolid returns whether block at (x, y, z) is solid
func (w *VoxelWorld) IsSolid(x, y, z int) bool {
	return BlockRegistry[w.GetBlock(x, y, z)].IsSolid
}

// SpreadWater simulates natural Minecraft water gravity and horizontal spreading
func (w *VoxelWorld) SpreadWater(x, y, z int, cm interface{ MarkBlockDirty(x, z int) }) {
	if y <= 1 || w.GetBlock(x, y, z) != BlockWater {
		return
	}

	// 1. Flow directly downwards if air
	if y > 1 && w.GetBlock(x, y-1, z) == BlockAir {
		w.SetBlock(x, y-1, z, BlockWater)
		cm.MarkBlockDirty(x, z)
		return
	}

	// 2. Spread horizontally into adjacent air blocks
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for _, d := range dirs {
		nx := x + d[0]
		nz := z + d[1]
		if w.GetBlock(nx, y, nz) == BlockAir {
			w.SetBlock(nx, y, nz, BlockWater)
			cm.MarkBlockDirty(nx, nz)
			if y > 1 && w.GetBlock(nx, y-1, nz) == BlockAir {
				w.SetBlock(nx, y-1, nz, BlockWater)
			}
		}
	}
}

// ScheduledUpdate handles delayed block physics for smooth sand falling and natural leaf decay
type ScheduledUpdate struct {
	X, Y, Z int
	Action  string // "fall_sand", "decay_leaf"
	Delay   float32
}

// QueueSandFall schedules an unsupported sand block to drop with a natural delay
func (w *VoxelWorld) QueueSandFall(x, y, z int) {
	if y <= 1 || w.GetBlock(x, y, z) != BlockSand {
		return
	}
	below := w.GetBlock(x, y-1, z)
	if below == BlockAir || below == BlockWater {
		// Avoid duplicate entries
		for _, u := range w.ScheduledUpdates {
			if u.X == x && u.Y == y && u.Z == z && u.Action == "fall_sand" {
				return
			}
		}
		w.ScheduledUpdates = append(w.ScheduledUpdates, ScheduledUpdate{
			X: x, Y: y, Z: z,
			Action: "fall_sand",
			Delay:  0.16,
		})
	}
}

// QueueLeafDecay schedules nearby disconnected leaves to crumble naturally over 0.2s - 2.5s
func (w *VoxelWorld) QueueLeafDecay(cx, cy, cz int) {
	for dx := -4; dx <= 4; dx++ {
		for dy := -3; dy <= 4; dy++ {
			for dz := -4; dz <= 4; dz++ {
				lx := cx + dx
				ly := cy + dy
				lz := cz + dz
				b := w.GetBlock(lx, ly, lz)
				if IsLeaf(b) {
					hasLog := false
					for ox := -3; ox <= 3 && !hasLog; ox++ {
						for oy := -3; oy <= 3 && !hasLog; oy++ {
							for oz := -3; oz <= 3 && !hasLog; oz++ {
								if ox*ox+oy*oy+oz*oz <= 12 {
									if IsLog(w.GetBlock(lx+ox, ly+oy, lz+oz)) {
										hasLog = true
									}
								}
							}
						}
					}
					if !hasLog {
						delay := 0.20 + float32((dx*7+dy*13+dz*19+100)%20)*0.08
						w.ScheduledUpdates = append(w.ScheduledUpdates, ScheduledUpdate{
							X: lx, Y: ly, Z: lz,
							Action: "decay_leaf",
							Delay:  delay,
						})
					}
				}
			}
		}
	}
}

// TickScheduledPhysics updates timed block physics (smooth falling sand and leaf decay)
func (w *VoxelWorld) TickScheduledPhysics(dt float32, cm interface{ MarkBlockDirty(x, z int) }, spawnParticles func(pos rl.Vector3, b BlockType)) {
	for i := len(w.ScheduledUpdates) - 1; i >= 0; i-- {
		u := &w.ScheduledUpdates[i]
		u.Delay -= dt
		if u.Delay <= 0 {
			action := u.Action
			x, y, z := u.X, u.Y, u.Z

			// Remove from queue
			w.ScheduledUpdates = append(w.ScheduledUpdates[:i], w.ScheduledUpdates[i+1:]...)

			if action == "fall_sand" {
				if w.GetBlock(x, y, z) == BlockSand && y > 1 {
					below := w.GetBlock(x, y-1, z)
					if below == BlockAir || below == BlockWater {
						w.SetBlock(x, y, z, BlockAir)
						w.SetBlock(x, y-1, z, BlockSand)
						cm.MarkBlockDirty(x, z)

						// Continue falling if still air below!
						if y-1 > 1 && (w.GetBlock(x, y-2, z) == BlockAir || w.GetBlock(x, y-2, z) == BlockWater) {
							w.ScheduledUpdates = append(w.ScheduledUpdates, ScheduledUpdate{
								X: x, Y: y - 1, Z: z,
								Action: "fall_sand",
								Delay:  0.14,
							})
						}
						// Check sand above
						if y+1 < WorldHeight && w.GetBlock(x, y+1, z) == BlockSand {
							w.QueueSandFall(x, y+1, z)
						}
					}
				}
			} else if action == "decay_leaf" {
				b := w.GetBlock(x, y, z)
				if IsLeaf(b) {
					hasLog := false
					for ox := -3; ox <= 3 && !hasLog; ox++ {
						for oy := -3; oy <= 3 && !hasLog; oy++ {
							for oz := -3; oz <= 3 && !hasLog; oz++ {
								if ox*ox+oy*oy+oz*oz <= 12 {
									if IsLog(w.GetBlock(x+ox, y+oy, z+oz)) {
										hasLog = true
									}
								}
							}
						}
					}
					if !hasLog {
						w.SetBlock(x, y, z, BlockAir)
						cm.MarkBlockDirty(x, z)
						if spawnParticles != nil {
							spawnParticles(rl.Vector3{X: float32(x), Y: float32(y), Z: float32(z)}, b)
						}
					}
				}
			}
		}
	}
}

// TickRandomBlocks simulates natural Minecraft random block ticks around the player (Grass spreading & growth, grass decay)
func (w *VoxelWorld) TickRandomBlocks(playerPos rl.Vector3, cm interface{ MarkBlockDirty(x, z int) }) {
	px := int(math.Floor(float64(playerPos.X)))
	py := int(math.Floor(float64(playerPos.Y)))
	pz := int(math.Floor(float64(playerPos.Z)))

	// Check 32 random blocks in active radius around player
	for i := 0; i < 32; i++ {
		rx := px + rand.Intn(49) - 24
		rz := pz + rand.Intn(49) - 24
		ry := py + rand.Intn(25) - 12

		if ry <= 0 || ry >= WorldHeight-1 {
			continue
		}

		b := w.GetBlock(rx, ry, rz)

		// 1. Grass Block Logic
		if b == BlockGrass {
			above := w.GetBlock(rx, ry+1, rz)
			aboveDef := BlockRegistry[above]
			// If grass is covered by an opaque solid block, it suffocates and turns into Dirt
			if aboveDef.IsSolid && !aboveDef.IsTransparent {
				w.SetBlock(rx, ry, rz, BlockDirt)
				cm.MarkBlockDirty(rx, rz)
			} else {
				// Attempt to spread to an adjacent exposed dirt block (dx, dz in [-1, 1], dy in [-2, 2])
				dx := rand.Intn(3) - 1
				dz := rand.Intn(3) - 1
				dy := rand.Intn(5) - 2
				tx, ty, tz := rx+dx, ry+dy, rz+dz
				if ty > 0 && ty < WorldHeight-1 {
					if w.GetBlock(tx, ty, tz) == BlockDirt {
						tabove := w.GetBlock(tx, ty+1, tz)
						taboveDef := BlockRegistry[tabove]
						// Dirt must have sunlight/air or transparent block above
						if !taboveDef.IsSolid || taboveDef.IsTransparent {
							w.SetBlock(tx, ty, tz, BlockGrass)
							cm.MarkBlockDirty(tx, tz)
						}
					}
				}
			}
		} else if b == BlockDirt {
			// 2. Dirt Block Spontaneous Growth if exposed to open sky/air and near grass
			above := w.GetBlock(rx, ry+1, rz)
			aboveDef := BlockRegistry[above]
			if !aboveDef.IsSolid || aboveDef.IsTransparent {
				// Check 3x3x3 neighbors for grass
				hasGrassNearby := false
				for ox := -1; ox <= 1 && !hasGrassNearby; ox++ {
					for oy := -1; oy <= 1 && !hasGrassNearby; oy++ {
						for oz := -1; oz <= 1 && !hasGrassNearby; oz++ {
							if w.GetBlock(rx+ox, ry+oy, rz+oz) == BlockGrass {
								hasGrassNearby = true
							}
						}
					}
				}
				if hasGrassNearby && rand.Float32() < 0.25 {
					w.SetBlock(rx, ry, rz, BlockGrass)
					cm.MarkBlockDirty(rx, rz)
				}
			}
		}
	}
}



