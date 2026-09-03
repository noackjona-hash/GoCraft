package voxel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChunkSaveAndLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gocraft_test_chunk_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	chunk := &ChunkData{
		Coord:    ChunkCoord{X: 3, Z: -5},
		Modified: true,
	}
	// Place some distinct test blocks
	chunk.Blocks[0][0][0] = BlockBedrock
	chunk.Blocks[5][12][7] = BlockDiamondOre
	chunk.Blocks[15][24][15] = BlockTorch

	chunkFile := filepath.Join(tempDir, "c.3.-5.bin")
	if err := SaveChunkGzip(chunkFile, chunk); err != nil {
		t.Fatalf("SaveChunkGzip failed: %v", err)
	}

	loaded, err := LoadChunkGzip(chunkFile, 3, -5)
	if err != nil {
		t.Fatalf("LoadChunkGzip failed: %v", err)
	}

	if loaded.Coord.X != 3 || loaded.Coord.Z != -5 {
		t.Fatalf("Coord mismatch: got (%d, %d), expected (3, -5)", loaded.Coord.X, loaded.Coord.Z)
	}
	if loaded.Blocks[0][0][0] != BlockBedrock {
		t.Fatalf("Bedrock mismatch at [0][0][0]")
	}
	if loaded.Blocks[5][12][7] != BlockDiamondOre {
		t.Fatalf("DiamondOre mismatch at [5][12][7]")
	}
	if loaded.Blocks[15][24][15] != BlockTorch {
		t.Fatalf("Torch mismatch at [15][24][15]")
	}
}

func TestLevelDataSaveAndLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gocraft_test_level_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	saveData := &LevelData{
		Version: 1,
		Player: PlayerSave{
			X:           10.5,
			Y:           25.0,
			Z:           -8.2,
			Yaw:         1.57,
			Pitch:       0.2,
			Health:      18.0,
			Hunger:      19.5,
			Oxygen:      10.0,
			Level:       7,
			ExpProgress: 0.65,
			Mode:        0,
		},
		Inventory: InventorySave{
			SelectedSlot: 2,
			Hotbar: []SlotSave{
				{Type: BlockDiamondOre, Count: 3},
				{Type: BlockTorch, Count: 64},
			},
			Main: []SlotSave{
				{Type: BlockCobblestone, Count: 32},
			},
		},
		TimeOfDay: 0.45,
		DayCount:  4,
		Torches: []TorchSave{
			{X: 10, Y: 25, Z: -8, LightLevel: 14},
		},
	}

	if err := SaveLevelData(tempDir, saveData); err != nil {
		t.Fatalf("SaveLevelData failed: %v", err)
	}

	loaded, err := LoadLevelData(tempDir)
	if err != nil {
		t.Fatalf("LoadLevelData failed: %v", err)
	}

	if loaded.Player.X != 10.5 || loaded.Player.Y != 25.0 || loaded.Player.Z != -8.2 {
		t.Fatalf("Player pos mismatch: got (%.1f, %.1f, %.1f)", loaded.Player.X, loaded.Player.Y, loaded.Player.Z)
	}
	if loaded.Player.Health != 18.0 || loaded.Player.Level != 7 {
		t.Fatalf("Player stats mismatch: health=%.1f, level=%d", loaded.Player.Health, loaded.Player.Level)
	}
	if len(loaded.Inventory.Hotbar) != 2 || loaded.Inventory.Hotbar[0].Type != BlockDiamondOre {
		t.Fatalf("Hotbar slot 0 mismatch: %v", loaded.Inventory.Hotbar)
	}
	if loaded.TimeOfDay != 0.45 || loaded.DayCount != 4 {
		t.Fatalf("Time/day mismatch: time=%.2f, days=%d", loaded.TimeOfDay, loaded.DayCount)
	}
	if len(loaded.Torches) != 1 || loaded.Torches[0].LightLevel != 14 {
		t.Fatalf("Torch mismatch: %v", loaded.Torches)
	}
}

func TestWorldChunkPersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gocraft_test_world_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	world1 := NewVoxelWorld()
	world1.SaveDir = tempDir

	// Modify a block in world1
	world1.SetBlock(10, 20, 10, BlockDiamondOre)
	savedCount := world1.SaveAllModifiedChunks()
	if savedCount != 1 {
		t.Fatalf("Expected 1 saved chunk, got %d", savedCount)
	}

	// Create new world instance with same saveDir
	world2 := NewVoxelWorld()
	world2.SaveDir = tempDir

	// Verify the modified block is loaded from disk
	b := world2.GetBlock(10, 20, 10)
	if b != BlockDiamondOre {
		t.Fatalf("Expected BlockDiamondOre from saved chunk, got %v", b)
	}
}
