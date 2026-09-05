package mcmob

import (
	"math"

	"gocraft/pkg/voxel"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// ItemEntity represents a dropped item/block in the world
type ItemEntity struct {
	Type        voxel.BlockType
	Count       int
	Pos         rl.Vector3
	Vel         rl.Vector3
	LifeTimer   float32
	PickupDelay float32
	HoverOffset float32
	RotationY   float32
	IsGrounded  bool
}

// NewItemEntity creates a new dropped item in the world
func NewItemEntity(bType voxel.BlockType, count int, pos rl.Vector3, vel rl.Vector3) *ItemEntity {
	return &ItemEntity{
		Type:        bType,
		Count:       count,
		Pos:         pos,
		Vel:         vel,
		LifeTimer:   300.0, // 5 minutes despawn
		PickupDelay: 1.5,   // Cannot be picked up immediately
	}
}

// Update handles gravity, collision, and bouncing for the item
func (item *ItemEntity) Update(dt float32, world *voxel.VoxelWorld) {
	if item.PickupDelay > 0 {
		item.PickupDelay -= dt
	}
	item.LifeTimer -= dt

	// Rotation & Hover animations
	item.RotationY += dt * 2.0 // rotate 2 radians per second (about 1 revolution per 3 seconds)
	item.HoverOffset += dt * 3.0

	// Gravity & Water Buoyancy
	ix := int(math.Floor(float64(item.Pos.X)))
	iy := int(math.Floor(float64(item.Pos.Y)))
	iz := int(math.Floor(float64(item.Pos.Z)))
	inWater := voxel.IsWater(world.GetBlock(ix, iy, iz))

	if inWater {
		// Floating buoyancy: items float to the surface
		item.Vel.Y += 12.0 * dt
		if item.Vel.Y > 2.2 {
			item.Vel.Y = 2.2
		}
		item.Vel.X *= 0.85
		item.Vel.Z *= 0.85
	} else {
		// Gravity in air
		item.Vel.Y -= 15.0 * dt
		if item.Vel.Y < -20.0 {
			item.Vel.Y = -20.0
		}
	}
	
	item.moveWithCollision(item.Vel.X*dt, item.Vel.Y*dt, item.Vel.Z*dt, world)
	
	// Friction
	if item.IsGrounded {
		item.Vel.X *= 0.8
		item.Vel.Z *= 0.8
	} else if !inWater {
		item.Vel.X *= 0.98
		item.Vel.Z *= 0.98
	}
}

// moveWithCollision moves item with AABB voxel collisions
func (item *ItemEntity) moveWithCollision(dx, dy, dz float32, world *voxel.VoxelWorld) {
	halfW := float32(0.125) // 0.25 size
	height := float32(0.25)

	// Y movement
	newY := item.Pos.Y + dy
	if checkItemCollision(item.Pos.X, newY, item.Pos.Z, halfW, height, world) {
		if dy < 0 {
			item.IsGrounded = true
			item.Pos.Y = float32(math.Floor(float64(newY))) + 1.0
			// Bounce a tiny bit if velocity is high enough
			if item.Vel.Y < -3.0 {
				item.Vel.Y = -item.Vel.Y * 0.4
				item.IsGrounded = false
			} else {
				item.Vel.Y = 0
			}
		} else {
			item.Vel.Y = 0
		}
	} else {
		item.Pos.Y = newY
		item.IsGrounded = false
	}

	// X movement
	newX := item.Pos.X + dx
	if !checkItemCollision(newX, item.Pos.Y, item.Pos.Z, halfW, height, world) {
		item.Pos.X = newX
	} else {
		item.Vel.X = -item.Vel.X * 0.5 // Bounce off wall
	}

	// Z movement
	newZ := item.Pos.Z + dz
	if !checkItemCollision(item.Pos.X, item.Pos.Y, newZ, halfW, height, world) {
		item.Pos.Z = newZ
	} else {
		item.Vel.Z = -item.Vel.Z * 0.5 // Bounce off wall
	}
}

func checkItemCollision(px, py, pz, halfW, height float32, world *voxel.VoxelWorld) bool {
	minX := int(math.Floor(float64(px - halfW)))
	maxX := int(math.Floor(float64(px + halfW)))
	minY := int(math.Floor(float64(py)))
	maxY := int(math.Floor(float64(py + height)))
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
