package voxel

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestBiomeAndTerrain(t *testing.T) {
	for x := -200; x <= 200; x += 37 {
		for z := -200; z <= 200; z += 43 {
			h, biome := GetTerrainHeight(float64(x), float64(z))
			if h < 2 || h >= WorldHeight-2 {
				t.Fatalf("Terrain height out of bounds at (%d, %d): %d", x, z, h)
			}
			if biome < BiomeOcean || biome > BiomeDesert {
				t.Fatalf("Invalid biome: %d", biome)
			}
		}
	}
}

func TestTreeGeneration(t *testing.T) {
	treeTypes := []TreeType{TreeSmallOak, TreeLargeOak, TreeBirch, TreeSpruce}
	for _, tt := range treeTypes {
		voxels := generateTreeVoxels(0, 16, 0, tt, 42)
		if len(voxels) == 0 {
			t.Fatalf("Tree type %d produced 0 voxels", tt)
		}
		hasLog := false
		hasLeaf := false
		for _, v := range voxels {
			if IsLog(v.Block) {
				hasLog = true
			}
			if IsLeaf(v.Block) {
				hasLeaf = true
			}
		}
		if !hasLog || !hasLeaf {
			t.Fatalf("Tree type %d missing log or leaf: log=%v, leaf=%v", tt, hasLog, hasLeaf)
		}
	}
}

func TestChunkGeneration(t *testing.T) {
	world := NewVoxelWorld()
	for cx := -2; cx <= 2; cx++ {
		for cz := -2; cz <= 2; cz++ {
			chunk := world.GetChunk(cx, cz)
			if chunk == nil {
				t.Fatalf("GetChunk(%d, %d) returned nil", cx, cz)
			}
			if chunk.Blocks[0][0][0] != BlockBedrock {
				t.Fatalf("Chunk (%d, %d) missing bedrock at [0][0][0]", cx, cz)
			}
		}
	}
}

func TestBlockRotation(t *testing.T) {
	// 1. Test IsLog for all variants
	logs := []BlockType{
		BlockOakLog, BlockOakLogX, BlockOakLogZ,
		BlockBirchLog, BlockBirchLogX, BlockBirchLogZ,
		BlockSpruceLog, BlockSpruceLogX, BlockSpruceLogZ,
	}
	for _, l := range logs {
		if !IsLog(l) {
			t.Fatalf("Expected IsLog to be true for block type %d", l)
		}
		if GetBlockDrop(l) != GetBaseLog(l) {
			t.Fatalf("Expected GetBlockDrop to return base log for %d, got %d", l, GetBlockDrop(l))
		}
	}

	// 2. Test Placement Normal Rotation
	// Normal along +X or -X -> X orientation
	if GetRotatedLogBlock(BlockBirchLog, rl.Vector3{X: 1, Y: 0, Z: 0}) != BlockBirchLogX {
		t.Fatalf("Expected BlockBirchLogX for X normal")
	}
	if GetRotatedLogBlock(BlockBirchLog, rl.Vector3{X: -1, Y: 0, Z: 0}) != BlockBirchLogX {
		t.Fatalf("Expected BlockBirchLogX for -X normal")
	}
	// Normal along +Z or -Z -> Z orientation
	if GetRotatedLogBlock(BlockOakLog, rl.Vector3{X: 0, Y: 0, Z: 1}) != BlockOakLogZ {
		t.Fatalf("Expected BlockOakLogZ for Z normal")
	}
	// Normal along +Y or -Y -> Vertical Y orientation
	if GetRotatedLogBlock(BlockSpruceLog, rl.Vector3{X: 0, Y: 1, Z: 0}) != BlockSpruceLog {
		t.Fatalf("Expected BlockSpruceLog for Y normal")
	}

	// 3. Test CycleBlockRotation
	if CycleBlockRotation(BlockBirchLog) != BlockBirchLogX {
		t.Fatalf("Expected BirchLog -> BirchLogX")
	}
	if CycleBlockRotation(BlockBirchLogX) != BlockBirchLogZ {
		t.Fatalf("Expected BirchLogX -> BirchLogZ")
	}
	if CycleBlockRotation(BlockBirchLogZ) != BlockBirchLog {
		t.Fatalf("Expected BirchLogZ -> BirchLog")
	}

	// 4. Test Texture Mapping (Rings vs Bark)
	// Vertical Birch Log: Rings on Top/Bottom (10, 2), Bark on Side (9, 2)
	col, row := GetBlockTextureAtlasPos(BlockBirchLog, FaceTop)
	if col != 10 || row != 2 {
		t.Fatalf("Expected (10, 2) for BirchLog Top face, got (%d, %d)", col, row)
	}
	col, row = GetBlockTextureAtlasPos(BlockBirchLog, FaceNorth)
	if col != 9 || row != 2 {
		t.Fatalf("Expected (9, 2) for BirchLog North face, got (%d, %d)", col, row)
	}

	// Horizontal Birch Log X: Rings on West/East (10, 2), Bark on Top/Bottom/North/South (9, 2)
	col, row = GetBlockTextureAtlasPos(BlockBirchLogX, FaceEast)
	if col != 10 || row != 2 {
		t.Fatalf("Expected (10, 2) for BirchLogX East face, got (%d, %d)", col, row)
	}
	col, row = GetBlockTextureAtlasPos(BlockBirchLogX, FaceTop)
	if col != 9 || row != 2 {
		t.Fatalf("Expected (9, 2) for BirchLogX Top face, got (%d, %d)", col, row)
	}
}

type mockChunkManager struct{}

func (m *mockChunkManager) MarkBlockDirty(x, z int) {}

func TestWaterSystem(t *testing.T) {
	// 1. Test Block Types and Helpers
	if !IsWater(BlockWater) || !IsWater(BlockWaterFlowing) {
		t.Fatalf("Expected IsWater to return true for BlockWater and BlockWaterFlowing")
	}
	if !IsLiquid(BlockWater) || !IsLiquid(BlockWaterFlowing) {
		t.Fatalf("Expected IsLiquid to return true for water blocks")
	}
	if IsWater(BlockStone) || IsWater(BlockAir) {
		t.Fatalf("Expected IsWater to return false for non-water blocks")
	}

	w := NewVoxelWorld()
	cm := &mockChunkManager{}

	// Setup a flat stone basin
	for x := 0; x < 10; x++ {
		for z := 0; z < 10; z++ {
			w.SetBlock(x, 10, z, BlockStone)
			w.SetBlock(x, 11, z, BlockAir)
			w.SetBlock(x, 12, z, BlockAir)
		}
	}

	// 2. Test Waterfall: Water dropping down into air
	w.SetBlock(5, 12, 5, BlockWater)
	w.SpreadWater(5, 12, 5, cm)
	if w.GetBlock(5, 11, 5) != BlockWaterFlowing {
		t.Fatalf("Expected vertical waterfall to create BlockWaterFlowing at (5, 11, 5), got %v", w.GetBlock(5, 11, 5))
	}

	// 3. Test Horizontal Spreading on solid ground
	w.SpreadWater(5, 11, 5, cm)
	hasSpread := w.GetBlock(6, 11, 5) == BlockWaterFlowing || w.GetBlock(4, 11, 5) == BlockWaterFlowing
	if !hasSpread {
		t.Fatalf("Expected water on ground to spread horizontally into adjacent air")
	}

	// 4. Test Water Recession: removing water source drains unsupported flowing water
	w.SetBlock(5, 12, 5, BlockAir)
	w.SetBlock(5, 11, 5, BlockAir)
	// Now (6, 11, 5) has no upstream water source within range
	w.SpreadWater(6, 11, 5, cm)
	if w.GetBlock(6, 11, 5) != BlockAir {
		t.Fatalf("Expected unsupported flowing water to recede and become BlockAir, got %v", w.GetBlock(6, 11, 5))
	}

	// 5. Test 2x2 Infinite Water Source Generation
	for x := 0; x < 4; x++ {
		for z := 0; z < 4; z++ {
			w.SetBlock(x, 11, z, BlockAir)
		}
	}
	w.SetBlock(1, 11, 1, BlockWater)
	w.SetBlock(2, 11, 2, BlockWater)
	w.SpreadWater(1, 11, 1, cm)
	w.SpreadWater(2, 11, 2, cm)

	corner := w.GetBlock(1, 11, 2)
	if corner != BlockWater {
		t.Fatalf("Expected 2x2 infinite source creation to produce BlockWater, got %v", corner)
	}

	// 6. Test Texture Atlas Pos
	col, row := GetBlockTextureAtlasPos(BlockWater, FaceTop)
	if col != 1 || row != 2 {
		t.Fatalf("Expected (1, 2) for BlockWater, got (%d, %d)", col, row)
	}
	col, row = GetBlockTextureAtlasPos(BlockWaterFlowing, FaceTop)
	if col != 0 || row != 14 {
		t.Fatalf("Expected (0, 14) for BlockWaterFlowing, got (%d, %d)", col, row)
	}
	col, row = GetBlockTextureAtlasPos(ItemWaterBucket, FaceTop)
	if col != 11 || row != 5 {
		t.Fatalf("Expected (11, 5) for ItemWaterBucket, got (%d, %d)", col, row)
	}
	col, row = GetBlockTextureAtlasPos(ItemBucket, FaceTop)
	if col != 12 || row != 5 {
		t.Fatalf("Expected (12, 5) for ItemBucket, got (%d, %d)", col, row)
	}
}

