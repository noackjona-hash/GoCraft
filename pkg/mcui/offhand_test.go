package mcui

import (
	"testing"

	"gocraft/pkg/voxel"
)

func TestOffhandSwapAndMinimap(t *testing.T) {
	gui := &InventoryGUI{
		SelectedSlot: 0,
		Minimap:      NewMinimap(),
	}

	// 1. Initial State
	if gui.GetOffhandBlock() != voxel.BlockAir {
		t.Fatalf("Expected empty offhand, got %v", gui.GetOffhandBlock())
	}

	// 2. Put Torch in Slot 0 and Swap
	gui.HotbarSlots[0] = ItemStack{Type: voxel.BlockTorch, Count: 16}
	gui.SwapOffhand()

	if gui.GetOffhandBlock() != voxel.BlockTorch {
		t.Fatalf("Expected BlockTorch in offhand after swap, got %v", gui.GetOffhandBlock())
	}
	if gui.HotbarSlots[0].Type != voxel.BlockAir {
		t.Fatalf("Expected slot 0 to be empty after swap, got %v", gui.HotbarSlots[0].Type)
	}

	// 3. Consume 1 item from offhand
	gui.ConsumeOffhandItem()
	if gui.OffhandSlot.Count != 15 {
		t.Fatalf("Expected 15 torches left in offhand, got %d", gui.OffhandSlot.Count)
	}

	// 4. Swap back
	gui.SwapOffhand()
	if gui.HotbarSlots[0].Type != voxel.BlockTorch || gui.HotbarSlots[0].Count != 15 {
		t.Fatalf("Expected slot 0 to have 15 torches, got %v", gui.HotbarSlots[0])
	}
	if gui.GetOffhandBlock() != voxel.BlockAir {
		t.Fatalf("Expected empty offhand after swap back, got %v", gui.GetOffhandBlock())
	}

	// 5. Minimap toggle
	if !gui.Minimap.IsVisible {
		t.Fatalf("Expected minimap to be visible by default")
	}
	gui.Minimap.Toggle()
	if gui.Minimap.IsVisible {
		t.Fatalf("Expected minimap to be hidden after toggle")
	}
	gui.Minimap.Toggle()
	if !gui.Minimap.IsVisible {
		t.Fatalf("Expected minimap to be visible after second toggle")
	}
}

func TestNewItemCraftingAndSmelting(t *testing.T) {
	// 1. Bow Crafting Test (3x3 grid)
	grid := []ItemStack{
		{Type: voxel.BlockAir, Count: 0}, {Type: voxel.ItemStick, Count: 1}, {Type: voxel.ItemString, Count: 1},
		{Type: voxel.ItemStick, Count: 1}, {Type: voxel.BlockAir, Count: 0}, {Type: voxel.ItemString, Count: 1},
		{Type: voxel.BlockAir, Count: 0}, {Type: voxel.ItemStick, Count: 1}, {Type: voxel.ItemString, Count: 1},
	}
	output, _ := MatchCraftingGrid(grid, 3, 3)
	if output.Type != voxel.ItemBow {
		t.Fatalf("Expected ItemBow output, got %v", output.Type)
	}

	// 2. Shield Crafting Test
	shieldGrid := []ItemStack{
		{Type: voxel.BlockOakPlanks, Count: 1}, {Type: voxel.ItemIronIngot, Count: 1}, {Type: voxel.BlockOakPlanks, Count: 1},
		{Type: voxel.BlockOakPlanks, Count: 1}, {Type: voxel.BlockOakPlanks, Count: 1}, {Type: voxel.BlockOakPlanks, Count: 1},
		{Type: voxel.BlockAir, Count: 0}, {Type: voxel.BlockOakPlanks, Count: 1}, {Type: voxel.BlockAir, Count: 0},
	}
	output, _ = MatchCraftingGrid(shieldGrid, 3, 3)
	if output.Type != voxel.ItemShield {
		t.Fatalf("Expected ItemShield output, got %v", output.Type)
	}

	// 3. Smelting Test
	smeltRes, ok := GetSmeltingResult(voxel.ItemPotato)
	if !ok || smeltRes != voxel.ItemBakedPotato {
		t.Fatalf("Expected Potato -> Baked Potato, got %v", smeltRes)
	}
	smeltRes, ok = GetSmeltingResult(voxel.ItemRawChicken)
	if !ok || smeltRes != voxel.ItemCookedChicken {
		t.Fatalf("Expected Raw Chicken -> Cooked Chicken, got %v", smeltRes)
	}
	burnTime := GetFuelBurnTime(voxel.ItemBlazeRod)
	if burnTime < 100.0 {
		t.Fatalf("Expected Blaze Rod to have long burn time, got %.1f", burnTime)
	}
}

