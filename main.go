package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"racing_game/pkg/mcaudio"
	"racing_game/pkg/mcplayer"
	"racing_game/pkg/mcui"
	"racing_game/pkg/voxel"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	rl.SetConfigFlags(rl.FlagMsaa4xHint | rl.FlagWindowResizable | rl.FlagWindowHighdpi)
	rl.InitWindow(1920, 1080, "GoCraft 3D — Survival & Crafting Voxel Engine")
	defer rl.CloseWindow()

	screenWidth := int32(rl.GetScreenWidth())
	screenHeight := int32(rl.GetScreenHeight())
	if screenWidth <= 0 || screenHeight <= 0 {
		screenWidth = 1920
		screenHeight = 1080
	}

	rl.SetTargetFPS(0) // Uncapped high-refresh framerate (500+ FPS)

	// Lock & hide mouse cursor for seamless first-person mouse look
	rl.DisableCursor()
	defer rl.EnableCursor()

	// Initialize Dynamic Texture Atlas (Auto-detects 128x128 HD or 16x16)
	atlas := voxel.GenerateTextureAtlas()
	defer atlas.Unload()

	world := voxel.NewVoxelWorld()
	chunkManager := voxel.NewChunkManager(world, atlas)

	// Spawn Player on top of highest terrain block
	spawnY := float32(world.GetHighestBlock(0, 0) + 2)
	player := mcplayer.NewMCPlayer(rl.Vector3{X: 0.5, Y: spawnY, Z: 0.5})

	// Pre-load initial chunks around spawn
	chunkManager.UpdatePlayerPos(player.Pos)

	gui := mcui.NewInventoryGUI(screenWidth, screenHeight)
	audioEngine := mcaudio.NewMCAudioEngine()
	defer audioEngine.Close()

	// Day/Night Cycle
	timeOfDay := float32(0.2) // 0.0 to 1.0 (0.2 = Morning, 0.5 = Noon, 0.8 = Sunset, 1.0 = Midnight)
	dayLengthSec := float32(300.0)
	cloudOffset := float32(0.0)

	// Block Mining State
	var currentMiningPos rl.Vector3
	var miningProgress float32
	var miningCrunchTimer float32

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
		}
		cloudOffset += dt * 1.2

		// 'E' Key: Toggle Player Inventory / Close Workbench / Close Furnace
		if rl.IsKeyPressed(rl.KeyE) && !gui.IsWorkbenchOpen && !gui.IsFurnaceOpen && !player.IsDead {
			gui.IsInventoryOpen = !gui.IsInventoryOpen
			if gui.IsInventoryOpen {
				rl.EnableCursor()
			} else {
				rl.DisableCursor()
			}
		}

		// Close Workbench, Furnace, or Inventory with ESC
		if rl.IsKeyPressed(rl.KeyEscape) {
			if gui.IsWorkbenchOpen || gui.IsFurnaceOpen || gui.IsInventoryOpen {
				gui.IsWorkbenchOpen = false
				gui.IsFurnaceOpen = false
				gui.IsInventoryOpen = false
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

		// 'G' Key: Toggle GameMode (Survival <-> Creative)
		if rl.IsKeyPressed(rl.KeyG) && !player.IsDead {
			player.ToggleGameMode()
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

		// --- 2. UPDATE PLAYER MOVEMENT & INFINITE CHUNK STREAMING ---
		if !isMenuOpen {
			player.Update(dt, world)
			chunkManager.UpdatePlayerPos(player.Pos)
		}

		// --- 3. DDA VOXEL RAYCAST (BLOCK MINING, WORKBENCH & PLACING) ---
		cosY := float32(math.Cos(float64(player.Yaw)))
		sinY := float32(math.Sin(float64(player.Yaw)))
		cosP := float32(math.Cos(float64(player.Pitch)))
		sinP := float32(math.Sin(float64(player.Pitch)))
		lookDir := rl.Vector3{X: -sinY * cosP, Y: sinP, Z: -cosY * cosP}

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
						gui.AddItem(brokenType, 1)

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
							if brokenType == voxel.BlockOakLog {
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
			// Right Click: Interact with Crafting Table / Place Block
			if rl.IsMouseButtonPressed(rl.MouseButtonRight) {
				player.TriggerSwing()
				if rayResult.Hit {
					bx := int(rayResult.BlockPos.X)
					by := int(rayResult.BlockPos.Y)
					bz := int(rayResult.BlockPos.Z)
					clickedBlock := world.GetBlock(bx, by, bz)

					// Right click on Crafting Table or Furnace opens interactive GUI!
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

						heldBlock := gui.GetActiveBlock()
						if heldBlock != voxel.BlockAir {
							playerBlockX := int(math.Floor(float64(player.Pos.X)))
							playerBlockY1 := int(math.Floor(float64(player.Pos.Y)))
							playerBlockY2 := int(math.Floor(float64(player.Pos.Y + 1.0)))
							playerBlockZ := int(math.Floor(float64(player.Pos.Z)))

							overlapPlayer := (px == playerBlockX && pz == playerBlockZ && (py == playerBlockY1 || py == playerBlockY2))
							if !overlapPlayer {
								world.SetBlock(px, py, pz, heldBlock)
								chunkManager.MarkBlockDirty(px, pz)
								audioEngine.TriggerBlockPlace()

								// Sand Falling Physics on placement!
								if heldBlock == voxel.BlockSand {
									world.QueueSandFall(px, py, pz)
								}

								// Survival Mode: Consume 1 item from hotbar
								if player.Mode == mcplayer.GameModeSurvival {
									gui.ConsumeActiveItem()
								}
							}
						}
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

		// Periodic Water Flow & Falling Sand Simulation around player
		if rand.Intn(20) == 0 {
			pcx := int(math.Floor(float64(player.Pos.X)))
			pcy := int(math.Floor(float64(player.Pos.Y)))
			pcz := int(math.Floor(float64(player.Pos.Z)))
			for ox := -4; ox <= 4; ox++ {
				for oz := -4; oz <= 4; oz++ {
					for oy := -2; oy <= 2; oy++ {
						tx := pcx + ox
						ty := pcy + oy
						tz := pcz + oz
						b := world.GetBlock(tx, ty, tz)
						if b == voxel.BlockWater {
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
		sunAngle := timeOfDay * 2.0 * math.Pi
		sunHeight := float32(math.Sin(float64(sunAngle)))

		skyCol := rl.NewColor(135, 206, 235, 255) // Azure Day
		if player.IsSubmerged {
			skyCol = rl.NewColor(8, 28, 65, 255) // Deep Ocean Vision
		} else if sunHeight < 0 {
			skyCol = rl.NewColor(10, 14, 28, 255) // Dark Night
		} else if sunHeight < 0.25 {
			skyCol = rl.NewColor(225, 125, 80, 255) // Golden Sunset
		}

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

			// Square Minecraft Sun
			rl.DrawCube(rl.Vector3{X: sunX, Y: sunY, Z: sunZ}, 28.0, 28.0, 28.0, rl.NewColor(255, 255, 220, 255))
			// Square Minecraft Moon
			rl.DrawCube(rl.Vector3{X: camPos.X - (sunX - camPos.X), Y: camPos.Y - (sunY - camPos.Y), Z: camPos.Z}, 24.0, 24.0, 24.0, rl.NewColor(220, 230, 245, 255))

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

		// 4. Targeted Block Wireframe Outline & 10-Stage Mining Cracks Overlay
		if rayResult.Hit && !isMenuOpen {
			voxel.DrawTargetBlockOutline(rayResult.BlockPos)
			if miningProgress > 0 {
				voxel.DrawMiningCracks(rayResult.BlockPos, miningProgress, atlas)
			}
		}

		// 5. Block Break Particles
		player.RenderParticles()

		// 6. 3D Steve Character Model (Rendered in 3rd Person views)
		player.Render3DSteveModel()

		// 7. First-Person Animated Right Hand / Textured 3D Held Block
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
		controlsText := fmt.Sprintf("%s  |  WASD: Move  |  SPACE: Jump/Swim Up  |  SHIFT/C: Dive  |  LEFT CLICK: Mine  |  RIGHT CLICK: Place/Workbench  |  E: Crafting  |  G: Mode", modeTag)
		cLen := rl.MeasureText(controlsText, 12)
		rl.DrawRectangleRounded(rl.NewRectangle(float32(screenWidth)*0.5-float32(cLen)*0.5-12, 10, float32(cLen)+24, 26), 0.3, 4, rl.NewColor(15, 20, 30, 210))
		rl.DrawText(controlsText, int32(screenWidth)/2-cLen/2, 16, 12, rl.NewColor(240, 245, 255, 255))

		// Coordinates & Chunk Info (Top Left)
		coordStr := fmt.Sprintf("XYZ: %.1f / %.1f / %.1f  |  Chunk: %d, 