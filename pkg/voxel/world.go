package voxel

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	ChunkSize   = 16
	WorldHeight = 48
	WaterLevel  = 14
)

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
}

// NewVoxelWorld initializes the infinite procedural voxel world
func NewVoxelWorld() *VoxelWorld {
	return &VoxelWorld{
		Chunks:           make(map[ChunkCoord]*ChunkData),
		ScheduledUpdates: make([]ScheduledUpdate, 0),
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

// generateChunk deterministically generates terrain, ores, sand, water, and trees using continuous world coordinates
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

			// Continuous procedural Perlin-like 2D terrain elevation
			nx := wx * 0.038
			nz := wz * 0.038

			elevation := math.Sin(nx)*3.2 + math.Cos(nz)*3.2 +
				math.Sin(nx*2.2+nz*1.8)*1.6 + math.Cos(nx*0.8-nz*1.4)*2.8

			height := int(18.0 + elevation)
			if height < 2 {
				height = 2
			}
			if height >= WorldHeight-6 {
				height = WorldHeight - 6
			}

			// Bedrock at y = 0
			chunk.Blocks[lx][0][lz] = BlockBedrock

			// Underground stone & Ore Veins
			for y := 1; y < height-3; y++ {
				// Deterministic pseudo-random hash for ores
				oreHash := int(math.Abs(math.Sin(wx*12.9898+wz*78.233+float64(y)*37.719) * 43758.5453)) % 1000

				if y < 8 && oreHash < 12 {
					chunk.Blocks[lx][y][lz] = BlockDiamondOre // Diamonds!
				} else if y < 10 && oreHash < 20 {
					chunk.Blocks[lx][y][lz] = BlockEmeraldOre // Emeralds!
				} else if y < 12 && oreHash < 35 {
					chunk.Blocks[lx][y][lz] = BlockRedstoneOre // Redstone
				} else if y < 16 && oreHash < 50 {
					chunk.Blocks[lx][y][lz] = BlockLapisOre // Lapis Lazuli
				} else if y < 18 && oreHash < 75 {
					chunk.Blocks[lx][y][lz] = BlockGoldOre // Gold
				} else if y < 28 && oreHash < 115 {
					chunk.Blocks[lx][y][lz] = BlockIronOre // Iron
				} else if oreHash < 175 {
					chunk.Blocks[lx][y][lz] = BlockCoalOre // Coal
				} else {
					chunk.Blocks[lx][y][lz] = BlockStone
				}
			}

			// Dirt / Sand layers
			for y := height - 3; y < height; y++ {
				if y >= 1 {
					if height <= WaterLevel+1 {
						chunk.Blocks[lx][y][lz] = BlockSand
					} else {
						chunk.Blocks[lx][y][lz] = BlockDirt
					}
				}
			}

			// Top surface block: Grass or Sand
			if height <= WaterLevel+1 {
				chunk.Blocks[lx][height][lz] = BlockSand
			} else {
				chunk.Blocks[lx][height][lz] = BlockGrass
			}

			// Crystal Blue Water Lakes & Oceans
			for y := height + 1; y <= WaterLevel; y++ {
				chunk.Blocks[lx][y][lz] = BlockWater
			}
		}
	}

	// Deterministic Oak Trees
	treeHash := int(math.Abs(math.Sin(float64(cx)*19.123+float64(cz)*57.456)*10000)) % 100
	if treeHash < 75 {
		tx := 4 + (treeHash % 8)
		tz := 4 + ((treeHash * 3) % 8)
		ty := chunk.GetHighestBlock(tx, tz)

		if ty > WaterLevel && chunk.Blocks[tx][ty][tz] == BlockGrass {
			growTreeInChunk(chunk, tx, ty+1, tz)
		}
	}

	return chunk
}

func growTreeInChunk(chunk *ChunkData, x, y, z int) {
	trunkHeight := 4

	for h := 0; h < trunkHeight; h++ {
		if y+h < WorldHeight {
			chunk.Blocks[x][y+h][z] = BlockOakLog
		}
	}

	topY := y + trunkHeight

	for lx := -2; lx <= 2; lx++ {
		for lz := -2; lz <= 2; lz++ {
			for ly := -1; ly <= 2; ly++ {
				if math.Abs(float64(lx)) == 2 && math.Abs(float64(lz)) == 2 && ly >= 1 {
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
	chunk.Blocks[lx][y][lz] = b
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


