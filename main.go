package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"gocraft/pkg/mcaudio"
	"gocraft/pkg/mcmob"
	"gocraft/pkg/mcplayer"
	"gocraft/pkg/mcui"
	"gocraft/pkg/voxel"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// loadCelestialTexture loads 24-bit/32-bit celestial textures and converts black background pixels to 100% transparent alpha
func loadCelestialTexture(filePath string) rl.Texture2D {
	if fi, err := os.Stat(filePath); err != nil || fi.IsDir() {
		return rl.Texture2D{}
	}
	img := rl.LoadImage(filePath)
	if img == nil || img.Width <= 0 {
		return rl.Texture2D{}
	}
	defer rl.UnloadImage(img)

	rl.ImageFormat(img, rl.UncompressedR8g8b8a8)
	rl.ImageAlphaClear(img, rl.Black, 0.12)

	return rl.LoadTextureFromImage(img)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// High-performance window configuration (MSAA 4x removed to eliminate fillrate choke on iGPUs)
	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagWindowHighdpi)
	rl.InitWindow(1920, 1080, "GoCraft 3D — Survival & Crafting Voxel Engine")
	defer rl.CloseWindow()

	screenWidth := int32(rl.GetScreenWidth())
	screenHeight := int32(rl.GetScreenHeight())
	if screenWidth <= 0 || screenHeight <= 0 {
		screenWidth = 1920
		screenHeight = 1080
	}

	rl.SetTargetFPS(300) // High-refresh rate cap — smooth 300 FPS without wasting GPU cycles

	// Lock & hide mouse cursor for seamless first-person mouse look
	rl.DisableCursor()
	defer rl.EnableCursor()

	// Initialize Dynamic Texture Atlas (Auto-detects 128x128 HD or 16x16)
	atlas := voxel.GenerateTextureAtlas()
	defer atlas.Unload()

	saveDir := "saves/world"
	world := voxel.NewVoxelWorld()
	world.SaveDir = saveDir

	chunkManager := voxel.NewChunkManager(world, atlas)

	// Environment Textures (Sun, Moon Phases) with Transparent Backgrounds
	sunTex := loadCelestialTexture("assets/textures/environment/sun.png")
	if sunTex.ID > 0 {
		defer rl.UnloadTexture(sunTex)
	}
	moonTex := loadCelestialTexture("assets/textures/environment/moon_phases.png")
	if moonTex.ID > 0 {
		defer rl.UnloadTexture(moonTex)
	}

	// Attempt to load existing saved game state
	levelData, loadErr := voxel.LoadLevelData(saveDir)
	hasSave := (loadErr == nil && levelData != nil)

	var player *mcplayer.MCPlayer
	if hasSave {
		player = mcplayer.NewMCPlayer(rl.Vector3{X: levelData.Player.X, Y: levelData.Player.Y, Z: levelData.Player.Z})
		player.Yaw = levelData.Player.Yaw
		player.Pitch = levelData.Player.Pitch
		player.Health = levelData.Player.Health
		player.Hunger = levelData.Player.Hunger
		player.Oxygen = levelData.Player.Oxygen
		player.Level = levelData.Player.Level
		player.ExpProgress = levelData.Player.ExpProgress
		if levelData.Player.Mode == 1 {
			player.Mode = mcplayer.GameModeCreative
		} else {
			player.Mode = mcplayer.GameModeSurvival
		}
		// Restore torches
		for _, t := range levelData.Torches {
			world.Torches[voxel.BlockPos{X: t.X, Y: t.Y, Z: t.Z}] = t.LightLevel
		}
		fmt.Printf("Loaded saved world! Player at (%.1f, %.1f, %.1f)\n", player.Pos.X, player.Pos.Y, player.Pos.Z)
	} else {
		// Spawn Player on top of highest terrain block
		spawnY := float32(world.GetHighestBlock(0, 0) + 2)
		if spawnY < float32(voxel.WaterLevel+2) {
			spawnY = float32(voxel.WaterLevel + 2)
		}
		player = mcplayer.NewMCPlayer(rl.Vector3{X: 0.5, Y: spawnY, Z: 0.5})
	}

	// Pre-load initial chunks around spawn
	chunkManager.UpdatePlayerPos(player.Pos)

	gui := mcui.NewInventoryGUI(screenWidth, screenHeight)
	defer gui.Unload()

	// Restore Inventory if save existed
	if hasSave {
		gui.SelectedSlot = levelData.Inventory.SelectedSlot
		for idx, s := range levelData.Inventory.Hotbar {
			if idx < len(gui.HotbarSlots) {
				gui.HotbarSlots[idx] = mcui.ItemStack{Type: s.Type, Count: s.Count}
			}
		}
		for idx, s := range levelData.Inventory.Main {
			if idx < len(gui.MainInventory) {
				gui.MainInventory[idx] = mcui.ItemStack{Type: s.Type, Count: s.Count}
			}
		}
		gui.UpdateCrafting()
	}

	audioEngine := mcaudio.NewMCAudioEngine()
	defer audioEngine.Close()

	// Initialize Authentic Minecraft Mob & AI Manager
	mobManager := mcmob.NewMobManager()
	defer mobManager.Unload()
	// Pre-spawn some peaceful animals around spawn area
	for i := 0; i < 6; i++ {
		mx := float32(-12 + rand.Intn(24))
		mz := float32(-12 + rand.Intn(24))
		my := float32(world.GetHighestBlock(int(mx), int(mz)) + 1)
		if my > float32(voxel.WaterLevel) {
			mType := mcmob.MobPig
			if i%3 == 1 {
				mType = mcmob.MobCow
			} else if i%3 == 2 {
				mType = mcmob.MobSheep
			}
			mobManager.SpawnMob(mType, rl.Vector3{X: mx, Y: my, Z: mz})
		}
	}

	// Day/Night Cycle
	timeOfDay := float32(0.2) // 0.0 to 1.0 (0.2 = Morning, 0.5 = Noon, 0.8 = Sunset, 1.0 = Midnight)
	dayCount := 0
	if hasSave {
		timeOfDay = levelData.TimeOfDay
		dayCount = levelData.DayCount
	}
	dayLengthSec := float32(300.0)
	cloudOffset := float32(0.0)

	// Block Mining State
	var currentMiningPos rl.Vector3
	var miningProgress float32
	var miningCrunchTimer float32

	// Save notification and timer
	var savedBannerTimer float32
	var autosaveTimer float32

	saveWorldState := func() {
		chunksSaved := world.SaveAllModifiedChunks()

		var hotbarSave []voxel.SlotSave
		for _, slot := range gui.HotbarSlots {
			hotbarSave = append(hotbarSave, voxel.SlotSave{Type: slot.Type, Count: slot.Count})
		}
		var mainSave []voxel.SlotSave
		for _, slot := range gui.MainInventory {
			mainSave = append(mainSave, voxel.SlotSave{Type: slot.Type, Count: slot.Count})
		}

		var torchSaves []voxel.TorchSave
		for pos, light := range world.Torches {
			torchSaves = append(torchSaves, voxel.TorchSave{X: pos.X, Y: pos.Y, Z: pos.Z, LightLevel: light})
		}

		modeVal := 0
		if player.Mode == mcplayer.GameModeCreative {
			modeVal = 1
		}

		data := &voxel.LevelData{
			Version: 1,
			Player: voxel.PlayerSave{
				X:           player.Pos.X,
				Y:           player.Pos.Y,
				Z:           player.Pos.Z,
				Yaw:         player.Yaw,
				Pitch:       player.Pitch,
				Health:      player.Health,
				Hunger:      player.Hunger,
				Oxygen:      player.Oxygen,
				Level:       player.Level,
				ExpProgress: player.ExpProgress,
				Mode:        modeVal,
			},
			Inventory: voxel.InventorySave{
				SelectedSlot: gui.SelectedSlot,
				Hotbar:       hotbarSave,
				Main:         mainSave,
			},
			TimeOfDay: timeOfDay,
			DayCount:  dayCount,
			Torches:   torchSaves,
		}

		if err := voxel.SaveLevelData(saveDir, data); err == nil {
			savedBannerTimer = 2.2
			if chunksSaved > 0 {
				fmt.Printf("World saved: %d chunks written to disk.\n", chunksSaved)
			}
		}
	}
	defer saveWorldState()

	// Background simulation timers for high FPS stability
	randomTickTimer := float32(0)
	waterSpreadTimer := float32(0)
	renderDistToastTimer := float32(0)
	renderDistToastMsg := ""
	wasInWater := false
	totalGameTime := float32(0)

	// Main Game Loop
	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()
		if dt > 0.05 {
			dt = 0.05
		}

		// Keep dynamic resolution up to date
		curW := int32(rl.GetScreenWidth())
		curH := int32(rl.GetScreenHeight())
		if curW > 0 && curH > 0 {
			screenWidth = curW
			screenHeight = curH
			gui.ScreenWidth = curW
			gui.ScreenHeight = curH
		}

		// Update Day/Night Cycle & Drifting Clouds
		timeOfDay += dt / dayLengthSec
		if timeOfDay >= 1.0 {
			timeOfDay -= 1.0
			dayCount++
		}
		cloudOffset += dt * 1.2

		sunAngle := timeOfDay * math.Pi * 2.0
		sunHeight := float32(math.Sin(float64(sunAngle)))

		// 'E' Key: Toggle Inventory OR Close Workbench / Furnace
		if rl.IsKeyPressed(rl.KeyE) && !player.IsDead {
			if gui.IsWorkbenchOpen || gui.IsFurnaceOpen || gui.IsInventoryOpen {
				gui.CloseMenu()
				rl.DisableCursor()
			} else {
				gui.IsInventoryOpen = true
				rl.EnableCursor()
			}
		}

		// Close Workbench, Furnace, or Inventory with ESC
		if rl.IsKeyPressed(rl.KeyEscape) {
			if gui.IsWorkbenchOpen || gui.IsFurnaceOpen || gui.IsInventoryOpen {
				gui.CloseMenu()
				rl.DisableCursor()
			}
		}

		// Damage / Kill Test key (K) in Survival mode
		if rl.IsKeyPressed(rl.KeyK) && player.Mode == mcplayer.GameModeSurvival {
			player.Health -= 8.0
			if player.Health < 0 {
				player.Health = 0
			}
		}

		isMenuOpen := gui.IsInventoryOpen || gui.IsWorkbenchOpen || gui.IsFurnaceOpen || player.IsDead

		// Tick Furnace smelting and burning physics
		gui.UpdateFurnace(dt)

		// Autosave every 60 seconds
		autosaveTimer += dt
		if autosaveTimer >= 60.0 && !player.IsDead {
			saveWorldState()
			autosaveTimer = 0
		}

		// Manual Save (F5)
		if rl.IsKeyPressed(rl.KeyF5) && !player.IsDead {
			saveWorldState()
			autosaveTimer = 0
			audioEngine.TriggerBlockPlace()
		}

		// 'F4' Key: Cycle Render Distance (3, 4, 5, 6 chunks) for instant FPS boost
		if rl.IsKeyPressed(rl.KeyF4) && !player.IsDead {
			newRadius := chunkManager.CycleRenderRadius()
			desc := "Fast (High FPS)"
			if newRadius == 3 {
				desc = "Ultra Performance (Low-End / Laptop)"
			} else if newRadius == 5 {
				desc = "Normal (Balanced)"
			} else if newRadius == 6 {
				desc = "Fancy (169 Chunks)"
			}
			renderDistToastMsg = fmt.Sprintf("Render Distance: %d Chunks — %s", newRadius, desc)
			renderDistToastTimer = 2.5
			audioEngine.TriggerBlockPlace()
		}

		// 'G' Key: Toggle GameMode (Survival <-> Creative)
		if rl.IsKeyPressed(rl.KeyG) && !player.IsDead {
			player.ToggleGameMode()
		}

		// 'Q' Key: Drop active item
		if rl.IsKeyPressed(rl.KeyQ) && !isMenuOpen && !player.IsDead {
			activeBlock := gui.GetActiveBlock()
			if activeBlock != voxel.BlockAir {
				removed := gui.RemoveActiveItem(1)
				if removed > 0 {
					cosY := float32(math.Cos(float64(player.Yaw)))
					sinY := float32(math.Sin(float64(player.Yaw)))
					cosP := float32(math.Cos(float64(player.Pitch)))
					sinP := float32(math.Sin(float64(player.Pitch)))
					lookDir := rl.Vector3{X: -sinY * cosP, Y: sinP, Z: -cosY * cosP}
					
					spawnPos := rl.Vector3{
						X: player.Pos.X,
						Y: player.Pos.Y + player.CurrentEyeHeight - 0.2,
						Z: player.Pos.Z,
					}
					
					vel := rl.Vector3Scale(lookDir, 6.0)
					mobManager.SpawnItem(activeBlock, removed, spawnPos, vel)
				}
			}
		}

		// 'F5' Key: Toggle Perspective (1st Person -> 3rd Person Back -> 3rd Person Front)
		if rl.IsKeyPressed(rl.KeyF5) && !player.IsDead {
			player.TogglePerspective()
		}

		// Hotbar Slot Selection (1-9)
		if !isMenuOpen {
			for key := rl.KeyOne; key <= rl.KeyNine; key++ {
				if rl.IsKeyPressed(int32(key)) {
					gui.SelectedSlot = int(key - rl.KeyOne)
				}
			}

			// Mouse Wheel Hotbar Cycling
			wheel := rl.GetMouseWheelMove()
			if wheel > 0 {
				gui.PrevSlot()
			} else if wheel < 0 {
				gui.NextSlot()
			}
		}

		// --- 2. UPDATE PLAYER MOVEMENT, FURNACE SMELTING & MOBS ---
		if !isMenuOpen {
			player.Update(dt, world)
			chunkManager.UpdatePlayerPos(player.Pos)
		}

		// Update interactive Furnace smelting
		gui.UpdateFurnace(dt)

		// Update 3D Mobs AI, Spawning & Combat
		mobManager.Update(
			dt,
			player.Pos,
			&player.Health,
			world,
			sunHeight,
			func(item voxel.BlockType, count int, pos rl.Vector3) {
				gui.AddItem(item, count)
				player.SpawnBlockBreakParticles(pos, item)
			},
			func(pos rl.Vector3, radius float32) {
				audioEngine.TriggerTNTExplosion()
				bx := int(math.Floor(float64(pos.X)))
				by := int(math.Floor(float64(pos.Y)))
				bz := int(math.Floor(float64(pos.Z)))
				for ex := -3; ex <= 3; ex++ {
					for ey := -3; ey <= 3; ey++ {
						for ez := -3; ez <= 3; ez++ {
							if ex*ex+ey*ey+ez*ez <= 9 {
								tx := bx + ex
								ty := by + ey
								tz := bz + ez
								tBlock := world.GetBlock(tx, ty, tz)
								if tBlock != voxel.BlockBedrock && tBlock != voxel.BlockAir {
									player.SpawnBlockBreakParticles(rl.Vector3{X: float32(tx), Y: float32(ty), Z: float32(tz)}, tBlock)
									world.SetBlock(tx, ty, tz, voxel.BlockAir)
									chunkManager.MarkBlockDirty(tx, tz)
								}
							}
						}
					}
				}
				// Damage player if close to Creeper blast
				pDist := rl.Vector3Distance(player.Pos, pos)
				if pDist < 6.0 {
					player.Health -= (6.0 - pDist) * 3.5
				}
			},
			func(item voxel.BlockType, count int) bool {
				added := gui.AddItem(item, count)
				if added {
					audioEngine.TriggerBlockPlace() // A subtle pop sound for picking up
				}
				return added
			},
		)

		// --- 3. DDA VOXEL RAYCAST & MOB COMBAT ---
		cosY := float32(math.Cos(float64(player.Yaw)))
		sinY := float32(math.Sin(float64(player.Yaw)))
		cosP := float32(math.Cos(float64(player.Pitch)))
		sinP := float32(math.Sin(float64(player.Pitch)))
		lookDir := rl.Vector3{X: -sinY * cosP, Y: sinP, Z: -cosY * cosP}

		// Mob Combat Hit Test (Sword & Tool melee)
		hitMob, _, mobHit := mobManager.RaycastMob(player.RLCamera.Position, lookDir, 4.2)
		if !isMenuOpen && mobHit && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			player.TriggerSwing()
			activeItem := gui.GetActiveBlock()
			dmg := voxel.BlockRegistry[activeItem].AttackDamage
			if dmg <= 0 {
				dmg = 1.0
			}
			hitMob.ApplyDamage(dmg, lookDir)
			audioEngine.TriggerBlockBreak()
			player.SpawnBlockBreakParticles(hitMob.Pos, voxel.BlockRedstoneOre) // Red damage spark
		}

		// Food & Eating Mechanics (Holding Right-Click)
		heldItem := gui.GetActiveBlock()
		heldDef := voxel.BlockRegistry[heldItem]
		if heldDef.IsFood && rl.IsMouseButtonDown(rl.MouseButtonRight) && !isMenuOpen && (player.Hunger < 20 || player.Health < 20) {
			player.EatingTimer += dt
			miningCrunchTimer += dt
			if miningCrunchTimer >= 0.22 {
				audioEngine.TriggerBlockBreak()
				eatPos := rl.Vector3Add(player.RLCamera.Position, rl.Vector3Scale(lookDir, 0.4))
				eatPos.Y -= 0.2 // Slightly lower for the mouth
				player.SpawnBlockBreakParticles(eatPos, heldItem) // eating particles!
				miningCrunchTimer = 0
			}
			if player.EatingTimer >= 1.4 {
				player.Hunger += heldDef.FoodPoints
				if player.Hunger > 20 {
					player.Hunger = 20
				}
				player.Health += heldDef.FoodPoints * 0.4
				if player.Health > 20 {
					player.Health = 20
				}
				gui.ConsumeActiveItem()
				player.EatingTimer = 0
				audioEngine.TriggerBlockBreak()
				eatPos := rl.Vector3Add(player.RLCamera.Position, rl.Vector3Scale(lookDir, 0.4))
				eatPos.Y -= 0.2
				player.SpawnBlockBreakParticles(eatPos, heldItem)
			}
		} else if player.EatingTimer > 0 && !rl.IsMouseButtonDown(rl.MouseButtonRight) {
			player.EatingTimer = 0
		}

		rayResult := voxel.RaycastVoxel(player.RLCamera.Position, lookDir, 5.5, world)

		// Track Block Mining Progress in Survival Mode
		if !isMenuOpen && rayResult.Hit && rl.IsMouseButtonDown(rl.MouseButtonLeft) {
			bx := int(rayResult.BlockPos.X)
			by := int(rayResult.BlockPos.Y)
			bz := int(rayResult.BlockPos.Z)
			targetPos := rayResult.BlockPos

			if targetPos != currentMiningPos {
				currentMiningPos = targetPos
				miningProgress = 0
			}

			brokenType := world.GetBlock(bx, by, bz)
			bDef := voxel.BlockRegistry[brokenType]

			if brokenType != voxel.BlockBedrock && brokenType != voxel.BlockAir {
				player.TriggerSwing()

				if player.Mode == mcplayer.GameModeCreative {
					// Creative Mode: Instant break on click
					if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
						player.SpawnBlockBreakParticles(rayResult.BlockPos, brokenType)
						world.SetBlock(bx, by, bz, voxel.BlockAir)
						chunkManager.MarkBlockDirty(bx, bz)
						audioEngine.TriggerBlockBreak()
					}
				} else {
					// Survival Mode: Break progress based on block hardness & tool efficiency
					hardness := bDef.Hardness
					if hardness <= 0 {
						hardness = 1.0
					}

					// Apply Tool Mining Speed Bonus!
					activeItem := gui.GetActiveBlock()
					activeDef := voxel.BlockRegistry[activeItem]
					if activeDef.IsTool {
						isPickaxeBlock := (brokenType == voxel.BlockStone || brokenType == voxel.BlockCobblestone ||
							brokenType == voxel.BlockCoalOre || brokenType == voxel.BlockIronOre ||
							brokenType == voxel.BlockGoldOre || brokenType == voxel.BlockDiamondOre ||
							brokenType == voxel.BlockRedstoneOre || brokenType == voxel.BlockEmeraldOre ||
							brokenType == voxel.BlockLapisOre || brokenType == voxel.BlockFurnace ||
							brokenType == voxel.BlockBricks || brokenType == voxel.BlockSandstone ||
							brokenType == voxel.BlockMossyCobblestone)

						isAxeBlock := (brokenType == voxel.BlockOakLog || brokenType == voxel.BlockOakPlanks ||
							brokenType == voxel.BlockCraftingTable || brokenType == voxel.BlockBookshelf)

						isShovelBlock := (brokenType == voxel.BlockDirt || brokenType == voxel.BlockGrass ||
							brokenType == voxel.BlockSand)

						if (activeDef.ToolType == "pickaxe" && isPickaxeBlock) ||
							(activeDef.ToolType == "axe" && isAxeBlock) ||
							(activeDef.ToolType == "shovel" && isShovelBlock) {
							hardness /= activeDef.MiningEfficiency
						}
					}

					miningProgress += dt / hardness
					miningCrunchTimer += dt
					if miningCrunchTimer >= 0.22 {
						audioEngine.TriggerBlockBreak()
						miningCrunchTimer = 0
					}

					if miningProgress >= 1.0 {
						player.SpawnBlockBreakParticles(rayResult.BlockPos, brokenType)
						dropItem := voxel.GetBlockDrop(brokenType)
						if dropItem != voxel.BlockAir {
							// Spawn item in the world with a small random velocity popup
							itemPos := rl.Vector3{
								X: float32(bx) + 0.5,
								Y: float32(by) + 0.5,
								Z: float32(bz) + 0.5,
							}
							itemVel := rl.Vector3{
								X: (rand.Float32() - 0.5) * 3.0,
								Y: 3.0 + rand.Float32()*2.0,
								Z: (rand.Float32() - 0.5) * 3.0,
							}
							mobManager.SpawnItem(dropItem, 1, itemPos, itemVel)
						}

						// TNT Explosion!
						if brokenType == voxel.BlockTNT {
							audioEngine.TriggerTNTExplosion()
							for ex := -3; ex <= 3; ex++ {
								for ey := -3; ey <= 3; ey++ {
									for ez := -3; ez <= 3; ez++ {
										if ex*ex+ey*ey+ez*ez <= 8 {
											tx := bx + ex
											ty := by + ey
											tz := bz + ez
											tBlock := world.GetBlock(tx, ty, tz)
											if tBlock != voxel.BlockBedrock && tBlock != voxel.BlockAir {
												player.SpawnBlockBreakParticles(rl.Vector3{X: float32(tx), Y: float32(ty), Z: float32(tz)}, tBlock)
												world.SetBlock(tx, ty, tz, voxel.BlockAir)
												chunkManager.MarkBlockDirty(tx, tz)
											}
										}
									}
								}
							}
						} else {
							world.SetBlock(bx, by, bz, voxel.BlockAir)
							chunkManager.MarkBlockDirty(bx, bz)
							audioEngine.TriggerBlockBreak()

							// Tree Leaf Decay when log is broken!
							if voxel.IsLog(brokenType) {
								world.QueueLeafDecay(bx, by, bz)
							}

							// Sand Gravity: drop sand if block beneath was mined!
							if by+1 < voxel.WorldHeight && world.GetBlock(bx, by+1, bz) == voxel.BlockSand {
								world.QueueSandFall(bx, by+1, bz)
							}
						}
						miningProgress = 0
					}
				}
			}
		} else {
			miningProgress = 0
			miningCrunchTimer = 0
		}

		if !isMenuOpen {
			heldBlock := gui.GetActiveBlock()
			bDef := voxel.BlockRegistry[heldBlock]

			// Right Click: Interact with Crafting Table, Furnace, Bucket, or Place Block
			if rl.IsMouseButtonPressed(rl.MouseButtonRight) && !bDef.IsFood {
				player.TriggerSwing()

				// 1. Bucket Interactions (Empty Bucket & Water Bucket)
				if heldBlock == voxel.ItemBucket {
					liquidRay := voxel.RaycastVoxelWithLiquids(player.RLCamera.Position, lookDir, 5.5, world)
					if liquidRay.Hit && voxel.IsWater(liquidRay.BlockType) {
						bx := int(liquidRay.BlockPos.X)
						by := int(liquidRay.BlockPos.Y)
						bz := int(liquidRay.BlockPos.Z)
						world.SetBlock(bx, by, bz, voxel.BlockAir)
						chunkManager.MarkBlockDirty(bx, bz)
						audioEngine.TriggerWaterSplash()
						player.SpawnBlockBreakParticles(liquidRay.BlockPos, voxel.BlockWater)
						if player.Mode == mcplayer.GameModeSurvival {
							gui.SetActiveSlotItem(voxel.ItemWaterBucket, 1)
						}
					}
				} else if heldBlock == voxel.ItemWaterBucket {
					liquidRay := voxel.RaycastVoxelWithLiquids(player.RLCamera.Position, lookDir, 5.5, world)
					targetRay := rayResult
					if liquidRay.Hit {
						targetRay = liquidRay
					}
					if targetRay.Hit {
						px := int(targetRay.PlacePos.X)
						py := int(targetRay.PlacePos.Y)
						pz := int(targetRay.PlacePos.Z)
						world.SetBlock(px, py, pz, voxel.BlockWater)
						chunkManager.MarkBlockDirty(px, pz)
						audioEngine.TriggerWaterSplash()
						player.SpawnBlockBreakParticles(targetRay.PlacePos, voxel.BlockWater)
						if player.Mode == mcplayer.GameModeSurvival {
							gui.SetActiveSlotItem(voxel.ItemBucket, 1)
						}
					}
				} else if rayResult.Hit {
					bx := int(rayResult.BlockPos.X)
					by := int(rayResult.BlockPos.Y)
					bz := int(rayResult.BlockPos.Z)
					clickedBlock := world.GetBlock(bx, by, bz)

					if clickedBlock == voxel.BlockCraftingTable && !player.IsSneaking {
						gui.IsWorkbenchOpen = true
						gui.IsInventoryOpen = false
						gui.IsFurnaceOpen = false
						rl.EnableCursor()
					} else if clickedBlock == voxel.BlockFurnace && !player.IsSneaking {
						gui.IsFurnaceOpen = true
						gui.IsWorkbenchOpen = false
						gui.IsInventoryOpen = false
						rl.EnableCursor()
					} else {
						px := int(rayResult.PlacePos.X)
						py := int(rayResult.PlacePos.Y)
						pz := int(rayResult.PlacePos.Z)

						if heldBlock != voxel.BlockAir && !bDef.IsTool && heldBlock != voxel.ItemStick && heldBlock != voxel.ItemCoal && heldBlock != voxel.ItemDiamond && heldBlock != voxel.ItemIronIngot && heldBlock != voxel.ItemGoldIngot {
							playerBlockX := int(math.Floor(float64(player.Pos.X)))
							playerBlockY1 := int(math.Floor(float64(player.Pos.Y)))
							playerBlockY2 := int(math.Floor(float64(player.Pos.Y + 1.0)))
							playerBlockZ := int(math.Floor(float64(player.Pos.Z)))

							overlapPlayer := (px == playerBlockX && pz == playerBlockZ && (py == playerBlockY1 || py == playerBlockY2))
							if !overlapPlayer {
								blockToPlace := heldBlock
								if voxel.IsLog(heldBlock) {
									blockToPlace = voxel.GetRotatedLogBlock(heldBlock, rayResult.HitNormal)
								}

								world.SetBlock(px, py, pz, blockToPlace)
								chunkManager.MarkBlockDirty(px, pz)
								audioEngine.TriggerBlockPlace()

								if heldBlock == voxel.BlockSand {
									world.QueueSandFall(px, py, pz)
								}

								if player.Mode == mcplayer.GameModeSurvival {
									gui.ConsumeActiveItem()
								}
							}
						}
					}
				}
			}

			// 'R' Key: Rotate targeted block (Logs Y <-> X <-> Z)
			if rayResult.Hit && rl.IsKeyPressed(rl.KeyR) {
				bx := int(rayResult.BlockPos.X)
				by := int(rayResult.BlockPos.Y)
				bz := int(rayResult.BlockPos.Z)
				targetBlock := world.GetBlock(bx, by, bz)

				if voxel.IsLog(targetBlock) {
					rotated := voxel.CycleBlockRotation(targetBlock)
					if rotated != targetBlock {
						world.SetBlock(bx, by, bz, rotated)
						chunkManager.MarkBlockDirty(bx, bz)
						audioEngine.TriggerBlockPlace()
						player.SpawnBlockBreakParticles(rayResult.BlockPos, targetBlock)
					}
				}
			}
		}

		// Update Audio Synthesizer & Underwater Status
		audioEngine.IsUnderwater = player.IsSubmerged
		audioEngine.Update(dt)
		audioEngine.UpdateAudioStream()

		// Update Scheduled Block Physics (Smooth falling sand and leaf decay)
		world.TickScheduledPhysics(dt, chunkManager, player.SpawnBlockBreakParticles)

		// Throttled Grass Spreading, Growth & Random Block Ticks (10x per sec instead of 300x per sec)
		randomTickTimer += dt
		if randomTickTimer >= 0.15 {
			randomTickTimer = 0
			world.TickRandomBlocks(player.Pos, chunkManager)
		}

		// Water Splash SFX and Particles on entering/jumping in water
		if player.IsSwimming && !wasInWater {
			audioEngine.TriggerWaterSplash()
			splashPos := player.Pos
			splashPos.Y += 0.2
			for s := 0; s < 12; s++ {
				player.SpawnBlockBreakParticles(splashPos, voxel.BlockWater)
			}
		}
		wasInWater = player.IsSwimming

		// Throttled Water Flow & Falling Sand Simulation around player (4x per sec)
		waterSpreadTimer += dt
		if waterSpreadTimer >= 0.25 {
			waterSpreadTimer = 0
			pcx := int(math.Floor(float64(player.Pos.X)))
			pcy := int(math.Floor(float64(player.Pos.Y)))
			pcz := int(math.Floor(float64(player.Pos.Z)))
			for ox := -4; ox <= 4; ox++ {
				for oz := -4; oz <= 4; oz++ {
					for oy := -3; oy <= 3; oy++ {
						tx := pcx + ox
						ty := pcy + oy
						tz := pcz + oz
						b := world.GetBlock(tx, ty, tz)
						if voxel.IsWater(b) {
							world.SpreadWater(tx, ty, tz, chunkManager)
						} else if b == voxel.BlockSand {
							world.QueueSandFall(tx, ty, tz)
						}
					}
				}
			}
		}

		// =========================================================================
		// --- 3D RENDERING PIPELINE ---
		// =========================================================================
		skyCol := rl.NewColor(135, 206, 235, 255) // Azure Day
		if player.IsSubmerged {
			skyCol = rl.NewColor(8, 28, 65, 255) // Deep Ocean Vision
		} else if sunHeight < 0 {
			skyCol = rl.NewColor(10, 14, 28, 255) // Dark Night
		} else if sunHeight < 0.25 {
			skyCol = rl.NewColor(225, 125, 80, 255) // Golden Sunset
		}

		// Update GPU Chunk Shader Fog and Minecraft Dynamic Lighting
		totalGameTime += dt
		heldBlock := gui.GetActiveBlock()
		heldTorchLevel := float32(0)
		if heldBlock == voxel.BlockTorch {
			heldTorchLevel = 1.0
		}
		chunkManager.UpdateFogAndSky(skyCol, player.IsSubmerged, player.RLCamera.Position, sunAngle, sunHeight, totalGameTime, player.Pos, heldTorchLevel)

		rl.BeginDrawing()
		rl.ClearBackground(skyCol)

		rl.BeginMode3D(player.RLCamera)

		// 1. Celestial Sun and Moon (visible when above water)
		camPos := player.RLCamera.Position
		if !player.IsSubmerged {
			sunDist := float32(320.0)
			sunX := camPos.X + float32(math.Cos(float64(sunAngle)))*sunDist
			sunY := camPos.Y + sunHeight*sunDist
			sunZ := camPos.Z
			sunPos := rl.Vector3{X: sunX, Y: sunY, Z: sunZ}

			moonX := camPos.X - (sunX - camPos.X)
			moonY := camPos.Y - (sunY - camPos.Y)
			moonZ := camPos.Z
			moonPos := rl.Vector3{X: moonX, Y: moonY, Z: moonZ}

			// Textured Minecraft Sun Billboard (Alpha-blended, clean golden square)
			if sunTex.ID > 0 {
				srcSun := rl.NewRectangle(0, 0, float32(sunTex.Width), float32(sunTex.Height))
				rl.DrawBillboardRec(player.RLCamera, sunTex, srcSun, sunPos, rl.Vector2{X: 60.0, Y: 60.0}, rl.White)
			} else {
				rl.DrawCube(sunPos, 28.0, 28.0, 28.0, rl.NewColor(255, 255, 220, 255))
			}

			// Textured Minecraft Moon Billboard with 8 Lunar Phases
			if moonTex.ID > 0 {
				phase := dayCount % 8
				phaseCol := phase % 4
				phaseRow := phase / 4
				srcMoon := rl.NewRectangle(float32(phaseCol*32), float32(phaseRow*32), 32, 32)
				rl.DrawBillboardRec(player.RLCamera, moonTex, srcMoon, moonPos, rl.Vector2{X: 48.0, Y: 48.0}, rl.White)
			} else {
				rl.DrawCube(moonPos, 24.0, 24.0, 24.0, rl.NewColor(220, 230, 245, 255))
			}

			// Twinkling Night Stars
			if sunHeight < 0 {
				for s := 0; s < 45; s++ {
					starAngle := float64(s) * 0.14
					starDist := float32(300.0)
					sx := camPos.X + float32(math.Cos(starAngle*13.7))*starDist
					sy := camPos.Y + float32(math.Abs(math.Sin(starAngle*7.3)))*starDist*0.8 + 30.0
					sz := camPos.Z + float32(math.Sin(starAngle*11.1))*starDist
					rl.DrawCube(rl.Vector3{X: sx, Y: sy, Z: sz}, 1.2, 1.2, 1.2, rl.NewColor(240, 245, 255, 220))
				}
			}

			// 2. Drifting Voxel Cloud Layer at y = 42
			cloudBaseY := float32(42.0)
			cloudCol := rl.NewColor(255, 255, 255, 180)
			if sunHeight < 0 {
				cloudCol = rl.NewColor(40, 45, 65, 180)
			}
			camGridX := int(math.Floor(float64(camPos.X))) >> 5
			camGridZ := int(math.Floor(float64(camPos.Z))) >> 5

			for cx := -3; cx <= 3; cx++ {
				for cz := -3; cz <= 3; cz++ {
					wx := float32((camGridX+cx)*32) + cloudOffset
					wz := float32((camGridZ + cz) * 32)
					cloudHash := int(math.Abs(math.Sin(float64((camGridX+cx)*17+(camGridZ+cz)*31))*1000)) % 100
					if cloudHash < 60 {
						rl.DrawCube(rl.Vector3{X: wx, Y: cloudBaseY, Z: wz}, 28.0, 3.5, 28.0, cloudCol)
					}
				}
			}
		} else {
			// Floating aquatic bubbles around player when submerged!
			for b := 0; b < 18; b++ {
				bx := camPos.X + float32(math.Sin(float64(b*7)+float64(cloudOffset*2)))*3.5
				by := camPos.Y + float32(math.Mod(float64(b*2)+float64(cloudOffset*1.8), 4.0)) - 1.5
				bz := camPos.Z + float32(math.Cos(float64(b*11)+float64(cloudOffset*2)))*3.5
				rl.DrawCube(rl.Vector3{X: bx, Y: by, Z: bz}, 0.05, 0.05, 0.05, rl.NewColor(200, 235, 255, 180))
			}
		}

		// 3. 3D Infinite Voxel World (Multi-pass Opaque, Cutout & Translucent Water on GPU)
		chunkManager.Render3D(camPos, lookDir)

		rl.BeginShaderMode(chunkManager.CutoutShader)
		// 4. 3D Living Mobs (Zombies, Skeletons, Creepers, Pigs, Cows, Sheep)
		mobManager.Render3D()

		// 4b. 3D Dropped Items
		mobManager.RenderItems(atlas)
		rl.EndShaderMode()

		// 5. Targeted Block Wireframe Outline & 10-Stage Mining Cracks Overlay
		if rayResult.Hit && !isMenuOpen {
			voxel.DrawTargetBlockOutline(rayResult.BlockPos)
			if miningProgress > 0 {
				voxel.DrawMiningCracks(rayResult.BlockPos, miningProgress, atlas)
			}
		}

		// 6. Block Break Particles
		player.RenderParticles(atlas)

		// 7. 3D Steve Character Model (Rendered in 3rd Person views)
		player.Render3DSteveModel()

		// 8. First-Person Animated Right Hand / Textured 3D Held Block
		player.RenderHandAndHeldBlock(gui.GetActiveBlock(), atlas)

		rl.EndMode3D()

		// =========================================================================
		// --- 2D MINECRAFT HUD, INVENTORY & DEATH SCREEN OVERLAY ---
		// =========================================================================
		// Underwater Blue Vignette & Screen Overlay
		if player.IsSubmerged && !player.IsDead {
			rl.DrawRectangle(0, 0, screenWidth, screenHeight, rl.NewColor(10, 55, 135, 95))
		}

		gui.Render(player, world, atlas)

		// Top Controls & Mode Tooltip (Centered)
		modeTag := "[SURVIVAL]"
		if player.Mode == mcplayer.GameModeCreative {
			modeTag = "[CREATIVE]"
		}
		controlsText := fmt.Sprintf("%s  |  WASD: Move  |  SPACE: Jump  |  L-CLICK: Mine  |  R-CLICK: Place  |  R: Rotate  |  F4: Dist (%d)  |  F5: Save", modeTag, chunkManager.RenderRadius)
		cLen := rl.MeasureText(controlsText, 12)
		rl.DrawRectangleRounded(rl.NewRectangle(float32(screenWidth)*0.5-float32(cLen)*0.5-12, 10, float32(cLen)+24, 26), 0.3, 4, rl.NewColor(15, 20, 30, 210))
		rl.DrawText(controlsText, int32(screenWidth)/2-cLen/2, 16, 12, rl.NewColor(240, 245, 255, 255))

		// Render Distance Toast
		if renderDistToastTimer > 0 {
			renderDistToastTimer -= dt
			alpha := float32(1.0)
			if renderDistToastTimer < 0.5 {
				alpha = renderDistToastTimer / 0.5
			}
			mLen := rl.MeasureText(renderDistToastMsg, 14)
			boxW := float32(mLen + 28)
			boxX := float32(screenWidth)*0.5 - boxW*0.5
			rl.DrawRectangleRounded(rl.NewRectangle(boxX, 74, boxW, 28), 0.3, 4, rl.NewColor(20, 30, 45, uint8(alpha*220)))
			rl.DrawText(renderDistToastMsg, int32(boxX)+14, 80, 14, rl.NewColor(120, 230, 255, uint8(alpha*255)))
		}

		// Save Notification Toast
		if savedBannerTimer > 0 {
			savedBannerTimer -= dt
			alpha := float32(1.0)
			if savedBannerTimer < 0.5 {
				alpha = savedBannerTimer / 0.5
			}
			msg := "✓ Game Saved"
			mLen := rl.MeasureText(msg, 14)
			boxW := float32(mLen + 28)
			boxX := float32(screenWidth)*0.5 - boxW*0.5
			rl.DrawRectangleRounded(rl.NewRectangle(boxX, 42, boxW, 28), 0.3, 4, rl.NewColor(15, 25, 35, uint8(alpha*220)))
			rl.DrawText(msg, int32(boxX)+14, 48, 14, rl.NewColor(255, 215, 60, uint8(alpha*255)))
		}

		// Coordinates & Chunk Info (Top Left)
		coordStr := fmt.Sprintf("XYZ: %.1f / %.1f / %.1f  |  Chunk: %d, %d", player.Pos.X, player.Pos.Y, player.Pos.Z, int(player.Pos.X)>>4, int(player.Pos.Z)>>4)
		c3Len := rl.MeasureText(coordStr, 13)
		rl.DrawRectangleRounded(rl.NewRectangle(12, 10, float32(c3Len)+16, 26), 0.3, 4, rl.NewColor(15, 20, 30, 200))
		rl.DrawText(coordStr, 20, 16, 13, rl.RayWhite)

		// FPS Counter (Top Right)
		rl.DrawFPS(int32(screenWidth)-75, 16)

		rl.EndDrawing()
	}

	fmt.Println("GoCraft 3D exited cleanly.")
}
