package voxel

import (
	"testing"
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
