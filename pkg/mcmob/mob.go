package mcmob

import (
	"math"
	"math/rand"

	"racing_game/pkg/voxel"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// MobType identifies the mob species
type MobType int

const (
	MobZombie MobType = iota
	MobSkeleton
	MobCreeper
	MobPig
	MobCow
	MobSheep
)

// MobState defines current AI activity
type MobState int

const (
	StateIdle MobState = iota
	StateWander
	StateChase
	StateExploding
	StateFlee
)

// Mob represents an active entity in the 3D world
type Mob struct {
	Type        MobType
	Pos         rl.Vector3
	Vel         rl.Vector3
	Yaw         float32
	Pitch       float32
	Health      float32
	MaxHealth   float32
	IsDead      bool
	IsOnFire    bool
	FireTimer   float32
	HurtTimer   float32
	AttackTimer float32
	FuseTimer   float32 // Creeper countdown (1.5s)
	State       MobState
	StateTimer  float32
	TargetDir   rl.Vector3
	WalkBobbing float32
	Width       float32
	Height      float32
	IsGrounded  bool
}

// NewMob constructs a mob with authentic attributes
func NewMob(mType MobType, pos rl.Vector3) *Mob {
	m := &Mob{
		Type:       mType,
		Pos:        pos,
		Health:     20,
		MaxHealth:  20,
		Width:      0.6,
		Height:     1.8,
		State:      StateIdle,
		StateTimer: 1.0 + rand.Float32()*2.0,
	}

	switch mType {
	case MobZombie, MobSkeleton, MobCreeper:
		m.Health = 20
		m.MaxHealth = 20
		m.Height = 1.8
		m.Width = 0.6
	case MobPig:
		m.Health = 10
		m.MaxHealth = 10
		m.Height = 0.9
		m.Width = 0.9
	case MobCow:
		m.Health = 10
		m.MaxHealth = 10
		m.Height = 1.3
		m.Width = 0.9
	case MobSheep:
		m.Health = 8
		m.MaxHealth = 8
		m.Height = 1.2
		m.Width = 0.9
	}

	return m
}

// IsHostile returns true for monsters
func (m *Mob) IsHostile() bool {
	return m.Type == MobZombie || m.Type == MobSkeleton || m.Type == MobCreeper
}

// Update ticks AI, movement physics, sunlight burning, and combat timers
func (m *Mob) Update(dt float32, playerPos rl.Vector3, playerHealth *float32, world *voxel.VoxelWorld, sunHeight float32) (exploded bool) {
	if m.IsDead {
		return false
	}

	// 1. Hurt & Fire Timers
	if m.HurtTimer > 0 {
		m.HurtTimer -= dt
	}
	if m.AttackTimer > 0 {
		m.AttackTimer -= dt
	}

	// Sunlight burning for Zombies & Skeletons
	if (m.Type == MobZombie || m.Type == MobSkeleton) && sunHeight > 0 {
		bx := int(math.Floor(float64(m.Pos.X)))
		by := int(math.Floor(float64(m.Pos.Y)))
		bz := int(math.Floor(float64(m.Pos.Z)))
		skyLight, _ := world.GetLightLevel(bx, by, bz)
		if skyLight >= 0.85 {
			m.IsOnFire = true
			m.Health -= dt * 2.0
			if m.Health <= 0 {
				m.IsDead = true
				return false
			}
		}
	}

	// 2. AI State Machine
	dx := playerPos.X - m.Pos.X
	dy := playerPos.Y - m.Pos.Y
	dz := playerPos.Z - m.Pos.Z
	distSq := dx*dx + dy*dy + dz*dz
	dist := float32(math.Sqrt(float64(distSq)))

	moveSpeed := float32(2.2)

	if m.IsHostile() {
		if dist < 16.0 {
			m.State = StateChase
			// Turn towards player
			m.Yaw = float32(math.Atan2(float64(dx), float64(dz)))
			dirX := dx / dist
			dirZ := dz / dist

			if m.Type == MobCreeper {
				moveSpeed = 2.6
				if dist < 2.8 {
					m.State = StateExploding
					m.FuseTimer += dt
					if m.FuseTimer >= 1.4 {
						m.IsDead = true
						return true // Trigger explosion!
					}
				} else {
					if m.FuseTimer > 0 {
						m.FuseTimer -= dt * 0.8
					}
				}
			}

			// Chase movement
			m.Vel.X = dirX * moveSpeed
			m.Vel.Z = dirZ * moveSpeed

			// Melee attack player
			if dist < 1.3 && m.AttackTimer <= 0 && m.Type != MobCreeper {
				if playerHealth != nil {
					*playerHealth -= 3.0
				}
				m.AttackTimer = 1.0
			}
		} else {
			m.State = StateIdle
			m.Vel.X *= 0.8
			m.Vel.Z *= 0.8
		}
	} else {
		// Rotation & Hover animations
		item.RotationY += dt * 1.0 
		item.HoverOffset += (float32(math.Sin(float64(item.HoverOffset))) * 0.1)
		
		m.StateTimer -= dt
		if m.StateTimer <= 0 {
			if rand.Float32() < 0.6 {
				m.State = StateWander
				angle := rand.Float32() * math.Pi * 2.0
				m.TargetDir = rl.Vector3{X: float32(math.Sin(float64(angle))), Y: 0, Z: float32(math.Cos(float64(angle)))}
				m.Yaw = angle
				m.StateTimer = 2.0 + rand.Float32()*3.0
			} else {
				m.State = StateIdle
				m.TargetDir = rl.Vector3{}
				m.StateTimer = 2.0 + rand.Float32()*2.0
			}
		}

		if m.State == StateWander || m.State == StateFlee {
			spd := float32(1.5)
			if m.State == StateFlee {
				spd = 3.2
			}
			m.Vel.X = m.TargetDir.X * spd
			m.Vel.Z = m.TargetDir.Z * spd
		} else {
			m.Vel.X *= 0.8
			m.Vel.Z *= 0.8
		}
	}

	// 3. Gravity & Jumping over 1-block obstacles
	m.Vel.Y -= 22.0 * dt
	if m.Vel.Y < -25.0 {
		m.Vel.Y = -25.0
	}

	// Auto step-up / jump when walking against a 1-block wall
	newX := m.Pos.X + m.Vel.X*dt
	newZ := m.Pos.Z + m.Vel.Z*dt

	if m.IsGrounded && (m.Vel.X != 0 || m.Vel.Z != 0) {
		bx := int(math.Floor(float64(newX)))
		by := int(math.Floor(float64(m.Pos.Y)))
		bz := int(math.Floor(float64(newZ)))
		if world.IsSolid(bx, by, bz) && !world.IsSolid(bx, by+1, bz) {
			m.Vel.Y = 6.8 // Jump over 1-block step!
		}
	}

	// 4. Movement & Collision
	m.moveWithCollision(m.Vel.X*dt, m.Vel.Y*dt, m.Vel.Z*dt, world)

	// 5. Walk cycle animation
	speedHoriz := float32(math.Sqrt(float64(m.Vel.X*m.Vel.X + m.Vel.Z*m.Vel.Z)))
	if speedHoriz > 0.1 && m.IsGrounded {
		m.WalkBobbing += dt * speedHoriz * 4.5
	} else {
		m.WalkBobbing *= 0.9
	}

	return false
}

// moveWithCollision moves mob with AABB voxel collisions
func (m *Mob) moveWithCollision(dx, dy, dz float32, world *voxel.VoxelWorld) {
	halfW := m.Width * 0.5

	// Y movement
	newY := m.Pos.Y + dy
	if m.checkCollision(m.Pos.X, newY, m.Pos.Z, halfW, m.Height, world) {
		if dy < 0 {
			m.IsGrounded = true
			m.Pos.Y = float32(math.Floor(float64(newY))) + 1.0
			m.Vel.Y = 0
		} else {
			m.Vel.Y = 0
		}
	} else {
		m.Pos.Y = newY
		m.IsGrounded = false
	}

	// X movement
	newX := m.Pos.X + dx
	if !m.checkCollision(newX, m.Pos.Y, m.Pos.Z, halfW, m.Height, world) {
		m.Pos.X = newX
	}

	// Z movement
	newZ := m.Pos.Z + dz
	if !m.checkCollision(m.Pos.X, m.Pos.Y, newZ, halfW, m.Height, world) {
		m.Pos.Z = newZ
	}
}

func (m *Mob) checkCollision(px, py, pz, halfW, height float32, world *voxel.VoxelWorld) bool {
	minX := int(math.Floor(float64(px - halfW)))
	maxX := int(math.Floor(float64(px + halfW)))
	minY := int(math.Floor(float64(py)))
	maxY := int(math.Floor(float64(py + height - 0.05)))
	minZ := int(math.Floor(float64(pz - halfW)))
	maxZ := int(math.Floor(float64(pz + halfW)))

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for z := minZ; z <= maxZ; z++ {
				if world.IsSolid(x, y, z) {
					return true
				}
			}
		}
	}
	return false
}

// ApplyDamage handles combat hits, weapon damage, knockback, and aggro
func (m *Mob) ApplyDamage(damage float32, knockbackDir rl.Vector3) {
	m.Health -= damage
	m.HurtTimer = 0.35 // Red hurt flash

	// Apply knockback
	m.Vel.X += knockbackDir.X * 5.5
	m.Vel.Y = 3.8
	m.Vel.Z += knockbackDir.Z * 5.5

	if !m.IsHostile() {
		m.State = StateFlee
		m.StateTimer = 4.0
		m.TargetDir = knockbackDir
	}

	if m.Health <= 0 {
		m.IsDead = true
	}
}

// Render3D draws the authentic 3D Minecraft cuboid model in the world
func (m *Mob) Render3D(mm *MobManager) {
	if m.IsDead {
		return
	}

	rl.PushMatrix()
	rl.Translatef(m.Pos.X, m.Pos.Y, m.Pos.Z)
	rl.Rotatef(m.Yaw*rl.Rad2deg, 0, 1, 0)

	// Hurt color flash (bright red)
	tint := rl.White
	if m.HurtTimer > 0 {
		tint = rl.NewColor(255, 90, 90, 255)
	}

	switch m.Type {
	case MobZombie:
		m.renderZombie(tint, mm.TexZombie)
	case MobSkeleton:
		m.renderSkeleton(tint, mm.TexSkeleton)
	case MobCreeper:
		m.renderCreeper(tint, mm.TexCreeper)
	case MobPig:
		m.renderPig(tint, mm.TexPig)
	case MobCow:
		m.renderCow(tint, mm.TexCow)
	case MobSheep:
		m.renderSheep(tint, mm.TexSheep)
	}

	rl.PopMatrix()
}

// drawBoxUV draws a textured box using standard Minecraft texture layout
func drawBoxUV(x, y, z, w, h, d float32, u, v, tw, th, td float32, imgW, imgH float32, tint rl.Color) {
	// Standard MC layout (starting at U,V):
	// Top:    (u+td, v)          size: (tw, td)
	// Bottom: (u+td+tw, v)       size: (tw, td)
	// Right:  (u, v+td)          size: (td, th)
	// Front:  (u+td, v+td)       size: (tw, th)
	// Left:   (u+td+tw, v+td)    size: (td, th)
	// Back:   (u+td+tw+td, v+td) size: (tw, th)

	hsW, hsH, hsD := w/2, h/2, d/2
	rl.Color4ub(tint.R, tint.G, tint.B, tint.A)
	rl.Begin(rl.Quads)

	// Helper to add quad with UVs
	addQuad := func(p0, p1, p2, p3 rl.Vector3, texX, texY, texW, texH float32) {
		u0 := texX / imgW
		v0 := texY / imgH
		u1 := (texX + texW) / imgW
		v1 := (texY + texH) / imgH

		rl.TexCoord2f(u0, v1); rl.Vertex3f(p0.X, p0.Y, p0.Z)
		rl.TexCoord2f(u1, v1); rl.Vertex3f(p1.X, p1.Y, p1.Z)
		rl.TexCoord2f(u1, v0); rl.Vertex3f(p2.X, p2.Y, p2.Z)
		rl.TexCoord2f(u0, v0); rl.Vertex3f(p3.X, p3.Y, p3.Z)
	}

	// Front (+Z) (MC Front)
	addQuad(
		rl.Vector3{X: x - hsW, Y: y - hsH, Z: z + hsD},
		rl.Vector3{X: x + hsW, Y: y - hsH, Z: z + hsD},
		rl.Vector3{X: x + hsW, Y: y + hsH, Z: z + hsD},
		rl.Vector3{X: x - hsW, Y: y + hsH, Z: z + hsD},
		u+td, v+td, tw, th,
	)

	// Back (-Z) (MC Back)
	addQuad(
		rl.Vector3{X: x + hsW, Y: y - hsH, Z: z - hsD},
		rl.Vector3{X: x - hsW, Y: y - hsH, Z: z - hsD},
		rl.Vector3{X: x - hsW, Y: y + hsH, Z: z - hsD},
		rl.Vector3{X: x + hsW, Y: y + hsH, Z: z - hsD},
		u+td+tw+td, v+td, tw, th,
	)

	// Right (+X) (MC Left side of entity, looking from front)
	addQuad(
		rl.Vector3{X: x + hsW, Y: y - hsH, Z: z + hsD},
		rl.Vector3{X: x + hsW, Y: y - hsH, Z: z - hsD},
		rl.Vector3{X: x + hsW, Y: y + hsH, Z: z - hsD},
		rl.Vector3{X: x + hsW, Y: y + hsH, Z: z + hsD},
		u+td+tw, v+td, td, th,
	)

	// Left (-X) (MC Right side of entity, looking from front)
	addQuad(
		rl.Vector3{X: x - hsW, Y: y - hsH, Z: z - hsD},
		rl.Vector3{X: x - hsW, Y: y - hsH, Z: z + hsD},
		rl.Vector3{X: x - hsW, Y: y + hsH, Z: z + hsD},
		rl.Vector3{X: x - hsW, Y: y + hsH, Z: z - hsD},
		u, v+td, td, th,
	)

	// Top (+Y) (MC Top)
	addQuad(
		rl.Vector3{X: x - hsW, Y: y + hsH, Z: z + hsD},
		rl.Vector3{X: x + hsW, Y: y + hsH, Z: z + hsD},
		rl.Vector3{X: x + hsW, Y: y + hsH, Z: z - hsD},
		rl.Vector3{X: x - hsW, Y: y + hsH, Z: z - hsD},
		u+td, v, tw, td,
	)

	// Bottom (-Y) (MC Bottom)
	addQuad(
		rl.Vector3{X: x - hsW, Y: y - hsH, Z: z - hsD},
		rl.Vector3{X: x + hsW, Y: y - hsH, Z: z - hsD},
		rl.Vector3{X: x + hsW, Y: y - hsH, Z: z + hsD},
		rl.Vector3{X: x - hsW, Y: y - hsH, Z: z + hsD},
		u+td+tw, v, tw, td,
	)

	rl.End()
}


func (m *Mob) renderZombie(tint rl.Color, tex rl.Texture2D) {
	if tex.ID > 0 {
		rl.SetTexture(tex.ID)
		
		// Head (8x8x8)
		drawBoxUV(0, 1.62, 0, 0.48, 0.48, 0.48, 0, 0, 8, 8, 8, 64, 64, tint)
		
		// Torso (8x12x4)
		drawBoxUV(0, 1.08, 0, 0.48, 0.68, 0.24, 16, 16, 8, 12, 4, 64, 64, tint)

		// Legs (4x12x4)
		legSwing := float32(math.Sin(float64(m.WalkBobbing))) * 0.28
		drawBoxUV(-0.13, 0.38, legSwing, 0.22, 0.72, 0.22, 0, 16, 4, 12, 4, 64, 64, tint)
		drawBoxUV(0.13, 0.38, -legSwing, 0.22, 0.72, 0.22, 0, 16, 4, 12, 4, 64, 64, tint)

		// Arms (4x12x4)
		drawBoxUV(-0.34, 1.25, 0.28, 0.20, 0.20, 0.55, 40, 16, 4, 12, 4, 64, 64, tint)
		drawBoxUV(0.34, 1.25, 0.28, 0.20, 0.20, 0.55, 40, 16, 4, 12, 4, 64, 64, tint)
		
		rl.SetTexture(0)
	} else {
		// Fallback
		rl.DrawCube(rl.Vector3{X: 0, Y: 1.62, Z: 0}, 0.48, 0.48, 0.48, tint)
		rl.DrawCube(rl.Vector3{X: 0, Y: 1.08, Z: 0}, 0.48, 0.68, 0.24, tint)
	}
}

func (m *Mob) renderSkeleton(tint rl.Color, tex rl.Texture2D) {
	if tex.ID > 0 {
		rl.SetTexture(tex.ID)
		
		drawBoxUV(0, 1.62, 0, 0.46, 0.46, 0.46, 0, 0, 8, 8, 8, 64, 32, tint) // Skull
		drawBoxUV(0, 1.08, 0, 0.42, 0.68, 0.18, 16, 16, 8, 12, 4, 64, 32, tint) // Ribcage

		legSwing := float32(math.Sin(float64(m.WalkBobbing))) * 0.28
		drawBoxUV(-0.12, 0.38, legSwing, 0.14, 0.72, 0.14, 0, 16, 2, 12, 2, 64, 32, tint)
		drawBoxUV(0.12, 0.38, -legSwing, 0.14, 0.72, 0.14, 0, 16, 2, 12, 2, 64, 32, tint)

		drawBoxUV(-0.28, 1.22, 0.25, 0.12, 0.12, 0.52, 40, 16, 2, 12, 2, 64, 32, tint)
		drawBoxUV(0.28, 1.22, 0.25, 0.12, 0.12, 0.52, 40, 16, 2, 12, 2, 64, 32, tint)
		
		rl.SetTexture(0)
	} else {
		rl.DrawCube(rl.Vector3{X: 0, Y: 1.62, Z: 0}, 0.46, 0.46, 0.46, tint)
		rl.DrawCube(rl.Vector3{X: 0, Y: 1.08, Z: 0}, 0.42, 0.68, 0.18, tint)
	}
}

func (m *Mob) renderCreeper(tint rl.Color, tex rl.Texture2D) {
	scale := float32(1.0)
	if m.FuseTimer > 0 {
		scale = 1.0 + (m.FuseTimer/1.4)*0.25
		if int(m.FuseTimer*14)%2 == 0 {
			tint = rl.White
		}
	}
	
	if tex.ID > 0 {
		rl.SetTexture(tex.ID)
		drawBoxUV(0, 1.35*scale, 0, 0.48*scale, 0.48*scale, 0.48*scale, 0, 0, 8, 8, 8, 64, 32, tint) // Head
		drawBoxUV(0, 0.80*scale, 0, 0.44*scale, 0.64*scale, 0.24*scale, 16, 16, 8, 12, 4, 64, 32, tint) // Torso
		
		legSwing := float32(math.Sin(float64(m.WalkBobbing))) * 0.22
		drawBoxUV(-0.15, 0.22, 0.16+legSwing, 0.18, 0.44, 0.18, 0, 16, 4, 6, 4, 64, 32, tint)
		drawBoxUV(0.15, 0.22, 0.16-legSwing, 0.18, 0.44, 0.18, 0, 16, 4, 6, 4, 64, 32, tint)
		drawBoxUV(-0.15, 0.22, -0.16-legSwing, 0.18, 0.44, 0.18, 0, 16, 4, 6, 4, 64, 32, tint)
		drawBoxUV(0.15, 0.22, -0.16+legSwing, 0.18, 0.44, 0.18, 0, 16, 4, 6, 4, 64, 32, tint)
		rl.SetTexture(0)
	} else {
		rl.DrawCube(rl.Vector3{X: 0, Y: 1.35 * scale, Z: 0}, 0.48*scale, 0.48*scale, 0.48*scale, tint)
		rl.DrawCube(rl.Vector3{X: 0, Y: 0.80 * scale, Z: 0}, 0.44*scale, 0.64*scale, 0.24*scale, tint)
	}
}

func (m *Mob) renderPig(tint rl.Color, tex rl.Texture2D) {
	if tex.ID > 0 {
		rl.SetTexture(tex.ID)
		drawBoxUV(0, 0.52, 0, 0.58, 0.48, 0.78, 28, 8, 10, 16, 8, 64, 32, tint) // Body
		drawBoxUV(0, 0.68, 0.48, 0.44, 0.44, 0.44, 0, 0, 8, 8, 8, 64, 32, tint) // Head

		legSwing := float32(math.Sin(float64(m.WalkBobbing))) * 0.24
		drawBoxUV(-0.18, 0.18, 0.24+legSwing, 0.18, 0.36, 0.18, 0, 16, 4, 6, 4, 64, 32, tint)
		drawBoxUV(0.18, 0.18, 0.24-legSwing, 0.18, 0.36, 0.18, 0, 16, 4, 6, 4, 64, 32, tint)
		drawBoxUV(-0.18, 0.18, -0.24-legSwing, 0.18, 0.36, 0.18, 0, 16, 4, 6, 4, 64, 32, tint)
		drawBoxUV(0.18, 0.18, -0.24+legSwing, 0.18, 0.36, 0.18, 0, 16, 4, 6, 4, 64, 32, tint)
		rl.SetTexture(0)
	} else {
		rl.DrawCube(rl.Vector3{X: 0, Y: 0.52, Z: 0}, 0.58, 0.48, 0.78, tint)
	}
}

func (m *Mob) renderCow(tint rl.Color, tex rl.Texture2D) {
	if tex.ID > 0 {
		rl.SetTexture(tex.ID)
		drawBoxUV(0, 0.78, 0, 0.68, 0.62, 0.98, 18, 4, 12, 18, 10, 64, 32, tint) // Body
		drawBoxUV(0, 1.08, 0.58, 0.44, 0.44, 0.44, 0, 0, 8, 8, 6, 64, 32, tint) // Head

		legSwing := float32(math.Sin(float64(m.WalkBobbing))) * 0.24
		drawBoxUV(-0.22, 0.24, 0.32+legSwing, 0.18, 0.48, 0.18, 0, 16, 4, 12, 4, 64, 32, tint)
		drawBoxUV(0.22, 0.24, 0.32-legSwing, 0.18, 0.48, 0.18, 0, 16, 4, 12, 4, 64, 32, tint)
		drawBoxUV(-0.22, 0.24, -0.32-legSwing, 0.18, 0.48, 0.18, 0, 16, 4, 12, 4, 64, 32, tint)
		drawBoxUV(0.22, 0.24, -0.32+legSwing, 0.18, 0.48, 0.18, 0, 16, 4, 12, 4, 64, 32, tint)
		rl.SetTexture(0)
	} else {
		rl.DrawCube(rl.Vector3{X: 0, Y: 0.78, Z: 0}, 0.68, 0.62, 0.98, tint)
	}
}

func (m *Mob) renderSheep(tint rl.Color, tex rl.Texture2D) {
	if tex.ID > 0 {
		rl.SetTexture(tex.ID)
		drawBoxUV(0, 0.72, 0, 0.74, 0.64, 0.94, 28, 8, 8, 16, 6, 64, 32, tint) // Body
		drawBoxUV(0, 0.95, 0.56, 0.36, 0.36, 0.44, 0, 0, 6, 6, 8, 64, 32, tint) // Head

		legSwing := float32(math.Sin(float64(m.WalkBobbing))) * 0.24
		drawBoxUV(-0.20, 0.20, 0.28+legSwing, 0.16, 0.40, 0.16, 0, 16, 4, 6, 4, 64, 32, tint)
		drawBoxUV(0.20, 0.20, 0.28-legSwing, 0.16, 0.40, 0.16, 0, 16, 4, 6, 4, 64, 32, tint)
		drawBoxUV(-0.20, 0.20, -0.28-legSwing, 0.16, 0.40, 0.16, 0, 16, 4, 6, 4, 64, 32, tint)
		drawBoxUV(0.20, 0.20, -0.28+legSwing, 0.16, 0.40, 0.16, 0, 16, 4, 6, 4, 64, 32, tint)
		rl.SetTexture(0)
	} else {
		rl.DrawCube(rl.Vector3{X: 0, Y: 0.72, Z: 0}, 0.74, 0.64, 0.94, tint)
	}
}
