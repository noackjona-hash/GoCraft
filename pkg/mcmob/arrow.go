package mcmob

import (
	"math"

	"gocraft/pkg/voxel"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Arrow represents an airborne or stuck projectile
type Arrow struct {
	Pos       rl.Vector3
	Vel       rl.Vector3
	Yaw       float32
	Pitch     float32
	IsStuck   bool
	LifeTimer float32 // Seconds until despawn
	Damage    float32
	IsFromMob bool
}

// NewArrow spawns an arrow with an initial velocity and ballistic angle
func NewArrow(pos, vel rl.Vector3, damage float32, isFromMob bool) *Arrow {
	yaw := float32(math.Atan2(float64(vel.X), float64(vel.Z)))
	horiz := float32(math.Sqrt(float64(vel.X*vel.X + vel.Z*vel.Z)))
	pitch := float32(math.Atan2(float64(vel.Y), float64(horiz)))

	return &Arrow{
		Pos:       pos,
		Vel:       vel,
		Yaw:       yaw,
		Pitch:     pitch,
		LifeTimer: 18.0,
		Damage:    damage,
		IsFromMob: isFromMob,
	}
}

// Update handles arrow gravity, trajectory rotation, and collision with player and voxels
func (a *Arrow) Update(dt float32, world *voxel.VoxelWorld, playerPos rl.Vector3, playerHealth *float32, onHitPlayer func()) bool {
	a.LifeTimer -= dt
	if a.LifeTimer <= 0 {
		return true // Despawn
	}

	if a.IsStuck {
		return false
	}

	// Gravity acceleration
	a.Vel.Y -= 14.0 * dt
	if a.Vel.Y < -35.0 {
		a.Vel.Y = -35.0
	}

	// Update rotation to match trajectory
	a.Yaw = float32(math.Atan2(float64(a.Vel.X), float64(a.Vel.Z)))
	horiz := float32(math.Sqrt(float64(a.Vel.X*a.Vel.X + a.Vel.Z*a.Vel.Z)))
	a.Pitch = float32(math.Atan2(float64(a.Vel.Y), float64(horiz)))

	nextPos := rl.Vector3Add(a.Pos, rl.Vector3Scale(a.Vel, dt))

	// 1. Collision with Player
	if a.IsFromMob && playerHealth != nil {
		targetEye := playerPos
		targetEye.Y += 0.9
		distToPlayer := rl.Vector3Distance(nextPos, targetEye)
		if distToPlayer < 0.85 {
			*playerHealth -= a.Damage
			if *playerHealth < 0 {
				*playerHealth = 0
			}
			if onHitPlayer != nil {
				onHitPlayer()
			}
			return true // Despawn upon hitting player
		}
	}

	// 2. Collision with solid voxels
	bx := int(math.Floor(float64(nextPos.X)))
	by := int(math.Floor(float64(nextPos.Y)))
	bz := int(math.Floor(float64(nextPos.Z)))
	if world != nil && world.IsSolid(bx, by, bz) {
		a.IsStuck = true
		a.Pos = nextPos
		a.Vel = rl.Vector3{}
		a.LifeTimer = 8.0 // Stick in block for 8s
		return false
	}

	a.Pos = nextPos
	return false
}

// Render3D draws the arrow in world space
func (a *Arrow) Render3D() {
	rl.PushMatrix()
	rl.Translatef(a.Pos.X, a.Pos.Y, a.Pos.Z)
	rl.Rotatef(-a.Yaw*rl.Rad2deg, 0, 1, 0)
	rl.Rotatef(a.Pitch*rl.Rad2deg, 1, 0, 0)

	woodCol := rl.NewColor(168, 134, 88, 255)
	flintCol := rl.NewColor(60, 60, 65, 255)
	featherCol := rl.NewColor(235, 235, 235, 255)

	// Shaft (0.04 x 0.04 x 0.6)
	rl.DrawCube(rl.Vector3{X: 0, Y: 0, Z: 0}, 0.04, 0.04, 0.65, woodCol)

	// Flint Arrowhead (+Z tip)
	rl.DrawCube(rl.Vector3{X: 0, Y: 0, Z: 0.35}, 0.08, 0.08, 0.08, flintCol)

	// Feather fletching (-Z tail)
	rl.DrawCube(rl.Vector3{X: 0, Y: 0.04, Z: -0.25}, 0.02, 0.08, 0.12, featherCol)
	rl.DrawCube(rl.Vector3{X: 0.04, Y: 0, Z: -0.25}, 0.08, 0.02, 0.12, featherCol)

	rl.PopMatrix()
}
