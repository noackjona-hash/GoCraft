package mcui

import (
	"fmt"
	"os"

	"gocraft/pkg/mcplayer"
	"gocraft/pkg/voxel"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// InventoryGUI manages the player's hotbar, main inventory, 2x2/3x3 crafting, recipe book, furnace smelting, and death screen
type InventoryGUI struct {
	ScreenWidth  int32
	ScreenHeight int32

	HotbarSlots   [9]ItemStack
	MainInventory [27]ItemStack
	SelectedSlot  int // 0 to 8

	IsInventoryOpen  bool
	IsWorkbenchOpen  bool
	IsFurnaceOpen    bool
	IsRecipeBookOpen bool

	Crafting2x2       [4]ItemStack
	Crafting2x2Output ItemStack

	Crafting3x3       [9]ItemStack
	Crafting3x3Output ItemStack

	// Furnace Smelting State
	FurnaceInput        ItemStack
	FurnaceFuel         ItemStack
	FurnaceOutput       ItemStack
	FurnaceBurnTime     float32
	FurnaceMaxBurnTime  float32
	FurnaceCookProgress float32 // 0.0 to 1.0

	CursorItem ItemStack

	// Authentic Minecraft GUI Textures from Resource Pack
	WidgetsTex   rl.Texture2D
	IconsTex     rl.Texture2D
	InventoryTex rl.Texture2D
	WorkbenchTex rl.Texture2D
	FurnaceTex   rl.Texture2D
	HasGUITex    bool
}

// NewInventoryGUI initializes empty inventory with starter items
func NewInventoryGUI(screenWidth, screenHeight int32) *InventoryGUI {
	gui := &InventoryGUI{
		ScreenWidth:      screenWidth,
		ScreenHeight:     screenHeight,
		SelectedSlot:     0,
		IsRecipeBookOpen: false,
	}

	gui.LoadTextures()

	// Starter items: Wooden Pickaxe, 16 Planks, 12 Torches, 8 Logs, 8 Sticks
	gui.HotbarSlots[0] = ItemStack{Type: voxel.ItemWoodPickaxe, Count: 1}
	gui.HotbarSlots[1] = ItemStack{Type: voxel.BlockOakPlanks, Count: 16}
	gui.HotbarSlots[2] = ItemStack{Type: voxel.BlockTorch, Count: 12}
	gui.HotbarSlots[3] = ItemStack{Type: voxel.BlockOakLog, Count: 8}
	gui.HotbarSlots[4] = ItemStack{Type: voxel.ItemStick, Count: 8}

	gui.UpdateCrafting()
	return gui
}

// LoadTextures loads authentic GUI texture sheets from the resource pack
func (gui *InventoryGUI) LoadTextures() {
	if !gui.HasGUITex {
		if fi, err := os.Stat("assets/textures/gui/widgets.png"); err == nil && !fi.IsDir() {
			gui.WidgetsTex = rl.LoadTexture("assets/textures/gui/widgets.png")
		}
		if fi, err := os.Stat("assets/textures/gui/icons.png"); err == nil && !fi.IsDir() {
			gui.IconsTex = rl.LoadTexture("assets/textures/gui/icons.png")
		}
		if fi, err := os.Stat("assets/textures/gui/container/inventory.png"); err == nil && !fi.IsDir() {
			gui.InventoryTex = rl.LoadTexture("assets/textures/gui/container/inventory.png")
		}
		if fi, err := os.Stat("assets/textures/gui/container/crafting_table.png"); err == nil && !fi.IsDir() {
			gui.WorkbenchTex = rl.LoadTexture("assets/textures/gui/container/crafting_table.png")
		}
		if fi, err := os.Stat("assets/textures/gui/container/furnace.png"); err == nil && !fi.IsDir() {
			gui.FurnaceTex = rl.LoadTexture("assets/textures/gui/container/furnace.png")
		}
		gui.HasGUITex = true
	}
}

// Unload releases GUI textures from GPU memory
func (gui *InventoryGUI) Unload() {
	if gui.WidgetsTex.ID > 0 {
		rl.UnloadTexture(gui.WidgetsTex)
	}
	if gui.IconsTex.ID > 0 {
		rl.UnloadTexture(gui.IconsTex)
	}
	if gui.InventoryTex.ID > 0 {
		rl.UnloadTexture(gui.InventoryTex)
	}
	if gui.WorkbenchTex.ID > 0 {
		rl.UnloadTexture(gui.WorkbenchTex)
	}
	if gui.FurnaceTex.ID > 0 {
		rl.UnloadTexture(gui.FurnaceTex)
	}
	gui.HasGUITex = false
}

// GetActiveBlock returns the block/item currently held in the active hotbar slot
func (gui *InventoryGUI) GetActiveBlock() voxel.BlockType {
	if gui.SelectedSlot >= 0 && gui.SelectedSlot < 9 {
		slot := gui.HotbarSlots[gui.SelectedSlot]
		if slot.Count > 0 {
			return slot.Type
		}
	}
	return voxel.BlockAir
}

// ConsumeActiveItem decrements stack in currently selected hotbar slot
func (gui *InventoryGUI) ConsumeActiveItem() {
	gui.RemoveActiveItem(1)
}

// SetActiveSlotItem sets the item type and count of the currently active hotbar slot
func (gui *InventoryGUI) SetActiveSlotItem(bType voxel.BlockType, count int) {
	if gui.SelectedSlot >= 0 && gui.SelectedSlot < 9 {
		gui.HotbarSlots[gui.SelectedSlot] = ItemStack{Type: bType, Count: count}
	}
}

// RemoveActiveItem removes up to count items from the currently selected hotbar slot and returns the amount actually removed
func (gui *InventoryGUI) RemoveActiveItem(count int) int {
	if gui.SelectedSlot >= 0 && gui.SelectedSlot < 9 {
		slot := &gui.HotbarSlots[gui.SelectedSlot]
		if slot.Count > 0 {
			removed := count
			if slot.Count < count {
				removed = slot.Count
			}
			slot.Count -= removed
			if slot.Count == 0 {
				slot.Type = voxel.BlockAir
			}
			return removed
		}
	}
	return 0
}

// AddItem tries to add count of bType into existing stacks, then empty slots
func (gui *InventoryGUI) AddItem(bType voxel.BlockType, count int) bool {
	if bType == voxel.BlockAir || count <= 0 {
		return false
	}

	// 1. Stack into existing hotbar slots
	for i := 0; i < 9; i++ {
		slot := &gui.HotbarSlots[i]
		if slot.Type == bType && slot.Count < 64 {
			space := 64 - slot.Count
			if count <= space {
				slot.Count += count
				return true
			}
			slot.Count = 64
			count -= space
		}
	}

	// 2. Stack into existing main inventory slots
	for i := 0; i < 27; i++ {
		slot := &gui.MainInventory[i]
		if slot.Type == bType && slot.Count < 64 {
			space := 64 - slot.Count
			if count <= space {
				slot.Count += count
				return true
			}
			slot.Count = 64
			count -= space
		}
	}

	// 3. Put into empty hotbar slot
	for i := 0; i < 9; i++ {
		slot := &gui.HotbarSlots[i]
		if slot.Type == voxel.BlockAir || slot.Count == 0 {
			slot.Type = bType
			slot.Count = count
			return true
		}
	}

	// 4. Put into empty main inventory slot
	for i := 0; i < 27; i++ {
		slot := &gui.MainInventory[i]
		if slot.Type == voxel.BlockAir || slot.Count == 0 {
			slot.Type = bType
			slot.Count = count
			return true
		}
	}

	return false
}

// NextSlot cycles hotbar slot with mouse wheel
func (gui *InventoryGUI) NextSlot() {
	gui.SelectedSlot = (gui.SelectedSlot + 1) % 9
}

// PrevSlot cycles hotbar slot backward
func (gui *InventoryGUI) PrevSlot() {
	gui.SelectedSlot = (gui.SelectedSlot - 1 + 9) % 9
}

// UpdateCrafting recalculates recipe match for 2x2 and 3x3 grids
func (gui *InventoryGUI) UpdateCrafting() {
	out2x2, _ := MatchCraftingGrid(gui.Crafting2x2[:], 2, 2)
	gui.Crafting2x2Output = out2x2

	out3x3, _ := MatchCraftingGrid(gui.Crafting3x3[:], 3, 3)
	gui.Crafting3x3Output = out3x3
}

// CloseMenu closes all open GUIs and safely returns any items in crafting grids back into player inventory
func (gui *InventoryGUI) CloseMenu() {
	gui.IsInventoryOpen = false
	gui.IsWorkbenchOpen = false
	gui.IsFurnaceOpen = false
	gui.IsRecipeBookOpen = false

	// Return items from 2x2 grid to player inventory
	for i := 0; i < 4; i++ {
		if gui.Crafting2x2[i].Count > 0 {
			gui.AddItem(gui.Crafting2x2[i].Type, gui.Crafting2x2[i].Count)
			gui.Crafting2x2[i] = ItemStack{Type: voxel.BlockAir, Count: 0}
		}
	}

	// Return items from 3x3 grid to player inventory
	for i := 0; i < 9; i++ {
		if gui.Crafting3x3[i].Count > 0 {
			gui.AddItem(gui.Crafting3x3[i].Type, gui.Crafting3x3[i].Count)
			gui.Crafting3x3[i] = ItemStack{Type: voxel.BlockAir, Count: 0}
		}
	}

	// Return cursor held item to player inventory
	if gui.CursorItem.Count > 0 && gui.CursorItem.Type != voxel.BlockAir {
		gui.AddItem(gui.CursorItem.Type, gui.CursorItem.Count)
		gui.CursorItem = ItemStack{Type: voxel.BlockAir, Count: 0}
	}

	gui.UpdateCrafting()
}

// AutoFillRecipe clears the crafting grid and auto-transfers ingredients from player inventory
func (gui *InventoryGUI) AutoFillRecipe(entry RecipeBookEntry, isWorkbench bool) {
	// 1. Return any existing crafting items back to player inventory
	if isWorkbench {
		for i := 0; i < 9; i++ {
			if gui.Crafting3x3[i].Count > 0 {
				gui.AddItem(gui.Crafting3x3[i].Type, gui.Crafting3x3[i].Count)
				gui.Crafting3x3[i] = ItemStack{Type: voxel.BlockAir, Count: 0}
			}
		}
	} else {
		for i := 0; i < 4; i++ {
			if gui.Crafting2x2[i].Count > 0 {
				gui.AddItem(gui.Crafting2x2[i].Type, gui.Crafting2x2[i].Count)
				gui.Crafting2x2[i] = ItemStack{Type: voxel.BlockAir, Count: 0}
			}
		}
	}

	// 2. Check if player has ingredients
	if !entry.CanCraft(gui.HotbarSlots, gui.MainInventory) {
		gui.UpdateCrafting()
		return
	}

	// Helper to consume 1 item of bType from inventory
	takeOne := func(bType voxel.BlockType) bool {
		// Try hotbar first
		for i := 0; i < 9; i++ {
			if gui.HotbarSlots[i].Type == bType && gui.HotbarSlots[i].Count > 0 {
				gui.HotbarSlots[i].Count--
				if gui.HotbarSlots[i].Count == 0 {
					gui.HotbarSlots[i].Type = voxel.BlockAir
				}
				return true
			}
		}
		// Try main inventory
		for i := 0; i < 27; i++ {
			if gui.MainInventory[i].Type == bType && gui.MainInventory[i].Count > 0 {
				gui.MainInventory[i].Count--
				if gui.MainInventory[i].Count == 0 {
					gui.MainInventory[i].Type = voxel.BlockAir
				}
				return true
			}
		}
		// Fallback for coal
		if bType == voxel.ItemCoal {
			for i := 0; i < 9; i++ {
				if gui.HotbarSlots[i].Type == voxel.BlockCoalOre && gui.HotbarSlots[i].Count > 0 {
					gui.HotbarSlots[i].Count--
					if gui.HotbarSlots[i].Count == 0 {
						gui.HotbarSlots[i].Type = voxel.BlockAir
					}
					return true
				}
			}
			for i := 0; i < 27; i++ {
				if gui.MainInventory[i].Type == voxel.BlockCoalOre && gui.MainInventory[i].Count > 0 {
					gui.MainInventory[i].Count--
					if gui.MainInventory[i].Count == 0 {
						gui.MainInventory[i].Type = voxel.BlockAir
					}
					return true
				}
			}
		}
		return false
	}

	// 3. Fill grid with recipe pattern
	if isWorkbench {
		for i := 0; i < 9; i++ {
			req := entry.Pattern3x3[i]
			if req != voxel.BlockAir {
				if takeOne(req) {
					gui.Crafting3x3[i] = ItemStack{Type: req, Count: 1}
				}
			}
		}
	} else {
		if entry.Is3x3Only {
			gui.UpdateCrafting()
			return
		}
		for i := 0; i < 4; i++ {
			req := entry.Pattern2x2[i]
			if req != voxel.BlockAir {
				if takeOne(req) {
					gui.Crafting2x2[i] = ItemStack{Type: req, Count: 1}
				}
			}
		}
	}

	gui.UpdateCrafting()
}

// GetSmeltingResult returns the smelted output item for given input
func GetSmeltingResult(input voxel.BlockType) (voxel.BlockType, bool) {
	switch input {
	case voxel.ItemRawPorkchop:
		return voxel.ItemCookedPorkchop, true
	case voxel.ItemRawBeef:
		return voxel.ItemCookedBeef, true
	case voxel.BlockIronOre:
		return voxel.ItemIronIngot, true
	case voxel.BlockGoldOre:
		return voxel.ItemGoldIngot, true
	case voxel.BlockSand:
		return voxel.BlockGlass, true
	case voxel.BlockCobblestone:
		return voxel.BlockStone, true
	case voxel.BlockOakLog:
		return voxel.ItemCoal, true // Charcoal
	default:
		return voxel.BlockAir, false
	}
}

// GetFuelBurnTime returns duration in seconds a fuel item burns
func GetFuelBurnTime(fuel voxel.BlockType) float32 {
	switch fuel {
	case voxel.ItemCoal, voxel.BlockCoalOre:
		return 16.0 // Smelts 4 items
	case voxel.BlockOakLog, voxel.BlockOakPlanks, voxel.BlockCraftingTable:
		return 6.0 // Smelts 1.5 items
	case voxel.ItemStick:
		return 2.0 // Smelts 0.5 items
	default:
		return 0.0
	}
}

// UpdateFurnace simulates the smelting cycle (burning fuel, cooking items)
func (gui *InventoryGUI) UpdateFurnace(dt float32) {
	result, canSmelt := GetSmeltingResult(gui.FurnaceInput.Type)
	canAcceptOutput := gui.FurnaceOutput.Count == 0 || (gui.FurnaceOutput.Type == result && gui.FurnaceOutput.Count < 64)

	// Burn down active fuel
	if gui.FurnaceBurnTime > 0 {
		gui.FurnaceBurnTime -= dt
		if gui.FurnaceBurnTime < 0 {
			gui.FurnaceBurnTime = 0
		}
	}

	// Ignite new fuel if needed and can smelt
	if gui.FurnaceBurnTime <= 0 && canSmelt && canAcceptOutput && gui.FurnaceInput.Count > 0 && gui.FurnaceFuel.Count > 0 {
		burnDuration := GetFuelBurnTime(gui.FurnaceFuel.Type)
		if burnDuration > 0 {
			gui.FurnaceFuel.Count--
			if gui.FurnaceFuel.Count == 0 {
				gui.FurnaceFuel.Type = voxel.BlockAir
			}
			gui.FurnaceBurnTime = burnDuration
			gui.FurnaceMaxBurnTime = burnDuration
		}
	}

	// Cook item if furnace is actively burning
	if gui.FurnaceBurnTime > 0 && canSmelt && canAcceptOutput && gui.FurnaceInput.Count > 0 {
		gui.FurnaceCookProgress += dt / 4.0 // 4.0 seconds per item
		if gui.FurnaceCookProgress >= 1.0 {
			gui.FurnaceCookProgress = 0
			gui.FurnaceInput.Count--
			if gui.FurnaceInput.Count == 0 {
				gui.FurnaceInput.Type = voxel.BlockAir
			}
			if gui.FurnaceOutput.Count == 0 {
				gui.FurnaceOutput.Type = result
			}
			gui.FurnaceOutput.Count++
		}
	} else if !canSmelt || !canAcceptOutput || gui.FurnaceInput.Count == 0 {
		gui.FurnaceCookProgress = 0
	}
}

// handleSlotInteraction implements true Minecraft left/right click slot mechanics
func (gui *InventoryGUI) handleSlotInteraction(slot *ItemStack, isRightClick bool) {
	if isRightClick {
		// Right Click: Place 1 item or split stack in half
		if gui.CursorItem.Count > 0 {
			if slot.Count == 0 || slot.Type == voxel.BlockAir {
				slot.Type = gui.CursorItem.Type
				slot.Count = 1
				gui.CursorItem.Count--
				if gui.CursorItem.Count == 0 {
					gui.CursorItem.Type = voxel.BlockAir
				}
			} else if slot.Type == gui.CursorItem.Type && slot.Count < 64 {
				slot.Count++
				gui.CursorItem.Count--
				if gui.CursorItem.Count == 0 {
					gui.CursorItem.Type = voxel.BlockAir
				}
			}
		} else if slot.Count > 0 {
			// Split in half
			take := (slot.Count + 1) / 2
			gui.CursorItem.Type = slot.Type
			gui.CursorItem.Count = take
			slot.Count -= take
			if slot.Count == 0 {
				slot.Type = voxel.BlockAir
			}
		}
	} else {
		// Left Click: Pick up / Swap / Combine
		if gui.CursorItem.Count == 0 {
			if slot.Count > 0 {
				gui.CursorItem = *slot
				slot.Type = voxel.BlockAir
				slot.Count = 0
			}
		} else {
			if slot.Count == 0 || slot.Type == voxel.BlockAir {
				*slot = gui.CursorItem
				gui.CursorItem = ItemStack{Type: voxel.BlockAir, Count: 0}
			} else if slot.Type == gui.CursorItem.Type {
				space := 64 - slot.Count
				if gui.CursorItem.Count <= space {
					slot.Count += gui.CursorItem.Count
					gui.CursorItem = ItemStack{Type: voxel.BlockAir, Count: 0}
				} else {
					slot.Count = 64
					gui.CursorItem.Count -= space
				}
			} else {
				// Swap items
				temp := *slot
				*slot = gui.CursorItem
				gui.CursorItem = temp
			}
		}
	}
	gui.UpdateCrafting()
}

// handleCraftingOutputClick takes crafted item and decrements recipe inputs
func (gui *InventoryGUI) handleCraftingOutputClick(isWorkbench bool) {
	var output *ItemStack
	var grid []ItemStack

	if isWorkbench {
		output = &gui.Crafting3x3Output
		grid = gui.Crafting3x3[:]
	} else {
		output = &gui.Crafting2x2Output
		grid = gui.Crafting2x2[:]
	}

	if output.Count == 0 || output.Type == voxel.BlockAir {
		return
	}

	if gui.CursorItem.Count == 0 {
		gui.CursorItem = *output
		for i := range grid {
			if grid[i].Count > 0 {
				grid[i].Count--
				if grid[i].Count == 0 {
					grid[i].Type = voxel.BlockAir
				}
			}
		}
		gui.UpdateCrafting()
	} else if gui.CursorItem.Type == output.Type && gui.CursorItem.Count+output.Count <= 64 {
		gui.CursorItem.Count += output.Count
		for i := range grid {
			if grid[i].Count > 0 {
				grid[i].Count--
				if grid[i].Count == 0 {
					grid[i].Type = voxel.BlockAir
				}
			}
		}
		gui.UpdateCrafting()
	}
}

// handleFurnaceOutputClick collects smelted item
func (gui *InventoryGUI) handleFurnaceOutputClick() {
	if gui.FurnaceOutput.Count == 0 || gui.FurnaceOutput.Type == voxel.BlockAir {
		return
	}

	if gui.CursorItem.Count == 0 {
		gui.CursorItem = gui.FurnaceOutput
		gui.FurnaceOutput = ItemStack{Type: voxel.BlockAir, Count: 0}
	} else if gui.CursorItem.Type == gui.FurnaceOutput.Type && gui.CursorItem.Count+gui.FurnaceOutput.Count <= 64 {
		gui.CursorItem.Count += gui.FurnaceOutput.Count
		gui.FurnaceOutput = ItemStack{Type: voxel.BlockAir, Count: 0}
	}
}

// Render draws the crosshair, survival bars, hotbar, 2x2/3x3 crafting, furnace GUI, and death screen
func (gui *InventoryGUI) Render(p *mcplayer.MCPlayer, world *voxel.VoxelWorld, atlas *voxel.TextureAtlas) {
	w := float32(gui.ScreenWidth)
	h := float32(gui.ScreenHeight)

	// GUI scale based on resolution
	guiScale := h / 720.0
	if guiScale < 1.0 {
		guiScale = 1.0
	} else if guiScale > 2.5 {
		guiScale = 2.5
	}

	// If player is dead, show Minecraft Death Screen!
	if p.IsDead {
		gui.renderDeathScreen(p, world, guiScale)
		return
	}

	// 1. Center Crosshair (+)
	if !gui.IsInventoryOpen && !gui.IsWorkbenchOpen && !gui.IsFurnaceOpen {
		gui.renderCrosshair(w*0.5, h*0.5)
	}

	// 2. Health (10 Hearts), Hunger (10 Drumsticks) & Oxygen Bubbles (10 Air)
	hotbarSlotSize := 44.0 * guiScale
	hotbarW := hotbarSlotSize * 9.0
	hotbarY := h - hotbarSlotSize - 20.0*guiScale
	hotbarX := w*0.5 - hotbarW*0.5

	gui.renderHealthHearts(hotbarX, hotbarY-32.0*guiScale, p.Health, guiScale)
	gui.renderHungerIcons(hotbarX+hotbarW-180.0*guiScale, hotbarY-32.0*guiScale, p.Hunger, guiScale)

	if p.Oxygen < 10.0 {
		gui.renderAirBubbles(hotbarX+hotbarW-180.0*guiScale, hotbarY-48.0*guiScale, p.Oxygen, guiScale)
	}

	// 3. Experience Bar & Level Number
	gui.renderExpBar(hotbarX, hotbarY-14.0*guiScale, hotbarW, 7.0*guiScale, p.Level, p.ExpProgress, guiScale)

	// 4. 9-Slot Hotbar with Stack Counts
	gui.renderHotbar(hotbarX, hotbarY, hotbarSlotSize, atlas, guiScale)

	// 5. Interactive GUIs
	if gui.IsFurnaceOpen {
		gui.renderFurnaceGUI(w*0.5, h*0.5, atlas, guiScale)
	} else if gui.IsWorkbenchOpen {
		gui.renderWorkbenchGUI(w*0.5, h*0.5, atlas, guiScale)
	} else if gui.IsInventoryOpen {
		gui.renderSurvivalInventoryGUI(w*0.5, h*0.5, p, atlas, guiScale)
	}

	// 6. Render Dragged Cursor Item
	if gui.CursorItem.Count > 0 && gui.CursorItem.Type != voxel.BlockAir {
		mousePos := rl.GetMousePosition()
		itemSize := 36.0 * guiScale
		gui.renderSlotItem(mousePos.X-itemSize*0.5, mousePos.Y-itemSize*0.5, itemSize, gui.CursorItem, atlas, guiScale)
	}
}

// renderDeathScreen draws the classic red Minecraft Death Screen with interactive Respawn button
func (gui *InventoryGUI) renderDeathScreen(p *mcplayer.MCPlayer, world *voxel.VoxelWorld, guiScale float32) {
	w := float32(gui.ScreenWidth)
	h := float32(gui.ScreenHeight)

	// 1. Red Screen Tint
	rl.DrawRectangle(0, 0, gui.ScreenWidth, gui.ScreenHeight, rl.NewColor(135, 15, 15, 190))

	// 2. "You Died!" Title
	title := "You Died!"
	fontSize := int32(48.0 * guiScale)
	tLen := rl.MeasureText(title, fontSize)
	rl.DrawText(title, int32(w*0.5)-tLen/2+3, int32(h*0.32)+3, fontSize, rl.NewColor(30, 0, 0, 255))
	rl.DrawText(title, int32(w*0.5)-tLen/2, int32(h*0.32), fontSize, rl.NewColor(255, 45, 45, 255))

	// 3. Score
	scoreStr := fmt.Sprintf("Score: %d", p.Level*10+int(p.ExpProgress*10))
	scoreSize := int32(20.0 * guiScale)
	sLen := rl.MeasureText(scoreStr, scoreSize)
	rl.DrawText(scoreStr, int32(w*0.5)-sLen/2, int32(h*0.42), scoreSize, rl.RayWhite)

	// 4. "Respawn" Button
	btnW := 220.0 * guiScale
	btnH := 44.0 * guiScale
	btnX := w*0.5 - btnW*0.5
	btnY := h * 0.52

	mouse := rl.GetMousePosition()
	isHover := mouse.X >= btnX && mouse.X <= btnX+btnW && mouse.Y >= btnY && mouse.Y <= btnY+btnH

	btnBg := rl.NewColor(50, 50, 50, 230)
	borderCol := rl.NewColor(120, 120, 120, 255)
	if isHover {
		btnBg = rl.NewColor(80, 80, 80, 255)
		borderCol = rl.RayWhite
	}

	rl.DrawRectangle(int32(btnX), int32(btnY), int32(btnW), int32(btnH), btnBg)
	rl.DrawRectangleLinesEx(rl.NewRectangle(btnX, btnY, btnW, btnH), 2.0*guiScale, borderCol)

	rText := "Respawn"
	rSize := int32(20.0 * guiScale)
	rLen := rl.MeasureText(rText, rSize)
	rl.DrawText(rText, int32(w*0.5)-rLen/2, int32(btnY+12.0*guiScale), rSize, rl.RayWhite)

	// Handle Respawn Click
	if isHover && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		p.Health = 20
		p.Hunger = 20
		p.Oxygen = 10
		p.IsDead = false
		p.DeathTimer = 0
		spawnY := float32(64)
		if world != nil {
			spawnY = float32(world.GetHighestBlock(0, 0) + 3)
		}
		p.Pos = rl.Vector3{X: 0, Y: spawnY, Z: 0}
		p.Vel = rl.Vector3{}
		rl.DisableCursor()
	}
}

func (gui *InventoryGUI) renderCrosshair(cx, cy float32) {
	if gui.IconsTex.ID > 0 {
		srcRec := rl.NewRectangle(0, 0, 16, 16)
		dstRec := rl.NewRectangle(cx-8, cy-8, 16, 16)
		rl.DrawTexturePro(gui.IconsTex, srcRec, dstRec, rl.Vector2{}, 0, rl.White)
	} else {
		col := rl.NewColor(240, 240, 240, 220)
		rl.DrawRectangle(int32(cx-8), int32(cy-1), 17, 3, col)
		rl.DrawRectangle(int32(cx-1), int32(cy-8), 3, 17, col)
	}
}

func (gui *InventoryGUI) renderHealthHearts(x, y float32, health float32, scale float32) {
	if gui.IconsTex.ID > 0 {
		heartSize := 9.0 * scale * 1.5
		spacing := 8.0 * scale * 1.5
		srcEmpty := rl.NewRectangle(16, 0, 9, 9)
		srcFull := rl.NewRectangle(52, 0, 9, 9)
		srcHalf := rl.NewRectangle(61, 0, 9, 9)

		for i := 0; i < 10; i++ {
			hx := x + float32(i)*spacing
			dst := rl.NewRectangle(hx, y, heartSize, heartSize)
			rl.DrawTexturePro(gui.IconsTex, srcEmpty, dst, rl.Vector2{}, 0, rl.White)

			if float32((i+1)*2) <= health {
				rl.DrawTexturePro(gui.IconsTex, srcFull, dst, rl.Vector2{}, 0, rl.White)
			} else if float32(i*2)+1 <= health {
				rl.DrawTexturePro(gui.IconsTex, srcHalf, dst, rl.Vector2{}, 0, rl.White)
			}
		}
	} else {
		spacing := 18.0 * scale
		heartW := 15.0 * scale
		heartH := 14.0 * scale
		for i := 0; i < 10; i++ {
			hx := x + float32(i)*spacing
			rl.DrawRectangle(int32(hx), int32(y), int32(heartW), int32(heartH), rl.NewColor(35, 10, 10, 220))
			rl.DrawRectangleLines(int32(hx), int32(y), int32(heartW), int32(heartH), rl.NewColor(15, 5, 5, 255))
			if float32((i+1)*2) <= health {
				rl.DrawRectangle(int32(hx+1.5*scale), int32(y+1.5*scale), int32(heartW-3*scale), int32(heartH-3*scale), rl.NewColor(235, 30, 30, 255))
			} else if float32(i*2)+1 <= health {
				rl.DrawRectangle(int32(hx+1.5*scale), int32(y+1.5*scale), int32((heartW-3*scale)*0.5), int32(heartH-3*scale), rl.NewColor(235, 30, 30, 255))
			}
		}
	}
}

func (gui *InventoryGUI) renderHungerIcons(x, y float32, hunger float32, scale float32) {
	if gui.IconsTex.ID > 0 {
		drumSize := 9.0 * scale * 1.5
		spacing := 8.0 * scale * 1.5
		srcEmpty := rl.NewRectangle(16, 27, 9, 9)
		srcFull := rl.NewRectangle(52, 27, 9, 9)
		srcHalf := rl.NewRectangle(61, 27, 9, 9)

		for i := 0; i < 10; i++ {
			fx := x + float32(9-i)*spacing
			dst := rl.NewRectangle(fx, y, drumSize, drumSize)
			rl.DrawTexturePro(gui.IconsTex, srcEmpty, dst, rl.Vector2{}, 0, rl.White)

			if float32((i+1)*2) <= hunger {
				rl.DrawTexturePro(gui.IconsTex, srcFull, dst, rl.Vector2{}, 0, rl.White)
			} else if float32(i*2)+1 <= hunger {
				rl.DrawTexturePro(gui.IconsTex, srcHalf, dst, rl.Vector2{}, 0, rl.White)
			}
		}
	} else {
		spacing := 18.0 * scale
		drumW := 15.0 * scale
		drumH := 14.0 * scale
		for i := 0; i < 10; i++ {
			fx := x + float32(i)*spacing
			rl.DrawRectangle(int32(fx), int32(y), int32(drumW), int32(drumH), rl.NewColor(35, 20, 10, 220))
			rl.DrawRectangleLines(int32(fx), int32(y), int32(drumW), int32(drumH), rl.NewColor(15, 10, 5, 255))
			if float32((i+1)*2) <= hunger {
				rl.DrawRectangle(int32(fx+2*scale), int32(y+2*scale), int32(drumW-4*scale), int32(drumH-4*scale), rl.NewColor(210, 125, 45, 255))
			}
		}
	}
}

func (gui *InventoryGUI) renderAirBubbles(x, y float32, oxygen float32, scale float32) {
	bubbles := int(oxygen)
	if gui.IconsTex.ID > 0 {
		bubbleSize := 9.0 * scale * 1.5
		spacing := 8.0 * scale * 1.5
		srcBubble := rl.NewRectangle(16, 18, 9, 9)
		for i := 0; i < bubbles; i++ {
			bx := x + float32(9-i)*spacing
			dst := rl.NewRectangle(bx, y, bubbleSize, bubbleSize)
			rl.DrawTexturePro(gui.IconsTex, srcBubble, dst, rl.Vector2{}, 0, rl.White)
		}
	} else {
		spacing := 18.0 * scale
		for i := 0; i < bubbles; i++ {
			bx := x + float32(i)*spacing
			rl.DrawCircle(int32(bx+7*scale), int32(y+7*scale), 6*scale, rl.NewColor(30, 130, 230, 240))
			rl.DrawCircle(int32(bx+5*scale), int32(y+5*scale), 2*scale, rl.RayWhite)
		}
	}
}

func (gui *InventoryGUI) renderExpBar(x, y, w, h float32, level int, progress float32, scale float32) {
	if gui.IconsTex.ID > 0 {
		srcBg := rl.NewRectangle(0, 64, 182, 5)
		dstBg := rl.NewRectangle(x, y, w, 6.0*scale)
		rl.DrawTexturePro(gui.IconsTex, srcBg, dstBg, rl.Vector2{}, 0, rl.White)

		if progress > 0 {
			progClamp := progress
			if progClamp > 1.0 {
				progClamp = 1.0
			}
			srcFill := rl.NewRectangle(0, 69, 182.0*progClamp, 5)
			dstFill := rl.NewRectangle(x, y, w*progClamp, 6.0*scale)
			rl.DrawTexturePro(gui.IconsTex, srcFill, dstFill, rl.Vector2{}, 0, rl.White)
		}
	} else {
		rl.DrawRectangle(int32(x), int32(y), int32(w), int32(h), rl.NewColor(15, 30, 10, 255))
		if progress > 0 {
			rl.DrawRectangle(int32(x+1), int32(y+1), int32((w-2)*progress), int32(h-2), rl.NewColor(128, 255, 32, 255))
		}
	}

	lvlStr := fmt.Sprintf("%d", level)
	fontSize := int32(16.0 * scale)
	lLen := rl.MeasureText(lvlStr, fontSize)
	rl.DrawText(lvlStr, int32(x+w*0.5)-lLen/2+1, int32(y-16.0*scale)+1, fontSize, rl.NewColor(0, 40, 0, 255))
	rl.DrawText(lvlStr, int32(x+w*0.5)-lLen/2, int32(y-16.0*scale), fontSize, rl.NewColor(128, 255, 32, 255))
}

func (gui *InventoryGUI) renderHotbar(x, y, slotSize float32, atlas *voxel.TextureAtlas, scale float32) {
	if gui.WidgetsTex.ID > 0 {
		hotbarW := 182.0 * scale * 2.0
		hotbarH := 22.0 * scale * 2.0
		startX := x + (slotSize*9.0-hotbarW)*0.5
		startY := y

		// 1. Authentic Hotbar Texture Background
		srcHotbar := rl.NewRectangle(0, 0, 182, 22)
		dstHotbar := rl.NewRectangle(startX, startY, hotbarW, hotbarH)
		rl.DrawTexturePro(gui.WidgetsTex, srcHotbar, dstHotbar, rl.Vector2{}, 0, rl.White)

		// 2. Render items in each slot
		innerSlotSize := 32.0 * scale
		for i := 0; i < 9; i++ {
			sx := startX + float32(3+i*20)*scale*2.0
			sy := startY + 3.0*scale*2.0
			gui.renderSlotItem(sx, sy, innerSlotSize, gui.HotbarSlots[i], atlas, scale)
		}

		// 3. Selected Slot Highlight Selector from widgets.png
		selX := startX + float32(gui.SelectedSlot*20)*scale*2.0 - 1.0*scale*2.0
		selY := startY - 1.0*scale*2.0
		srcSel := rl.NewRectangle(0, 22, 24, 24)
		dstSel := rl.NewRectangle(selX, selY, 24.0*scale*2.0, 24.0*scale*2.0)
		rl.DrawTexturePro(gui.WidgetsTex, srcSel, dstSel, rl.Vector2{}, 0, rl.White)
	} else {
		totalW := slotSize * 9.0
		rl.DrawRectangle(int32(x), int32(y), int32(totalW), int32(slotSize), rl.NewColor(45, 45, 45, 240))
		for i := 0; i < 9; i++ {
			sx := x + float32(i)*slotSize
			itemPad := 4.0 * scale
			gui.renderSlotItem(sx+itemPad, y+itemPad, slotSize-itemPad*2, gui.HotbarSlots[i], atlas, scale)
			if i == gui.SelectedSlot {
				rl.DrawRectangleLinesEx(rl.NewRectangle(sx-2*scale, y-2*scale, slotSize+4*scale, slotSize+4*scale), 3.0*scale, rl.RayWhite)
			}
		}
	}

	// Active block name popup
	activeBlock := gui.GetActiveBlock()
	if activeBlock != voxel.BlockAir {
		nameStr := voxel.BlockRegistry[activeBlock].Name
		fontSize := int32(16.0 * scale)
		nLen := rl.MeasureText(nameStr, fontSize)
		totalW := slotSize * 9.0
		rl.DrawRectangleRounded(rl.NewRectangle(x+totalW*0.5-float32(nLen)*0.5-12*scale, y-60*scale, float32(nLen)+24*scale, 26*scale), 0.3, 4, rl.NewColor(20, 20, 30, 220))
		rl.DrawText(nameStr, int32(x+totalW*0.5)-nLen/2, int32(y-55*scale), fontSize, rl.RayWhite)
	}
}

// renderSlotItem draws the authentic 2D texture pack sprite and stack count text
func (gui *InventoryGUI) renderSlotItem(x, y, size float32, item ItemStack, atlas *voxel.TextureAtlas, scale float32) {
	if item.Count <= 0 || item.Type == voxel.BlockAir {
		return
	}

	// 1. Draw Textured Minecraft Item / Block Sprite from Atlas
	if atlas != nil && atlas.Texture.ID > 0 {
		col, row := voxel.GetBlockTextureAtlasPos(item.Type, voxel.FaceNorth)
		bSize := float32(atlas.BlockSize)
		srcRec := rl.NewRectangle(float32(col)*bSize, float32(row)*bSize, bSize, bSize)
		dstRec := rl.NewRectangle(x, y, size, size)

		rl.DrawTexturePro(atlas.Texture, srcRec, dstRec, rl.Vector2{}, 0, rl.White)
	}

	// 2. Draw Stack Count (bottom-right of slot)
	if item.Count > 1 {
		countStr := fmt.Sprintf("%d", item.Count)
		fontSize := int32(14.0 * scale)
		tLen := rl.MeasureText(countStr, fontSize)

		tx := int32(x + size - float32(tLen) - 2*scale)
		ty := int32(y + size - float32(fontSize) - 1*scale)

		// Shadow
		rl.DrawText(countStr, tx+1, ty+1, fontSize, rl.NewColor(30, 30, 30, 255))
		// White text
		rl.DrawText(countStr, tx, ty, fontSize, rl.RayWhite)
	}
}

// renderSurvivalInventoryGUI renders the authentic 2x2 player crafting inventory GUI
func (gui *InventoryGUI) renderSurvivalInventoryGUI(cx, cy float32, p *mcplayer.MCPlayer, atlas *voxel.TextureAtlas, scale float32) {
	s := scale * 2.0
	winW := 176.0 * s
	winH := 166.0 * s
	slotSize := 16.0 * s

	winX := cx - winW*0.5
	if gui.IsRecipeBookOpen {
		winX = cx - winW*0.5 + 85.0*scale
	}
	winY := cy - winH*0.5

	if gui.InventoryTex.ID > 0 {
		src := rl.NewRectangle(0, 0, 176, 166)
		dst := rl.NewRectangle(winX, winY, winW, winH)
		rl.DrawTexturePro(gui.InventoryTex, src, dst, rl.Vector2{}, 0, rl.White)
	} else {
		rl.DrawRectangle(int32(winX), int32(winY), int32(winW), int32(winH), rl.NewColor(198, 198, 198, 255))
		rl.DrawRectangleLinesEx(rl.NewRectangle(winX, winY, winW, winH), 3.0*scale, rl.NewColor(60, 60, 60, 255))
	}

	// 2x2 Crafting Grid (x: 98, y: 18)
	for r := 0; r < 2; r++ {
		for c := 0; c < 2; c++ {
			sx := winX + float32(98+c*18)*s
			sy := winY + float32(18+r*18)*s
			idx := r*2 + c
			gui.renderInteractiveSlot(sx, sy, slotSize, &gui.Crafting2x2[idx], atlas, scale, false, false)
		}
	}

	// 2x2 Output Slot (x: 154, y: 28)
	gui.renderInteractiveSlot(winX+154.0*s, winY+28.0*s, slotSize, &gui.Crafting2x2Output, atlas, scale, true, false)

	// Main 27-slot Inventory (x: 8, y: 84)
	for r := 0; r < 3; r++ {
		for c := 0; c < 9; c++ {
			sx := winX + float32(8+c*18)*s
			sy := winY + float32(84+r*18)*s
			idx := r*9 + c
			gui.renderInteractiveSlot(sx, sy, slotSize, &gui.MainInventory[idx], atlas, scale, false, false)
		}
	}

	// Hotbar in GUI (x: 8, y: 142)
	for c := 0; c < 9; c++ {
		sx := winX + float32(8+c*18)*s
		sy := winY + 142.0*s
		gui.renderInteractiveSlot(sx, sy, slotSize, &gui.HotbarSlots[c], atlas, scale, false, false)
	}

	// Recipe Book Toggle Button
	bookBtnX := winX + 104.0*s
	bookBtnY := winY + 61.0*s
	bookBtnS := 18.0 * s
	gui.renderRecipeBookToggleButton(bookBtnX, bookBtnY, bookBtnS, scale)

	// Render Recipe Book Panel on Left
	if gui.IsRecipeBookOpen {
		gui.renderRecipeBookPanel(winX-175.0*scale, winY, 170.0*scale, winH, atlas, scale, false)
	}
}

// renderWorkbenchGUI renders the authentic 3x3 crafting table GUI
func (gui *InventoryGUI) renderWorkbenchGUI(cx, cy float32, atlas *voxel.TextureAtlas, scale float32) {
	s := scale * 2.0
	winW := 176.0 * s
	winH := 166.0 * s
	slotSize := 16.0 * s

	winX := cx - winW*0.5
	if gui.IsRecipeBookOpen {
		winX = cx - winW*0.5 + 85.0*scale
	}
	winY := cy - winH*0.5

	if gui.WorkbenchTex.ID > 0 {
		src := rl.NewRectangle(0, 0, 176, 166)
		dst := rl.NewRectangle(winX, winY, winW, winH)
		rl.DrawTexturePro(gui.WorkbenchTex, src, dst, rl.Vector2{}, 0, rl.White)
	} else {
		rl.DrawRectangle(int32(winX), int32(winY), int32(winW), int32(winH), rl.NewColor(198, 198, 198, 255))
		rl.DrawRectangleLinesEx(rl.NewRectangle(winX, winY, winW, winH), 3.0*scale, rl.NewColor(60, 60, 60, 255))
	}

	// 3x3 Crafting Grid (x: 30, y: 17)
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			sx := winX + float32(30+c*18)*s
			sy := winY + float32(17+r*18)*s
			idx := r*3 + c
			gui.renderInteractiveSlot(sx, sy, slotSize, &gui.Crafting3x3[idx], atlas, scale, false, true)
		}
	}

	// 3x3 Output Slot (x: 124, y: 35)
	gui.renderInteractiveSlot(winX+124.0*s, winY+35.0*s, slotSize, &gui.Crafting3x3Output, atlas, scale, true, true)

	// Main 27-slot Inventory (x: 8, y: 84)
	for r := 0; r < 3; r++ {
		for c := 0; c < 9; c++ {
			sx := winX + float32(8+c*18)*s
			sy := winY + float32(84+r*18)*s
			idx := r*9 + c
			gui.renderInteractiveSlot(sx, sy, slotSize, &gui.MainInventory[idx], atlas, scale, false, true)
		}
	}

	// Hotbar in Workbench (x: 8, y: 142)
	for c := 0; c < 9; c++ {
		sx := winX + float32(8+c*18)*s
		sy := winY + 142.0*s
		gui.renderInteractiveSlot(sx, sy, slotSize, &gui.HotbarSlots[c], atlas, scale, false, true)
	}

	// Recipe Book Toggle Button
	bookBtnX := winX + 8.0*s
	bookBtnY := winY + 45.0*s
	bookBtnS := 18.0 * s
	gui.renderRecipeBookToggleButton(bookBtnX, bookBtnY, bookBtnS, scale)

	// Render Recipe Book Panel on Left
	if gui.IsRecipeBookOpen {
		gui.renderRecipeBookPanel(winX-175.0*scale, winY, 170.0*scale, winH, atlas, scale, true)
	}
}

// renderRecipeBookToggleButton renders the recipe book button
func (gui *InventoryGUI) renderRecipeBookToggleButton(x, y, size float32, scale float32) {
	mouse := rl.GetMousePosition()
	isHover := mouse.X >= x && mouse.X <= x+size && mouse.Y >= y && mouse.Y <= y+size

	btnBg := rl.NewColor(55, 140, 45, 255)
	borderCol := rl.NewColor(30, 80, 25, 255)
	if isHover {
		btnBg = rl.NewColor(75, 180, 60, 255)
		borderCol = rl.RayWhite
	}

	rl.DrawRectangle(int32(x), int32(y), int32(size), int32(size), btnBg)
	rl.DrawRectangleLinesEx(rl.NewRectangle(x, y, size, size), 1.5*scale, borderCol)

	// Book Icon Drawing
	rl.DrawRectangle(int32(x+3*scale), int32(y+3*scale), int32(size-6*scale), int32(size-6*scale), rl.NewColor(140, 30, 25, 255))
	rl.DrawRectangle(int32(x+5*scale), int32(y+5*scale), int32(size-10*scale), int32(size-10*scale), rl.NewColor(240, 230, 200, 255))

	if isHover && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		gui.IsRecipeBookOpen = !gui.IsRecipeBookOpen
	}
}

// renderRecipeBookPanel renders the scrollable list of craftable items in the Recipe Book
func (gui *InventoryGUI) renderRecipeBookPanel(x, y, w, h float32, atlas *voxel.TextureAtlas, scale float32, isWorkbench bool) {
	rl.DrawRectangle(int32(x), int32(y), int32(w), int32(h), rl.NewColor(185, 185, 185, 255))
	rl.DrawRectangleLinesEx(rl.NewRectangle(x, y, w, h), 3.0*scale, rl.NewColor(60, 60, 60, 255))

	titleSize := int32(15.0 * scale)
	rl.DrawText("Rezeptbuch", int32(x+14*scale), int32(y+14*scale), titleSize, rl.NewColor(50, 50, 50, 255))

	slotSize := 32.0 * scale
	slotPad := 4.0 * scale
	cols := 4
	startX := x + 12.0*scale
	startY := y + 42.0*scale

	mouse := rl.GetMousePosition()
	var hoveredEntry *RecipeBookEntry
	var hoveredPos rl.Vector2

	for i, entry := range RecipeBookList {
		c := i % cols
		r := i / cols
		sx := startX + float32(c)*(slotSize+slotPad)
		sy := startY + float32(r)*(slotSize+slotPad)

		canCraft := entry.CanCraft(gui.HotbarSlots, gui.MainInventory)
		is3x3Locked := !isWorkbench && entry.Is3x3Only

		isHover := mouse.X >= sx && mouse.X <= sx+slotSize && mouse.Y >= sy && mouse.Y <= sy+slotSize

		slotBg := rl.NewColor(135, 135, 135, 255)
		borderCol := rl.NewColor(55, 55, 55, 255)

		if canCraft && !is3x3Locked {
			slotBg = rl.NewColor(150, 190, 140, 255)
			borderCol = rl.NewColor(40, 120, 30, 255)
		} else if is3x3Locked {
			slotBg = rl.NewColor(105, 105, 105, 255)
		}

		if isHover {
			slotBg = rl.NewColor(180, 220, 170, 255)
			borderCol = rl.RayWhite
			hoveredEntry = &RecipeBookList[i]
			hoveredPos = mouse
		}

		rl.DrawRectangle(int32(sx), int32(sy), int32(slotSize), int32(slotSize), slotBg)
		rl.DrawRectangleLinesEx(rl.NewRectangle(sx, sy, slotSize, slotSize), 1.5*scale, borderCol)

		itemPad := 3.0 * scale
		gui.renderSlotItem(sx+itemPad, sy+itemPad, slotSize-itemPad*2, entry.Output, atlas, scale)

		if isHover && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			gui.AutoFillRecipe(entry, isWorkbench)
		}
	}

	if hoveredEntry != nil {
		nameStr := hoveredEntry.Name
		if !isWorkbench && hoveredEntry.Is3x3Only {
			nameStr = fmt.Sprintf("%s (Requires 3x3 Table)", hoveredEntry.Name)
		}
		tipSize := int32(14.0 * scale)
		tLen := rl.MeasureText(nameStr, tipSize)
		tipW := float32(tLen) + 16.0*scale
		tipH := 24.0 * scale
		tipX := hoveredPos.X + 12.0*scale
		tipY := hoveredPos.Y - 28.0*scale

		rl.DrawRectangle(int32(tipX), int32(tipY), int32(tipW), int32(tipH), rl.NewColor(20, 20, 30, 240))
		rl.DrawRectangleLinesEx(rl.NewRectangle(tipX, tipY, tipW, tipH), 1.5*scale, rl.NewColor(130, 130, 200, 255))
		rl.DrawText(nameStr, int32(tipX+8*scale), int32(tipY+5*scale), tipSize, rl.RayWhite)
	}
}

// renderFurnaceGUI renders the authentic Minecraft furnace smelting GUI
func (gui *InventoryGUI) renderFurnaceGUI(cx, cy float32, atlas *voxel.TextureAtlas, scale float32) {
	s := scale * 2.0
	winW := 176.0 * s
	winH := 166.0 * s
	slotSize := 16.0 * s

	winX := cx - winW*0.5
	winY := cy - winH*0.5

	if gui.FurnaceTex.ID > 0 {
		src := rl.NewRectangle(0, 0, 176, 166)
		dst := rl.NewRectangle(winX, winY, winW, winH)
		rl.DrawTexturePro(gui.FurnaceTex, src, dst, rl.Vector2{}, 0, rl.White)

		// 1. Animated Smelting Flame (src: 176, 0, 14, 14) at (56, 36)
		if gui.FurnaceBurnTime > 0 && gui.FurnaceMaxBurnTime > 0 {
			burnPct := gui.FurnaceBurnTime / gui.FurnaceMaxBurnTime
			if burnPct > 1.0 {
				burnPct = 1.0
			}
			flameH := 14.0 * burnPct
			srcFlame := rl.NewRectangle(176, 14.0-flameH, 14, flameH)
			dstFlame := rl.NewRectangle(winX+56.0*s, winY+float32(36.0+14.0-flameH)*s, 14.0*s, flameH*s)
			rl.DrawTexturePro(gui.FurnaceTex, srcFlame, dstFlame, rl.Vector2{}, 0, rl.White)
		}

		// 2. Animated Cooking Progress Arrow (src: 176, 14, 24, 17) at (79, 34)
		if gui.FurnaceCookProgress > 0 {
			arrowW := 24.0 * gui.FurnaceCookProgress
			if arrowW > 24.0 {
				arrowW = 24.0
			}
			srcArrow := rl.NewRectangle(176, 14, arrowW, 17)
			dstArrow := rl.NewRectangle(winX+79.0*s, winY+34.0*s, arrowW*s, 17.0*s)
			rl.DrawTexturePro(gui.FurnaceTex, srcArrow, dstArrow, rl.Vector2{}, 0, rl.White)
		}
	} else {
		rl.DrawRectangle(int32(winX), int32(winY), int32(winW), int32(winH), rl.NewColor(198, 198, 198, 255))
		rl.DrawRectangleLinesEx(rl.NewRectangle(winX, winY, winW, winH), 3.0*scale, rl.NewColor(60, 60, 60, 255))
	}

	// 1. Input Slot (x: 56, y: 17)
	gui.renderInteractiveFurnaceSlot(winX+56.0*s, winY+17.0*s, slotSize, &gui.FurnaceInput, atlas, scale, false)

	// 2. Fuel Slot (x: 56, y: 53)
	gui.renderInteractiveFurnaceSlot(winX+56.0*s, winY+53.0*s, slotSize, &gui.FurnaceFuel, atlas, scale, false)

	// 3. Output Slot (x: 116, y: 35)
	gui.renderInteractiveFurnaceSlot(winX+116.0*s, winY+35.0*s, slotSize+8.0*scale, &gui.FurnaceOutput, atlas, scale, true)

	// Main 27-slot Inventory (x: 8, y: 84)
	for r := 0; r < 3; r++ {
		for c := 0; c < 9; c++ {
			sx := winX + float32(8+c*18)*s
			sy := winY + float32(84+r*18)*s
			idx := r*9 + c
			gui.renderInteractiveSlot(sx, sy, slotSize, &gui.MainInventory[idx], atlas, scale, false, false)
		}
	}

	// Hotbar in Furnace (x: 8, y: 142)
	for c := 0; c < 9; c++ {
		sx := winX + float32(8+c*18)*s
		sy := winY + 142.0*s
		gui.renderInteractiveSlot(sx, sy, slotSize, &gui.HotbarSlots[c], atlas, scale, false, false)
	}
}

func (gui *InventoryGUI) renderInteractiveFurnaceSlot(x, y, size float32, slot *ItemStack, atlas *voxel.TextureAtlas, scale float32, isOutput bool) {
	mouse := rl.GetMousePosition()
	isHover := mouse.X >= x && mouse.X <= x+size && mouse.Y >= y && mouse.Y <= y+size

	if isHover {
		rl.DrawRectangle(int32(x), int32(y), int32(size), int32(size), rl.NewColor(255, 255, 255, 75))
	}

	// Render item
	gui.renderSlotItem(x+1.5*scale, y+1.5*scale, size-3*scale, *slot, atlas, scale)

	// Click Handling
	if isHover {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			if isOutput {
				gui.handleFurnaceOutputClick()
			} else {
				gui.handleSlotInteraction(slot, false)
			}
		} else if rl.IsMouseButtonPressed(rl.MouseButtonRight) {
			if isOutput {
				gui.handleFurnaceOutputClick()
			} else {
				gui.handleSlotInteraction(slot, true)
			}
		}
	}
}

func (gui *InventoryGUI) renderInteractiveSlot(x, y, size float32, slot *ItemStack, atlas *voxel.TextureAtlas, scale float32, isOutput, isWorkbench bool) {
	mouse := rl.GetMousePosition()
	isHover := mouse.X >= x && mouse.X <= x+size && mouse.Y >= y && mouse.Y <= y+size

	if isHover {
		rl.DrawRectangle(int32(x), int32(y), int32(size), int32(size), rl.NewColor(255, 255, 255, 75))
	}

	// Render item
	gui.renderSlotItem(x+1.5*scale, y+1.5*scale, size-3*scale, *slot, atlas, scale)

	// Click Handling
	if isHover {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			if isOutput {
				gui.handleCraftingOutputClick(isWorkbench)
			} else {
				gui.handleSlotInteraction(slot, false)
			}
		} else if rl.IsMouseButtonPressed(rl.MouseButtonRight) {
			if isOutput {
				gui.handleCraftingOutputClick(isWorkbench)
			} else {
				gui.handleSlotInteraction(slot, true)
			}
		}
	}
}
