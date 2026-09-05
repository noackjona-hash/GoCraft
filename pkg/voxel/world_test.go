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
	if col != 12 || row != 8 {
		t.Fatalf("Expected (12, 8) for ItemWaterBucket, got (%d, %d)", col, row)
	}
	col, row = GetBlockTextureAtlasPos(ItemBucket, FaceTop)
	if col != 13 || row != 8 {
		t.Fatalf("Expected (13, 8) for ItemBucket, got (%d, %d)", col, row)
	}
}

func TestWaterLevelsAndCornerSmoothing(t *testing.T) {
	w := NewVoxelWorld()
	// Solid base at y=10
	for x := 0; x <= 10; x++ {
		for z := 0; z <= 10; z++ {
			w.SetBlock(x, 10, z, BlockStone)
		}
	}

	// Source block at (5, 11, 5)
	w.SetBlock(5, 11, 5, BlockWater)
	srcH := w.GetWaterHeight(5, 11, 5)
	if srcH != 0.88 {
		t.Fatalf("Expected source water height 0.88, got %.2f", srcH)
	}

	// Flowing water blocks spreading out horizontally
	w.SetBlock(6, 11, 5, BlockWaterFlowing)
	dist1H := w.GetWaterHeight(6, 11, 5)
	if dist1H >= srcH {
		t.Fatalf("Expected flowing water height (%.2f) to be lower than source (%.2f)", dist1H, srcH)
	}

	w.SetBlock(7, 11, 5, BlockWaterFlowing)
	dist2H := w.GetWaterHeight(7, 11, 5)
	if dist2H >= dist1H {
		t.Fatalf("Expected distance 2 water height (%.2f) to be lower than distance 1 (%.2f)", dist2H, dist1H)
	}

	// Corner smoothing
	cornerH := w.GetWaterCornerHeight(5, 11, 5, 1) // (+X, +Z) corner between source and flowing
	if cornerH <= dist1H || cornerH >= 1.0 {
		t.Fatalf("Expected corner height between source and flowing, got %.2f", cornerH)
	}

	// Waterfall above forces corner to 1.0
	w.SetBlock(5, 12, 5, BlockWater)
	wfCornerH := w.GetWaterCornerHeight(5, 11, 5, 0)
	if wfCornerH != 1.0 {
		t.Fatalf("Expected waterfall corner height to be 1.0, got %.2f", wfCornerH)
	}
}

func TestMiningTiersAndHarvestability(t *testing.T) {
	// 1. Instant break blocks
	dur, harvest := GetMiningSpeedAndHarvest(BlockTorch, BlockAir)
	if dur > 0.1 || !harvest {
		t.Fatalf("Expected instant break for torch with hand, got dur=%.2f, harvest=%v", dur, harvest)
	}

	// 2. Stone with bare hand -> No harvest, 7.5s
	dur, harvest = GetMiningSpeedAndHarvest(BlockStone, BlockAir)
	if harvest {
		t.Fatalf("Expected stone with bare hand to NOT drop items!")
	}
	if dur != 7.5 {
		t.Fatalf("Expected stone with bare hand to take 7.5s, got %.2f", dur)
	}

	// 3. Stone with Wooden Pickaxe -> Harvests, 1.125s
	dur, harvest = GetMiningSpeedAndHarvest(BlockStone, ItemWoodPickaxe)
	if !harvest {
		t.Fatalf("Expected stone with wood pickaxe to harvest cobblestone!")
	}
	if dur != 1.125 {
		t.Fatalf("Expected stone with wood pickaxe to take 1.125s, got %.2f", dur)
	}

	// 4. Iron Ore with Wooden Pickaxe -> No harvest (tier 1 < 2)
	dur, harvest = GetMiningSpeedAndHarvest(BlockIronOre, ItemWoodPickaxe)
	if harvest {
		t.Fatalf("Expected iron ore with wooden pickaxe to NOT drop items!")
	}

	// 5. Iron Ore with Stone Pickaxe -> Harvests (tier 2 >= 2)
	dur, harvest = GetMiningSpeedAndHarvest(BlockIronOre, ItemStonePickaxe)
	if !harvest {
		t.Fatalf("Expected iron ore with stone pickaxe to drop items!")
	}

	// 6. Diamond Ore with Stone Pickaxe -> No harvest (tier 2 < 3)
	dur, harvest = GetMiningSpeedAndHarvest(BlockDiamondOre, ItemStonePickaxe)
	if harvest {
		t.Fatalf("Expected diamond ore with stone pickaxe to NOT drop items!")
	}

	// 7. Diamond Ore with Iron Pickaxe -> Harvests (tier 3 >= 3)
	dur, harvest = GetMiningSpeedAndHarvest(BlockDiamondOre, ItemIronPickaxe)
	if !harvest {
		t.Fatalf("Expected diamond ore with iron pickaxe to drop diamond!")
	}

	// 8. Obsidian with Diamond Pickaxe -> Harvests
	dur, harvest = GetMiningSpeedAndHarvest(BlockObsidian, ItemDiamondPickaxe)
	if !harvest {
		t.Fatalf("Expected obsidian with diamond pickaxe to harvest!")
	}
	if dur > 10.0 {
		t.Fatalf("Expected obsidian with diamond pickaxe to take < 10s, got %.2f", dur)
	}

	// 9. Obsidian with Iron Pickaxe -> No harvest, 41.67s (penalty with pickaxe speed)
	dur, harvest = GetMiningSpeedAndHarvest(BlockObsidian, ItemIronPickaxe)
	if harvest {
		t.Fatalf("Expected obsidian with iron pickaxe to NOT harvest!")
	}
	if dur < 40.0 || dur > 45.0 {
		t.Fatalf("Expected obsidian with iron pickaxe to take ~41.67s, got %.2f", dur)
	}

	// 10. Obsidian with Bare Fist -> No harvest, 250s
	dur, harvest = GetMiningSpeedAndHarvest(BlockObsidian, BlockAir)
	if harvest {
		t.Fatalf("Expected obsidian with bare fist to NOT harvest!")
	}
	if dur != 250.0 {
		t.Fatalf("Expected obsidian with bare fist to take 250s, got %.2f", dur)
	}

	// 11. Wood log with bare fist vs Stone Axe
	durFist, harvestFist := GetMiningSpeedAndHarvest(BlockOakLog, BlockAir)
	durAxe, harvestAxe := GetMiningSpeedAndHarvest(BlockOakLog, ItemStoneAxe)
	if !harvestFist || !harvestAxe {
		t.Fatalf("Expected wood log to always harvest!")
	}
	if durAxe >= durFist {
		t.Fatalf("Expected stone axe (%.2fs) to be much faster than fist (%.2fs)", durAxe, durFist)
	}
}

func TestNewItemRegistryAndAtlas(t *testing.T) {
	newItems := []BlockType{
		ItemBow, ItemShield, ItemFlintAndSteel, ItemShears, ItemFishingRod,
		ItemCompass, ItemClock, ItemGoldenPickaxe, ItemGoldenAxe, ItemGoldenShovel,
		ItemGoldenSword, ItemGoldenApple, ItemCarrot, ItemGoldenCarrot, ItemPotato,
		ItemBakedPotato, ItemMushroomStew, ItemCookie, ItemRawChicken, ItemCookedChicken,
		ItemRawMutton, ItemCookedMutton, ItemWheat, ItemWheatSeeds, ItemMelonSlice,
		ItemSweetBerries, ItemFlint, ItemGoldNugget, ItemIronNugget, ItemRedstone,
		ItemLapisLazuli, ItemEmerald, ItemString, ItemFeather, ItemLeather,
		ItemSpiderEye, ItemEnderPearl, ItemSlimeball, ItemBlazeRod, ItemBook,
		ItemIronHelmet, ItemIronChestplate, ItemIronLeggings, ItemIronBoots,
		ItemDiamondHelmet, ItemDiamondChestplate, ItemDiamondLeggings, ItemDiamondBoots,
	}

	for _, item := range newItems {
		def, exists := BlockRegistry[item]
		if !exists {
			t.Fatalf("Expected item %d to be in BlockRegistry", item)
		}
		if def.Name == "" {
			t.Fatalf("Expected item %d to have non-empty name", item)
		}
		col, row := GetBlockTextureAtlasPos(item, FaceNorth)
		if col < 0 || col >= 16 || row < 0 || row >= 16 {
			t.Fatalf("Expected valid atlas pos for %s, got (%d, %d)", def.Name, col, row)
		}
	}
}

type dummyChunkManager struct{}

func (d *dummyChunkManager) MarkBlockDirty(x, z int) {}

func TestBlockUpdateSystem(t *testing.T) {
	w := NewVoxelWorld()
	dcm := &dummyChunkManager{}

	// 1. Floating sand test:
	// Place sand at (5, 12, 5) with air beneath
	w.SetBlock(5, 12, 5, BlockSand)
	w.SetBlock(5, 11, 5, BlockAir)
	if len(w.ScheduledUpdates) != 0 {
		t.Fatalf("Expected no physics updates queued on initial placement without block update")
	}

	// Trigger block update on neighbor
	w.NotifyNeighbors(5, 11, 5, dcm)
	if len(w.ScheduledUpdates) == 0 {
		t.Fatalf("Expected scheduled sand fall update after neighbor update")
	}

	// Step physics: sand falls down to (5, 11, 5)
	w.TickScheduledPhysics(0.2, dcm, nil)
	if w.GetBlock(5, 11, 5) != BlockSand || w.GetBlock(5, 12, 5) != BlockAir {
		t.Fatalf("Expected sand to fall from (5, 12, 5) to (5, 11, 5), got %v at 12 and %v at 11",
			w.GetBlock(5, 12, 5), w.GetBlock(5, 11, 5))
	}

	// 2. Cactus block update test:
	// Clear 3x3 area around cactus to air
	for dx := -1; dx <= 1; dx++ {
		for dz := -1; dz <= 1; dz++ {
			w.SetBlock(10+dx, 11, 10+dz, BlockAir)
		}
	}
	// Cactus on sand is stable
	w.SetBlock(10, 10, 10, BlockSand)
	w.SetBlock(10, 11, 10, BlockCactus)
	w.TriggerBlockUpdate(10, 11, 10, dcm)
	if w.GetBlock(10, 11, 10) != BlockCactus {
		t.Fatalf("Expected cactus on sand to remain stable, got %v", w.GetBlock(10, 11, 10))
	}

	// Placing solid block next to cactus causes cactus to pop off!
	w.SetBlock(11, 11, 10, BlockStone)
	w.NotifyNeighbors(11, 11, 10, dcm)
	if w.GetBlock(10, 11, 10) != BlockAir {
		t.Fatalf("Expected cactus to break when solid block is placed adjacent to it, got %v", w.GetBlock(10, 11, 10))
	}
}



