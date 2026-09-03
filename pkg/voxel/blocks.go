package voxel

import (
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
	IsLightSource    bool
	LightLevel       uint8   // Emitted light level (0 to 15)
	Hardness         float32 // Mining time
	DropItem         BlockType
	IsTool           bool
	ToolType         string  // "pickaxe", "axe", "shovel", "sword"
	MiningEfficiency float32 // Speed multiplier (2.0 to 8.0)
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

// IsLeaf returns true for any tree leaves
func IsLeaf(b BlockType) bool {
	return b == BlockOakLeaves || b == BlockBirchLeaves || b == BlockSpruceLeaves
}

// IsLog returns true for any tree log
func IsLog(b BlockType) bool {
	return b == BlockOakLog || b == BlockBirchLog || b == BlockSpruceLog
}

// GetBlockDrop returns the item dropped when a block is mined
func GetBlockDrop(b BlockType) BlockType {
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

// GetLightOpacity returns how much light is subtracted when passing through the block.
func GetLightOpacity(b BlockType) uint8 {
	if b == BlockAir {
		return 1
	}
	def, exists := BlockRegistry[b]
	if !exists || (def.IsSolid && !def.IsTransparent) {
		return 15 // Completely opaque
	}
	if b == BlockWater {
		return 3 // Water absorbs 3 light levels per block
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
		Type:        BlockGrass,
		Name:        "Grass Block",
		TopColor:    rl.NewColor(92, 168, 56, 255),
		SideColor:   rl.NewColor(134, 96, 67, 255),
		BottomColor: rl.NewColor(122, 85, 58, 255),
		IsSolid:     true,
		Hardness:    0.6,
		DropItem:    BlockDirt, // Grass drops Dirt!
	},
	BlockDirt: {
		Type:        BlockDirt,
		Name:        "Dirt",
		TopColor:    rl.NewColor(134, 96, 67, 255),
		SideColor:   rl.NewColor(134, 96, 67, 255),
		BottomColor: rl.NewColor(134, 96, 67, 255),
		IsSolid:     true,
		Hardness:    0.5,
		DropItem:    BlockDirt,
	},
	BlockStone: {
		Type:        BlockStone,
		Name:        "Stone",
		TopColor:    rl.NewColor(125, 125, 125, 255),
		SideColor:   rl.NewColor(120, 120, 120, 255),
		BottomColor: rl.NewColor(115, 115, 115, 255),
		IsSolid:     true,
		Hardness:    2.5,
		DropItem:    BlockCobblestone, // Stone drops Cobblestone!
	},
	BlockCobblestone: {
		Type:        BlockCobblestone,
		Name:        "Cobblestone",
		TopColor:    rl.NewColor(105, 105, 105, 255),
		SideColor:   rl.NewColor(95, 95, 95, 255),
		BottomColor: rl.NewColor(90, 90, 90, 255),
		IsSolid:     true,
		Hardness:    2.0,
		DropItem:    BlockCobblestone,
	},
	BlockMossyCobblestone: {
		Type:        BlockMossyCobblestone,
		Name:        "Mossy Cobblestone",
		TopColor:    rl.NewColor(85, 115, 85, 255),
		SideColor:   rl.NewColor(75, 105, 75, 255),
		BottomColor: rl.NewColor(70, 95, 70, 255),
		IsSolid:     true,
		Hardness:    2.0,
		DropItem:    BlockMossyCobblestone,
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
		Type:        BlockSand,
		Name:        "Sand",
		TopColor:    rl.NewColor(218, 204, 150, 255),
		SideColor:   rl.NewColor(210, 195, 140, 255),
		BottomColor: rl.NewColor(200, 185, 130, 255),
		IsSolid:     true,
		Hardness:    0.5,
	},
	BlockSandstone: {
		Type:        BlockSandstone,
		Name:        "Sandstone",
		TopColor:    rl.NewColor(225, 215, 168, 255),
		SideColor:   rl.NewColor(212, 198, 148, 255),
		BottomColor: rl.NewColor(198, 185, 136, 255),
		IsSolid:     true,
		Hardness:    1.4,
	},
	BlockWater: {
		Type:          BlockWater,
		Name:          "Water",
		TopColor:      rl.NewColor(35, 110, 220, 175),
		SideColor:     rl.NewColor(30, 95, 200, 175),
		BottomColor:   rl.NewColor(25, 80, 180, 175),
		IsSolid:       false,
		IsTransparent: true,
		Hardness:      100.0,
	},
	BlockCoalOre: {
		Type:        BlockCoalOre,
		Name:        "Coal Ore",
		TopColor:    rl.NewColor(115, 115, 115, 255),
		SideColor:   rl.NewColor(110, 110, 110, 255),
		BottomColor: rl.NewColor(105, 105, 105, 255),
		IsSolid:     true,
		Hardness:    2.5,
		DropItem:    ItemCoal, // Coal Ore drops Coal item!
	},
	BlockIronOre: {
		Type:        BlockIronOre,
		Name:        "Iron Ore",
		TopColor:    rl.NewColor(125, 120, 115, 255),
		SideColor:   rl.NewColor(120, 115, 110, 255),
		BottomColor: rl.NewColor(115, 110, 105, 255),
		IsSolid:     true,
		Hardness:    3.0,
		DropItem:    BlockIronOre, // Iron Ore drops raw Iron Ore (smelted in furnace into Iron Ingot)
	},
	BlockGoldOre: {
		Type:        BlockGoldOre,
		Name:        "Gold Ore",
		TopColor:    rl.NewColor(130, 125, 115, 255),
		SideColor:   rl.NewColor(125, 120, 110, 255),
		BottomColor: rl.NewColor(120, 115, 105, 255),
		IsSolid:     true,
		Hardness:    3.0,
		DropItem:    BlockGoldOre, // Gold Ore drops raw Gold Ore (smelted in furnace into Gold Ingot)
	},
	BlockDiamondOre: {
		Type:        BlockDiamondOre,
		Name:        "Diamond Ore",
		TopColor:    rl.NewColor(120, 130, 135, 255),
		SideColor:   rl.NewColor(115, 125, 130, 255),
		BottomColor: rl.NewColor(110, 120, 125, 255),
		IsSolid:     true,
		Hardness:    3.5,
		DropItem:    ItemDiamond, // Diamond Ore drops sparkling Diamond!
	},
	BlockRedstoneOre: {
		Type:        BlockRedstoneOre,
		Name:        "Redstone Ore",
		TopColor:    rl.NewColor(125, 115, 115, 255),
		SideColor:   rl.NewColor(120, 110, 110, 255),
		BottomColor: rl.NewColor(115, 105, 105, 255),
		IsSolid:     true,
		IsLightSource: true,
		LightLevel:  9,
		Hardness:    3.0,
	},
	BlockEmeraldOre: {
		Type:        BlockEmeraldOre,
		Name:        "Emerald Ore",
		TopColor:    rl.NewColor(115, 130, 120, 255),
		SideColor:   rl.NewColor(110, 125, 115, 255),
		BottomColor: rl.NewColor(105, 120, 110, 255),
		IsSolid:     true,
		Hardness:    3.5,
	},
	BlockLapisOre: {
		Type:        BlockLapisOre,
		Name:        "Lapis Lazuli Ore",
		TopColor:    rl.NewColor(115, 120, 135, 255),
		SideColor:   rl.NewColor(110, 115, 130, 255),
		BottomColor: rl.NewColor(105, 110, 125, 255),
		IsSolid:     true,
		Hardness:    3.0,
	},
	BlockBricks: {
		Type:        BlockBricks,
		Name:        "Bricks",
		TopColor:    rl.NewColor(155, 78, 62, 255),
		SideColor:   rl.NewColor(148, 72, 56, 255),
		BottomColor: rl.NewColor(140, 68, 52, 255),
		IsSolid:     true,
		Hardness:    2.0,
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
		Type:        BlockFurnace,
		Name:        "Furnace",
		TopColor:    rl.NewColor(125, 125, 125, 255),
		SideColor:   rl.NewColor(105, 105, 105, 255),
		BottomColor: rl.NewColor(100, 100, 100, 255),
		IsSolid:     true,
		IsLightSource: true,
		LightLevel:  13,
		Hardness:    2.5,
	},
	BlockBookshelf: {
		Type:        BlockBookshelf,
		Name:        "Bookshelf",
		TopColor:    rl.NewColor(162, 130, 78, 255),
		SideColor:   rl.NewColor(145, 110, 68, 255),
		BottomColor: rl.NewColor(150, 118, 68, 255),
		IsSolid:     true,
		Hardness:    1.5,
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
		Type:        BlockObsidian,
		Name:        "Obsidian",
		TopColor:    rl.NewColor(20, 15, 30, 255),
		SideColor:   rl.NewColor(15, 10, 25, 255),
		BottomColor: rl.NewColor(15, 10, 25, 255),
		IsSolid:     true,
		Hardness:    9.0,
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
}
