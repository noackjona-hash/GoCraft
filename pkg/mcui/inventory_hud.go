package mcui

import (
	"fmt"

	"racing_game/pkg/mcplayer"
	"racing_game/pkg/voxel"

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
}

// NewInventoryGUI initializes empty inventory with starter items
func NewInventoryGUI(screenWidth, screenHeight int32) *InventoryGUI {
	gui := &InventoryGUI{
		ScreenWidth:      screenWidth,
		ScreenHeight:     screenHeight,
		SelectedSlot:     0,
		IsRecipeBookOpen: false,
	}

	// Starter items: Wooden Pickaxe, 16 Planks, 12 Torches, 8 Logs, 8 Sticks
	gui.HotbarSlots[0] = ItemStack{Type: voxel.ItemWoodPickaxe, Count: 1}
	gui.HotbarSlots[1] = ItemStack{Type: voxel.BlockOakPlanks, Count: 16}
	gui.HotbarSlots[2] = ItemStack{Type: voxel.BlockTorch, Count: 12}
	gui.HotbarSlots[3] = ItemStack{Type: voxel.BlockOakLog, Count: 8}
	gui.HotbarSlots[4] = ItemStack{Type: voxel.ItemStick, Count: 8}

	gui.UpdateCrafting()
	return gui
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
	if gui.SelectedSlot >= 0 && gui.SelectedSlot < 9 {
		slot := &gui.HotbarSlots[gui.SelectedSlot]
		if slot.Count > 0 {
			slot.Count--
			if slot.Count == 0 {
				slot.Type = voxel.BlockAir
			}
		}
	}
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
	col := rl.NewColor(240, 240, 240, 220)
	rl.DrawRectangle(int32(cx-8), int32(cy-1), 17, 3, col)
	rl.DrawRectangle(int32(cx-1), int32(cy-8), 3, 17, col)
}

func (gui *InventoryGUI) renderHealthHearts(x, y float32, health float32, scale float32) {
	spacing := 18.0 * scale
	heartW := 15.0 * scale
	heartH := 14.0 * scale

	for i := 0; i < 10; i++ {
		hx := x + float32(i)*spacing
		rl.DrawRectangle(int32(hx), int32(y), int32(heartW), int32(heartH), rl.NewColor(35, 10, 10, 220))
		rl.DrawRectangleLines(int32(hx), int32(y), int32(heartW), int32(heartH), rl.NewColor(15, 5, 5, 255))

		if float32((i+1)*2) <= health {
			rl.DrawRectangle(int32(hx+1.5*scale), int32(y+1.5*scale), int32(heartW-3*scale), int32(heartH-3*scale), rl.NewColor(235, 30, 30, 255))
			rl.DrawRectangle(int32(hx+3*scale), int32(y+3*scale), int32(3*scale), int32(3*scale), rl.NewColor(255, 200, 200, 255))
		} else if float32(i*2)+1 <= health {
			rl.DrawRectangle(int32(hx+1.5*scale), int32(y+1.5*scale), int32((heartW-3*scale)*0.5), int32(heartH-3*scale), rl.NewColor(235, 30, 30, 255))
		}
	}
}

func (gui *InventoryGUI) renderHungerIcons(x, y float32, hunger float32, scale float32) {
	spacing := 18.0 * scale
	drumW := 15.0 * scale
	drumH := 14.0 * scale

	for i := 0; i < 10; i++ {
		fx := x + float32(i)*spacing
		rl.DrawRectangle(int32(fx), int32(y), int32(drumW), int32(drumH), rl.NewColor(35, 20, 10, 220))
		rl.DrawRectangleLines(int32(fx), int32(y), int32(drumW), int32(drumH), rl.NewColor(15, 10, 5, 255))
		if float32((i+1)*2) <= hunger {
			rl.DrawRectangle(int32(fx+2*scale), int32(y+2*scale), int32(drumW-4*scale), int32(drumH-4*scale), rl.NewColor(210, 125, 45, 255))
			rl.DrawRectangle(int32(fx+3*scale), int32(y+3*scale), int32(2*scale), int32(2*scale), rl.NewColor(255, 220, 170, 255))
		}
	}
}

func (gui *InventoryGUI) renderAirBubbles(x, y float32, oxygen float32, scale float32) {
	bubbles := int(oxygen)
	spacing := 18.0 * scale
	for i := 0; i < bubbles; i++ {
		bx := x + float32(i)*spacing
		rl.DrawCircle(int32(bx+7*scale), int32(y+7*scale), 6*scale, rl.NewColor(30, 130, 230, 240))
		rl.DrawCircle(int32(bx+5*scale), int32(y+5*scale), 2*scale, rl.RayWhite)
	}
}

func (gui *InventoryGUI) renderExpBar(x, y, w, h float32, level int, progress float32, scale float32) {
	rl.DrawRectangle(int32(x), int32(y), int32(w), int32(h), rl.NewColor(15, 30, 10, 255))
	rl.DrawRectangleLines(int32(x), int32(y), int32(w), int32(h), rl.NewColor(5, 15, 5, 255))
	if progress > 0 {
		rl.DrawRectangle(int32(x+1), int32(y+1), int32((w-2)*progress), int32(h-2), rl.NewColor(128, 255, 32, 255))
	}

	lvlStr := fmt.Sprintf("%d", level)
	fontSize := int32(18.0 * scale)
	lLen := rl.MeasureText(lvlStr, fontSize)
	rl.DrawText(lvlStr, int32(x+w*0.5)-lLen/2+1, int32(y-18.0*scale)+1, fontSize, rl.NewColor(0, 40, 0, 255))
	rl.DrawText(lvlStr, int32(x+w*0.5)-lLen/2, int32(y-18.0*scale), fontSize, rl.NewColor(128, 255, 32, 255))
}

func (gui *InventoryGUI) renderHotbar(x, y, slotSize float32, atlas *voxel.TextureAtlas, scale float32) {
	totalW := slotSize * 9.0

	// Hotbar Background
	rl.DrawRectangle(int32(x), int32(y), int32(totalW), int32(slotSize), rl.NewColor(45, 45, 45, 240))
	rl.DrawRectangleLinesEx(rl.NewRectangle(x, y, totalW, slotSize), 2.0*scale, rl.NewColor(25, 25, 25, 255))

	for i := 0; i < 9; i++ {
		sx := x + float32(i)*slotSize
		rl.DrawRectangleLinesEx(rl.NewRectangle(sx, y, slotSize, slotSize), 1.5*scale, rl.NewColor(65, 65, 65, 255))

		// Render authentic HD item sprite!
		itemPad := 4.0 * scale
		gui.renderSlotItem(sx+itemPad, y+itemPad, slotSize-itemPad*2, gui.HotbarSlots[i], atlas, scale)

		// Selected Slot Outline
		if i == gui.SelectedSlot {
			rl.DrawRectangleLinesEx(rl.NewRectangle(sx-2*scale, y-2*scale, slotSize+4*scale, slotSize+4*scale), 3.0*scale, rl.RayWhite)
		}
	}

	// Active block name popup
	activeBlock := gui.GetActiveBlock()
	if activeBlock != voxel.BlockAir {
		nameStr := voxel.BlockRegistry[activeBlock].Name
		fontSize := int32(16.0 * scale)
		nLen := rl.MeasureText(nameStr, fontSize)
		rl.DrawRectangleRounded(rl.NewRectangle(x+totalW*0.5-float32(nLen)*0.5-12*scale, y-60*scale, float32(nLen)+24*scale, 26*scale), 0.3, 4, rl.NewColor(20, 20, 30, 220))
		rl.DrawText(nameStr, int32(x+totalW*0.5)-nLen/2, int32(y-55*scale),