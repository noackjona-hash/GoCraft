package mcui

import "racing_game/pkg/voxel"

// ItemStack represents a stack of items in an inventory slot
type ItemStack struct {
	Type  voxel.BlockType
	Count int
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
		// 4 Redstone + 5 Sand -> TNT
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
			return r.Output, rIdx
		}
	}

	return ItemStack{Type: voxel.BlockAir, Count: 0}, -1
}
