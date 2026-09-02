package mcplayer

import (
	"math"

	"racing_game/pkg/voxel"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// ViewPerspective defines 1st or 3rd person view
type ViewPerspective int

const (
	PerspectiveFirstPerson ViewPerspective = iota
	PerspectiveThirdPersonBack
	PerspectiveThirdPersonFront
)

// GameMode defines Survival vs Creative
type GameMode int

const (
	GameModeSurvival GameMode = iota
	GameModeCreative
)

// MCPlayer represents the Minecraft Steve avatar
type MCPlayer struct {
	Pos          rl.Vector3
	Vel          rl.Vector3
	Yaw          float32 // Horizontal look angle (radians)
	Pitch        float32 // Vertical look angle (radians)
	Perspective  ViewPerspective
	Mode         GameMode
	IsGrounded   bool
	IsSprinting  bool
	IsSneaking   bool
	IsSwimming   bool
	IsSubmerged  bool
	Health       float32 // 20 HP (10 Hearts)
	Hunger       float32 // 20 Food (10 Drumsticks)
	Oxygen       float32 // 10.0 (10 Air Bubbles)
	Level        int
	ExpProgress  float32
	FallDistance float32
	RegenTimer   float32
	DrownTimer   float32
	IsDead       bool
	DeathTimer   float32

	// Hand / Tool Swing Animation
	SwingTimer  float32
	WalkBobbing float32
	BaseFov     float32

	// Raylib Camera
	RLCamera rl.Camera3D

	// Particle System for block mining
	Particles []BlockParticle
}

// BlockParticle represents a small flying 3D voxel cube particle
type BlockParticle struct {
	Pos     rl.Vector3
	Vel     rl.Vector3
	Color   rl.Color
	Life    float32
	MaxLife float32
	Size    float32
}

// NewMCPlayer spawns player on top of highest terrain block
func NewMCPlayer(spawnPos rl.Vector3) *MCPlayer {
	p := &MCPlayer{
		Pos:         spawnPos,
		Yaw:         0,
		Pitch:       0,
		Perspective: PerspectiveFirstPerson,
		Mode:        GameModeSurvival,
		Health:      20,
		Hunger:      20,
		Oxygen:      10.0,
		Level:       30,
		ExpProgress: 0.65,
		BaseFov:     70.0,
		Particles:   make([]BlockParticle, 0, 128),
	}

	p.RLCamera = rl.Camera3D{
		Position:   rl.Vector3{X: spawnPos.X, Y: spawnPos.Y + 1.62, Z: spawnPos.Z},
		Target:     rl.Vector3{X: spawnPos.X, Y: spawnPos.Y + 1.62, Z: spawnPos.Z + 1.0},
		Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
		Fovy:       70.0,
		Projection: rl.CameraPerspective,
	}

	return p
}

// TogglePerspective cycles with F5 key
func (p *MCPlayer) TogglePerspective() {
	p.Perspective = (p.Perspective + 1) % 3
}

// ToggleGameMode switches between Survival and Creative
func (p *MCPlayer) ToggleGameMode() {
	if p.Mode == GameModeSurvival {
		p.Mode = GameModeCreative
		p.Health = 20
		p.Hunger = 20
		p.Oxygen = 10.0
	} else {
		p.Mode = GameModeSurvival
	}
}

// TriggerSwing triggers arm swinging animation
func (p *MCPlayer) TriggerSwing() {
	p.SwingTimer = 0.25
}

// SpawnBlockBreakParticles creates bursting block voxel debris
func (p *MCPlayer) SpawnBlockBreakParticles(pos rl.Vector3, bType voxel.BlockType) {
	bDef := voxel.BlockRegistry[bType]
	colors := []rl.Color{bDef.TopColor, bDef.SideColor, bDef.BottomColor}

	for i := 0; i < 16; i++ {
		col := colors[i%len(colors)]
		rx := float32(i%4)*0.22 - 0.33
		ry := float32((i/4)%4)*0.22 - 0.33
		rz := float32((i*7)%4)*0.22 - 0.33

		vx := rx*4.5 + float32(math.Sin(float64(i*3)))*1.5
		vy := 2.5 + float32(math.Cos(float64(i*5)))*2.0
		vz := rz*4.5 + float32(math.Cos(float64(i*3)))*1.5

		p.Particles = append(p.Particles, BlockParticle{
			Pos:     rl.Vector3{X: pos.X + 0.5 + rx, Y: pos.Y + 0.5 + ry, Z: pos.Z + 0.5 + rz},
			Vel:     rl.Vector3{X: vx, Y: vy, Z: vz},
			Color:   col,
			Life:    0.6,
			MaxLife: 0.6,
			Size:    0.08 + float32(i%3)*0.03,
		})
	}
}

// Update handles mouse look, WASD movement, voxel AABB collisions, jumping, camera, survival, and particles
func (p *MCPlayer) Update(dt float32, world *voxel.VoxelWorld) {
	if dt <= 0 || dt > 0.05 {
		dt = 0.016667
	}

	if p.Health <= 0 {
		p.Health = 0
		p.IsDead = true
		p.DeathTimer += dt
		p.Vel = rl.Vector3{}
		rl.EnableCursor()
		p.updateCamera(dt)
		return
	}

	if p.SwingTimer > 0 {
		p.SwingTimer -= dt
		if p.SwingTimer < 0 {
			p.SwingTimer = 0
		}
	}

	// 1. Mouse Look Camera
	mouseDelta := rl.GetMouseDelta()
	mouseSens := float32(0.0028)

	p.Yaw -= mouseDelta.X * mouseSens
	p.Pitch -= mouseDelta.Y * mouseSens

	// Clamp pitch [-89° .. +89°]
	if p.Pitch > 89.0*rl.Deg2rad {
		p.Pitch = 89.0 * rl.Deg2rad
	} else if p.Pitch < -89.0*rl.Deg2rad {
		p.Pitch = -89.0 * rl.Deg2rad
	}

	// Check if in water
	feetBlock := world.GetBlock(int(math.Floor(float64(p.Pos.X))), int(math.Floor(float64(p.Pos.Y))), int(math.Floor(float64(p.Pos.Z))))
	waistBlock := world.GetBlock(int(math.Floor(float64(p.Pos.X))), int(math.Floor(float64(p.Pos.Y+0.8))), int(math.Floor(float64(p.Pos.Z))))
	headBlock := world.GetBlock(int(math.Floor(float64(p.Pos.X))), int(math.Floor(float64(p.Pos.Y+1.62))), int(math.Floor(float64(p.Pos.Z))))

	p.IsSwimming = (feetBlock == voxel.BlockWater || waistBlock == voxel.BlockWater || headBlock == voxel.BlockWater)
	p.IsSubmerged = (headBlock == voxel.BlockWater)

	// 2. Movement Inputs (WASD)
	cosY := float32(math.Cos(float64(p.Yaw)))
	sinY := float32(math.Sin(float64(p.Yaw)))

	forward := rl.Vector3{X: -sinY, Y: 0, Z: -cosY}
	right := rl.Vector3{X: cosY, Y: 0, Z: -sinY}

	moveVec := rl.Vector3{}
	isMoving := false

	if rl.IsKeyDown(rl.KeyW) {
		moveVec.X += forward.X
		moveVec.Z += forward.Z
		isMoving = true
	}
	if rl.IsKeyDown(rl.KeyS) {
		moveVec.X -= forward.X
		moveVec.Z -= forward.Z
		isMoving = true
	}
	if rl.IsKeyDown(rl.KeyA) {
		moveVec.X -= right.X
		moveVec.Z -= right.Z
		isMoving = true
	}
	if rl.IsKeyDown(rl.KeyD) {
		moveVec.X += right.X
		moveVec.Z += right.Z
		isMoving = true
	}

	// Sprint & Sneak
	p.IsSprinting = rl.IsKeyDown(rl.KeyLeftControl) || (rl.IsKeyDown(rl.KeyLeftShift) && isMoving)
	p.IsSneaking = rl.IsKeyDown(rl.KeyC)

	speed := float32(4.35) // Normal walk speed
	if p.IsSwimming {
		// Water drag / Swimming speed
		if p.IsSprinting {
			speed = 4.2 // Fast swimming
		} else {
			speed = 2.4 // Gentle wading
		}
	} else if p.IsSprinting {
		speed = 6.8 // Sprint speed
		if p.Mode == GameModeSurvival {
			p.Hunger -= dt * 0.08
			if p.Hunger < 0 {
				p.Hunger = 0
			}
		}
	} else if p.IsSneaking {
		speed = 1.6 // Sneak speed
	}

	lenSq := moveVec.X*moveVec.X + moveVec.Z*moveVec.Z
	if lenSq > 0.001 {
		lenM := float32(math.Sqrt(float64(lenSq)))
		moveVec.X = (moveVec.X / lenM) * speed
		moveVec.Z = (moveVec.Z / lenM) * speed
		if p.IsSwimming {
			p.WalkBobbing += dt * speed * 1.5
		} else {
			p.WalkBobbing += dt * speed * 2.2
		}
	} else {
		p.WalkBobbing *= 0.85
	}

	// 3. Jump & Water Buoyancy / Diving
	if p.IsSwimming {
		p.FallDistance = 0 // Water negates fall damage!

		if rl.IsKeyDown(rl.KeySpace) {
			// Swim upward toward surface
			p.Vel.Y = 2.8

			// Surface hop onto land
			if !p.IsSubmerged && waistBlock == voxel.BlockWater {
				shoreX := int(math.Floor(float64(p.Pos.X + forward.X*0.6)))
				shoreZ := int(math.Floor(float64(p.Pos.Z + forward.Z*0.6)))
				shoreY := int(math.Floor(float64(p.Pos.Y + 0.5)))
				if voxel.BlockRegistry[world.GetBlock(shoreX, shoreY, shoreZ)].IsSolid {
					p.Vel.Y = 6.8 // Hop onto riverbank / beach!
				}
			}
		} else if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyC) {
			// Dive downward
			p.Vel.Y = -2.6
		} else {
			// Natural slow sinking / neutral buoyancy
			p.Vel.Y = -1.0
		}
	} else {
		if rl.IsKeyDown(rl.KeySpace) && p.IsGrounded {
			p.Vel.Y = 8.5
			p.IsGrounded = false
		}
		p.Vel.Y -= 26.0 * dt
		if p.Vel.Y < -35.0 {
			p.Vel.Y = -35.0
		}
	}

	// Track Fall Distance for Survival Damage
	if !p.IsGrounded && p.Vel.Y < 0 && !p.IsSwimming {
		p.FallDistance += -p.Vel.Y * dt
	}

	// 4. Voxel AABB Collisions & Resolution
	p.moveWithVoxelCollision(moveVec.X*dt, p.Vel.Y*dt, moveVec.Z*dt, world)

	// 5. Survival Mechanics (Drowning, Hunger, Regen)
	if p.Mode == GameModeSurvival {
		if p.IsSubmerged {
			p.Oxygen -= dt * 0.8
			if p.Oxygen <= 0 {
				p.Oxygen = 0
				p.DrownTimer += dt
				if p.DrownTimer >= 1.0 {
					p.Health -= 2.0
					p.DrownTimer = 0
					if p.Health < 0 {
						p.Health = 0
					}
				}
			}
		} else {
			p.Oxygen = 10.0
			p.DrownTimer = 0
		}

		if p.Hunger >= 18 && p.Health < 20 {
			p.RegenTimer += dt
			if p.RegenTimer >= 3.5 {
				p.Health += 1.0
				if p.Health > 20 {
					p.Health = 20
				}
				p.Hunger -= 0.5
				p.RegenTimer = 0
			}
		} else if p.Hunger == 0 && p.Health > 1.0 {
			p.RegenTimer += dt
			if p.RegenTimer >= 4.0 {
				p.Health -= 1.0
				p.RegenTimer = 0
			}
		}
	}

	// 6. Update Particles
	for i := len(p.Particles) - 1; i >= 0; i-- {
		part := &p.Particles[i]
		part.Life -= dt
		if part.Life <= 0 {
			p.Particles = append(p.Particles[:i], p.Particles[i+1:]...)
			continue
		}
		part.Vel.Y -= 14.0 * dt
		part.Pos.X += part.Vel.X * dt
		part.Pos.Y += part.Vel.Y * dt
		part.Pos.Z += part.Vel.Z * dt
	}

	// 7. Update Camera
	p.updateCamera(dt)
}

// moveWithVoxelCollision applies AABB sliding collision on X, Y, Z axes
func (p *MCPlayer) moveWithVoxelCollision(dx, dy, dz float32, world *voxel.VoxelWorld) {
	playerW := float32(0.6)
	playerH := float32(1.8)
	if p.IsSneaking {
		playerH = 1.5
	}
	halfW := playerW * 0.5

	wasGrounded := p.IsGrounded

	// Y-axis Movement & Collision
	newY := p.Pos.Y + dy
	if p.checkAABBCollision(p.Pos.X, newY, p.Pos.Z, halfW, playerH, world) {
		if dy < 0 {
			p.IsGrounded = true
			p.Vel.Y = 0
			p.Pos.Y = float32(math.Floor(float64(newY))) + 1.0

			// Fall damage in Survival
			if !wasGrounded && p.Mode == GameModeSurvival && p.FallDistance > 3.8 {
				damage := (p.FallDistance - 3.5) * 1.0
				p.Health -= damage
				if p.Health < 0 {
					p.Health = 0
				}
			}
			p.FallDistance = 0
		} else {
			p.Vel.Y = 0 // Hit ceiling
		}
	} else {
		p.Pos.Y = newY
		p.IsGrounded = false
	}

	// X-axis Movement & Collision (with Step-up on 1-block obstacles)
	newX := p.Pos.X + dx
	if p.checkAABBCollision(newX, p.Pos.Y, p.Pos.Z, halfW, playerH, world) {
		if p.IsGrounded && !p.checkAABBCollision(newX, p.Pos.Y+0.6, p.Pos.Z, halfW, playerH, world) {
			p.Pos.Y += 0.6
			p.Pos.X = newX
		}
	} else {
		p.Pos.X = newX
	}

	// Z-axis Movement & Collision
	newZ := p.Pos.Z + dz
	if p.checkAABBCollision(p.Pos.X, p.Pos.Y, newZ, halfW, playerH, world) {
		if p.IsGrounded && !p.checkAABBCollision(p.Pos.X, p.Pos.Y+0.6, newZ, halfW, playerH, world) {
			p.Pos.Y += 0.6
			p.Pos.Z = newZ
		}
	} else {
		p.Pos.Z = newZ
	}
}

func (p *MCPlayer) checkAABBCollision(px, py, pz, halfW, height float32, world *voxel.VoxelWorld) bool {
	minX := int(math.Floor(float64(px - halfW)))
	maxX := int(math.Floor(float64(px + halfW)))
	minY := int(math.Floor(float64(py)))
	maxY := int(math.Floor(float64(py + height - 0.05)))
	minZ := int(math.Floor(float64(pz - halfW)))
	maxZ := int(math.Floor(float64(pz + halfW)))

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for z := minZ; z <= maxZ; z++ {
				bType := world.GetBlock(x, y, z)
				if voxel.BlockRegistry[bType].IsSolid {
					return true
				}
			}
		}
	}
	return false
}

// updateCamera positions the camera for 1st person or 3rd person views
func (p *MCPlayer) updateCamera(dt float32) {
	cosY := float32(math.Cos(float64(p.Yaw)))
	sinY := float32(math.Sin(float64(p.Yaw)))
	cosP := float32(math.Cos(float64(p.Pitch)))
	sinP := float32(math.Sin(float64(p.Pitch)))

	lookDir := rl.Vector3{
		X: -sinY * cosP,
		Y: sinP,
		Z: -cosY * cosP,
	}

	eyeHeight := float32(1.62)
	if p.IsSneaking {
		eyeHeight = 1.35
	}

	bobY := float32(math.Sin(float64(p.WalkBobbing))) * 0.04
	bobX := float32(math.Cos(float64(p.WalkBobbing*0.5))) * 0.025

	// Sprint FOV effect
	targetFov := p.BaseFov
	if p.IsSprinting {
		targetFov = 82.0
	}
	p.RLCamera.Fovy += (targetFov - p.RLCamera.Fovy) * dt * 10.0

	eyePos := rl.Vector3{X: p.Pos.X + bobX, Y: p.Pos.Y + eyeHeight + bobY, Z: p.Pos.Z}

	if p.IsDead {
		tiltAngle := float32(math.Min(math.Pi*0.45, float64(p.DeathTimer*3.2)))
		p.RLCamera.Up = rl.Vector3{
			X: float32(math.Sin(float64(tiltAngle))),
			Y: float32(math.Cos(float64(tiltAngle))),
			Z: 0,
		}
	} else {
		p.RLCamera.Up = rl.Vector3{X: 0, Y: 1, Z: 0}
	}

	switch p.Perspective {
	case PerspectiveFirstPerson:
		p.RLCamera.Position = eyePos
		p.RLCamera.Target = rl.Vector3{
			X: eyePos.X + lookDir.X,
			Y: eyePos.Y + lookDir.Y,
			Z: eyePos.Z + lookDir.Z,
		}

	case PerspectiveThirdPersonBack:
		dist := float32(3.6)
		p.RLCamera.Position = rl.Vector3{
			X: eyePos.X - lookDir.X*dist,
			Y: eyePos.Y - lookDir.Y*dist + 0.4,
			Z: eyePos.Z - lookDir.Z*dist,
		}
		p.RLCamera.Target = eyePos

	case PerspectiveThirdPersonFront:
		dist := float32(3.6)
		p.RLCamera.Position = rl.Vector3{
			X: eyePos.X + lookDir.X*dist,
			Y: eyePos.Y + lookDir.Y*dist + 0.4,
			Z: eyePos.Z + lookDir.Z*dist,
		}
		p.RLCamera.Target = eyePos
	}
}

// RenderParticles renders all active flying block break debris
func (p *MCPlayer) RenderParticles() {
	for i := 0; i < len(p.Particles); i++ {
		part := &p.Particles[i]
		rl.DrawCube(part.Pos, part.Size, part.Size, part.Size, part.Color)
	}
}

// RenderHandAndHeldBlock draws the authentic Minecraft first-person viewmodel (textured arm, 2D tools, 3D blocks)
func (p *MCPlayer) RenderHandAndHeldBlock(heldBlock voxel.BlockType, atlas *voxel.TextureAtlas) {
	if p.Perspective != PerspectiveFirstPerson || atlas == nil || atlas.Texture.ID == 0 {
		return
	}

	cam := p.RLCamera
	forward := rl.Vector3Normalize(rl.Vector3Subtract(cam.Target, cam.Position))
	right := rl.Vector3Normalize(rl.Vector3CrossProduct(forward, cam.Up))
	up := rl.Vector3CrossProduct(right, forward)

	// Swing animation progress
	swingProgress := float32(0.0)
	if p.SwingTimer > 0 {
		swingProgress = 1.0 - (p.SwingTimer / 0.25)
	}
	swing := float32(math.Sin(float64(swingProgress * math.Pi)))

	// Walk bobbing
	bobX := float32(math.Sin(float64(p.WalkBobbing*0.5))) * 0.008
	bobY := float32(math.Cos(float64(p.WalkBobbing))) * 0.008

	// Helper to emit a world-space vertex from camera-relative coordinate (lx, ly, lz)
	// lx: screen right, ly: screen up, lz: forward into screen
	vtx := func(lx, ly, lz float32, u, v float32) {
		wx := cam.Position.X + right.X*lx + up.X*ly + forward.X*lz
		wy := cam.Position.Y + right.Y*lx + up.Y*ly + forward.Y*lz
		wz := cam.Position.Z + right.Z*lx + up.Z*ly + forward.Z*lz
		rl.TexCoord2f(u, v)
		rl.Vertex3f(wx, wy, wz)
	}

	bDef := voxel.BlockRegistry[heldBlock]
	isFlatItem := bDef.IsTool || heldBlock == voxel.ItemStick || heldBlock == voxel.ItemDiamond ||
		heldBlock == voxel.ItemCoal || heldBlock == voxel.ItemIronIngot || heldBlock == voxel.ItemGoldIngot ||
		heldBlock == voxel.BlockTorch

	shirtU0, shirtV0, shirtU1, shirtV1 := float32(6)/16.0, float32(2)/16.0, float32(7)/16.0, float32(3)/16.0
	skinU0, skinV0, skinU1, skinV1 := float32(7)/16.0, float32(2)/16.0, float32(8)/16.0, float32(3)/16.0

	drawBox := func(x0, y0, z0, x1, y1, z1 float32, u0, v0, u1, v1 float32, col rl.Color) {
		rl.Color4ub(col.R, col.G, col.B, col.A)

		// Top Face (+Y)
		vtx(x0, y1, z1, u0, v1)
		vtx(x1, y1, z1, u1, v1)
		vtx(x1, y1, z0, u1, v0)
		vtx(x0, y1, z0, u0, v0)

		// Bottom Face (-Y)
		vtx(x0, y0, z0, u0, v1)
		vtx(x1, y0, z0, u1, v1)
		vtx(x1, y0, z1, u1, v0)
		vtx(x0, y0, z1, u0, v0)

		// Front (+Z)
		vtx(x1, y0, z1, u1, v1)
		vtx(x0, y0, z1, u0, v1)
		vtx(x0, y1, z1, u0, v0)
		vtx(x1, y1, z1, u1, v0)

		// Back (-Z)
		vtx(x0, y0, z0, u0, v1)
		vtx(x1, y0, z0, u1, v1)
		vtx(x1, y1, z0, u1, v0)
		vtx(x0, y1, z0, u0, v0)

		// Left (-X)
		vtx(x0, y0, z0, u0, v1)
		vtx(x0, y0, z1, u1, v1)
		vtx(x0, y1, z1, u1, v0)
		vtx(x0, y1, z0, u0, v0)

		// Right (+X)
		vtx(x1, y0, z1, u0, v1)
		vtx(x1, y0, z0, u1, v1)
		vtx(x1, y1, z0, u1, v0)
		vtx(x1, y1, z1, u0, v0)
	}

	rl.Begin(rl.Quads)
	rl.SetTexture(atlas.Texture.ID)

	if heldBlock == voxel.BlockAir {
		// 1. Empty Steve Arm / Fist
		ax := 0.24 + bobX - swing*0.06
		ay := -0.22 + bobY - swing*0.08
		az := 0.36 + swing*0.12

		// Sleeve (Cyan Shirt)
		drawBox(ax-0.05, ay-0.08, az-0.12, ax+0.05, ay+0.04, az+0.04, shirtU0, shirtV0, shirtU1, shirtV1, rl.White)
		// Forearm & Fist (Skin)
		drawBox(ax-0.04, ay-0.07, az+0.04, ax+0.04, ay+0.03, az+0.16, skinU0, skinV0, skinU1, skinV1, rl.White)
	} else if isFlatItem {
		// 2. 2D Tool Sprite (Pickaxes, Swords, Axes, Shovels, Sticks)
		cx := 0.22 + bobX - swing*0.08
		cy := -0.15 + bobY - swing*0.12
		cz := 0.44 + swing*0.10

		u0, v0, u1, v1 := voxel.GetBlockTextureUVs(heldBlock, voxel.FaceNorth)
		s := float32(0.16)

		rl.Color4ub(255, 255, 255, 255)

		// Front Face
		vtx(cx-s*0.8, cy-s*0.9, cz-0.04, u0, v1)
		vtx(cx+s*0.9, cy-s*0.4, cz+0.04, u1, v1)
		vtx(cx+s*0.5, cy+s*0.9, cz+0.08, u1, v0)
		vtx(cx-s*0.9, cy+s*0.4, cz, u0, v0)

		// Back Face
		vtx(cx+s*0.9, cy-s*0.4, cz+0.04, u1, v1)
		vtx(cx-s*0.8, cy-s*0.9, cz-0.04, u0, v1)
		vtx(cx-s*0.9, cy+s*0.4, cz, u0, v0)
		vtx(cx+s*0.5, cy+s*0.9, cz+0.08, u1, v0)

		// Arm holding the base
		drawBox(cx+0.04, cy-0.14, cz-0.14, cx+0.12, cy-0.04, cz, shirtU0, shirtV0, shirtU1, shirtV1, rl.White)
		drawBox(cx+0.03, cy-0.12, cz, cx+0.09, cy-0.03, cz+0.08, skinU0, skinV0, skinU1, skinV1, rl.White)
	} else {
		// 3. Mini 3D Voxel Block
		cx := 0.22 + bobX - swing*0.06
		cy := -0.16 + bobY - swing*0.10
		cz := 0.44 + swing*0.10

		hs := float32(0.065)

		// Top Face (+Y)
		u0, v0, u1, v1 := voxel.GetBlockTextureUVs(heldBlock, voxel.FaceTop)
		rl.Color4ub(255, 255, 255, 255)
		vtx(cx-hs, cy+hs, cz+hs, u0, v1)
		vtx(cx+hs, cy+hs, cz+hs, u1, v1)
		vtx(cx+hs, cy+hs, cz-hs, u1, v0)
		vtx(cx-hs, cy+hs, cz-hs, u0, v0)

		// Bottom Face (-Y)
		u0, v0, u1, v1 = voxel.GetBlockTextureUVs(heldBlock, voxel.FaceBottom)
		rl.Color4ub(160, 160, 160, 255)
		vtx(cx-hs, cy-hs, cz-hs, u0, v1)
		vtx(cx+hs, cy-hs, cz-hs, u1, v1)
		vtx(cx+hs, cy-hs, cz+hs, u1, v0)
		vtx(cx-hs, cy-hs, cz+hs, u0, v0)

		// Front Face (+Z)
		u0, v0, u1, v1 = voxel.GetBlockTextureUVs(heldBlock, voxel.FaceSouth)
		rl.Color4ub(220, 220, 220, 255)
		vtx(cx+hs, cy-hs, cz+hs, u1, v1)
		vtx(cx-hs, cy-hs, cz+hs, u0, v1)
		vtx(cx-hs, cy+hs, cz+hs, u0, v0)
		vtx(cx+hs, cy+hs, cz+hs, u1, v0)

		// Back Face (-Z)
		u0, v0, u1, v1 = voxel.GetBlockTextureUVs(heldBlock, voxel.FaceNorth)
		rl.Color4ub(200, 200, 200, 255)
		vtx(cx-hs, cy-hs, cz-hs, u0, v1)
		vtx(cx+hs, cy-hs, cz-hs, u1, v1)
		vtx(cx+hs, cy+hs, cz-hs, u1, v0)
		vtx(cx-hs, cy+hs, cz-hs, u0, v0)

		// Left Face (-X)
		u0, v0, u1, v1 = voxel.GetBlockTextureUVs(heldBlock, voxel.FaceWest)
		rl.Color4ub(190, 190, 190, 255)
		vtx(cx-hs, cy-hs, cz-hs, u0, v1)
		vtx(cx-hs, cy-hs, cz+hs, u1, v1)
		vtx(cx-hs, cy+hs, cz+hs, u1, v0)
		vtx(cx-hs, cy+hs, cz-hs, u0, v0)

		// Right Face (+X)
		u0, v0, u1, v1 = voxel.GetBlockTextureUVs(heldBlock, voxel.FaceEast)
		rl.Color4ub(210, 210, 210, 255)
		vtx(cx+hs, cy-hs, cz+hs, u0, v1)
		vtx(cx+hs, cy-hs, cz-hs, u1, v1)
		vtx(cx+hs, cy+hs, cz-hs, u1, v0)
		vtx(cx+hs, cy+hs, cz+hs, u0, v0)

		// Arm holding block
		drawBox(cx+0.04, cy-0.14, cz-0.12, cx+0.12, cy-0.04, cz+0.02, shirtU0, shirtV0, shirtU1, shirtV1, rl.White)
		drawBox(cx+0.03, cy-0.12, cz+0.02, cx+0.09, cy-0.03, cz+0.10, skinU0, skinV0, skinU1, skinV1, rl.White)
	}

	rl.End()
}

// Render3DSteveModel draws the 3D Steve character in 3rd person view
func (p *MCPlayer) Render3DSteveModel() {
	if p.Perspective == PerspectiveFirstPerson {
		return
	}

	rl.PushMatrix()
	rl.Translatef(p.Pos.X, p.Pos.Y, p.Pos.Z)
	rl.Rotatef(-p.Yaw*rl.Rad2deg, 0, 1, 0)

	skinCol := rl.NewColor(198, 140, 102, 255)
	cyanShirt := rl.NewColor(0, 175, 185, 255) // Cyan Steve Shirt
	blueJeans := rl.NewColor(42, 52, 135, 255) // Blue Jeans
	hairCol := rl.NewColor(75, 48, 28, 255)
	eyeWhite := rl.NewColor(245, 245, 245, 255)
	eyeBlue := rl.NewColor(45, 75, 195, 255)
	shoeCol := rl.NewColor(80, 80, 85, 255)

	// Head (8x8 pixels)
	rl.DrawCube(rl.Vector3{X: 0, Y: 1.62, Z: 0}, 0.48, 0.48, 0.48, skinCol)
	rl.DrawCube(rl.Vector3{X: 0, Y: 1.80, Z: 0}, 0.50, 0.14, 0.50, hairCol)

	// Face Features (Eyes & Mouth)
	rl.DrawCube(rl.Vector3{X: -0.12, Y: 1.62, Z: 0.245}, 0.08, 0.06, 0.02, eyeWhite)
	rl.DrawCube(rl.Vector3{X: 0.12, Y: 1.62, Z: 0.245}, 0.08, 0.06, 0.02, eyeWhite)
	rl.DrawCube(rl.Vector3{X: -0.09, Y: 1.62, Z: 0.25}, 0.04, 0.06, 0.02, eyeBlue)
	rl.DrawCube(rl.Vector3{X: 0.09, Y: 1.62, Z: 0.25}, 0.04, 0.06, 0.02, eyeBlue)

	// Torso & Cyan Shirt
	rl.DrawCube(rl.Vector3{X: 0, Y: 1.08, Z: 0}, 0.48, 0.68, 0.24, cyanShirt)

	// Legs
	legSwing := float32(math.Sin(float64(p.WalkBobbing))) * 0.24
	rl.DrawCube(rl.Vector3{X: -0.13, Y: 0.44, Z: legSwing}, 0.22, 0.60, 0.22, blueJeans)
	rl.DrawCube(rl.Vector3{X: 0.13, Y: 0.44, Z: -legSwing}, 0.22, 0.60, 0.22, blueJeans)
	// Shoes
	rl.DrawCube(rl.Vector3{X: -0.13, Y: 0.08, Z: legSwing}, 0.22, 0.16, 0.22, shoeCol)
	rl.DrawCube(rl.Vector3{X: 0.13, Y: 0.08, Z: -legSwing}, 0.22, 0.16, 0.22, shoeCol)

	// Arms (Cyan Sleeve + Skin Hand)
	rl.DrawCube(rl.Vector3{X: -0.36, Y: 1.18, Z: -legSwing}, 0.22, 0.32, 0.22, cyanShirt)
	rl.DrawCube(rl.Vector3{X: -0.36, Y: 0.94, Z: -legSwing}, 0.20, 0.36, 0.20, skinCol)

	rl.DrawCube(rl.Vector3{X: 0.36, Y: 1.18, Z: legSwing}, 0.22, 0.32, 0.22, cyanShirt)
	rl.DrawCube(rl.Vector3{X: 0.36, Y: 0.94, Z: legSwing}, 0.20, 0.36, 0.20, skinCol)

	rl.PopMatrix()
}
