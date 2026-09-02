package voxel

import (
	"math"
	"math/rand"

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
	Coord  ChunkCoord
	Blocks [ChunkSize][WorldHeight][ChunkSize]BlockType
}

// VoxelWorld manages infinite chunk generation and block queries
type VoxelWorld struct {
	Chunks           map[ChunkCoord]*ChunkData
	ScheduledUpdates []ScheduledUpdate
	Torches          map[BlockPos]uint8
}

// NewVoxelWorld initializes the infinite procedural voxel world
func NewVoxelWorld() *VoxelWorld {
	return &VoxelWorld{
		Chunks:           make(map[ChunkCoord]*ChunkData),
		ScheduledUpdates: make([]ScheduledUpdate, 0),
		Torches:          make(map[BlockPos]uint8),
	}
}

// GetChunk returns chunk at (cx, cz), generating it if not already created
func (w *VoxelWorld) GetChunk(cx, cz int) *ChunkData {
	coord := ChunkCoord{X: cx, Z: cz}
	chunk, exists := w.Chunks[coord]
	if !exists {
		chunk = w.generateChunk(cx, cz)
		w.Chunks[coord] = chunk
	}
	return chunk
}

// generateChunk deterministically generates rich biomes, mountains, valleys, 3D caves, balanced ores, and trees
func (w *VoxelWorld) generateChunk(cx, cz int) *ChunkData {
	chunk := &ChunkData{
		Coord: ChunkCoord{X: cx, Z: cz},
	}

	startX := cx * ChunkSize
	startZ := cz * ChunkSize

	for lx := 0; lx < ChunkSize; lx++ {
		for lz := 0; lz < ChunkSize; lz++ {
			wx := float64(startX + lx)
			wz := float64(startZ + lz)

			// 1. Biome Factors: Continentalness, Moisture, and Mountain Roughness
			continental := FractalNoise2D(wx*0.005, wz*0.005, 3, 0.5, 2.0)
			moisture := FractalNoise2D((wx+600.0)*0.006, (wz+600.0)*0.006, 2, 0.5, 2.0)
			roughness := FractalNoise2D((wx-400.0)*0.012, (wz-400.0)*0.012, 3, 0.5, 2.0)
			detail := Perlin2D(wx*0.04, wz*0.04) * 1.8

			isOcean := continental < -0.28
			isDesert := !isOcean && moisture < -0.22
			isMountain := !isOcean && continental > 0.15 && roughness > 0.20

			// 2. Continuous Biome-Driven Elevation
			var baseH float64
			if isOcean {
				// Deep water basin & gradual beach slope
				baseH = 9.0 + (continental+0.6)*8.0 + detail*0.5
			} else if isMountain {
				// Dramatic Mountain Peaks and Ridges
				peakBoost := (roughness - 0.20) * 26.0
				if peakBoost < 0 {
					peakBoost = 0
				}
				baseH = 22.0 + peakBoost + detail*2.2
			} else if isDesert {
				// Rolling Desert Dunes
				baseH = 16.0 + continental*4.0 + roughness*3.0 + detail
			} else {
				// Plains / Forest Rolling Hills
				baseH = 17.0 + continental*6.0 + roughness*4.0 + detail
			}

			height := int(math.Round(baseH))
			if height < 2 {
				height = 2
			}
			if height >= WorldHeight-4 {
				height = WorldHeight - 4
			}

			// 3. Bedrock at y = 0
			chunk.Blocks[lx][0][lz] = BlockBedrock

			// 4. Underground Stone, 3D Caves, and Balanced Realistic Ore Distribution
			for y := 1; y < height; y++ {
				// 3D Cave carving (natural subterranean tunnels)
				caveNoise := Perlin3D(wx*0.048, float64(y)*0.08, wz*0.048)
				if caveNoise > 0.62 && y > 2 && y < height-2 {
					chunk.Blocks[lx][y][lz] = BlockAir
					continue
				}

				// Top layers are handled separately
				if y >= height-3 {
					if height <= WaterLevel+1 || isDesert {
						if y == height-3 && !isDesert {
							chunk.Blocks[lx][y][lz] = BlockSandstone
						} else {
							chunk.Blocks[lx][y][lz] = BlockSand
						}
					} else if isMountain && height > 32 {
						chunk.Blocks[lx][y][lz] = BlockStone
					} else {
						chunk.Blocks[lx][y][lz] = BlockDirt
					}
					continue
				}

				// Rebalanced Authentic Ore Generation (Rare Veins)
				oreHash := int(math.Abs(math.Sin(wx*12.9898+wz*78.233+float64(y)*37.719)*43758.5453)) % 1000

				if y <= 8 && oreHash == 24 {
					// Diamond Ore: Rare deep layer (~0.1%)
					chunk.Blocks[lx][y][lz] = BlockDiamondOre
				} else if isMountain && y >= 12 && y <= 36 && oreHash == 25 {
					// Emerald Ore: Rare in high mountains (~0.1%)
					chunk.Blocks[lx][y][lz] = BlockEmeraldOre
				} else if y <= 10 && oreHash >= 20 && oreHash < 23 {
					// Redstone Ore: Deep layer (~0.3%)
					chunk.Blocks[lx][y][lz] = BlockRedstoneOre
				} else if y >= 4 && y <= 16 && oreHash >= 18 && oreHash < 20 {
					// Lapis Lazuli Ore: Mid-deep layer (~0.2%)
					chunk.Blocks[lx][y][lz] = BlockLapisOre
				} else if y <= 14 && oreHash >= 15 && oreHash < 18 {
					// Gold Ore: Mid-deep layer (~0.3%)
					chunk.Blocks[lx][y][lz] = BlockGoldOre
				} else if y <= 26 && oreHash >= 8 && oreHash < 15 {
					// Iron Ore: Common underground (~0.7%)
					chunk.Blocks[lx][y][lz] = BlockIronOre
				} else if y <= 38 && oreHash < 8 {
					// Coal Ore: Found throughout mountains & caves (~0.8%)
					chunk.Blocks[lx][y][lz] = BlockCoalOre
				} else {
					chunk.Blocks[lx][y][lz] = BlockStone
				}
			}

			// 5. Top Surface Block (Grass, Sand, or Mountain Stone)
			if height <= WaterLevel+1 || isDesert {
				chunk.Blocks[lx][height][lz] = BlockSand
			} else if isMountain && height > 32 {
				if roughness > 0.45 {
					chunk.Blocks[lx][height][lz] = BlockCobblestone
				} else {
					chunk.Blocks[lx][height][lz] = BlockStone
				}
			} else {
				chunk.Blocks[lx][height][lz] = BlockGrass
			}

			// 6. Water Body Filling
			for y := height + 1; y <= WaterLevel; y++ {
				chunk.Blocks[lx][y][lz] = BlockWater
			}
		}
	}

	// 7. Trees generation (Forests, Plains, and Meadows)
	chunkCenterX := float64(startX + 8)
	chunkCenterZ := float64(startZ + 8)
	cMoisture := FractalNoise2D((chunkCenterX+600.0)*0.006, (chunkCenterZ+600.0)*0.006, 2, 0.5, 2.0)
	cRoughness := FractalNoise2D((chunkCenterX-400.0)*0.012, (chunkCenterZ-400.0)*0.012, 3, 0.5, 2.0)
	cContinental := FractalNoise2D(chunkCenterX*0.005, chunkCenterZ*0.005, 3, 0.5, 2.0)

	if cContinental > -0.25 && cMoisture > -0.20 && (cRoughness <= 0.35 || cContinental <= 0.15) {
		// Tree Count based on forest density
		numTrees := 1
		if cMoisture > 0.12 {
			numTrees = 2 + int(math.Abs(math.Sin(float64(cx*17+cz*31))*3.0)) // 2 to 4 trees in dense forest
		} else {
			// Plains: 30% chance for 1 tree
			treeChance := int(math.Abs(math.Sin(float64(cx*13+cz*47))*1000.0)) % 100
			if treeChance > 35 {
				numTrees = 0
			}
		}

		for t := 0; t < numTrees; t++ {
			seed := int(math.Abs(math.Sin(float64(cx*101+cz*73+t*37))*10000.0))
			tx := 3 + (seed % 10)
			tz := 3 + ((seed / 10) % 10)
			ty := chunk.GetHighestBlock(tx, tz)

			if ty > WaterLevel && chunk.Blocks[tx][ty][tz] == BlockGrass {
				treeHeight := 4 + (seed % 3) // 4 to 6 block tall trees
				growTreeInChunk(chunk, tx, ty+1, tz, treeHeight)
			}
		}
	}

	return chunk
}

func growTreeInChunk(chunk *ChunkData, x, y, z, height int) {
	for h := 0; h < height; h++ {
		if y+h < WorldHeight {
			chunk.Blocks[x][y+h][z] = BlockOakLog
		}
	}

	topY := y + height - 1

	for lx := -2; lx <= 2; lx++ {
		for lz := -2; lz <= 2; lz++ {
			for ly := -1; ly <= 2; ly++ {
				// Corner rounding for natural spherical tree leaf canopies
				if (math.Abs(float64(lx)) == 2 && math.Abs(float64(lz)) == 2) && (ly >= 1 || math.Abs(float64(lx))+math.Abs(float64(lz)) > 3) {
					continue
				}
				leafX := x + lx
				leafY := topY + ly
				leafZ := z + lz

				if leafX >= 0 && leafX < ChunkSize && leafZ >= 0 && leafZ < ChunkSize && leafY >= 0 && leafY < WorldHeight {
					if chunk.Blocks[leafX][leafY][leafZ] == BlockAir {
						chunk.Blocks[leafX][leafY][leafZ] = BlockOakLeaves
					}
				}
			}
		}
	}
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
	chunk.Blocks[lx][y][lz] = b

	pos := BlockPos{X: x, Y: y, Z: z}
	if b == BlockTorch {
		w.Torches[pos] = 14
	} else if b == BlockFurnace {
		w.Torches[pos] = 13
	} else if oldB == BlockTorch || oldB == BlockFurnace {
		delete(w.Torches, pos)
	}
}

// GetHighestOpaqueBlock finds the top solid non-transparent block (ignoring leaves, glass, torches, water)
func (w *VoxelWorld) GetHighestOpaqueBlock(x, z int) int {
	for y := WorldHeight - 1; y >= 0; y-- {
		b := w.GetBlock(x, y, z)
		if b == BlockAir || b == BlockWater || b == BlockOakLeaves || b == BlockGlass || b == BlockTorch {
			continue
		}
		if BlockRegistry[b].IsSolid {
			return y
		}
	}
	return 0
}

// GetLightLevel returns authentic Minecraft SkyLight (0.0..1.0) and TorchLight (0.0..1.0)
func (w *VoxelWorld) GetLightLevel(x, y, z int) (float32, float32) {
	if y >= WorldHeight-1 {
		return 1.0, 0.0
	}

	topOpaque := w.GetHighestOpaqueBlock(x, z)
	var skyLight float32 = 1.0

	if y < topOpaque {
		// Inside solid cave or under solid overhang: calculate soft horizontal ambient sky diffusion
		depth := topOpaque - y
		if depth <= 4 {
			skyLight = 0.70 - float32(depth)*0.08
		} else if depth <= 10 {
			skyLight = 0.38 - float32(depth-4)*0.04
		} else {
			skyLight = 0.12 // Deep cave darkness
		}

		// Horizontal diffusion from nearby open sky columns within 3 blocks
		for ox := -2; ox <= 2; ox++ {
			for oz := -2; oz <= 2; oz++ {
				if ox == 0 && oz == 0 {
					continue
				}
				dist := float32(math.Sqrt(float64(ox*ox + oz*oz)))
				neighborOpaque := w.GetHighestOpaqueBlock(x+ox, z+oz)
				if y >= neighborOpaque {
					diffused := 1.0 - dist*0.14
					if diffused > skyLight {
						skyLight = diffused
					}
				}
			}
		}
	} else {
		// Above all opaque terrain: check for soft tree leaf shade
		leafCount := 0
		for checkY := y + 1; checkY < WorldHeight; checkY++ {
			if w.GetBlock(x, checkY, z) == BlockOakLeaves {
				leafCount++
			}
		}
		if leafCount > 0 {
			// Soft gentle tree shade: 0.85 to 0.94 (never black!)
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
			tLight := float32(float64(strength)-manhattan) / float32(strength)
			if tLight > maxTorchLight {
				maxTorchLight = tLight
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
				if w.GetBlock(lx, ly, lz) == BlockOakLeaves {
					hasLog := false
					for ox := -3; ox <= 3 && !hasLog; ox++ {
						for oy := -3; oy <= 3 && !hasLog; oy++ {
							for oz := -3; oz <= 3 && !hasLog; oz++ {
								if ox*ox+oy*oy+oz*oz <= 11 {
									if w.GetBlock(lx+ox, ly+oy, lz+oz) == BlockOakLog {
										hasLog = true
									}
								}
							}
						}
					}
					if !hasLog {
						delay := 0.25 + float32((dx*7+dy*13+dz*19+100)%20)*0.10
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
				if w.GetBlock(x, y, z) == BlockOakLeaves {
					hasLog := false
					for ox := -3; ox <= 3 && !hasLog; ox++ {
						for oy := -3; oy <= 3 && !hasLog; oy++ {
							for oz := -3; oz <= 3 && !hasLog; oz++ {
								if ox*ox+oy*oy+oz*oz <= 11 {
									if w.GetBlock(x+ox, y+oy, z+oz) == BlockOakLog {
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
							spawnParticles(rl.Vector3{X: float32(x), Y: float32(y), Z: float32(z)}, BlockOakLeaves)
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



