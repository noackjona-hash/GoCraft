package voxel

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// SlotSave represents a saved item slot
type SlotSave struct {
	Type  BlockType `json:"type"`
	Count int       `json:"count"`
}

// PlayerSave holds persistent player position, stats, and mode
type PlayerSave struct {
	X           float32 `json:"x"`
	Y           float32 `json:"y"`
	Z           float32 `json:"z"`
	Yaw         float32 `json:"yaw"`
	Pitch       float32 `json:"pitch"`
	Health      float32 `json:"health"`
	Hunger      float32 `json:"hunger"`
	Oxygen      float32 `json:"oxygen"`
	Level       int     `json:"level"`
	ExpProgress float32 `json:"exp_progress"`
	Mode        int     `json:"mode"` // 0 = Survival, 1 = Creative
}

// InventorySave holds hotbar and main inventory slots
type InventorySave struct {
	SelectedSlot int        `json:"selected_slot"`
	Hotbar       []SlotSave `json:"hotbar"`
	Main         []SlotSave `json:"main"`
}

// TorchSave represents a placed torch location and light emission
type TorchSave struct {
	X          int   `json:"x"`
	Y          int   `json:"y"`
	Z          int   `json:"z"`
	LightLevel uint8 `json:"light_level"`
}

// LevelData bundles all world metadata and player progress into level.json
type LevelData struct {
	Version   int           `json:"version"`
	Player    PlayerSave    `json:"player"`
	Inventory InventorySave `json:"inventory"`
	TimeOfDay float32       `json:"time_of_day"`
	DayCount  int           `json:"day_count"`
	Torches   []TorchSave   `json:"torches"`
}

// SaveLevelData writes LevelData to saves/<world>/level.json atomically
func SaveLevelData(saveDir string, data *LevelData) error {
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize level data: %w", err)
	}

	tmpFile := filepath.Join(saveDir, "level.json.tmp")
	finalFile := filepath.Join(saveDir, "level.json")

	if err := os.WriteFile(tmpFile, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write temp level file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, finalFile); err != nil {
		// Fallback for Windows if target exists
		_ = os.Remove(finalFile)
		return os.Rename(tmpFile, finalFile)
	}

	return nil
}

// LoadLevelData reads saves/<world>/level.json if present
func LoadLevelData(saveDir string) (*LevelData, error) {
	levelPath := filepath.Join(saveDir, "level.json")
	bytes, err := os.ReadFile(levelPath)
	if err != nil {
		return nil, err
	}

	var data LevelData
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, fmt.Errorf("corrupt level file: %w", err)
	}

	return &data, nil
}

// SaveChunkGzip compresses 12,288 voxel bytes into a gzip binary file
func SaveChunkGzip(filePath string, chunk *ChunkData) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	// 1 byte header version
	if _, err := gz.Write([]byte{1}); err != nil {
		return err
	}

	raw := make([]byte, ChunkSize*WorldHeight*ChunkSize)
	idx := 0
	for x := 0; x < ChunkSize; x++ {
		for y := 0; y < WorldHeight; y++ {
			for z := 0; z < ChunkSize; z++ {
				raw[idx] = byte(chunk.Blocks[x][y][z])
				idx++
			}
		}
	}

	_, err = gz.Write(raw)
	return err
}

// LoadChunkGzip decompresses a chunk binary file from disk
func LoadChunkGzip(filePath string, cx, cz int) (*ChunkData, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	header := make([]byte, 1)
	if _, err := io.ReadFull(gz, header); err != nil {
		return nil, err
	}

	expectedLen := ChunkSize * WorldHeight * ChunkSize
	raw := make([]byte, expectedLen)
	if _, err := io.ReadFull(gz, raw); err != nil {
		return nil, err
	}

	chunk := &ChunkData{
		Coord:    ChunkCoord{X: cx, Z: cz},
		Modified: false,
	}

	idx := 0
	for x := 0; x < ChunkSize; x++ {
		for y := 0; y < WorldHeight; y++ {
			for z := 0; z < ChunkSize; z++ {
				chunk.Blocks[x][y][z] = BlockType(raw[idx])
				idx++
			}
		}
	}

	return chunk, nil
}
