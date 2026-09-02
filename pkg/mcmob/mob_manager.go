package mcmob

import (
	"math"
	"math/rand"

	"gocraft/pkg/voxel"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// MobManager coordinates mob spawning, updates, combat raycasts, and loot drops
type MobManager struct {
	Mobs         []*Mob
	MaxMobs      int
	SpawnTimer   float32
	ItemEntities []*ItemEntity

	// Mob Textures
	TexZombie   rl.Texture2D
	TexSkeleton rl.Texture2D
	TexCreeper  rl.Texture2D
	TexPig      rl.Texture2D
	TexCow      rl.Texture2D
	TexSheep    rl.Texture2D
}

// NewMobManager initializes the mob management system and loads textures
func NewMobManager() *MobManager {
	mm := &MobManager{
		Mobs:         make([]*Mob, 0, 32),
		MaxMobs:      24,
		ItemEntities: make([]*ItemEntity, 0, 128),
	}

	mm.TexZombie = rl.LoadTexture("assets/textures/entity/zombie/zombie.png")
	mm.TexSkeleton = rl.LoadTexture("assets/textures/entity/skeleton/skeleton.png")
	mm.TexCreeper = rl.LoadTexture("assets/textures/entity/creeper/creeper.png")
	mm.TexPig = rl.LoadTexture("assets/textures/entity/pig/pig.png")
	mm.TexCow = rl.LoadTexture("assets/textures/entity/cow/cow.png")
	mm.TexSheep = rl.LoadTexture("assets/textures/entity/sheep/sheep.png")

	return mm
}

// Unload frees up VRAM used by mob textures
func (mm *MobManager) Unload() {
	if mm.TexZombie.ID > 0 { rl.UnloadTexture(mm.TexZombie) }
	if mm.TexSkeleton.ID > 0 { rl.UnloadTexture(mm.TexSkeleton) }
	if mm.TexCreeper.ID > 0 { rl.UnloadTexture(mm.TexCreeper) }
	if mm.TexPig.ID > 0 { rl.UnloadTexture(mm.TexPig) }
	if mm.TexCow.ID > 0 { rl.UnloadTexture(mm.TexCow) }
	if mm.TexSheep.ID > 0 { rl.UnloadTexture(mm.TexSheep) }
}

// SpawnMob creates and tracks a new mob
func (mm *MobManager) SpawnMob(mType MobType, pos rl.Vector3) *Mob {
	m := NewMob(mType, pos)
	mm.Mobs = append(mm.Mobs, m)
	return m
}

// SpawnItem drops a new item entity into the world
func (mm *MobManager) SpawnItem(bType voxel.BlockType, count int, pos rl.Vector3, initialVel rl.Vector3) {
	if bType == voxel.BlockAir || count <= 0 {
		return
	}
	item := NewItemEntity(bType, count, pos, initialVel)
	mm.ItemEntities = append(mm.ItemEntities, item)
}

// Update ticks all active mobs and manages natural procedural spawning
func (mm *MobManager) Update(
	dt float32,
	playerPos rl.Vector3,
	playerHealth *float32,
	world *voxel.VoxelWorld,
	sunHeight float32,
	onDropLoot func(item voxel.BlockType, count int, pos rl.Vector3),
	onExplode func(pos rl.Vector3, radius float32),
	onPickupLoot func(item voxel.BlockType, count int) bool,
) {
	// 1. Natural Spawning
	mm.SpawnTimer += dt
	if mm.SpawnTimer >= 3.5 && len(mm.Mobs) < mm.MaxMobs {
		mm.SpawnTimer = 0

		// Pick random spawn point 18 to 36 blocks away from player
		angle := rand.Float32() * math.Pi * 2.0
		dist := 18.0 + rand.Float32()*18.0
		sx := int(math.Floor(float64(playerPos.X + float32(math.Sin(float64(angle)))*dist)))
		sz := int(math.Floor(float64(playerPos.Z + float32(math.Cos(float64(angle)))*dist)))

		sy := world.GetHighestBlock(sx, sz)
		if sy > 0 && sy < voxel.WorldHeight-2 {
			skyLight, blockLight := world.GetLightLevel(sx, sy+1, sz)
			spawnPos := rl.Vector3{X: float32(sx) + 0.5, Y: float32(sy + 1), Z: float32(sz) + 0.5}

			if sunHeight < 0 || (skyLight < 0.4 && blockLight < 0.4) {
				// Night or Dark Cave: Hostile Mobs
				r := rand.Float32()
				if r < 0.45 {
					mm.SpawnMob(MobZombie, spawnPos)
				} else if r < 0.75 {
					mm.SpawnMob(MobSkeleton, spawnPos)
				} else {
					mm.SpawnMob(MobCreeper, spawnPos)
				}
			} else {
				// Day Surface: Passive Animals on Grass
				surfaceBlock := world.GetBlock(sx, sy, sz)
				if surfaceBlock == voxel.BlockGrass {
					r := rand.Float32()
					if r < 0.38 {
						mm.SpawnMob(MobCow, spawnPos)
					} else if r < 0.74 {
						mm.SpawnMob(MobPig, spawnPos)
					} else {
						mm.SpawnMob(MobSheep, spawnPos)
					}
				}
			}
		}
	}

	// 2. Update all active mobs & process deaths / explosions
	activeMobs := mm.Mobs[:0]
	for _, m := range mm.Mobs {
		if m.IsDead {
			// Loot drop on death
			if onDropLoot != nil {
				switch m.Type {
				case MobZombie:
					onDropLoot(voxel.ItemRottenFlesh, 1+rand.Intn(2), m.Pos)
				case MobSkeleton:
					onDropLoot(voxel.ItemBone, 1+rand.Intn(2), m.Pos)
					onDropLoot(voxel.ItemArrow, 1+rand.Intn(3), m.Pos)
				case MobCreeper:
					if m.FuseTimer < 1.3 { // Only drop gunpowder if killed before exploding!
						onDropLoot(voxel.ItemGunpowder, 1+rand.Intn(2), m.Pos)
					}
				case MobPig:
					onDropLoot(voxel.ItemRawPorkchop, 1+rand.Intn(2), m.Pos)
				case MobCow:
					onDropLoot(voxel.ItemRawBeef, 1+rand.Intn(2), m.Pos)
				case MobSheep:
					onDropLoot(voxel.BlockWool, 1+rand.Intn(2), m.Pos)
					onDropLoot(voxel.ItemRawBeef, 1, m.Pos)
				}
			}
			continue
		}

		// Despawn mobs that are too far away (> 64 blocks)
		dx := m.Pos.X - playerPos.X
		dz := m.Pos.Z - playerPos.Z
		if dx*dx+dz*dz > 64.0*64.0 {
			continue
		}

		exploded := m.Update(dt, playerPos, playerHealth, world, sunHeight)
		if exploded {
			if onExplode != nil {
				onExplode(m.Pos, 3.5)
			}
			continue
		}

		activeMobs = append(activeMobs, m)
	}
	mm.Mobs = activeMobs

	// 3. Update Item Entities
	activeItems := mm.ItemEntities[:0]
	for _, item := range mm.ItemEntities {
		item.Update(dt, world)
		if item.LifeTimer <= 0 {
			continue // Despawn
		}

		// Pickup Logic
		if item.PickupDelay <= 0 && onPickupLoot != nil {
			dx := item.Pos.X - playerPos.X
			dy := item.Pos.Y - playerPos.Y
			dz := item.Pos.Z - playerPos.Z
			distSq := dx*dx + dy*dy + dz*dz
			if distSq < 1.5*1.5 {
				if onPickupLoot(item.Type, item.Count) {
					continue // Picked up, remove from world
				}
			}
		}

		activeItems = append(activeItems, item)
	}
	mm.ItemEntities = activeItems
}

// RaycastMob checks if the player's crosshair hits any mob
func (mm *MobManager) RaycastMob(rayOrigin, rayDir rl.Vector3, maxDist float32) (*Mob, float32, bool) {
	var closestMob *Mob
	closestDist := maxDist

	for _, m := range mm.Mobs {
		if m.IsDead {
			continue
		}

		// Mob AABB center
		mobCenter := rl.Vector3{
			X: m.Pos.X,
			Y: m.Pos.Y + m.Height*0.5,
			Z: m.Pos.Z,
		}

		// Vector from ray origin to mob
		toMob := rl.Vector3Subtract(mobCenter, rayOrigin)
		// Projection of toMob onto rayDir
		proj := toMob.X*rayDir.X + toMob.Y*rayDir.Y + toMob.Z*rayDir.Z
		if proj <= 0 || proj >= closestDist {
			continue
		}

		// Distance from ray line to mob center
		closestPoint := rl.Vector3Add(rayOrigin, rl.Vector3Scale(rayDir, proj))
		distToLine := rl.Vector3Distance(closestPoint, mobCenter)

		hitRadius := float32(m.Width * 0.75)
		if distToLine <= hitRadius {
			closestDist = proj
			closestMob = m
		}
	}

	if closestMob != nil {
		return closestMob, closestDist, true
	}
	return nil, 0, false
}

// Render3D draws all living mobs
func (mm *MobManager) Render3D() {
	for _, m := range mm.Mobs {
		m.Render3D(mm)
	}
}

// RenderItems draws dropped items in the world
func (mm *MobManager) RenderItems(atlas *voxel.TextureAtlas) {
	if atlas == nil || atlas.Texture.ID == 0 {
		return
	}

	rl.SetTexture(atlas.Texture.ID)
	rl.Begin(rl.Quads)

	for _, item := range mm.ItemEntities {
		bDef := voxel.BlockRegistry[item.Type]
		
		isFlatItem := bDef.IsTool || item.Type == voxel.ItemStick || item.Type == voxel.ItemDiamond ||
			item.Type == voxel.ItemCoal || item.Type == voxel.ItemIronIngot || item.Type == voxel.ItemGoldIngot ||
			item.Type == voxel.BlockTorch || bDef.IsFood || item.Type == voxel.ItemGunpowder ||
			item.Type == voxel.ItemBone || item.Type == voxel.ItemArrow

		cx := item.Pos.X
		cy := item.Pos.Y + 0.15 + float32(math.Sin(float64(item.HoverOffset)))*0.08
		cz := item.Pos.Z

		// Item rendering
		if isFlatItem {
			// Billboard-like rotation for flat items, or just rotating flat on ground
			u0, v0, u1, v1 := voxel.GetBlockTextureUVs(item.Type, voxel.FaceNorth)
			s := float32(0.18)

			// Simple rotation matrix around Y axis
			cosR := float32(math.Cos(float64(item.RotationY)))
			sinR := float32(math.Sin(float64(item.RotationY)))

			rotX := func(x, z float32) float32 { return x*cosR - z*sinR }
			rotZ := func(x, z float32) float32 { return x*sinR + z*cosR }

			rl.Color4ub(255, 255, 255, 255)

			// Front
			p1x, p1z := rotX(-s, 0), rotZ(-s, 0)
			p2x, p2z := rotX(s, 0), rotZ(s, 0)

			rl.TexCoord2f(u0, v1); rl.Vertex3f(cx+p1x, cy-s, cz+p1z)
			rl.TexCoord2f(u1, v1); rl.Vertex3f(cx+p2x, cy-s, cz+p2z)
			rl.TexCoord2f(u1, v0); rl.Vertex3f(cx+p2x, cy+s, cz+p2z)
			rl.TexCoord2f(u0, v0); rl.Vertex3f(cx+p1x, cy+s, cz+p1z)

			// Back
			rl.TexCoord2f(u1, v1); rl.Vertex3f(cx+p2x, cy-s, cz+p2z)
			rl.TexCoord2f(u0, v1); rl.Vertex3f(cx+p1x, cy-s, cz+p1z)
			rl.TexCoord2f(u0, v0); rl.Vertex3f(cx+p1x, cy+s, cz+p1z)
			rl.TexCoord2f(u1, v0); rl.Vertex3f(cx+p2x, cy+s, cz+p2z)

		} else {
			// 3D Mini Block
			hs := float32(0.12)
			
			cosR := float32(math.Cos(float64(item.RotationY)))
			sinR := float32(math.Sin(float64(item.RotationY)))

			vtx := func(x, y, z, u, v float32) {
				rx := x*cosR - z*sinR
				rz := x*sinR + z*cosR
				rl.TexCoord2f(u, v)
				rl.Vertex3f(cx+rx, cy+y, cz+rz)
			}

			// Top Face
			u0, v0, u1, v1 := voxel.GetBlockTextureUVs(item.Type, voxel.FaceTop)
			rl.Color4ub(255, 255, 255, 255)
			vtx(-hs, hs, hs, u0, v1)
			vtx(hs, hs, hs, u1, v1)
			vtx(hs, hs, -hs, u1, v0)
			vtx(-hs, hs, -hs, u0, v0)

			// Bottom
			u0, v0, u1, v1 = voxel.GetBlockTextureUVs(item.Type, voxel.FaceBottom)
			rl.Color4ub(160, 160, 160, 255)
			vtx(-hs, -hs, -hs, u0, v1)
			vtx(hs, -hs, -hs, u1, v1)
			vtx(hs, -hs, hs, u1, v0)
			vtx(-hs, -hs, hs, u0, v0)

			// Front
			u0, v0, u1, v1 = voxel.GetBlockTextureUVs(item.Type, voxel.FaceSouth)
			rl.Color4ub(220, 220, 220, 255)
			vtx(hs, -hs, hs, u1, v1)
			vtx(-hs, -hs, hs, u0, v1)
			vtx(-hs, hs, hs, u0, v0)
			vtx(hs, hs, hs, u1, v0)

			// Back
			u0, v0, u1, v1 = voxel.GetBlockTextureUVs(item.Type, voxel.FaceNorth)
			rl.Color4ub(200, 200, 200, 255)
			vtx(-hs, -hs, -hs, u0, v1)
			vtx(hs, -hs, -hs, u1, v1)
			vtx(hs, hs, -hs, u1, v0)
			vtx(-hs, hs, -hs, u0, v0)

			// Left
			u0, v0, u1, v1 = voxel.GetBlockTextureUVs(item.Type, voxel.FaceWest)
			rl.Color4ub(190, 190, 190, 255)
			vtx(-hs, -hs, -hs, u0, v1)
			vtx(-hs, -hs, hs, u1, v1)
			vtx(-hs, hs, hs, u1, v0)
			vtx(-hs, hs, -hs, u0, v0)

			// Right
			u0, v0, u1, v1 = voxel.GetBlockTextureUVs(item.Type, voxel.FaceEast)
			rl.Color4ub(210, 210, 210, 255)
			vtx(hs, -hs, hs, u0, v1)
			vtx(hs, -hs, -hs, u1, v1)
			vtx(hs, hs, -hs, u1, v0)
			vtx(hs, hs, hs, u0, v0)
		}
	}

	rl.End()
}
