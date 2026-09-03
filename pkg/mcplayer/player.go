package mcplayer

import (
	"math"
	"math/rand"

	"gocraft/pkg/voxel"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// ---------------------------------------------------------------------------
// Float32 math helpers — eliminate noisy float64 round-trips.
// ---------------------------------------------------------------------------

func floor32(v float32) float32 { return float32(math.Floor(float64(v))) }
func clamp32(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
func sin32(v float32) float32 { return float32(math.Sin(float64(v))) }
func cos32(v float32) float32 { return float32(math.Cos(float64(v))) }
func sqrt32(v float32) float32 { return float32(math.Sqrt(float64(v))) }

// ---------------------------------------------------------------------------
// Physics & gameplay constants.
// ---------------------------------------------------------------------------

const (
	// Mouse / camera.
	MouseSensitivity = float32(0.0028)
	MaxPitch         = 89.0 * rl.Deg2rad

	// Movement speeds (blocks/sec).
	WalkSpeed       = float32(4.35)
	SprintSpeed     = float32(6.8)
	SneakSpeed      = float32(1.6)
	SwimSpeed       = float32(2.4)
	SwimSprintSpeed = float32(4.2)

	// Vertical physics.
	JumpVelocity    = float32(8.5)
	Gravity         = float32(26.0)
	TerminalVelocity = float32(35.0)
	SwimUpSpeed     = float32(2.8)
	SwimDiveSpeed   = float32(2.6)
	SwimSinkSpeed   = float32(1.0)
	SurfaceHopSpeed = float32(6.8)

	// Player AABB.
	PlayerWidth     = float32(0.6)
	PlayerHeight    = float32(1.8)
	SneakHeight     = float32(1.5)
	StepUpHeight    = float32(0.6)

	// Eye heights.
	StandEyeHeight = float32(1.62)
	SneakEyeHeight = float32(1.35)
	EyeHeightLerpSpeed = float32(12.0) // Smooth transition speed.

	// FOV.
	DefaultFov  = float32(70.0)
	SprintFov   = float32(82.0)
	FovLerpRate = float32(10.0)

	// Swing animation.
	SwingDuration = float32(0.25)

	// Third-person camera offset.
	ThirdPersonDist      = float32(3.6)
	ThirdPersonYOffset   = float32(0.4)

	// Fall damage (survival).
	FallDamageThreshold  = float32(3.5)
	FallDamageCheckDist  = float32(3.8) // Must exceed threshold before damage applies.

	// Survival timers.
	RegenHungerThreshold = float32(18.0)
	RegenInterval        = float32(3.5)
	RegenHungerCost      = float32(0.5)
	StarvationInterval   = float32(4.0)
	DrownTickInterval    = float32(1.0)
	DrownDamage          = float32(2.0)
	OxygenDrainRate      = float32(0.8)
	SprintHungerRate     = float32(0.08)

	// Coyote time — grace window to jump after leaving an edge.
	CoyoteGracePeriod = float32(0.10)

	// Double-tap sprint detection window.
	DoubleTapWindow = float32(0.30)

	// Head bobbing.
	BobSpeedGround = float32(2.2)
	BobSpeedSwim   = float32(1.5)
	BobDecay       = float32(0.85)
	BobAmplitudeY  = float32(0.04)
	BobAmplitudeX  = float32(0.025)

	// Particle physics.
	ParticleGravity = float32(14.0)
)

// ---------------------------------------------------------------------------
// View & game-mode enums.
// ---------------------------------------------------------------------------

// ViewPerspective defines 1st or 3rd person view.
type ViewPerspective int

const (
	PerspectiveFirstPerson ViewPerspective = iota
	PerspectiveThirdPersonBack
	PerspectiveThirdPersonFront
)

// GameMode defines Survival vs Creative.
type GameMode int

const (
	GameModeSurvival GameMode = iota
	GameModeCreative
)

// ---------------------------------------------------------------------------
// MCPlayer — the Minecraft Steve avatar.
// ---------------------------------------------------------------------------

// MCPlayer represents the Minecraft Steve avatar.
type MCPlayer struct {
	// Position & velocity.
	Pos rl.Vector3
	Vel rl.Vector3

	// Look angles (radians).
	Yaw   float32
	Pitch float32

	// Cached trig values — updated once per frame in handleMouseLook.
	cosYaw, sinYaw float32
	cosPitch, sinPitch float32

	// View & mode.
	Perspective ViewPerspective
	Mode        GameMode

	// Movement state.
	IsGrounded  bool
	IsSprinting bool
	IsSneaking  bool
	IsSwimming  bool
	IsSubmerged bool

	// Coyote time — allows jumping briefly after walking off an edge.
	CoyoteTimer float32

	// Double-tap sprint — tracks time since last forward-key press.
	DoubleTapTimer float32
	WasPressedW    bool

	// Survival stats.
	Health      float32 // 20 HP (10 Hearts)
	Hunger      float32 // 20 Food (10 Drumsticks)
	Oxygen      float32 // 10.0 (10 Air Bubbles)
	Level       int
	ExpProgress float32

	// Fall damage tracking.
	FallDistance float32

	// Survival timers.
	RegenTimer float32
	DrownTimer float32

	// Death state.
	IsDead     bool
	DeathTimer float32

	// Hand / tool swing animation.
	SwingTimer    float32
	WalkBobbing   float32
	BaseFov       float32
	EquipProgress float32 // 0.0 to 1.0 (item slide down & up on switch)
	PrevHeldBlock voxel.BlockType
	EatingTimer   float32 // 0 to 1.4s for food eating

	// Smooth camera eye-height (lerps between stand/sneak).
	CurrentEyeHeight float32

	// Step-up smoothing residual.
	StepLerp float32

	// Raylib Camera.
	RLCamera rl.Camera3D

	// Particle system for block mining.
	Particles []BlockParticle
}

// BlockParticle represents a small flying 3D voxel cube particle.
type BlockParticle struct {
	Pos       rl.Vector3
	Vel       rl.Vector3
	Color     rl.Color
	SourceRec rl.Rectangle
	Life      float32
	MaxLife   float32
	Size      float32
}

// ---------------------------------------------------------------------------
// Constructor.
// ---------------------------------------------------------------------------

// NewMCPlayer spawns player on top of highest terrain block.
func NewMCPlayer(spawnPos rl.Vector3) *MCPlayer {
	p := &MCPlayer{
		Pos:              spawnPos,
		Perspective:      PerspectiveFirstPerson,
		Mode:             GameModeSurvival,
		Health:           20,
		Hunger:           20,
		Oxygen:           10.0,
		Level:            30,
		ExpProgress:      0.65,
		BaseFov:          DefaultFov,
		CurrentEyeHeight: StandEyeHeight,
		EquipProgress:    1.0,
		Particles:        make([]BlockParticle, 0, 128),
	}

	p.RLCamera = rl.Camera3D{
		Position:   rl.Vector3{X: spawnPos.X, Y: spawnPos.Y + StandEyeHeight, Z: spawnPos.Z},
		Target:     rl.Vector3{X: spawnPos.X, Y: spawnPos.Y + StandEyeHeight, Z: spawnPos.Z + 1.0},
		Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
		Fovy:       DefaultFov,
		Projection: rl.CameraPerspective,
	}

	return p
}

// ---------------------------------------------------------------------------
// Public actions.
// ---------------------------------------------------------------------------

// TogglePerspective cycles with F5 key.
func (p *MCPlayer) TogglePerspective() {
	p.Perspective = (p.Perspective + 1) % 3
}

// ToggleGameMode switches between Survival and Creative.
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

// TriggerSwing triggers arm swinging animation.
func (p *MCPlayer) TriggerSwing() {
	p.SwingTimer = SwingDuration
}

// SpawnBlockBreakParticles creates bursting block voxel debris.
func (p *MCPlayer) SpawnBlockBreakParticles(pos rl.Vector3, bType voxel.BlockType) {
	for i := 0; i < 16; i++ {
		rx := float32(i%4)*0.22 - 0.33
		ry := float32((i/4)%4)*0.22 - 0.33
		rz := float32((i*7)%4)*0.22 - 0.33

		vx := rx*4.5 + sin32(float32(i*3))*1.5
		vy := float32(2.5) + cos32(float32(i*5))*2.0
		vz := rz*4.5 + cos32(float32(i*3))*1.5

		txCol, row := voxel.GetBlockTextureAtlasPos(bType, voxel.FaceNorth)
		pixelSize := float32(16)
		offsetX := float32(i%4) * 4.0 // 4x4 pixel chunks
		offsetY := float32((i/4)%4) * 4.0
		srcRec := rl.NewRectangle(float32(txCol)*pixelSize+offsetX, float32(row)*pixelSize+offsetY, 4, 4)

		p.Particles = append(p.Particles, BlockParticle{
			Pos:       rl.Vector3{X: pos.X + 0.5 + rx, Y: pos.Y + 0.5 + ry, Z: pos.Z + 0.5 + rz},
			Vel:       rl.Vector3{X: vx, Y: vy, Z: vz},
			Color:     rl.White, // Textured particles don't need solid color tinting
			SourceRec: srcRec,
			Life:      0.0,
			MaxLife:   0.6 + rand.Float32()*0.4,
			Size:      0.15 + rand.Float32()*0.08,
		})
	}
}

// ---------------------------------------------------------------------------
// Update — top-level coordinator (calls focused sub-methods).
// ---------------------------------------------------------------------------

// Update handles mouse look, WASD movement, voxel AABB collisions, jumping,
// camera, survival, and particles.
func (p *MCPlayer) Update(dt float32, world *voxel.VoxelWorld) {
	if dt <= 0 || dt > 0.05 {
		dt = 0.016667
	}

	// Dead — freeze movement, animate death camera.
	if p.Health <= 0 {
		p.Health = 0
		p.IsDead = true
		p.DeathTimer += dt
		p.Vel = rl.Vector3{}
		rl.EnableCursor()
		p.updateCamera(dt)
		return
	}

	// Tick swing animation.
	if p.SwingTimer > 0 {
		p.SwingTimer = clamp32(p.SwingTimer-dt, 0, SwingDuration)
	}

	// 1. Mouse look & trig cache.
	p.handleMouseLook(dt)

	// 2. Water state detection.
	p.checkWaterState(world)

	// 3. Movement input (WASD + sprint/sneak detection).
	moveVec, isMoving := p.handleMovementInput(dt)

	// 4. Calculate speed based on current state.
	speed := p.calculateMoveSpeed()

	// Normalize and scale movement vector.
	lenSq := moveVec.X*moveVec.X + moveVec.Z*moveVec.Z
	if lenSq > 0.001 {
		inv := speed / float32(math.Sqrt(float64(lenSq)))
		moveVec.X *= inv
		moveVec.Z *= inv
		if p.IsSwimming {
			p.WalkBobbing += dt * speed * BobSpeedSwim
		} else {
			p.WalkBobbing += dt * speed * BobSpeedGround
		}
	} else {
		p.WalkBobbing *= BobDecay
	}

	// Drain hunger while sprinting (survival).
	if p.IsSprinting && isMoving && p.Mode == GameModeSurvival {
		p.Hunger = clamp32(p.Hunger-dt*SprintHungerRate, 0, 20)
	}

	// 5. Jump, swim buoyancy & diving.
	forward := rl.Vector3{X: -p.sinYaw, Y: 0, Z: -p.cosYaw}
	p.handleJumpAndSwim(dt, forward, world)

	// Track fall distance for survival damage.
	if !p.IsGrounded && p.Vel.Y < 0 && !p.IsSwimming {
		p.FallDistance += -p.Vel.Y * dt
	}

	// 6. Voxel AABB collision & resolution.
	p.moveWithVoxelCollision(moveVec.X*dt, p.Vel.Y*dt, moveVec.Z*dt, world)

	// 7. Survival mechanics (drowning, hunger, regen).
	if p.Mode == GameModeSurvival {
		p.handleSurvivalMechanics(dt)
	}

	// 8. Particle & Animation update.
	p.updateParticles(dt)
	if p.EquipProgress < 1.0 {
		p.EquipProgress += dt * 5.0
		if p.EquipProgress > 1.0 {
			p.EquipProgress = 1.0
		}
	}

	// 9. Camera.
	p.updateCamera(dt)
}

// ---------------------------------------------------------------------------
// Sub-method: mouse look.
// ---------------------------------------------------------------------------

// handleMouseLook reads mouse delta, updates yaw/pitch, and caches trig.
func (p *MCPlayer) handleMouseLook(dt float32) {
	_ = dt // Unused but kept for signature consistency.
	mouseDelta := rl.GetMouseDelta()

	p.Yaw -= mouseDelta.X * MouseSensitivity
	p.Pitch = clamp32(p.Pitch-mouseDelta.Y*MouseSensitivity, -MaxPitch, MaxPitch)

	// Cache trig values for the frame — avoids recomputation in multiple methods.
	p.cosYaw = cos32(p.Yaw)
	p.sinYaw = sin32(p.Yaw)
	p.cosPitch = cos32(p.Pitch)
	p.sinPitch = sin32(p.Pitch)
}

// ---------------------------------------------------------------------------
// Sub-method: water state detection.
// ---------------------------------------------------------------------------

// checkWaterState samples blocks at feet, waist, and head to determine swimming/submerged.
func (p *MCPlayer) checkWaterState(world *voxel.VoxelWorld) {
	bx := int(floor32(p.Pos.X))
	bz := int(floor32(p.Pos.Z))

	feetY := int(floor32(p.Pos.Y))
	waistY := int(floor32(p.Pos.Y + 0.8))
	headY := int(floor32(p.Pos.Y + StandEyeHeight))

	feetBlock := world.GetBlock(bx, feetY, bz)
	waistBlock := world.GetBlock(bx, waistY, bz)
	headBlock := world.GetBlock(bx, headY, bz)

	p.IsSwimming = (voxel.IsWater(feetBlock) || voxel.IsWater(waistBlock) || voxel.IsWater(headBlock))
	p.IsSubmerged = voxel.IsWater(headBlock)
}

// ---------------------------------------------------------------------------
// Sub-method: movement input.
// ---------------------------------------------------------------------------

// handleMovementInput reads WASD keys, detects sprint (double-tap or ctrl),
// detects sneak, and returns a raw (un-normalized) movement vector.
func (p *MCPlayer) handleMovementInput(dt float32) (rl.Vector3, bool) {
	forward := rl.Vector3{X: -p.sinYaw, Y: 0, Z: -p.cosYaw}
	right := rl.Vector3{X: p.cosYaw, Y: 0, Z: -p.sinYaw}

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

	// Double-tap W to sprint.
	wDown := rl.IsKeyDown(rl.KeyW)
	if rl.IsKeyPressed(rl.KeyW) {
		if p.DoubleTapTimer > 0 {
			p.IsSprinting = true
		}
		p.DoubleTapTimer = DoubleTapWindow
	}
	if p.DoubleTapTimer > 0 {
		p.DoubleTapTimer -= dt
	}
	// Ctrl also activates sprint; shift sprint kept for backward compat.
	if rl.IsKeyDown(rl.KeyLeftControl) || (rl.IsKeyDown(rl.KeyLeftShift) && isMoving) {
		p.IsSprinting = true
	}
	// Cancel sprint if W is released or player stops moving.
	if !wDown || !isMoving {
		p.IsSprinting = false
	}

	// Sneak.
	p.IsSneaking = rl.IsKeyDown(rl.KeyC)

	return moveVec, isMoving
}

// ---------------------------------------------------------------------------
// Sub-method: speed calculation.
// ---------------------------------------------------------------------------

// calculateMoveSpeed returns the current movement speed based on player state.
func (p *MCPlayer) calculateMoveSpeed() float32 {
	if p.IsSwimming {
		if p.IsSprinting {
			return SwimSprintSpeed
		}
		return SwimSpeed
	}
	if p.IsSprinting {
		return SprintSpeed
	}
	if p.IsSneaking {
		return SneakSpeed
	}
	return WalkSpeed
}

// ---------------------------------------------------------------------------
// Sub-method: jump & swim.
// ---------------------------------------------------------------------------

// handleJumpAndSwim manages jumping, coyote time, water buoyancy, surface
// hopping, and diving.
func (p *MCPlayer) handleJumpAndSwim(dt float32, forward rl.Vector3, world *voxel.VoxelWorld) {
	if p.IsSwimming {
		p.FallDistance = 0 // Water negates fall damage.

		if rl.IsKeyDown(rl.KeySpace) {
			// Swim upward toward surface.
			p.Vel.Y = SwimUpSpeed

			// Surface hop onto land (head above water, waist still in).
			waistInWater := voxel.IsWater(world.GetBlock(int(floor32(p.Pos.X)), int(floor32(p.Pos.Y+0.8)), int(floor32(p.Pos.Z))))
			if !p.IsSubmerged && waistInWater {
				shoreX := int(floor32(p.Pos.X + forward.X*0.6))
				shoreZ := int(floor32(p.Pos.Z + forward.Z*0.6))
				shoreY := int(floor32(p.Pos.Y + 0.5))
				if voxel.BlockRegistry[world.GetBlock(shoreX, shoreY, shoreZ)].IsSolid {
					p.Vel.Y = SurfaceHopSpeed
				}
			}
		} else if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyC) {
			// Dive downward.
			p.Vel.Y = -SwimDiveSpeed
		} else {
			// Natural slow sinking / neutral buoyancy.
			p.Vel.Y = -SwimSinkSpeed
		}
		return
	}

	// --- Ground / air physics ---

	// Coyote time management.
	if p.IsGrounded {
		p.CoyoteTimer = CoyoteGracePeriod
	} else if p.CoyoteTimer > 0 {
		p.CoyoteTimer -= dt
	}

	// Jump — allowed if grounded OR within coyote window.
	if rl.IsKeyDown(rl.KeySpace) && (p.IsGrounded || p.CoyoteTimer > 0) {
		p.Vel.Y = JumpVelocity
		p.IsGrounded = false
		p.CoyoteTimer = 0 // Consume the coyote window.
	}

	// Gravity.
	p.Vel.Y -= Gravity * dt
	if p.Vel.Y < -TerminalVelocity {
		p.Vel.Y = -TerminalVelocity
	}
}

// ---------------------------------------------------------------------------
// Sub-method: survival mechanics.
// ---------------------------------------------------------------------------

// handleSurvivalMechanics manages drowning, oxygen, hunger-based regen,
// and starvation damage.
func (p *MCPlayer) handleSurvivalMechanics(dt float32) {
	// Drowning.
	if p.IsSubmerged {
		p.Oxygen -= dt * OxygenDrainRate
		if p.Oxygen <= 0 {
			p.Oxygen = 0
			p.DrownTimer += dt
			if p.DrownTimer >= DrownTickInterval {
				p.Health = clamp32(p.Health-DrownDamage, 0, 20)
				p.DrownTimer = 0
			}
		}
	} else {
		p.Oxygen = 10.0
		p.DrownTimer = 0
	}

	// Health regeneration (high hunger).
	if p.Hunger >= RegenHungerThreshold && p.Health < 20 {
		p.RegenTimer += dt
		if p.RegenTimer >= RegenInterval {
			p.Health = clamp32(p.Health+1.0, 0, 20)
			p.Hunger -= RegenHungerCost
			p.RegenTimer = 0
		}
	} else if p.Hunger == 0 && p.Health > 1.0 {
		// Starvation damage.
		p.RegenTimer += dt
		if p.RegenTimer >= StarvationInterval {
			p.Health -= 1.0
			p.RegenTimer = 0
		}
	}
}

// ---------------------------------------------------------------------------
// Sub-method: particle update.
// ---------------------------------------------------------------------------

// updateParticles ticks particle lifetime and physics with zero-allocation
// removal (swap-with-last pattern).
func (p *MCPlayer) updateParticles(dt float32) {
	n := len(p.Particles)
	for i := 0; i < n; {
		part := &p.Particles[i]
		part.Life -= dt
		if part.Life <= 0 {
			// Swap with last, shrink slice — no allocation.
			p.Particles[i] = p.Particles[n-1]
			n--
			p.Particles = p.Particles[:n]
			continue
		}
		part.Vel.Y -= ParticleGravity * dt
		part.Pos.X += part.Vel.X * dt
		part.Pos.Y += part.Vel.Y * dt
		part.Pos.Z += part.Vel.Z * dt
		i++
	}
}

// ---------------------------------------------------------------------------
// Collision.
// ---------------------------------------------------------------------------

// moveWithVoxelCollision applies AABB sliding collision on X, Y, Z axes
// with step-up, fall damage, and sneak edge-guard.
func (p *MCPlayer) moveWithVoxelCollision(dx, dy, dz float32, world *voxel.VoxelWorld) {
	playerH := PlayerHeight
	if p.IsSneaking {
		playerH = SneakHeight
	}
	halfW := PlayerWidth * 0.5

	wasGrounded := p.IsGrounded

	// --- Y-axis movement & collision ---
	newY := p.Pos.Y + dy
	if p.checkAABBCollision(p.Pos.X, newY, p.Pos.Z, halfW, playerH, world) {
		if dy < 0 {
			p.IsGrounded = true
			p.Vel.Y = 0
			// Snap to the surface of the block we landed on.
			p.Pos.Y = floor32(newY) + 1.0

			// Fall damage in survival.
			if !wasGrounded && p.Mode == GameModeSurvival && p.FallDistance > FallDamageCheckDist {
				damage := (p.FallDistance - FallDamageThreshold) * 1.0
				p.Health = clamp32(p.Health-damage, 0, 20)
			}
			p.FallDistance = 0
		} else {
			p.Vel.Y = 0 // Hit ceiling.
		}
	} else {
		p.Pos.Y = newY
		p.IsGrounded = false
	}

	// --- X-axis movement & collision (with step-up) ---
	newX := p.Pos.X + dx
	if p.checkAABBCollision(newX, p.Pos.Y, p.Pos.Z, halfW, playerH, world) {
		// Try stepping up.
		if p.IsGrounded && !p.checkAABBCollision(newX, p.Pos.Y+StepUpHeight, p.Pos.Z, halfW, playerH, world) {
			p.StepLerp += StepUpHeight // Accumulate for smooth camera.
			p.Pos.Y += StepUpHeight
			p.Pos.X = newX
		}
		// Else: blocked, don't move on X.
	} else {
		// Sneak edge-guard: prevent walking off edges while sneaking.
		if p.IsSneaking && p.IsGrounded && !p.IsSwimming {
			if !p.checkAABBCollision(newX, p.Pos.Y-1.0, p.Pos.Z, halfW, playerH, world) {
				// Would walk over air — block X movement.
				goto skipX
			}
		}
		p.Pos.X = newX
	}
skipX:

	// --- Z-axis movement & collision (with step-up) ---
	newZ := p.Pos.Z + dz
	if p.checkAABBCollision(p.Pos.X, p.Pos.Y, newZ, halfW, playerH, world) {
		// Try stepping up.
		if p.IsGrounded && !p.checkAABBCollision(p.Pos.X, p.Pos.Y+StepUpHeight, newZ, halfW, playerH, world) {
			p.StepLerp += StepUpHeight
			p.Pos.Y += StepUpHeight
			p.Pos.Z = newZ
		}
	} else {
		// Sneak edge-guard: prevent walking off edges while sneaking.
		if p.IsSneaking && p.IsGrounded && !p.IsSwimming {
			if !p.checkAABBCollision(p.Pos.X, p.Pos.Y-1.0, newZ, halfW, playerH, world) {
				goto skipZ
			}
		}
		p.Pos.Z = newZ
	}
skipZ:
}

// checkAABBCollision tests player AABB against solid voxels.
func (p *MCPlayer) checkAABBCollision(px, py, pz, halfW, height float32, world *voxel.VoxelWorld) bool {
	minX := int(floor32(px - halfW))
	maxX := int(floor32(px + halfW))
	minY := int(floor32(py))
	maxY := int(floor32(py + height - 0.05))
	minZ := int(floor32(pz - halfW))
	maxZ := int(floor32(pz + halfW))

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

// ---------------------------------------------------------------------------
// Camera.
// ---------------------------------------------------------------------------

// updateCamera positions the camera for 1st person or 3rd person views with
// smooth eye-height transitions, walk bobbing, sprint FOV, and step-up lerp.
func (p *MCPlayer) updateCamera(dt float32) {
	lookDir := rl.Vector3{
		X: -p.sinYaw * p.cosPitch,
		Y: p.sinPitch,
		Z: -p.cosYaw * p.cosPitch,
	}

	// Smooth eye-height transition (stand ↔ sneak).
	targetEyeHeight := StandEyeHeight
	if p.IsSneaking {
		targetEyeHeight = SneakEyeHeight
	}
	p.CurrentEyeHeight += (targetEyeHeight - p.CurrentEyeHeight) * dt * EyeHeightLerpSpeed

	// Smooth step-up residual decay.
	if p.StepLerp > 0 {
		decay := dt * 15.0
		if decay > p.StepLerp {
			decay = p.StepLerp
		}
		p.StepLerp -= decay
	}

	// Walk/swim bobbing.
	bobY := sin32(p.WalkBobbing) * BobAmplitudeY
	bobX := cos32(p.WalkBobbing*0.5) * BobAmplitudeX

	// Sprint FOV lerp.
	targetFov := p.BaseFov
	if p.IsSprinting {
		targetFov = SprintFov
	}
	p.RLCamera.Fovy += (targetFov - p.RLCamera.Fovy) * dt * FovLerpRate

	eyePos := rl.Vector3{
		X: p.Pos.X + bobX,
		Y: p.Pos.Y + p.CurrentEyeHeight + bobY - p.StepLerp,
		Z: p.Pos.Z,
	}

	// Death tilt.
	if p.IsDead {
		tiltAngle := float32(math.Min(math.Pi*0.45, float64(p.DeathTimer*3.2)))
		p.RLCamera.Up = rl.Vector3{
			X: sin32(tiltAngle),
			Y: cos32(tiltAngle),
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
		p.RLCamera.Position = rl.Vector3{
			X: eyePos.X - lookDir.X*ThirdPersonDist,
			Y: eyePos.Y - lookDir.Y*ThirdPersonDist + ThirdPersonYOffset,
			Z: eyePos.Z - lookDir.Z*ThirdPersonDist,
		}
		p.RLCamera.Target = eyePos

	case PerspectiveThirdPersonFront:
		p.RLCamera.Position = rl.Vector3{
			X: eyePos.X + lookDir.X*ThirdPersonDist,
			Y: eyePos.Y + lookDir.Y*ThirdPersonDist + ThirdPersonYOffset,
			Z: eyePos.Z + lookDir.Z*ThirdPersonDist,
		}
		p.RLCamera.Target = eyePos
	}
}

// ---------------------------------------------------------------------------
// Rendering.
// ---------------------------------------------------------------------------

// RenderParticles renders all active flying block break debris and eating particles.
func (p *MCPlayer) RenderParticles(atlas *voxel.TextureAtlas) {
	if atlas == nil || atlas.Texture.ID == 0 {
		return
	}
	
	for i := 0; i < len(p.Particles); i++ {
		part := &p.Particles[i]
		// Draw 2D textured particle billboard always facing the camera
		rl.DrawBillboardRec(p.RLCamera, atlas.Texture, part.SourceRec, part.Pos, rl.NewVector2(part.Size, part.Size), part.Color)
	}
}

// RenderHandAndHeldBlock draws the authentic Minecraft first-person viewmodel
// with authentic 3D swing kinematics (sin of sqrt progress, pitch chop, roll, yaw),
// item switch equip slide, and walking bobbing.
func (p *MCPlayer) RenderHandAndHeldBlock(heldBlock voxel.BlockType, atlas *voxel.TextureAtlas) {
	if p.Perspective != PerspectiveFirstPerson || atlas == nil || atlas.Texture.ID == 0 {
		return
	}

	if heldBlock != p.PrevHeldBlock {
		p.PrevHeldBlock = heldBlock
		p.EquipProgress = 0.0 // trigger item equip rise animation
	}

	cam := p.RLCamera
	forward := rl.Vector3Normalize(rl.Vector3Subtract(cam.Target, cam.Position))
	right := rl.Vector3Normalize(rl.Vector3CrossProduct(forward, cam.Up))
	up := rl.Vector3CrossProduct(right, forward)

	// Authentic Minecraft swing progress math (sin of sqrt progress)
	swingProgress := float32(0.0)
	if p.SwingTimer > 0 {
		swingProgress = 1.0 - (p.SwingTimer / SwingDuration)
	}
	f1 := sin32(swingProgress * math.Pi)
	f2 := sin32(sqrt32(swingProgress) * math.Pi)

	// Base angles for iconic Minecraft diagonal first-person arm posture
	baseRotX := float32(-18.0 * (math.Pi / 180.0)) // Tilted upward
	baseRotY := float32(-28.0 * (math.Pi / 180.0)) // Angled inward towards center
	baseRotZ := float32(-10.0 * (math.Pi / 180.0)) // Slight inward roll

	// Dynamic swing additions (pitch chop, yaw inward, roll)
	rotX := baseRotX - f2*62.0*(math.Pi/180.0)
	rotY := baseRotY - f1*24.0*(math.Pi/180.0)
	rotZ := baseRotZ - f1*16.0*(math.Pi/180.0)

	swingX := -f2 * 0.16
	swingY := sin32(sqrt32(swingProgress)*math.Pi*2.0) * 0.09
	swingZ := -f1 * 0.12

	// Item equip dip & rise
	equipY := (1.0 - p.EquipProgress) * -0.30

	// Walk bobbing (sinusoidal camera-relative bob)
	bobX := sin32(p.WalkBobbing*0.5) * 0.008
	bobY := -abs32(cos32(p.WalkBobbing)) * 0.008

	// Pivot point for hand / item rotation (around shoulder / elbow)
	pivotX := float32(0.28)
	pivotY := float32(-0.24)
	pivotZ := float32(0.34)

	// Helper to apply 3D transformation and emit world vertex
	vtxTransformed := func(lx, ly, lz float32, u, v float32) {
		dx := lx
		dy := ly
		dz := lz

		// Rotate Z (roll)
		x1 := dx*cos32(rotZ) - dy*sin32(rotZ)
		y1 := dx*sin32(rotZ) + dy*cos32(rotZ)
		z1 := dz

		// Rotate X (pitch downward chop)
		x2 := x1
		y2 := y1*cos32(rotX) - z1*sin32(rotX)
		z2 := y1*sin32(rotX) + z1*cos32(rotX)

		// Rotate Y (yaw inward)
		x3 := x2*cos32(rotY) + z2*sin32(rotY)
		y3 := y2
		z3 := -x2*sin32(rotY) + z2*cos32(rotY)

		// Eating chew motion
		chewY := float32(0.0)
		chewZ := float32(0.0)
		if p.EatingTimer > 0 {
			chewY = sin32(p.EatingTimer*26.0)*0.018 - 0.04
			chewZ = cos32(p.EatingTimer*26.0)*0.012 + 0.02
		}

		finalLX := pivotX + x3 + swingX + bobX
		finalLY := pivotY + y3 + swingY + bobY + equipY + chewY
		finalLZ := pivotZ + z3 + swingZ + chewZ

		wx := cam.Position.X + right.X*finalLX + up.X*finalLY + forward.X*finalLZ
		wy := cam.Position.Y + right.Y*finalLX + up.Y*finalLY + forward.Y*finalLZ
		wz := cam.Position.Z + right.Z*finalLX + up.Z*finalLY + forward.Z*finalLZ

		rl.TexCoord2f(u, v)
		rl.Vertex3f(wx, wy, wz)
	}

	bDef := voxel.BlockRegistry[heldBlock]
	isFlatItem := bDef.IsTool || bDef.IsFood || heldBlock == voxel.ItemStick || heldBlock == voxel.ItemDiamond ||
		heldBlock == voxel.ItemCoal || heldBlock == voxel.ItemIronIngot || heldBlock == voxel.ItemGoldIngot ||
		heldBlock == voxel.BlockTorch || heldBlock == voxel.ItemGunpowder || heldBlock == voxel.ItemBone ||
		heldBlock == voxel.ItemArrow

	// Dedicated Row 15 Steve textures (0, 15 = Shirt, 1, 15 = Skin)
	shirtU0, shirtV0, shirtU1, shirtV1 := float32(0)/16.0, float32(15)/16.0, float32(1)/16.0, float32(16)/16.0
	skinU0, skinV0, skinU1, skinV1 := float32(1)/16.0, float32(15)/16.0, float32(2)/16.0, float32(16)/16.0

	eps := float32(0.5) / float32(atlas.Texture.Width)
	shirtU0 += eps
	shirtV0 += eps
	shirtU1 -= eps
	shirtV1 -= eps
	skinU0 += eps
	skinV0 += eps
	skinU1 -= eps
	skinV1 -= eps

	drawBox := func(x0, y0, z0, x1, y1, z1 float32, u0, v0, u1, v1 float32, baseCol rl.Color) {
		r, g, b := float32(baseCol.R), float32(baseCol.G), float32(baseCol.B)

		// Top Face (+Y) - 100% light
		rl.Color4ub(uint8(r*1.0), uint8(g*1.0), uint8(b*1.0), baseCol.A)
		vtxTransformed(x0, y1, z1, u0, v1)
		vtxTransformed(x1, y1, z1, u1, v1)
		vtxTransformed(x1, y1, z0, u1, v0)
		vtxTransformed(x0, y1, z0, u0, v0)

		// Bottom Face (-Y) - 55% light
		rl.Color4ub(uint8(r*0.55), uint8(g*0.55), uint8(b*0.55), baseCol.A)
		vtxTransformed(x0, y0, z0, u0, v1)
		vtxTransformed(x1, y0, z0, u1, v1)
		vtxTransformed(x1, y0, z1, u1, v0)
		vtxTransformed(x0, y0, z1, u0, v0)

		// Front Face (+Z) - 85% light
		rl.Color4ub(uint8(r*0.85), uint8(g*0.85), uint8(b*0.85), baseCol.A)
		vtxTransformed(x1, y0, z1, u1, v1)
		vtxTransformed(x0, y0, z1, u0, v1)
		vtxTransformed(x0, y1, z1, u0, v0)
		vtxTransformed(x1, y1, z1, u1, v0)

		// Back Face (-Z) - 80% light
		rl.Color4ub(uint8(r*0.80), uint8(g*0.80), uint8(b*0.80), baseCol.A)
		vtxTransformed(x0, y0, z0, u0, v1)
		vtxTransformed(x1, y0, z0, u1, v1)
		vtxTransformed(x1, y1, z0, u1, v0)
		vtxTransformed(x0, y1, z0, u0, v0)

		// Left Face (-X) - 65% light
		rl.Color4ub(uint8(r*0.65), uint8(g*0.65), uint8(b*0.65), baseCol.A)
		vtxTransformed(x0, y0, z0, u0, v1)
		vtxTransformed(x0, y0, z1, u1, v1)
		vtxTransformed(x0, y1, z1, u1, v0)
		vtxTransformed(x0, y1, z0, u0, v0)

		// Right Face (+X) - 75% light
		rl.Color4ub(uint8(r*0.75), uint8(g*0.75), uint8(b*0.75), baseCol.A)
		vtxTransformed(x1, y0, z1, u0, v1)
		vtxTransformed(x1, y0, z0, u1, v1)
		vtxTransformed(x1, y1, z0, u1, v0)
		vtxTransformed(x1, y1, z1, u0, v0)
	}

	rl.Begin(rl.Quads)
	rl.SetTexture(atlas.Texture.ID)

	if heldBlock == voxel.BlockAir {
		// 1. Empty Steve Arm / Fist (comes cleanly from bottom-right towards center)
		// Sleeve (Cyan Shirt)
		drawBox(-0.045, -0.045, -0.16, 0.045, 0.045, -0.05, shirtU0, shirtV0, shirtU1, shirtV1, rl.White)
		// Forearm & Fist (Skin)
		drawBox(-0.040, -0.040, -0.05, 0.040, 0.040, 0.16, skinU0, skinV0, skinU1, skinV1, rl.White)
	} else if isFlatItem {
		// 2. 2D Tool Sprite (Pickaxes, Swords, Axes, Shovels, Sticks)
		u0, v0, u1, v1 := voxel.GetBlockTextureUVs(heldBlock, voxel.FaceNorth)
		s := float32(0.14)

		rl.Color4ub(255, 255, 255, 255)

		// Front Face (45 deg diagonal Minecraft tool posture)
		vtxTransformed(-s*0.35, -s*0.65+0.06, 0.08-0.015, u0, v1)
		vtxTransformed(+s*0.75, -s*0.15+0.06, 0.08+0.015, u1, v1)
		vtxTransformed(+s*0.45, +s*0.85+0.06, 0.08+0.035, u1, v0)
		vtxTransformed(-s*0.65, +s*0.35+0.06, 0.08+0.005, u0, v0)

		// Back Face
		vtxTransformed(+s*0.75, -s*0.15+0.06, 0.08+0.015, u1, v1)
		vtxTransformed(-s*0.35, -s*0.65+0.06, 0.08-0.015, u0, v1)
		vtxTransformed(-s*0.65, +s*0.35+0.06, 0.08+0.005, u0, v0)
		vtxTransformed(+s*0.45, +s*0.85+0.06, 0.08+0.035, u1, v0)

		// Steve Arm holding the tool handle:
		drawBox(0.01, -0.16, -0.16, 0.09, -0.08, -0.04, shirtU0, shirtV0, shirtU1, shirtV1, rl.White)
		drawBox(0.015, -0.15, -0.04, 0.085, -0.09, 0.08, skinU0, skinV0, skinU1, skinV1, rl.White)
	} else {
		// 3. Mini 3D Voxel Block
		hs := float32(0.06)
		by := float32(0.04)
		bz := float32(0.06)

		// Top Face (+Y)
		u0, v0, u1, v1 := voxel.GetBlockTextureUVs(heldBlock, voxel.FaceTop)
		rl.Color4ub(255, 255, 255, 255)
		vtxTransformed(-hs, by+hs, bz+hs, u0, v1)
		vtxTransformed(+hs, by+hs, bz+hs, u1, v1)
		vtxTransformed(+hs, by+hs, bz-hs, u1, v0)
		vtxTransformed(-hs, by+hs, bz-hs, u0, v0)

		// Bottom Face (-Y)
		u0, v0, u1, v1 = voxel.GetBlockTextureUVs(heldBlock, voxel.FaceBottom)
		rl.Color4ub(160, 160, 160, 255)
		vtxTransformed(-hs, by-hs, bz-hs, u0, v1)
		vtxTransformed(+hs, by-hs, bz-hs, u1, v1)
		vtxTransformed(+hs, by-hs, bz+hs, u1, v0)
		vtxTransformed(-hs, by-hs, bz+hs, u0, v0)

		// Front Face (+Z)
		u0, v0, u1, v1 = voxel.GetBlockTextureUVs(heldBlock, voxel.FaceSouth)
		rl.Color4ub(220, 220, 220, 255)
		vtxTransformed(+hs, by-hs, bz+hs, u1, v1)
		vtxTransformed(-hs, by-hs, bz+hs, u0, v1)
		vtxTransformed(-hs, by+hs, bz+hs, u0, v0)
		vtxTransformed(+hs, by+hs, bz+hs, u1, v0)

		// Back Face (-Z)
		u0, v0, u1, v1 = voxel.GetBlockTextureUVs(heldBlock, voxel.FaceNorth)
		rl.Color4ub(200, 200, 200, 255)
		vtxTransformed(-hs, by-hs, bz-hs, u0, v1)
		vtxTransformed(+hs, by-hs, bz-hs, u1, v1)
		vtxTransformed(+hs, by+hs, bz-hs, u1, v0)
		vtxTransformed(-hs, by+hs, bz-hs, u0, v0)

		// Left Face (-X)
		u0, v0, u1, v1 = voxel.GetBlockTextureUVs(heldBlock, voxel.FaceWest)
		rl.Color4ub(190, 190, 190, 255)
		vtxTransformed(-hs, by-hs, bz-hs, u0, v1)
		vtxTransformed(-hs, by-hs, bz+hs, u1, v1)
		vtxTransformed(-hs, by+hs, bz+hs, u1, v0)
		vtxTransformed(-hs, by+hs, bz-hs, u0, v0)

		// Right Face (+X)
		u0, v0, u1, v1 = voxel.GetBlockTextureUVs(heldBlock, voxel.FaceEast)
		rl.Color4ub(210, 210, 210, 255)
		vtxTransformed(+hs, by-hs, bz+hs, u0, v1)
		vtxTransformed(+hs, by-hs, bz-hs, u1, v1)
		vtxTransformed(+hs, by+hs, bz-hs, u1, v0)
		vtxTransformed(+hs, by+hs, bz+hs, u0, v0)

		// Steve Arm supporting block:
		drawBox(0.01, -0.14, -0.16, 0.09, -0.06, -0.04, shirtU0, shirtV0, shirtU1, shirtV1, rl.White)
		drawBox(0.015, -0.13, -0.04, 0.085, -0.07, 0.06, skinU0, skinV0, skinU1, skinV1, rl.White)
	}

	rl.End()
}

// Render3DSteveModel draws the 3D Steve character in 3rd person view with authentic walking, sneaking, and mining animations.
func (p *MCPlayer) Render3DSteveModel() {
	if p.Perspective == PerspectiveFirstPerson {
		return
	}

	rl.PushMatrix()
	yOffset := float32(0.0)
	if p.IsSneaking {
		yOffset = -0.15
	}
	rl.Translatef(p.Pos.X, p.Pos.Y+yOffset, p.Pos.Z)
	rl.Rotatef(-p.Yaw*rl.Rad2deg, 0, 1, 0)

	skinCol := rl.NewColor(198, 140, 102, 255)
	cyanShirt := rl.NewColor(0, 175, 185, 255) // Cyan Steve Shirt
	blueJeans := rl.NewColor(42, 52, 135, 255)  // Blue Jeans
	hairCol := rl.NewColor(75, 48, 28, 255)
	eyeWhite := rl.NewColor(245, 245, 245, 255)
	eyeBlue := rl.NewColor(45, 75, 195, 255)
	shoeCol := rl.NewColor(80, 80, 85, 255)

	// Head (8x8 pixels).
	headY := float32(1.62)
	headZ := float32(0.0)
	if p.IsSneaking {
		headZ = 0.08
	}
	rl.DrawCube(rl.Vector3{X: 0, Y: headY, Z: headZ}, 0.48, 0.48, 0.48, skinCol)
	rl.DrawCube(rl.Vector3{X: 0, Y: headY + 0.18, Z: headZ}, 0.50, 0.14, 0.50, hairCol)

	// Face Features (Eyes & Mouth).
	rl.DrawCube(rl.Vector3{X: -0.12, Y: headY, Z: headZ + 0.245}, 0.08, 0.06, 0.02, eyeWhite)
	rl.DrawCube(rl.Vector3{X: 0.12, Y: headY, Z: headZ + 0.245}, 0.08, 0.06, 0.02, eyeWhite)
	rl.DrawCube(rl.Vector3{X: -0.09, Y: headY, Z: headZ + 0.25}, 0.04, 0.06, 0.02, eyeBlue)
	rl.DrawCube(rl.Vector3{X: 0.09, Y: headY, Z: headZ + 0.25}, 0.04, 0.06, 0.02, eyeBlue)

	// Torso & Cyan Shirt (tilts forward 15 deg when sneaking).
	torsoY := float32(1.08)
	torsoZ := float32(0.0)
	if p.IsSneaking {
		torsoZ = 0.04
	}
	rl.DrawCube(rl.Vector3{X: 0, Y: torsoY, Z: torsoZ}, 0.48, 0.68, 0.24, cyanShirt)

	// Legs (sinusoidal pendulum walk cycle).
	legSwing := sin32(p.WalkBobbing) * 0.28
	rl.DrawCube(rl.Vector3{X: -0.13, Y: 0.44, Z: legSwing}, 0.22, 0.60, 0.22, blueJeans)
	rl.DrawCube(rl.Vector3{X: 0.13, Y: 0.44, Z: -legSwing}, 0.22, 0.60, 0.22, blueJeans)
	// Shoes.
	rl.DrawCube(rl.Vector3{X: -0.13, Y: 0.08, Z: legSwing}, 0.22, 0.16, 0.22, shoeCol)
	rl.DrawCube(rl.Vector3{X: 0.13, Y: 0.08, Z: -legSwing}, 0.22, 0.16, 0.22, shoeCol)

	// Arms (Cyan Sleeve + Skin Hand).
	armRightZ := legSwing
	armRightY := float32(1.18)
	// 3rd Person Mining/Punching Swing
	if p.SwingTimer > 0 {
		sp := 1.0 - (p.SwingTimer / SwingDuration)
		armRightZ = sin32(sqrt32(sp)*math.Pi) * 0.45
		armRightY += sin32(sp*math.Pi) * 0.15
	}

	rl.DrawCube(rl.Vector3{X: -0.36, Y: 1.18, Z: -legSwing}, 0.22, 0.32, 0.22, cyanShirt)
	rl.DrawCube(rl.Vector3{X: -0.36, Y: 0.94, Z: -legSwing}, 0.20, 0.36, 0.20, skinCol)

	rl.DrawCube(rl.Vector3{X: 0.36, Y: armRightY, Z: armRightZ}, 0.22, 0.32, 0.22, cyanShirt)
	rl.DrawCube(rl.Vector3{X: 0.36, Y: armRightY - 0.24, Z: armRightZ}, 0.20, 0.36, 0.20, skinCol)

	rl.PopMatrix()
}
