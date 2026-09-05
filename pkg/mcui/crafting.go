package mcui

import "gocraft/pkg/voxel"

// ItemStack represents a stack of items in an inventory slot
type ItemStack struct {
	Type          voxel.BlockType
	Count         int
	Durability    int // Current remaining durability (0 = unused or non-tool)
	MaxDurability int // Maximum durability (59 for wood, 131 for stone, 250 for iron, 1561 for diamond)
}

// Recipe represents a 2x2 or 3x3 crafting pattern
type Recipe struct {
	Width   int
	Height  int
	Pattern []voxel.BlockType
	Output  ItemStack
}

// Global recipe registry
var CraftingRecipes []Recipe

func init() {
	CraftingRecipes = []Recipe{
		// 1. 1 Log -> 4 Planks (1x1)
		{
			Width:   1,
			Height:  1,
			Pattern: []voxel.BlockType{voxel.BlockOakLog},
			Output:  ItemStack{Type: voxel.BlockOakPlanks, Count: 4},
		},
		{
			Width:   1,
			Height:  1,
			Pattern: []voxel.BlockType{voxel.BlockBirchLog},
			Output:  ItemStack{Type: voxel.BlockOakPlanks, Count: 4},
		},
		{
			Width:   1,
			Height:  1,
			Pattern: []voxel.BlockType{voxel.BlockSpruceLog},
			Output:  ItemStack{Type: voxel.BlockSprucePlanks, Count: 4},
		},
		{
			Width:  1,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.BlockSprucePlanks,
				voxel.BlockSprucePlanks,
			},
			Output: ItemStack{Type: voxel.ItemStick, Count: 4},
		},
		{
			Width:  2,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.BlockSprucePlanks, voxel.BlockSprucePlanks,
				voxel.BlockSprucePlanks, voxel.BlockSprucePlanks,
			},
			Output: ItemStack{Type: voxel.BlockCraftingTable, Count: 1},
		},
		// 2. 2 Planks (1x2 vertical) -> 4 Sticks
		{
			Width:  1,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.BlockOakPlanks,
				voxel.BlockOakPlanks,
			},
			Output: ItemStack{Type: voxel.ItemStick, Count: 4},
		},
		// 3. 4 Planks (2x2) -> 1 Crafting Table
		{
			Width:  2,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.BlockOakPlanks, voxel.BlockOakPlanks,
				voxel.BlockOakPlanks, voxel.BlockOakPlanks,
			},
			Output: ItemStack{Type: voxel.BlockCraftingTable, Count: 1},
		},
		// 4. 1 Coal + 1 Stick -> 4 Torches
		{
			Width:  1,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.BlockCoalOre,
				voxel.ItemStick,
			},
			Output: ItemStack{Type: voxel.BlockTorch, Count: 4},
		},
		{
			Width:  1,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.ItemCoal,
				voxel.ItemStick,
			},
			Output: ItemStack{Type: voxel.BlockTorch, Count: 4},
		},
		// 5. 4 Sand (2x2) -> 4 Sandstone
		{
			Width:  2,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.BlockSand, voxel.BlockSand,
				voxel.BlockSand, voxel.BlockSand,
			},
			Output: ItemStack{Type: voxel.BlockSandstone, Count: 4},
		},
		// 6. 1 Cobble + 1 Leaves -> 1 Mossy Cobblestone
		{
			Width:  1,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.BlockOakLeaves,
				voxel.BlockCobblestone,
			},
			Output: ItemStack{Type: voxel.BlockMossyCobblestone, Count: 1},
		},

		// --- WOODEN TOOLS ---
		// Wooden Pickaxe (3 Planks + 2 Sticks)
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockOakPlanks, voxel.BlockOakPlanks, voxel.BlockOakPlanks,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemWoodPickaxe, Count: 1},
		},
		// Wooden Axe (3 Planks + 2 Sticks)
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockOakPlanks, voxel.BlockOakPlanks, voxel.BlockAir,
				voxel.BlockOakPlanks, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemWoodAxe, Count: 1},
		},
		// Wooden Shovel (1 Plank + 2 Sticks)
		{
			Width:  1,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockOakPlanks,
				voxel.ItemStick,
				voxel.ItemStick,
			},
			Output: ItemStack{Type: voxel.ItemWoodShovel, Count: 1},
		},
		// Wooden Sword (2 Planks + 1 Stick)
		{
			Width:  1,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockOakPlanks,
				voxel.BlockOakPlanks,
				voxel.ItemStick,
			},
			Output: ItemStack{Type: voxel.ItemWoodSword, Count: 1},
		},

		// --- STONE TOOLS ---
		// Stone Pickaxe (3 Cobblestone + 2 Sticks)
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockCobblestone, voxel.BlockCobblestone, voxel.BlockCobblestone,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemStonePickaxe, Count: 1},
		},
		// Stone Axe (3 Cobblestone + 2 Sticks)
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockCobblestone, voxel.BlockCobblestone, voxel.BlockAir,
				voxel.BlockCobblestone, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemStoneAxe, Count: 1},
		},
		// Stone Shovel (1 Cobblestone + 2 Sticks)
		{
			Width:  1,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockCobblestone,
				voxel.ItemStick,
				voxel.ItemStick,
			},
			Output: ItemStack{Type: voxel.ItemStoneShovel, Count: 1},
		},
		// Stone Sword (2 Cobblestone + 1 Stick)
		{
			Width:  1,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockCobblestone,
				voxel.BlockCobblestone,
				voxel.ItemStick,
			},
			Output: ItemStack{Type: voxel.ItemStoneSword, Count: 1},
		},

		// --- IRON TOOLS ---
		// Iron Pickaxe (3 Iron + 2 Sticks)
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockIronOre, voxel.BlockIronOre, voxel.BlockIronOre,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemIronPickaxe, Count: 1},
		},
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemIronIngot, voxel.ItemIronIngot, voxel.ItemIronIngot,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemIronPickaxe, Count: 1},
		},
		// Iron Axe
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemIronIngot, voxel.ItemIronIngot, voxel.BlockAir,
				voxel.ItemIronIngot, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemIronAxe, Count: 1},
		},
		// Iron Sword
		{
			Width:  1,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemIronIngot,
				voxel.ItemIronIngot,
				voxel.ItemStick,
			},
			Output: ItemStack{Type: voxel.ItemIronSword, Count: 1},
		},

		// Bucket (3 Iron Ingots in V-shape)
		{
			Width:  3,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.ItemIronIngot, voxel.BlockAir, voxel.ItemIronIngot,
				voxel.BlockAir, voxel.ItemIronIngot, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemBucket, Count: 1},
		},

		// --- DIAMOND TOOLS ---
		// Diamond Pickaxe (3 Diamonds + 2 Sticks)
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockDiamondOre, voxel.BlockDiamondOre, voxel.BlockDiamondOre,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemDiamondPickaxe, Count: 1},
		},
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemDiamond, voxel.ItemDiamond, voxel.ItemDiamond,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemDiamondPickaxe, Count: 1},
		},
		// Diamond Sword
		{
			Width:  1,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemDiamond,
				voxel.ItemDiamond,
				voxel.ItemStick,
			},
			Output: ItemStack{Type: voxel.ItemDiamondSword, Count: 1},
		},
		// Diamond Axe
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemDiamond, voxel.ItemDiamond, voxel.BlockAir,
				voxel.ItemDiamond, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemDiamondAxe, Count: 1},
		},

		// 8 Cobblestone -> 1 Furnace
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockCobblestone, voxel.BlockCobblestone, voxel.BlockCobblestone,
				voxel.BlockCobblestone, voxel.BlockAir, voxel.BlockCobblestone,
				voxel.BlockCobblestone, voxel.BlockCobblestone, voxel.BlockCobblestone,
			},
			Output: ItemStack{Type: voxel.BlockFurnace, Count: 1},
		},
		// 6 Planks + 3 Books/Ores -> Bookshelf
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockOakPlanks, voxel.BlockOakPlanks, voxel.BlockOakPlanks,
				voxel.BlockOakLog, voxel.BlockOakLog, voxel.BlockOakLog,
				voxel.BlockOakPlanks, voxel.BlockOakPlanks, voxel.BlockOakPlanks,
			},
			Output: ItemStack{Type: voxel.BlockBookshelf, Count: 1},
		},
		// 4 Redstone / 5 Gunpowder + Sand -> TNT
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockSand, voxel.BlockRedstoneOre, voxel.BlockSand,
				voxel.BlockRedstoneOre, voxel.BlockSand, voxel.BlockRedstoneOre,
				voxel.BlockSand, voxel.BlockRedstoneOre, voxel.BlockSand,
			},
			Output: ItemStack{Type: voxel.BlockTNT, Count: 1},
		},
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemGunpowder, voxel.BlockSand, voxel.ItemGunpowder,
				voxel.BlockSand, voxel.ItemGunpowder, voxel.BlockSand,
				voxel.ItemGunpowder, voxel.BlockSand, voxel.ItemGunpowder,
			},
			Output: ItemStack{Type: voxel.BlockTNT, Count: 1},
		},

		// 22. Bow (3 Sticks + 3 Strings)
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockAir, voxel.ItemStick, voxel.ItemString,
				voxel.ItemStick, voxel.BlockAir, voxel.ItemString,
				voxel.BlockAir, voxel.ItemStick, voxel.ItemString,
			},
			Output: ItemStack{Type: voxel.ItemBow, Count: 1},
		},

		// 23. Shield (1 Iron Ingot + 6 Planks)
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockOakPlanks, voxel.ItemIronIngot, voxel.BlockOakPlanks,
				voxel.BlockOakPlanks, voxel.BlockOakPlanks, voxel.BlockOakPlanks,
				voxel.BlockAir, voxel.BlockOakPlanks, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemShield, Count: 1},
		},

		// 24. Flint and Steel (Iron Ingot + Flint)
		{
			Width:  2,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.ItemIronIngot, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemFlint,
			},
			Output: ItemStack{Type: voxel.ItemFlintAndSteel, Count: 1},
		},

		// 25. Shears (2 Iron Ingots)
		{
			Width:  2,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.BlockAir, voxel.ItemIronIngot,
				voxel.ItemIronIngot, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemShears, Count: 1},
		},

		// 26. Fishing Rod (3 Sticks + 2 Strings)
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockAir, voxel.BlockAir, voxel.ItemStick,
				voxel.BlockAir, voxel.ItemStick, voxel.ItemString,
				voxel.ItemStick, voxel.BlockAir, voxel.ItemString,
			},
			Output: ItemStack{Type: voxel.ItemFishingRod, Count: 1},
		},

		// 27. Compass (4 Iron + 1 Redstone)
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockAir, voxel.ItemIronIngot, voxel.BlockAir,
				voxel.ItemIronIngot, voxel.ItemRedstone, voxel.ItemIronIngot,
				voxel.BlockAir, voxel.ItemIronIngot, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemCompass, Count: 1},
		},

		// 28. Clock (4 Gold + 1 Redstone)
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.BlockAir, voxel.ItemGoldIngot, voxel.BlockAir,
				voxel.ItemGoldIngot, voxel.ItemRedstone, voxel.ItemGoldIngot,
				voxel.BlockAir, voxel.ItemGoldIngot, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemClock, Count: 1},
		},

		// 29. Golden Tools
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemGoldIngot, voxel.ItemGoldIngot, voxel.ItemGoldIngot,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
			Output: ItemStack{Type: voxel.ItemGoldenPickaxe, Count: 1},
		},
		{
			Width:  2,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemGoldIngot, voxel.ItemGoldIngot,
				voxel.ItemGoldIngot, voxel.ItemStick,
				voxel.BlockAir, voxel.ItemStick,
			},
			Output: ItemStack{Type: voxel.ItemGoldenAxe, Count: 1},
		},
		{
			Width:  1,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemGoldIngot,
				voxel.ItemStick,
				voxel.ItemStick,
			},
			Output: ItemStack{Type: voxel.ItemGoldenShovel, Count: 1},
		},
		{
			Width:  1,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemGoldIngot,
				voxel.ItemGoldIngot,
				voxel.ItemStick,
			},
			Output: ItemStack{Type: voxel.ItemGoldenSword, Count: 1},
		},

		// 30. Iron Armor
		{
			Width:  3,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.ItemIronIngot, voxel.ItemIronIngot, voxel.ItemIronIngot,
				voxel.ItemIronIngot, voxel.BlockAir, voxel.ItemIronIngot,
			},
			Output: ItemStack{Type: voxel.ItemIronHelmet, Count: 1},
		},
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemIronIngot, voxel.BlockAir, voxel.ItemIronIngot,
				voxel.ItemIronIngot, voxel.ItemIronIngot, voxel.ItemIronIngot,
				voxel.ItemIronIngot, voxel.ItemIronIngot, voxel.ItemIronIngot,
			},
			Output: ItemStack{Type: voxel.ItemIronChestplate, Count: 1},
		},
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemIronIngot, voxel.ItemIronIngot, voxel.ItemIronIngot,
				voxel.ItemIronIngot, voxel.BlockAir, voxel.ItemIronIngot,
				voxel.ItemIronIngot, voxel.BlockAir, voxel.ItemIronIngot,
			},
			Output: ItemStack{Type: voxel.ItemIronLeggings, Count: 1},
		},
		{
			Width:  3,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.ItemIronIngot, voxel.BlockAir, voxel.ItemIronIngot,
				voxel.ItemIronIngot, voxel.BlockAir, voxel.ItemIronIngot,
			},
			Output: ItemStack{Type: voxel.ItemIronBoots, Count: 1},
		},

		// 31. Diamond Armor
		{
			Width:  3,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.ItemDiamond, voxel.ItemDiamond, voxel.ItemDiamond,
				voxel.ItemDiamond, voxel.BlockAir, voxel.ItemDiamond,
			},
			Output: ItemStack{Type: voxel.ItemDiamondHelmet, Count: 1},
		},
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemDiamond, voxel.BlockAir, voxel.ItemDiamond,
				voxel.ItemDiamond, voxel.ItemDiamond, voxel.ItemDiamond,
				voxel.ItemDiamond, voxel.ItemDiamond, voxel.ItemDiamond,
			},
			Output: ItemStack{Type: voxel.ItemDiamondChestplate, Count: 1},
		},
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemDiamond, voxel.ItemDiamond, voxel.ItemDiamond,
				voxel.ItemDiamond, voxel.BlockAir, voxel.ItemDiamond,
				voxel.ItemDiamond, voxel.BlockAir, voxel.ItemDiamond,
			},
			Output: ItemStack{Type: voxel.ItemDiamondLeggings, Count: 1},
		},
		{
			Width:  3,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.ItemDiamond, voxel.BlockAir, voxel.ItemDiamond,
				voxel.ItemDiamond, voxel.BlockAir, voxel.ItemDiamond,
			},
			Output: ItemStack{Type: voxel.ItemDiamondBoots, Count: 1},
		},

		// 32. Golden Apple (Apple + 8 Gold Ingots)
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemGoldIngot, voxel.ItemGoldIngot, voxel.ItemGoldIngot,
				voxel.ItemGoldIngot, voxel.ItemApple, voxel.ItemGoldIngot,
				voxel.ItemGoldIngot, voxel.ItemGoldIngot, voxel.ItemGoldIngot,
			},
			Output: ItemStack{Type: voxel.ItemGoldenApple, Count: 1},
		},

		// 33. Golden Carrot (Carrot + 8 Gold Nuggets)
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemGoldNugget, voxel.ItemGoldNugget, voxel.ItemGoldNugget,
				voxel.ItemGoldNugget, voxel.ItemCarrot, voxel.ItemGoldNugget,
				voxel.ItemGoldNugget, voxel.ItemGoldNugget, voxel.ItemGoldNugget,
			},
			Output: ItemStack{Type: voxel.ItemGoldenCarrot, Count: 1},
		},

		// 34. Gold Ingot <-> 9 Nuggets
		{
			Width:   1,
			Height:  1,
			Pattern: []voxel.BlockType{voxel.ItemGoldIngot},
			Output:  ItemStack{Type: voxel.ItemGoldNugget, Count: 9},
		},
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemGoldNugget, voxel.ItemGoldNugget, voxel.ItemGoldNugget,
				voxel.ItemGoldNugget, voxel.ItemGoldNugget, voxel.ItemGoldNugget,
				voxel.ItemGoldNugget, voxel.ItemGoldNugget, voxel.ItemGoldNugget,
			},
			Output: ItemStack{Type: voxel.ItemGoldIngot, Count: 1},
		},

		// 35. Iron Ingot <-> 9 Nuggets
		{
			Width:   1,
			Height:  1,
			Pattern: []voxel.BlockType{voxel.ItemIronIngot},
			Output:  ItemStack{Type: voxel.ItemIronNugget, Count: 9},
		},
		{
			Width:  3,
			Height: 3,
			Pattern: []voxel.BlockType{
				voxel.ItemIronNugget, voxel.ItemIronNugget, voxel.ItemIronNugget,
				voxel.ItemIronNugget, voxel.ItemIronNugget, voxel.ItemIronNugget,
				voxel.ItemIronNugget, voxel.ItemIronNugget, voxel.ItemIronNugget,
			},
			Output: ItemStack{Type: voxel.ItemIronIngot, Count: 1},
		},

		// 36. Bread (3 Wheat)
		{
			Width:  3,
			Height: 1,
			Pattern: []voxel.BlockType{
				voxel.ItemWheat, voxel.ItemWheat, voxel.ItemWheat,
			},
			Output: ItemStack{Type: voxel.ItemBread, Count: 1},
		},

		// 37. Cookie (2 Wheat + 1 Cocoa Bean/Brown Dye or Sugar Cane)
		{
			Width:  3,
			Height: 1,
			Pattern: []voxel.BlockType{
				voxel.ItemWheat, voxel.BlockSugarCane, voxel.ItemWheat,
			},
			Output: ItemStack{Type: voxel.ItemCookie, Count: 8},
		},

		// 38. Book (1 Leather + 3 Sugar Cane)
		{
			Width:  2,
			Height: 2,
			Pattern: []voxel.BlockType{
				voxel.BlockSugarCane, voxel.BlockSugarCane,
				voxel.BlockSugarCane, voxel.ItemLeather,
			},
			Output: ItemStack{Type: voxel.ItemBook, Count: 1},
		},
	}
}

// MatchCraftingGrid scans the 2x2 or 3x3 slot array and checks if it matches any recipe
func MatchCraftingGrid(grid []ItemStack, gridW, gridH int) (ItemStack, int) {
	// Find bounding box of non-empty items
	minX, minY := 99, 99
	maxX, maxY := -1, -1
	itemCount := 0

	for y := 0; y < gridH; y++ {
		for x := 0; x < gridW; x++ {
			idx := y*gridW + x
			if idx < len(grid) && grid[idx].Count > 0 && grid[idx].Type != voxel.BlockAir {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
				itemCount++
			}
		}
	}

	if itemCount == 0 {
		return ItemStack{Type: voxel.BlockAir, Count: 0}, -1
	}

	subW := maxX - minX + 1
	subH := maxY - minY + 1

	for rIdx, r := range CraftingRecipes {
		if r.Width != subW || r.Height != subH {
			continue
		}

		match := true
		for ry := 0; ry < r.Height; ry++ {
			for rx := 0; rx < r.Width; rx++ {
				gridIdx := (minY+ry)*gridW + (minX + rx)
				recIdx := ry*r.Width + rx

				gType := voxel.BlockAir
				if gridIdx < len(grid) && grid[gridIdx].Count > 0 {
					gType = grid[gridIdx].Type
				}
				rType := r.Pattern[recIdx]

				if gType != rType {
					match = false
					break
				}
			}
			if !match {
				break
			}
		}

		if match {
			out := r.Output
			if bDef, ok := voxel.BlockRegistry[out.Type]; ok && bDef.IsTool && bDef.MaxDurability > 0 {
				out.Durability = bDef.MaxDurability
				out.MaxDurability = bDef.MaxDurability
			}
			return out, rIdx
		}
	}

	return ItemStack{Type: voxel.BlockAir, Count: 0}, -1
}

// RecipeBookEntry describes a craftable recipe in the Recipe Book
type RecipeBookEntry struct {
	Name        string
	Output      ItemStack
	Is3x3Only   bool
	Ingredients map[voxel.BlockType]int
	Pattern2x2  [4]voxel.BlockType
	Pattern3x3  [9]voxel.BlockType
}

// Global recipe book catalogue
var RecipeBookList []RecipeBookEntry

func init() {
	RecipeBookList = []RecipeBookEntry{
		{
			Name:        "Oak Planks",
			Output:      ItemStack{Type: voxel.BlockOakPlanks, Count: 4},
			Is3x3Only:   false,
			Ingredients: map[voxel.BlockType]int{voxel.BlockOakLog: 1},
			Pattern2x2:  [4]voxel.BlockType{voxel.BlockOakLog, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir},
			Pattern3x3:  [9]voxel.BlockType{voxel.BlockOakLog, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir},
		},
		{
			Name:        "Sticks",
			Output:      ItemStack{Type: voxel.ItemStick, Count: 4},
			Is3x3Only:   false,
			Ingredients: map[voxel.BlockType]int{voxel.BlockOakPlanks: 2},
			Pattern2x2:  [4]voxel.BlockType{voxel.BlockOakPlanks, voxel.BlockAir, voxel.BlockOakPlanks, voxel.BlockAir},
			Pattern3x3:  [9]voxel.BlockType{voxel.BlockOakPlanks, voxel.BlockAir, voxel.BlockAir, voxel.BlockOakPlanks, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir},
		},
		{
			Name:        "Crafting Table",
			Output:      ItemStack{Type: voxel.BlockCraftingTable, Count: 1},
			Is3x3Only:   false,
			Ingredients: map[voxel.BlockType]int{voxel.BlockOakPlanks: 4},
			Pattern2x2:  [4]voxel.BlockType{voxel.BlockOakPlanks, voxel.BlockOakPlanks, voxel.BlockOakPlanks, voxel.BlockOakPlanks},
			Pattern3x3:  [9]voxel.BlockType{voxel.BlockOakPlanks, voxel.BlockOakPlanks, voxel.BlockAir, voxel.BlockOakPlanks, voxel.BlockOakPlanks, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir},
		},
		{
			Name:        "Torches",
			Output:      ItemStack{Type: voxel.BlockTorch, Count: 4},
			Is3x3Only:   false,
			Ingredients: map[voxel.BlockType]int{voxel.ItemCoal: 1, voxel.ItemStick: 1},
			Pattern2x2:  [4]voxel.BlockType{voxel.ItemCoal, voxel.BlockAir, voxel.ItemStick, voxel.BlockAir},
			Pattern3x3:  [9]voxel.BlockType{voxel.ItemCoal, voxel.BlockAir, voxel.BlockAir, voxel.ItemStick, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir},
		},
		{
			Name:        "Sandstone",
			Output:      ItemStack{Type: voxel.BlockSandstone, Count: 4},
			Is3x3Only:   false,
			Ingredients: map[voxel.BlockType]int{voxel.BlockSand: 4},
			Pattern2x2:  [4]voxel.BlockType{voxel.BlockSand, voxel.BlockSand, voxel.BlockSand, voxel.BlockSand},
			Pattern3x3:  [9]voxel.BlockType{voxel.BlockSand, voxel.BlockSand, voxel.BlockAir, voxel.BlockSand, voxel.BlockSand, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir, voxel.BlockAir},
		},
		{
			Name:        "Furnace",
			Output:      ItemStack{Type: voxel.BlockFurnace, Count: 1},
			Is3x3Only:   true,
			Ingredients: map[voxel.BlockType]int{voxel.BlockCobblestone: 8},
			Pattern3x3: [9]voxel.BlockType{
				voxel.BlockCobblestone, voxel.BlockCobblestone, voxel.BlockCobblestone,
				voxel.BlockCobblestone, voxel.BlockAir, voxel.BlockCobblestone,
				voxel.BlockCobblestone, voxel.BlockCobblestone, voxel.BlockCobblestone,
			},
		},
		{
			Name:        "Wooden Pickaxe",
			Output:      ItemStack{Type: voxel.ItemWoodPickaxe, Count: 1},
			Is3x3Only:   true,
			Ingredients: map[voxel.BlockType]int{voxel.BlockOakPlanks: 3, voxel.ItemStick: 2},
			Pattern3x3: [9]voxel.BlockType{
				voxel.BlockOakPlanks, voxel.BlockOakPlanks, voxel.BlockOakPlanks,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
		},
		{
			Name:        "Wooden Axe",
			Output:      ItemStack{Type: voxel.ItemWoodAxe, Count: 1},
			Is3x3Only:   true,
			Ingredients: map[voxel.BlockType]int{voxel.BlockOakPlanks: 3, voxel.ItemStick: 2},
			Pattern3x3: [9]voxel.BlockType{
				voxel.BlockOakPlanks, voxel.BlockOakPlanks, voxel.BlockAir,
				voxel.BlockOakPlanks, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
		},
		{
			Name:        "Wooden Sword",
			Output:      ItemStack{Type: voxel.ItemWoodSword, Count: 1},
			Is3x3Only:   true,
			Ingredients: map[voxel.BlockType]int{voxel.BlockOakPlanks: 2, voxel.ItemStick: 1},
			Pattern3x3: [9]voxel.BlockType{
				voxel.BlockOakPlanks, voxel.BlockAir, voxel.BlockAir,
				voxel.BlockOakPlanks, voxel.BlockAir, voxel.BlockAir,
				voxel.ItemStick, voxel.BlockAir, voxel.BlockAir,
			},
		},
		{
			Name:        "Wooden Shovel",
			Output:      ItemStack{Type: voxel.ItemWoodShovel, Count: 1},
			Is3x3Only:   true,
			Ingredients: map[voxel.BlockType]int{voxel.BlockOakPlanks: 1, voxel.ItemStick: 2},
			Pattern3x3: [9]voxel.BlockType{
				voxel.BlockOakPlanks, voxel.BlockAir, voxel.BlockAir,
				voxel.ItemStick, voxel.BlockAir, voxel.BlockAir,
				voxel.ItemStick, voxel.BlockAir, voxel.BlockAir,
			},
		},
		{
			Name:        "Stone Pickaxe",
			Output:      ItemStack{Type: voxel.ItemStonePickaxe, Count: 1},
			Is3x3Only:   true,
			Ingredients: map[voxel.BlockType]int{voxel.BlockCobblestone: 3, voxel.ItemStick: 2},
			Pattern3x3: [9]voxel.BlockType{
				voxel.BlockCobblestone, voxel.BlockCobblestone, voxel.BlockCobblestone,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
		},
		{
			Name:        "Stone Axe",
			Output:      ItemStack{Type: voxel.ItemStoneAxe, Count: 1},
			Is3x3Only:   true,
			Ingredients: map[voxel.BlockType]int{voxel.BlockCobblestone: 3, voxel.ItemStick: 2},
			Pattern3x3: [9]voxel.BlockType{
				voxel.BlockCobblestone, voxel.BlockCobblestone, voxel.BlockAir,
				voxel.BlockCobblestone, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
		},
		{
			Name:        "Stone Sword",
			Output:      ItemStack{Type: voxel.ItemStoneSword, Count: 1},
			Is3x3Only:   true,
			Ingredients: map[voxel.BlockType]int{voxel.BlockCobblestone: 2, voxel.ItemStick: 1},
			Pattern3x3: [9]voxel.BlockType{
				voxel.BlockCobblestone, voxel.BlockAir, voxel.BlockAir,
				voxel.BlockCobblestone, voxel.BlockAir, voxel.BlockAir,
				voxel.ItemStick, voxel.BlockAir, voxel.BlockAir,
			},
		},
		{
			Name:        "Iron Pickaxe",
			Output:      ItemStack{Type: voxel.ItemIronPickaxe, Count: 1},
			Is3x3Only:   true,
			Ingredients: map[voxel.BlockType]int{voxel.ItemIronIngot: 3, voxel.ItemStick: 2},
			Pattern3x3: [9]voxel.BlockType{
				voxel.ItemIronIngot, voxel.ItemIronIngot, voxel.ItemIronIngot,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
		},
		{
			Name:        "Iron Axe",
			Output:      ItemStack{Type: voxel.ItemIronAxe, Count: 1},
			Is3x3Only:   true,
			Ingredients: map[voxel.BlockType]int{voxel.ItemIronIngot: 3, voxel.ItemStick: 2},
			Pattern3x3: [9]voxel.BlockType{
				voxel.ItemIronIngot, voxel.ItemIronIngot, voxel.BlockAir,
				voxel.ItemIronIngot, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
		},
		{
			Name:        "Iron Sword",
			Output:      ItemStack{Type: voxel.ItemIronSword, Count: 1},
			Is3x3Only:   true,
			Ingredients: map[voxel.BlockType]int{voxel.ItemIronIngot: 2, voxel.ItemStick: 1},
			Pattern3x3: [9]voxel.BlockType{
				voxel.ItemIronIngot, voxel.BlockAir, voxel.BlockAir,
				voxel.ItemIronIngot, voxel.BlockAir, voxel.BlockAir,
				voxel.ItemStick, voxel.BlockAir, voxel.BlockAir,
			},
		},
		{
			Name:        "Diamond Pickaxe",
			Output:      ItemStack{Type: voxel.ItemDiamondPickaxe, Count: 1},
			Is3x3Only:   true,
			Ingredients: map[voxel.BlockType]int{voxel.ItemDiamond: 3, voxel.ItemStick: 2},
			Pattern3x3: [9]voxel.BlockType{
				voxel.ItemDiamond, voxel.ItemDiamond, voxel.ItemDiamond,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
				voxel.BlockAir, voxel.ItemStick, voxel.BlockAir,
			},
		},
		{
			Name:        "Diamond Sword",
			Output:      ItemStack{Type: voxel.ItemDiamondSword, Count: 1},
			Is3x3Only:   true,
			Ingredients: map[voxel.BlockType]int{voxel.ItemDiamond: 2, voxel.ItemStick: 1},
			Pattern3x3: [9]voxel.BlockType{
				voxel.ItemDiamond, voxel.BlockAir, voxel.BlockAir,
				voxel.ItemDiamond, voxel.BlockAir, voxel.BlockAir,
				voxel.ItemStick, voxel.BlockAir, voxel.BlockAir,
			},
		},
		{
			Name:        "Bookshelf",
			Output:      ItemStack{Type: voxel.BlockBookshelf, Count: 1},
			Is3x3Only:   true,
			Ingredients: map[voxel.BlockType]int{voxel.BlockOakPlanks: 6, voxel.BlockOakLog: 3},
			Pattern3x3: [9]voxel.BlockType{
				voxel.BlockOakPlanks, voxel.BlockOakPlanks, voxel.BlockOakPlanks,
				voxel.BlockOakLog, voxel.BlockOakLog, voxel.BlockOakLog,
				voxel.BlockOakPlanks, voxel.BlockOakPlanks, voxel.BlockOakPlanks,
			},
		},
		{
			Name:        "TNT",
			Output:      ItemStack{Type: voxel.BlockTNT, Count: 1},
			Is3x3Only:   true,
			Ingredients: map[voxel.BlockType]int{voxel.BlockSand: 5, voxel.BlockRedstoneOre: 4},
			Pattern3x3: [9]voxel.BlockType{
				voxel.BlockSand, voxel.BlockRedstoneOre, voxel.BlockSand,
				voxel.BlockRedstoneOre, voxel.BlockSand, voxel.BlockRedstoneOre,
				voxel.BlockSand, voxel.BlockRedstoneOre, voxel.BlockSand,
			},
		},
	}
}

// CanCraftEntry checks whether player has required ingredients in hotbar or main inventory
func (entry *RecipeBookEntry) CanCraft(hotbar [9]ItemStack, mainInv [27]ItemStack) bool {
	counts := make(map[voxel.BlockType]int)
	for _, s := range hotbar {
		if s.Count > 0 {
			counts[s.Type] += s.Count
		}
	}
	for _, s := range mainInv {
		if s.Count > 0 {
			counts[s.Type] += s.Count
		}
	}

	for reqType, reqCount := range entry.Ingredients {
		// Special check: CoalOre also works for Coal
		if reqType == voxel.ItemCoal {
			if counts[voxel.ItemCoal]+counts[voxel.BlockCoalOre] < reqCount {
				return false
			}
		} else if counts[reqType] < reqCount {
			return false
		}
	}
	return true
}
