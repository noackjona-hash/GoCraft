package voxel

import (
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

	// Items & Materials
	ItemStick
	ItemDiamond
	ItemIronIngot
	ItemGoldIngot
	ItemCoal

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
	Hardness         float32 // Mining time
	IsTool           bool
	ToolType         string  // "pickaxe", "axe", "shovel", "sword"
	MiningEfficiency float32 // Speed multiplier (2.0 to 8.0)
	AttackDamage     float32
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
	},
	BlockDirt: {
		Type:        BlockDirt,
		Name:        "Dirt",
		TopColor:    rl.NewColor(134, 96, 67, 255),
		SideColor:   rl.NewColor(134, 96, 67, 255),
		BottomColor: rl.NewColor(134, 96, 67, 255),
		IsSolid:     true,
		Hardness:    0.5,
	},
	BlockStone: {
		Type:        BlockStone,
		Name:        "Stone",
		TopColor:    rl.NewColor(125, 125, 125, 255),
		SideColor:   rl.NewColor(120, 120, 120, 255),
		BottomColor: rl.NewColor(115, 115, 115, 255),
		IsSolid:     true,
		Hardness:    2.5,
	},
	BlockCobblestone: {
		Type:        BlockCobblestone,
		Name:        "Cobblestone",
		TopColor:    rl.NewColor(105, 105, 105, 255),
		SideColor:   rl.NewColor(95, 95, 95, 255),
		BottomColor: rl.NewColor(90, 90, 90, 255),
		IsSolid:     true,
		Hardness:    2.0,
	},
	BlockMossyCobblestone: {
		Type:        BlockMossyCobblestone,
		Name:        "Mossy Cobblestone",
		TopColor:    rl.NewColor(85, 115, 85, 255),
		SideColor:   rl.NewColor(75, 105, 75, 255),
		BottomColor: rl.NewColor(70, 95, 70, 255),
		IsSolid:     true,
		Hardness:    2.0,
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
	},
	BlockIronOre: {
		Type:        BlockIronOre,
		Name:        "Iron Ore",
		TopColor:    rl.NewColor(125, 120, 115, 255),
		SideColor:   rl.NewColor(120, 115, 110, 255),
		BottomColor: rl.NewColor(115, 110, 105, 255),
		IsSolid:     true,
		Hardness:    3.0,
	},
	BlockGoldOre: {
		Type:        BlockGoldOre,
		Name:        "Gold Ore",
		TopColor:    rl.NewColor(130, 125, 115, 255),
		SideColor:   rl.NewColor(125, 120, 110, 255),
		BottomColor: rl.NewColor(120, 115, 105, 255),
		IsSolid:     true,
		Hardness:    3.0,
	},
	BlockDiamondOre: {
		Type:        BlockDiamondOre,
		Name:        "Diamond Ore",
		TopColor:    rl.NewColor(120, 130, 135, 255),
		SideColor:   rl.NewColor(115, 125, 130, 255),
		BottomColor: rl.NewColor(110, 120, 125, 255),
		IsSolid:     true,
		Hardness:    3.5,
	},
	BlockRedstoneOre: {
		Type:        BlockRedstoneOre,
		Name:        "Redstone Ore",
		TopColor:    rl.NewColor(125, 115, 115, 255),
		SideColor:   rl.NewColor(120, 110, 110, 255),
		BottomColor: rl.NewColor(115, 105, 105, 255),
		IsSolid:     true,
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
		Hardness:      0.05,
	},
	BlockFurnace: {
		Type:        BlockFurnace,
		Name:        "Furnace",
		TopColor:    rl.NewColor(125, 125, 125, 255),
		SideColor:   rl.NewColor(105, 105, 105, 255),
		BottomColor: rl.NewColor(100, 100, 100, 255),
		IsSolid:     true,
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
}
