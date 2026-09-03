package voxel

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// BlockType defines the voxel material or inventory item
type BlockType uint8

const (
	BlockAir BlockType = iota
	BlockGrass
	BlockDirt
	BlockStone
	BlockCobblestone
	BlockBedrock
	BlockOakLog
	BlockOakPlanks
	BlockOakLeaves
	BlockBirchLog
	BlockBirchLeaves
	BlockGlass
	BlockSand
	BlockSandstone
	BlockWater
	BlockWaterFlowing
	BlockCoalOre
	BlockIronOre
	BlockGoldOre
	BlockDiamondOre
	BlockBricks
	BlockTNT
	BlockCraftingTable
	BlockTorch
	BlockRedstoneOre
	BlockEmeraldOre
	BlockLapisOre
	BlockFurnace
	BlockBookshelf
	BlockMossyCobblestone
	BlockWool
	BlockObsidian

	// Spruce Wood (Taiga)
	BlockSpruceLog
	BlockSprucePlanks
	BlockSpruceLeaves

	// Rotated Log Orientations (Horizontal X and Z axes)
	BlockOakLogX
	BlockOakLogZ
	BlockBirchLogX
	BlockBirchLogZ
	BlockSpruceLogX
	BlockSpruceLogZ

	// Wildflowers
	BlockDandelion
	BlockPoppy
	BlockCornflower
	BlockAllium

	// Surface Foliage & Desert Plants
	BlockTallGrass
	BlockDeadBush
	BlockCactus
	BlockSugarCane

	// Nature & Special Blocks
	BlockPumpkin
	BlockRedMushroom
	BlockBrownMushroom

	// Terrain & Mineral Deposits
	BlockGravel
	BlockClay
	BlockSnow

	// Items & Materials
	ItemStick
	ItemDiamond
	ItemIronIngot
	ItemGoldIngot
	ItemCoal

	// Food & Mob Drops
	ItemRawBeef
	ItemCookedBeef
	ItemRawPorkchop
	ItemCookedPorkchop
	ItemApple
	ItemBread
	ItemRottenFlesh
	ItemGunpowder
	ItemBone
	ItemArrow

	// Wooden Tools
	ItemWoodPickaxe
	ItemWoodAxe
	ItemWoodShovel
	ItemWoodSword

	// Stone Tools
	ItemStonePickaxe
	ItemStoneAxe
	ItemStoneShovel
	ItemStoneSword

	// Iron Tools
	ItemIronPickaxe
	ItemIronAxe
	ItemIronShovel
	ItemIronSword

	// Diamond Tools
	ItemDiamondPickaxe
	ItemDiamondAxe
	ItemDiamondShovel
	ItemDiamondSword

	// Buckets
	ItemBucket
	ItemWaterBucket
)

// BlockDef contains attributes, face colors, and tool mechanics
type BlockDef struct {
	Type             BlockType
	Name             string
	TopColor         rl.Color
	SideColor        rl.Color
	BottomColor      rl.Color
	IsSolid          bool
	IsTransparent    bool
	IsLiquid         bool
	IsLightSource    bool
	LightLevel       uint8   // Emitted light level (0 to 15)
	Hardness         float32 // Mining time
	DropItem         BlockType
	RequiredTool     string  // "pickaxe", "axe", "shovel", "sword", or ""
	RequiredTier     int     // Minimum tool tier to drop: 0=none/fist, 1=wood, 2=stone, 3=iron, 4=diamond
	IsTool           bool
	ToolType         string  // "pickaxe", "axe", "shovel", "sword"
	ToolTier         int     // Tool tier: 1=wood, 2=stone, 3=iron, 4=diamond
	MaxDurability    int     // Durability uses: 59, 131, 250, 1561
	MiningEfficiency float32 // Speed multiplier (2.0 to 8.5)
	AttackDamage     float32
	IsFood           bool
	FoodPoints       float32 // Hunger restored (2 to 8)
	Saturation       float32
}

// IsPlant returns true if block is rendered as crossed quads foliage
func IsPlant(b BlockType) bool {
	return b == BlockDandelion || b == BlockPoppy || b == BlockCornflower || b == BlockAllium ||
		b == BlockTallGrass || b == BlockDeadBush || b == BlockSugarCane ||
		b == BlockRedMushroom || b == BlockBrownMushroom
}

// IsWater returns true for still water source or flowing water
func IsWater(b BlockType) bool {
	return b == BlockWater || b == BlockWaterFlowing
}

// IsLiquid returns true for any fluid block
func IsLiquid(b BlockType) bool {
	return b == BlockWater || b == BlockWaterFlowing
}

// IsLeaf returns true for any tree leaves
func IsLeaf(b BlockType) bool {
	return b == BlockOakLeaves || b == BlockBirchLeaves || b == BlockSpruceLeaves
}

// IsLog returns true for any tree log (vertical or rotated)
func IsLog(b BlockType) bool {
	return b == BlockOakLog || b == BlockOakLogX || b == BlockOakLogZ ||
		b == BlockBirchLog || b == BlockBirchLogX || b == BlockBirchLogZ ||
		b == BlockSpruceLog || b == BlockSpruceLogX || b == BlockSpruceLogZ
}

// GetBaseLog returns the standard vertical log block type
func GetBaseLog(b BlockType) BlockType {
	switch b {
	case BlockOakLog, BlockOakLogX, BlockOakLogZ:
		return BlockOakLog
	case BlockBirchLog, BlockBirchLogX, BlockBirchLogZ:
		return BlockBirchLog
	case BlockSpruceLog, BlockSpruceLogX, BlockSpruceLogZ:
		return BlockSpruceLog
	default:
		return b
	}
}

// GetRotatedLogBlock determines the log orientation based on placement face normal
func GetRotatedLogBlock(baseLog BlockType, norm rl.Vector3) BlockType {
	base := GetBaseLog(baseLog)
	absX := math.Abs(float64(norm.X))
	absY := math.Abs(float64(norm.Y))
	absZ := math.Abs(float64(norm.Z))

	if absX > absY && absX > absZ {
		switch base {
		case BlockOakLog:
			return BlockOakLogX
		case BlockBirchLog:
			return BlockBirchLogX
		case BlockSpruceLog:
			return BlockSpruceLogX
		}
	} else if absZ > absY && absZ > absX {
		switch base {
		case BlockOakLog:
			return BlockOakLogZ
		case BlockBirchLog:
			return BlockBirchLogZ
		case BlockSpruceLog:
			return BlockSpruceLogZ
		}
	}

	return base
}

// CycleBlockRotation cycles a rotatable block (Logs Y -> X -> Z -> Y)
func CycleBlockRotation(b BlockType) BlockType {
	switch b {
	case BlockOakLog:
		return BlockOakLogX
	case BlockOakLogX:
		return BlockOakLogZ
	case BlockOakLogZ:
		return BlockOakLog

	case BlockBirchLog:
		return BlockBirchLogX
	case BlockBirchLogX:
		return BlockBirchLogZ
	case BlockBirchLogZ:
		return BlockBirchLog

	case BlockSpruceLog:
		return BlockSpruceLogX
	case BlockSpruceLogX:
		return BlockSpruceLogZ
	case BlockSpruceLogZ:
		return BlockSpruceLog

	default:
		return b
	}
}

// GetBlockDrop returns the item dropped when a block is mined
func GetBlockDrop(b BlockType) BlockType {
	if IsLog(b) {
		return GetBaseLog(b)
	}
	if def, exists := BlockRegistry[b]; exists && def.DropItem != BlockAir {
		return def.DropItem
	}
	if IsLeaf(b) || b == BlockGlass || b == BlockWater || b == BlockBedrock {
		if b == BlockOakLeaves && rand.Float32() < 0.05 {
			return ItemApple
		}
		if rand.Float32() < 0.08 {
			return ItemStick
		}
		return BlockAir
	}
	if b == BlockTallGrass {
		return BlockAir // Tall grass occasionally drops seeds in vanilla, or air
	}
	return b // Default: drops itself
}

// GetMiningSpeedAndHarvest returns the mining duration in seconds and whether the block will drop items.
func GetMiningSpeedAndHarvest(blockType, heldItem BlockType) (float32, bool) {
	bDef, bExists := BlockRegistry[blockType]
	if !bExists {
		return 1.0, true
	}

	hardness := bDef.Hardness
	if hardness <= 0.08 {
		return 0.05, true // Instant break (torches, flowers, plants)
	}
	if hardness >= 1000.0 || hardness < 0 {
		return 999999.0, false // Bedrock is indestructible
	}

	heldDef, heldExists := BlockRegistry[heldItem]
	isCorrectTool := heldExists && heldDef.IsTool && heldDef.ToolType == bDef.RequiredTool && bDef.RequiredTool != ""

	if isCorrectTool {
		speed := heldDef.MiningEfficiency
		if speed <= 1.0 {
			speed = 2.0
		}

		if heldDef.ToolTier >= bDef.RequiredTier {
			return (hardness * 1.5) / speed, true
		} else {
			// Correct tool type (e.g. pickaxe), but too low tier (e.g. wooden pickaxe on diamond ore) -> no drop!
			return (hardness * 5.0) / speed, false
		}
	}

	// Wrong tool or bare hand:
	// Base time is hardness * 5.0 (slow penalty)
	// Can only harvest if required tier is 0 (wood, dirt, sand, etc.)
	canHarvest := (bDef.RequiredTier == 0)
	return hardness * 5.0, canHarvest
}

// GetLightOpacity returns how much light is subtracted when passing through the block.
func GetLightOpacity(b BlockType) uint8 {
	if b == BlockAir {
		return 1
	}
	def, exists := BlockRegistry[b]
	if !exists || (def.IsSolid && !def.IsTransparent) {
		return 15 // Completely opaque
	}
	if IsWater(b) {
		return 2 // Water absorbs 2 light levels per block
	}
	if IsLeaf(b) {
		return 2 // Leaves absorb 2 light levels per block
	}
	// Glass, Torches, Flowers, Grass, etc.
	return 1
}

// BlockRegistry holds all registered block & item definitions
var BlockRegistry = map[BlockType]BlockDef{
	BlockAir: {
		Type:          BlockAir,
		Name:          "Air",
		IsSolid:       false,
		IsTransparent: true,
	},
	BlockGrass: {
		Type:         BlockGrass,
		Name:         "Grass Block",
		TopColor:     rl.NewColor(92, 168, 56, 255),
		SideColor:    rl.NewColor(134, 96, 67, 255),
		BottomColor:  rl.NewColor(122, 85, 58, 255),
		IsSolid:      true,
		Hardness:     0.6,
		DropItem:     BlockDirt, // Grass drops Dirt!
		RequiredTool: "shovel",
		RequiredTier: 0,
	},
	BlockDirt: {
		Type:         BlockDirt,
		Name:         "Dirt",
		TopColor:     rl.NewColor(134, 96, 67, 255),
		SideColor:    rl.NewColor(134, 96, 67, 255),
		BottomColor:  rl.NewColor(134, 96, 67, 255),
		IsSolid:      true,
		Hardness:     0.5,
		DropItem:     BlockDirt,
		RequiredTool: "shovel",
		RequiredTier: 0,
	},
	BlockStone: {
		Type:         BlockStone,
		Name:         "Stone",
		TopColor:     rl.NewColor(125, 125, 125, 255),
		SideColor:    rl.NewColor(120, 120, 120, 255),
		BottomColor:  rl.NewColor(115, 115, 115, 255),
		IsSolid:      true,
		Hardness:     1.5,
		DropItem:     BlockCobblestone, // Stone drops Cobblestone!
		RequiredTool: "pickaxe",
		RequiredTier: 1, // Requires Wooden Pickaxe or better
	},
	BlockCobblestone: {
		Type:         BlockCobblestone,
		Name:         "Cobblestone",
		TopColor:     rl.NewColor(105, 105, 105, 255),
		SideColor:    rl.NewColor(95, 95, 95, 255),
		BottomColor:  rl.NewColor(90, 90, 90, 255),
		IsSolid:      true,
		Hardness:     2.0,
		DropItem:     BlockCobblestone,
		RequiredTool: "pickaxe",
		RequiredTier: 1,
	},
	BlockMossyCobblestone: {
		Type:         BlockMossyCobblestone,
		Name:         "Mossy Cobblestone",
		TopColor:     rl.NewColor(85, 115, 85, 255),
		SideColor:    rl.NewColor(75, 105, 75, 255),
		BottomColor:  rl.NewColor(70, 95, 70, 255),
		IsSolid:      true,
		Hardness:     2.0,
		DropItem:     BlockMossyCobblestone,
		RequiredTool: "pickaxe",
		RequiredTier: 1,
	},
	BlockBedrock: {
		Type:        BlockBedrock,
		Name:        "Bedrock",
		TopColor:    rl.NewColor(45, 45, 45, 255),
		SideColor:   rl.NewColor(40, 40, 40, 255),
		BottomColor: rl.NewColor(35, 35, 35, 255),
		IsSolid:     true,
		Hardness:    -1.0, // Indestructible
	},
	BlockOakLog: {
		Type:        BlockOakLog,
		Name:        "Oak Wood Log",
		TopColor:    rl.NewColor(168, 134, 88, 255),
		SideColor:   rl.NewColor(103, 82, 49, 255),
		BottomColor: rl.NewColor(168, 134, 88, 255),
		IsSolid:     true,
		Hardness:    1.8,
	},
	BlockOakLogX: {
		Type:        BlockOakLogX,
		Name:        "Oak Wood Log",
		TopColor:    rl.NewColor(103, 82, 49, 255),
		SideColor:   rl.NewColor(168, 134, 88, 255),
		BottomColor: rl.NewColor(103, 82, 49, 255),
		IsSolid:     true,
		Hardness:    1.8,
		DropItem:    BlockOakLog,
	},
	BlockOakLogZ: {
		Type:        BlockOakLogZ,
		Name:        "Oak Wood Log",
		TopColor:    rl.NewColor(103, 82, 49, 255),
		SideColor:   rl.NewColor(168, 134, 88, 255),
		BottomColor: rl.NewColor(103, 82, 49, 255),
		IsSolid:     true,
		Hardness:    1.8,
		DropItem:    BlockOakLog,
	},
	BlockOakPlanks: {
		Type:        BlockOakPlanks,
		Name:        "Oak Planks",
		TopColor:    rl.NewColor(162, 130, 78, 255),
		SideColor:   rl.NewColor(155, 124, 73, 255),
		BottomColor: rl.NewColor(150, 118, 68, 255),
		IsSolid:     true,
		Hardness:    1.2,
	},
	BlockOakLeaves: {
		Type:          BlockOakLeaves,
		Name:          "Oak Leaves",
		TopColor:      rl.NewColor(58, 120, 32, 220),
		SideColor:     rl.NewColor(52, 112, 28, 220),
		BottomColor:   rl.NewColor(48, 105, 25, 220),
		IsSolid:       true,
		IsTransparent: true,
		Hardness:      0.25,
	},
	BlockBirchLog: {
		Type:        BlockBirchLog,
		Name:        "Birch Log",
		TopColor:    rl.NewColor(190, 180, 150, 255),
		SideColor:   rl.NewColor(220, 225, 215, 255),
		BottomColor: rl.NewColor(190, 180, 150, 255),
		IsSolid:     true,
		Hardness:    2.0,
	},
	BlockBirchLogX: {
		Type:        BlockBirchLogX,
		Name:        "Birch Log",
		TopColor:    rl.NewColor(220, 225, 215, 255),
		SideColor:   rl.NewColor(190, 180, 150, 255),
		BottomColor: rl.NewColor(220, 225, 215, 255),
		IsSolid:     true,
		Hardness:    2.0,
		DropItem:    BlockBirchLog,
	},
	BlockBirchLogZ: {
		Type:        BlockBirchLogZ,
		Name:        "Birch Log",
		TopColor:    rl.NewColor(220, 225, 215, 255),
		SideColor:   rl.NewColor(190, 180, 150, 255),
		BottomColor: rl.NewColor(220, 225, 215, 255),
		IsSolid:     true,
		Hardness:    2.0,
		DropItem:    BlockBirchLog,
	},
	BlockBirchLeaves: {
		Type:          BlockBirchLeaves,
		Name:          "Birch Leaves",
		TopColor:      rl.NewColor(100, 150, 70, 255),
		SideColor:     rl.NewColor(100, 150, 70, 255),
		BottomColor:   rl.NewColor(100, 150, 70, 255),
		IsSolid:       true,
		IsTransparent: true,
		Hardness:      0.2,
	},
	BlockGlass: {
		Type:          BlockGlass,
		Name:          "Glass",
		TopColor:      rl.NewColor(200, 235, 255, 90),
		SideColor:     rl.NewColor(190, 225, 255, 90),
		BottomColor:   rl.NewColor(190, 225, 255, 90),
		IsSolid:       true,
		IsTransparent: true,
		Hardness:      0.3,
	},
	BlockSand: {
		Type:         BlockSand,
		Name:         "Sand",
		TopColor:     rl.NewColor(218, 204, 150, 255),
		SideColor:    rl.NewColor(210, 195, 140, 255),
		BottomColor:  rl.NewColor(200, 185, 130, 255),
		IsSolid:      true,
		Hardness:     0.5,
		RequiredTool: "shovel",
		RequiredTier: 0,
	},
	BlockSandstone: {
		Type:         BlockSandstone,
		Name:         "Sandstone",
		TopColor:     rl.NewColor(225, 215, 168, 255),
		SideColor:    rl.NewColor(212, 198, 148, 255),
		BottomColor:  rl.NewColor(198, 185, 136, 255),
		IsSolid:      true,
		Hardness:     1.4,
		RequiredTool: "pickaxe",
		RequiredTier: 1,
	},
	BlockWater: {
		Type:          BlockWater,
		Name:          "Water Source",
		TopColor:      rl.NewColor(44, 115, 235, 210),
		SideColor:     rl.NewColor(40, 105, 220, 210),
		BottomColor:   rl.NewColor(35, 95, 205, 210),
		IsSolid:       false,
		IsTransparent: true,
		IsLiquid:      true,
		Hardness:      100.0,
		DropItem:      BlockAir,
	},
	BlockWaterFlowing: {
		Type:          BlockWaterFlowing,
		Name:          "Flowing Water",
		TopColor:      rl.NewColor(44, 115, 235, 195),
		SideColor:     rl.NewColor(40, 105, 220, 195),
		BottomColor:   rl.NewColor(35, 95, 205, 195),
		IsSolid:       false,
		IsTransparent: true,
		IsLiquid:      true,
		Hardness:      100.0,
		DropItem:      BlockAir,
	},
	BlockCoalOre: {
		Type:         BlockCoalOre,
		Name:         "Coal Ore",
		TopColor:     rl.NewColor(115, 115, 115, 255),
		SideColor:    rl.NewColor(110, 110, 110, 255),
		BottomColor:  rl.NewColor(105, 105, 105, 255),
		IsSolid:      true,
		Hardness:     3.0,
		DropItem:     ItemCoal, // Coal Ore drops Coal item!
		RequiredTool: "pickaxe",
		RequiredTier: 1, // Wooden Pickaxe or better
	},
	BlockIronOre: {
		Type:         BlockIronOre,
		Name:         "Iron Ore",
		TopColor:     rl.NewColor(125, 120, 115, 255),
		SideColor:    rl.NewColor(120, 115, 110, 255),
		BottomColor:  rl.NewColor(115, 110, 105, 255),
		IsSolid:      true,
		Hardness:     3.0,
		DropItem:     BlockIronOre, // Iron Ore drops raw Iron Ore (smelted in furnace into Iron Ingot)
		RequiredTool: "pickaxe",
		RequiredTier: 2, // Requires Stone Pickaxe or better!
	},
	BlockGoldOre: {
		Type:         BlockGoldOre,
		Name:         "Gold Ore",
		TopColor:     rl.NewColor(130, 125, 115, 255),
		SideColor:    rl.NewColor(125, 120, 110, 255),
		BottomColor:  rl.NewColor(120, 115, 105, 255),
		IsSolid:      true,
		Hardness:     3.0,
		DropItem:     BlockGoldOre, // Gold Ore drops raw Gold Ore (smelted in furnace into Gold Ingot)
		RequiredTool: "pickaxe",
		RequiredTier: 3, // Requires Iron Pickaxe or better!
	},
	BlockDiamondOre: {
		Type:         BlockDiamondOre,
		Name:         "Diamond Ore",
		TopColor:     rl.NewColor(120, 130, 135, 255),
		SideColor:    rl.NewColor(115, 125, 130, 255),
		BottomColor:  rl.NewColor(110, 120, 125, 255),
		IsSolid:      true,
		Hardness:     3.0,
		DropItem:     ItemDiamond, // Diamond Ore drops sparkling Diamond!
		RequiredTool: "pickaxe",
		RequiredTier: 3, // Requires Iron Pickaxe or better!
	},
	BlockRedstoneOre: {
		Type:          BlockRedstoneOre,
		Name:          "Redstone Ore",
		TopColor:      rl.NewColor(125, 115, 115, 255),
		SideColor:     rl.NewColor(120, 110, 110, 255),
		BottomColor:   rl.NewColor(115, 105, 105, 255),
		IsSolid:       true,
		IsLightSource: true,
		LightLevel:    9,
		Hardness:      3.0,
		RequiredTool:  "pickaxe",
		RequiredTier:  3, // Requires Iron Pickaxe or better!
	},
	BlockEmeraldOre: {
		Type:         BlockEmeraldOre,
		Name:         "Emerald Ore",
		TopColor:     rl.NewColor(115, 130, 120, 255),
		SideColor:    rl.NewColor(110, 125, 115, 255),
		BottomColor:  rl.NewColor(105, 120, 110, 255),
		IsSolid:      true,
		Hardness:     3.0,
		RequiredTool: "pickaxe",
		RequiredTier: 3, // Requires Iron Pickaxe or better!
	},
	BlockLapisOre: {
		Type:         BlockLapisOre,
		Name:         "Lapis Lazuli Ore",
		TopColor:     rl.NewColor(115, 120, 135, 255),
		SideColor:    rl.NewColor(110, 115, 130, 255),
		BottomColor:  rl.NewColor(105, 110, 125, 255),
		IsSolid:      true,
		Hardness:     3.0,
		RequiredTool: "pickaxe",
		RequiredTier: 2, // Requires Stone Pickaxe or better!
	},
	BlockBricks: {
		Type:         BlockBricks,
		Name:         "Bricks",
		TopColor:     rl.NewColor(155, 78, 62, 255),
		SideColor:    rl.NewColor(148, 72, 56, 255),
		BottomColor:  rl.NewColor(140, 68, 52, 255),
		IsSolid:      true,
		Hardness:     2.0,
		RequiredTool: "pickaxe",
		RequiredTier: 1,
	},
	BlockTNT: {
		Type:        BlockTNT,
		Name:        "TNT",
		TopColor:    rl.NewColor(185, 45, 35, 255),
		SideColor:   rl.NewColor(210, 50, 40, 255),
		BottomColor: rl.NewColor(165, 40, 30, 255),
		IsSolid:     true,
		Hardness:    0.1,
	},
	BlockCraftingTable: {
		Type:        BlockCraftingTable,
		Name:        "Crafting Table",
		TopColor:    rl.NewColor(182, 142, 88, 255),
		SideColor:   rl.NewColor(145, 105, 65, 255),
		BottomColor: rl.NewColor(135, 95, 58, 255),
		IsSolid:     true,
		Hardness:    1.5,
	},
	BlockTorch: {
		Type:          BlockTorch,
		Name:          "Torch",
		TopColor:      rl.NewColor(255, 215, 60, 255),
		SideColor:     rl.NewColor(145, 105, 55, 255),
		BottomColor:   rl.NewColor(115, 80, 40, 255),
		IsSolid:       false,
		IsTransparent: true,
		IsLightSource: true,
		LightLevel:    14,
		Hardness:      0.05,
	},
	BlockFurnace: {
		Type:          BlockFurnace,
		Name:          "Furnace",
		TopColor:      rl.NewColor(125, 125, 125, 255),
		SideColor:     rl.NewColor(105, 105, 105, 255),
		BottomColor:   rl.NewColor(100, 100, 100, 255),
		IsSolid:       true,
		IsLightSource: true,
		LightLevel:    13,
		Hardness:      3.5,
		RequiredTool:  "pickaxe",
		RequiredTier:  1,
	},
	BlockBookshelf: {
		Type:         BlockBookshelf,
		Name:         "Bookshelf",
		TopColor:     rl.NewColor(162, 130, 78, 255),
		SideColor:    rl.NewColor(145, 110, 68, 255),
		BottomColor:  rl.NewColor(150, 118, 68, 255),
		IsSolid:      true,
		Hardness:     1.5,
		DropItem:     BlockOakPlanks, // Drops planks
		RequiredTool: "axe",
		RequiredTier: 0,
	},

	// --- SPRUCE WOOD ---
	BlockSpruceLog: {
		Type:        BlockSpruceLog,
		Name:        "Spruce Log",
		TopColor:    rl.NewColor(110, 85, 55, 255),
		SideColor:   rl.NewColor(60, 45, 30, 255),
		BottomColor: rl.NewColor(110, 85, 55, 255),
		IsSolid:     true,
		Hardness:    2.0,
	},
	BlockSpruceLogX: {
		Type:        BlockSpruceLogX,
		Name:        "Spruce Log",
		TopColor:    rl.NewColor(60, 45, 30, 255),
		SideColor:   rl.NewColor(110, 85, 55, 255),
		BottomColor: rl.NewColor(60, 45, 30, 255),
		IsSolid:     true,
		Hardness:    2.0,
		DropItem:    BlockSpruceLog,
	},
	BlockSpruceLogZ: {
		Type:        BlockSpruceLogZ,
		Name:        "Spruce Log",
		TopColor:    rl.NewColor(60, 45, 30, 255),
		SideColor:   rl.NewColor(110, 85, 55, 255),
		BottomColor: rl.NewColor(60, 45, 30, 255),
		IsSolid:     true,
		Hardness:    2.0,
		DropItem:    BlockSpruceLog,
	},
	BlockSprucePlanks: {
		Type:        BlockSprucePlanks,
		Name:        "Spruce Planks",
		TopColor:    rl.NewColor(115, 85, 55, 255),
		SideColor:   rl.NewColor(110, 80, 50, 255),
		BottomColor: rl.NewColor(105, 75, 45, 255),
		IsSolid:     true,
		Hardness:    1.2,
	},
	BlockSpruceLeaves: {
		Type:          BlockSpruceLeaves,
		Name:          "Spruce Leaves",
		TopColor:      rl.NewColor(50, 85, 50, 255),
		SideColor:     rl.NewColor(45, 80, 45, 255),
		BottomColor:   rl.NewColor(40, 75, 40, 255),
		IsSolid:       true,
		IsTransparent: true,
		Hardness:      0.2,
	},

	// --- WILDFLOWERS ---
	BlockDandelion: {
		Type:          BlockDandelion,
		Name:          "Dandelion",
		TopColor:      rl.NewColor(255, 240, 50, 255),
		SideColor:     rl.NewColor(255, 240, 50, 255),
		BottomColor:   rl.NewColor(255, 240, 50, 255),
		IsSolid:       false,
		IsTransparent: true,
		Hardness:      0.05,
	},
	BlockPoppy: {
		Type:          BlockPoppy,
		Name:          "Poppy",
		TopColor:      rl.NewColor(230, 40, 40, 255),
		SideColor:     rl.NewColor(230, 40, 40, 255),
		BottomColor:   rl.NewColor(230, 40, 40, 255),
		IsSolid:       false,
		IsTransparent: true,
		Hardness:      0.05,
	},
	BlockCornflower: {
		Type:          BlockCornflower,
		Name:          "Cornflower",
		TopColor:      rl.NewColor(70, 130, 240, 255),
		SideColor:     rl.NewColor(70, 130, 240, 255),
		BottomColor:   rl.NewColor(70, 130, 240, 255),
		IsSolid:       false,
		IsTransparent: true,
		Hardness:      0.05,
	},
	BlockAllium: {
		Type:          BlockAllium,
		Name:          "Allium",
		TopColor:      rl.NewColor(180, 130, 210, 255),
		SideColor:     rl.NewColor(180, 130, 210, 255),
		BottomColor:   rl.NewColor(180, 130, 210, 255),
		IsSolid:       false,
		IsTransparent: true,
		Hardness:      0.05,
	},

	// --- FOLIAGE & DESERT PLANTS ---
	BlockTallGrass: {
		Type:          BlockTallGrass,
		Name:          "Grass",
		TopColor:      rl.NewColor(92, 168, 56, 255),
		SideColor:     rl.NewColor(92, 168, 56, 255),
		BottomColor:   rl.NewColor(92, 168, 56, 255),
		IsSolid:       false,
		IsTransparent: true,
		Hardness:      0.05,
	},
	BlockDeadBush: {
		Type:          BlockDeadBush,
		Name:          "Dead Bush",
		TopColor:      rl.NewColor(145, 110, 65, 255),
		SideColor:     rl.NewColor(145, 110, 65, 255),
		BottomColor:   rl.NewColor(145, 110, 65, 255),
		IsSolid:       false,
		IsTransparent: true,
		Hardness:      0.05,
	},
	BlockCactus: {
		Type:        BlockCactus,
		Name:        "Cactus",
		TopColor:    rl.NewColor(65, 125, 45, 255),
		SideColor:   rl.NewColor(55, 115, 38, 255),
		BottomColor: rl.NewColor(45, 105, 30, 255),
		IsSolid:     true,
		Hardness:    0.4,
	},
	BlockSugarCane: {
		Type:          BlockSugarCane,
		Name:          "Sugar Cane",
		TopColor:      rl.NewColor(135, 185, 75, 255),
		SideColor:     rl.NewColor(135, 185, 75, 255),
		BottomColor:   rl.NewColor(135, 185, 75, 255),
		IsSolid:       false,
		IsTransparent: true,
		Hardness:      0.05,
	},

	// --- NATURE & SPECIALS ---
	BlockPumpkin: {
		Type:        BlockPumpkin,
		Name:        "Pumpkin",
		TopColor:    rl.NewColor(215, 135, 35, 255),
		SideColor:   rl.NewColor(205, 125, 28, 255),
		BottomColor: rl.NewColor(195, 115, 20, 255),
		IsSolid:     true,
		Hardness:    1.0,
	},
	BlockRedMushroom: {
		Type:          BlockRedMushroom,
		Name:          "Red Mushroom",
		TopColor:      rl.NewColor(230, 50, 45, 255),
		SideColor:     rl.NewColor(230, 50, 45, 255),
		BottomColor:   rl.NewColor(230, 50, 45, 255),
		IsSolid:       false,
		IsTransparent: true,
		Hardness:      0.05,
	},
	BlockBrownMushroom: {
		Type:          BlockBrownMushroom,
		Name:          "Brown Mushroom",
		TopColor:      rl.NewColor(165, 125, 90, 255),
		SideColor:     rl.NewColor(165, 125, 90, 255),
		BottomColor:   rl.NewColor(165, 125, 90, 255),
		IsSolid:       false,
		IsTransparent: true,
		Hardness:      0.05,
	},

	// --- TERRAIN & MINERAL DEPOSITS ---
	BlockGravel: {
		Type:        BlockGravel,
		Name:        "Gravel",
		TopColor:    rl.NewColor(135, 130, 130, 255),
		SideColor:   rl.NewColor(130, 125, 125, 255),
		BottomColor: rl.NewColor(125, 120, 120, 255),
		IsSolid:     true,
		Hardness:    0.6,
	},
	BlockClay: {
		Type:        BlockClay,
		Name:        "Clay",
		TopColor:    rl.NewColor(160, 165, 175, 255),
		SideColor:   rl.NewColor(155, 160, 170, 255),
		BottomColor: rl.NewColor(150, 155, 165, 255),
		IsSolid:     true,
		Hardness:    0.6,
	},
	BlockSnow: {
		Type:        BlockSnow,
		Name:        "Snow Block",
		TopColor:    rl.NewColor(245, 250, 255, 255),
		SideColor:   rl.NewColor(240, 245, 250, 255),
		BottomColor: rl.NewColor(235, 240, 245, 255),
		IsSolid:     true,
		Hardness:    0.3,
	},

	// --- BASIC ITEMS ---
	ItemStick: {
		Type:      ItemStick,
		Name:      "Stick",
		TopColor:  rl.NewColor(135, 95, 50, 255),
		SideColor: rl.NewColor(135, 95, 50, 255),
		IsSolid:   false,
	},
	ItemDiamond: {
		Type:      ItemDiamond,
		Name:      "Diamond",
		TopColor:  rl.NewColor(90, 235, 245, 255),
		SideColor: rl.NewColor(80, 220, 235, 255),
		IsSolid:   false,
	},
	ItemIronIngot: {
		Type:      ItemIronIngot,
		Name:      "Iron Ingot",
		TopColor:  rl.NewColor(220, 220, 220, 255),
		SideColor: rl.NewColor(200, 200, 200, 255),
		IsSolid:   false,
	},
	ItemGoldIngot: {
		Type:      ItemGoldIngot,
		Name:      "Gold Ingot",
		TopColor:  rl.NewColor(255, 215, 40, 255),
		SideColor: rl.NewColor(235, 195, 30, 255),
		IsSolid:   false,
	},
	ItemCoal: {
		Type:      ItemCoal,
		Name:      "Coal",
		TopColor:  rl.NewColor(35, 35, 35, 255),
		SideColor: rl.NewColor(25, 25, 25, 255),
		IsSolid:   false,
	},

	// --- WOODEN TOOLS ---
	ItemWoodPickaxe: {
		Type:             ItemWoodPickaxe,
		Name:             "Wooden Pickaxe",
		TopColor:         rl.NewColor(150, 115, 65, 255),
		SideColor:        rl.NewColor(140, 105, 55, 255),
		IsTool:           true,
		ToolType:         "pickaxe",
		ToolTier:         1,
		MaxDurability:    59,
		MiningEfficiency: 2.0,
		AttackDamage:     2.0,
	},
	ItemWoodAxe: {
		Type:             ItemWoodAxe,
		Name:             "Wooden Axe",
		TopColor:         rl.NewColor(150, 115, 65, 255),
		SideColor:        rl.NewColor(140, 105, 55, 255),
		IsTool:           true,
		ToolType:         "axe",
		ToolTier:         1,
		MaxDurability:    59,
		MiningEfficiency: 2.0,
		AttackDamage:     3.0,
	},
	ItemWoodShovel: {
		Type:             ItemWoodShovel,
		Name:             "Wooden Shovel",
		TopColor:         rl.NewColor(150, 115, 65, 255),
		SideColor:        rl.NewColor(140, 105, 55, 255),
		IsTool:           true,
		ToolType:         "shovel",
		ToolTier:         1,
		MaxDurability:    59,
		MiningEfficiency: 2.0,
		AttackDamage:     1.5,
	},
	ItemWoodSword: {
		Type:             ItemWoodSword,
		Name:             "Wooden Sword",
		TopColor:         rl.NewColor(150, 115, 65, 255),
		SideColor:        rl.NewColor(140, 105, 55, 255),
		IsTool:           true,
		ToolType:         "sword",
		ToolTier:         1,
		MaxDurability:    59,
		MiningEfficiency: 1.5,
		AttackDamage:     4.0,
	},

	// --- STONE TOOLS ---
	ItemStonePickaxe: {
		Type:             ItemStonePickaxe,
		Name:             "Stone Pickaxe",
		TopColor:         rl.NewColor(130, 130, 130, 255),
		SideColor:        rl.NewColor(115, 115, 115, 255),
		IsTool:           true,
		ToolType:         "pickaxe",
		ToolTier:         2,
		MaxDurability:    131,
		MiningEfficiency: 4.0,
		AttackDamage:     3.0,
	},
	ItemStoneAxe: {
		Type:             ItemStoneAxe,
		Name:             "Stone Axe",
		TopColor:         rl.NewColor(130, 130, 130, 255),
		SideColor:        rl.NewColor(115, 115, 115, 255),
		IsTool:           true,
		ToolType:         "axe",
		ToolTier:         2,
		MaxDurability:    131,
		MiningEfficiency: 4.0,
		AttackDamage:     4.0,
	},
	ItemStoneShovel: {
		Type:             ItemStoneShovel,
		Name:             "Stone Shovel",
		TopColor:         rl.NewColor(130, 130, 130, 255),
		SideColor:        rl.NewColor(115, 115, 115, 255),
		IsTool:           true,
		ToolType:         "shovel",
		ToolTier:         2,
		MaxDurability:    131,
		MiningEfficiency: 4.0,
		AttackDamage:     2.5,
	},
	ItemStoneSword: {
		Type:             ItemStoneSword,
		Name:             "Stone Sword",
		TopColor:         rl.NewColor(130, 130, 130, 255),
		SideColor:        rl.NewColor(115, 115, 115, 255),
		IsTool:           true,
		ToolType:         "sword",
		ToolTier:         2,
		MaxDurability:    131,
		MiningEfficiency: 2.0,
		AttackDamage:     5.0,
	},

	// --- IRON TOOLS ---
	ItemIronPickaxe: {
		Type:             ItemIronPickaxe,
		Name:             "Iron Pickaxe",
		TopColor:         rl.NewColor(230, 230, 230, 255),
		SideColor:        rl.NewColor(205, 205, 205, 255),
		IsTool:           true,
		ToolType:         "pickaxe",
		ToolTier:         3,
		MaxDurability:    250,
		MiningEfficiency: 6.0,
		AttackDamage:     4.0,
	},
	ItemIronAxe: {
		Type:             ItemIronAxe,
		Name:             "Iron Axe",
		TopColor:         rl.NewColor(230, 230, 230, 255),
		SideColor:        rl.NewColor(205, 205, 205, 255),
		IsTool:           true,
		ToolType:         "axe",
		ToolTier:         3,
		MaxDurability:    250,
		MiningEfficiency: 6.0,
		AttackDamage:     5.0,
	},
	ItemIronShovel: {
		Type:             ItemIronShovel,
		Name:             "Iron Shovel",
		TopColor:         rl.NewColor(230, 230, 230, 255),
		SideColor:        rl.NewColor(205, 205, 205, 255),
		IsTool:           true,
		ToolType:         "shovel",
		ToolTier:         3,
		MaxDurability:    250,
		MiningEfficiency: 6.0,
		AttackDamage:     3.5,
	},
	ItemIronSword: {
		Type:             ItemIronSword,
		Name:             "Iron Sword",
		TopColor:         rl.NewColor(230, 230, 230, 255),
		SideColor:        rl.NewColor(205, 205, 205, 255),
		IsTool:           true,
		ToolType:         "sword",
		ToolTier:         3,
		MaxDurability:    250,
		MiningEfficiency: 2.5,
		AttackDamage:     6.0,
	},

	// --- DIAMOND TOOLS ---
	ItemDiamondPickaxe: {
		Type:             ItemDiamondPickaxe,
		Name:             "Diamond Pickaxe",
		TopColor:         rl.NewColor(95, 240, 250, 255),
		SideColor:        rl.NewColor(80, 220, 230, 255),
		IsTool:           true,
		ToolType:         "pickaxe",
		ToolTier:         4,
		MaxDurability:    1561,
		MiningEfficiency: 8.5,
		AttackDamage:     5.0,
	},
	ItemDiamondAxe: {
		Type:             ItemDiamondAxe,
		Name:             "Diamond Axe",
		TopColor:         rl.NewColor(95, 240, 250, 255),
		SideColor:        rl.NewColor(80, 220, 230, 255),
		IsTool:           true,
		ToolType:         "axe",
		ToolTier:         4,
		MaxDurability:    1561,
		MiningEfficiency: 8.5,
		AttackDamage:     6.0,
	},
	ItemDiamondShovel: {
		Type:             ItemDiamondShovel,
		Name:             "Diamond Shovel",
		TopColor:         rl.NewColor(95, 240, 250, 255),
		SideColor:        rl.NewColor(80, 220, 230, 255),
		IsTool:           true,
		ToolType:         "shovel",
		ToolTier:         4,
		MaxDurability:    1561,
		MiningEfficiency: 8.5,
		AttackDamage:     4.5,
	},
	ItemDiamondSword: {
		Type:             ItemDiamondSword,
		Name:             "Diamond Sword",
		TopColor:         rl.NewColor(95, 240, 250, 255),
		SideColor:        rl.NewColor(80, 220, 230, 255),
		IsTool:           true,
		ToolType:         "sword",
		ToolTier:         4,
		MaxDurability:    1561,
		MiningEfficiency: 3.0,
		AttackDamage:     7.0,
	},

	// --- NEW BLOCKS ---
	BlockWool: {
		Type:        BlockWool,
		Name:        "White Wool",
		TopColor:    rl.NewColor(235, 235, 235, 255),
		SideColor:   rl.NewColor(225, 225, 225, 255),
		BottomColor: rl.NewColor(225, 225, 225, 255),
		IsSolid:     true,
		Hardness:    0.8,
	},
	BlockObsidian: {
		Type:         BlockObsidian,
		Name:         "Obsidian",
		TopColor:     rl.NewColor(20, 15, 30, 255),
		SideColor:    rl.NewColor(15, 10, 25, 255),
		BottomColor:  rl.NewColor(15, 10, 25, 255),
		IsSolid:      true,
		Hardness:     50.0,
		DropItem:     BlockObsidian,
		RequiredTool: "pickaxe",
		RequiredTier: 4, // Requires Diamond Pickaxe!
	},

	// --- FOOD ITEMS ---
	ItemRawBeef: {
		Type:        ItemRawBeef,
		Name:        "Raw Beef",
		TopColor:    rl.NewColor(185, 45, 45, 255),
		SideColor:   rl.NewColor(160, 35, 35, 255),
		IsFood:      true,
		FoodPoints:  3.0,
		Saturation:  1.8,
	},
	ItemCookedBeef: {
		Type:        ItemCookedBeef,
		Name:        "Steak",
		TopColor:    rl.NewColor(115, 60, 40, 255),
		SideColor:   rl.NewColor(95, 45, 30, 255),
		IsFood:      true,
		FoodPoints:  8.0,
		Saturation:  12.8,
	},
	ItemRawPorkchop: {
		Type:        ItemRawPorkchop,
		Name:        "Raw Porkchop",
		TopColor:    rl.NewColor(235, 130, 130, 255),
		SideColor:   rl.NewColor(215, 110, 110, 255),
		IsFood:      true,
		FoodPoints:  3.0,
		Saturation:  1.8,
	},
	ItemCookedPorkchop: {
		Type:        ItemCookedPorkchop,
		Name:        "Cooked Porkchop",
		TopColor:    rl.NewColor(150, 90, 60, 255),
		SideColor:   rl.NewColor(130, 75, 45, 255),
		IsFood:      true,
		FoodPoints:  8.0,
		Saturation:  12.8,
	},
	ItemApple: {
		Type:        ItemApple,
		Name:        "Apple",
		TopColor:    rl.NewColor(220, 30, 30, 255),
		SideColor:   rl.NewColor(200, 20, 20, 255),
		IsFood:      true,
		FoodPoints:  4.0,
		Saturation:  2.4,
	},
	ItemBread: {
		Type:        ItemBread,
		Name:        "Bread",
		TopColor:    rl.NewColor(190, 130, 60, 255),
		SideColor:   rl.NewColor(170, 110, 45, 255),
		IsFood:      true,
		FoodPoints:  5.0,
		Saturation:  6.0,
	},
	ItemRottenFlesh: {
		Type:        ItemRottenFlesh,
		Name:        "Rotten Flesh",
		TopColor:    rl.NewColor(140, 80, 50, 255),
		SideColor:   rl.NewColor(110, 60, 35, 255),
		IsFood:      true,
		FoodPoints:  2.0,
		Saturation:  0.8,
	},

	// --- MOB DROPS & ITEMS ---
	ItemGunpowder: {
		Type:      ItemGunpowder,
		Name:      "Gunpowder",
		TopColor:  rl.NewColor(110, 110, 110, 255),
		SideColor: rl.NewColor(90, 90, 90, 255),
	},
	ItemBone: {
		Type:      ItemBone,
		Name:      "Bone",
		TopColor:  rl.NewColor(230, 230, 220, 255),
		SideColor: rl.NewColor(210, 210, 200, 255),
	},
	ItemArrow: {
		Type:      ItemArrow,
		Name:      "Arrow",
		TopColor:  rl.NewColor(160, 150, 140, 255),
		SideColor: rl.NewColor(140, 130, 120, 255),
	},
	ItemBucket: {
		Type:      ItemBucket,
		Name:      "Bucket",
		TopColor:  rl.NewColor(220, 220, 220, 255),
		SideColor: rl.NewColor(180, 180, 180, 255),
		IsSolid:   false,
		Hardness:  0.1,
	},
	ItemWaterBucket: {
		Type:      ItemWaterBucket,
		Name:      "Water Bucket",
		TopColor:  rl.NewColor(60, 130, 240, 255),
		SideColor: rl.NewColor(180, 180, 180, 255),
		IsSolid:   false,
		Hardness:  0.1,
	},
}
