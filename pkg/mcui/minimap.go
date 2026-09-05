package mcui

import (
	"fmt"
	"math"

	"gocraft/pkg/mcmob"
	"gocraft/pkg/voxel"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Minimap displays a tactical top-down radar, player arrow, mob tracking, and world coordinates
type Minimap struct {
	IsVisible bool
}

// NewMinimap creates a new minimap instance enabled by default
func NewMinimap() *Minimap {
	return &Minimap{
		IsVisible: true,
	}
}

// Toggle toggles minimap visibility ('M' key)
func (m *Minimap) Toggle() {
	m.IsVisible = !m.IsVisible
}

// Render draws the minimap in the top-right corner of the screen
func (m *Minimap) Render(world *voxel.VoxelWorld, playerPos rl.Vector3, playerYaw float32, mobs []*mcmob.Mob, screenW, screenH int32) {
	if !m.IsVisible || world == nil {
		return
	}

	size := float32(130.0)
	margin := float32(12.0)
	mapX := float32(screenW) - size - margin
	mapY := margin

	centerX := mapX + size*0.5
	centerY := mapY + size*0.5
	radius := size * 0.46

	// 1. Background dark panel & border
	rl.DrawRectangleRounded(rl.NewRectangle(mapX-4, mapY-4, size+8, size+28), 0.12, 6, rl.NewColor(18, 20, 26, 215))
	rl.DrawRectangleRoundedLines(rl.NewRectangle(mapX-4, mapY-4, size+8, size+28), 0.12, 6, rl.NewColor(65, 75, 95, 255))

	// Circular radar mask
	rl.DrawCircle(int32(centerX), int32(centerY), radius, rl.NewColor(12, 14, 18, 240))

	// 2. Sample and draw top-down terrain voxels
	gridRadius := 22 // 44x44 blocks area
	stepPx := (radius * 1.8) / float32(gridRadius*2)
	plX := int(math.Floor(float64(playerPos.X)))
	plZ := int(math.Floor(float64(playerPos.Z)))

	for gx := -gridRadius; gx <= gridRadius; gx++ {
		for gz := -gridRadius; gz <= gridRadius; gz++ {
			// Circular clipping check
			fx := float32(gx) * stepPx
			fz := float32(gz) * stepPx
			if fx*fx+fz*fz > radius*radius*0.92 {
				continue
			}

			wx := plX + gx
			wz := plZ + gz
			hy := world.GetHighestBlock(wx, wz)
			b := world.GetBlock(wx, hy, wz)

			var col rl.Color
			if voxel.IsWater(b) {
				col = rl.NewColor(44, 115, 235, 240) // Azure Water
			} else if b == voxel.BlockGrass {
				col = rl.NewColor(85, 155, 52, 255) // Lush Grass
			} else if b == voxel.BlockDirt {
				col = rl.NewColor(134, 96, 67, 255) // Dirt
			} else if b == voxel.BlockSand || b == voxel.BlockSandstone {
				col = rl.NewColor(218, 204, 145, 255) // Sand
			} else if b == voxel.BlockStone || b == voxel.BlockCobblestone || b == voxel.BlockMossyCobblestone {
				col = rl.NewColor(125, 125, 125, 255) // Stone
			} else if voxel.IsLog(b) {
				col = rl.NewColor(168, 134, 88, 255) // Wood Log
			} else if voxel.IsLeaf(b) {
				col = rl.NewColor(42, 98, 22, 255) // Leaves
			} else {
				col = rl.NewColor(80, 80, 80, 255)
			}

			// Height hill shading
			if hy > int(playerPos.Y) {
				col.R = uint8(math.Min(255, float64(col.R)+18))
				col.G = uint8(math.Min(255, float64(col.G)+18))
				col.B = uint8(math.Min(255, float64(col.B)+18))
			} else if hy < int(playerPos.Y) {
				col.R = uint8(math.Max(0, float64(col.R)-18))
				col.G = uint8(math.Max(0, float64(col.G)-18))
				col.B = uint8(math.Max(0, float64(col.B)-18))
			}

			px := centerX + fx
			py := centerY + fz
			rl.DrawRectangle(int32(px), int32(py), int32(math.Ceil(float64(stepPx))), int32(math.Ceil(float64(stepPx))), col)
		}
	}

	// 3. Mob Radar Tracking Dots
	for _, mob := range mobs {
		if mob.IsDead {
			continue
		}
		dx := mob.Pos.X - playerPos.X
		dz := mob.Pos.Z - playerPos.Z
		distSq := dx*dx + dz*dz
		if distSq > float32(gridRadius*gridRadius) {
			continue
		}

		dotX := centerX + (dx/float32(gridRadius))*radius*0.85
		dotY := centerY + (dz/float32(gridRadius))*radius*0.85

		if mob.IsHostile() {
			// Red dot for monsters (Zombie, Skeleton, Creeper)
			rl.DrawCircle(int32(dotX), int32(dotY), 2.5, rl.NewColor(245, 45, 45, 255))
			rl.DrawCircle(int32(dotX), int32(dotY), 1.2, rl.White)
		} else {
			// Green dot for passive animals (Cow, Pig, Sheep)
			rl.DrawCircle(int32(dotX), int32(dotY), 2.0, rl.NewColor(55, 225, 75, 255))
		}
	}

	// 4. Player Indicator Arrow (Rotated according to playerYaw)
	sinY := float32(math.Sin(float64(playerYaw)))
	cosY := float32(math.Cos(float64(playerYaw)))

	// Arrow tip (pointing in look direction: X=-sinY, Z=-cosY)
	tipX := centerX - sinY*7.0
	tipY := centerY - cosY*7.0
	leftX := centerX + sinY*4.0 - cosY*5.0
	leftY := centerY + cosY*4.0 + sinY*5.0
	rightX := centerX + sinY*4.0 + cosY*5.0
	rightY := centerY + cosY*4.0 - sinY*5.0

	rl.DrawTriangle(
		rl.Vector2{X: tipX, Y: tipY},
		rl.Vector2{X: leftX, Y: leftY},
		rl.Vector2{X: rightX, Y: rightY},
		rl.NewColor(255, 40, 40, 255), // Red player arrow
	)
	rl.DrawTriangleLines(
		rl.Vector2{X: tipX, Y: tipY},
		rl.Vector2{X: leftX, Y: leftY},
		rl.Vector2{X: rightX, Y: rightY},
		rl.White,
	)

	// Radar circle rim
	rl.DrawCircleLines(int32(centerX), int32(centerY), radius, rl.NewColor(180, 195, 215, 255))

	// 5. Cardinal Compass Markers (N, S, E, W)
	rl.DrawText("N", int32(centerX)-3, int32(centerY-radius-9), 10, rl.NewColor(255, 65, 55, 255)) // Red North
	rl.DrawText("S", int32(centerX)-3, int32(centerY+radius+1), 10, rl.RayWhite)
	rl.DrawText("E", int32(centerX+radius+2), int32(centerY)-4, 10, rl.RayWhite)
	rl.DrawText("W", int32(centerX-radius-10), int32(centerY)-4, 10, rl.RayWhite)

	// 6. Coordinates display under the radar
	coordStr := fmt.Sprintf("X:%d Y:%d Z:%d", int(playerPos.X), int(playerPos.Y), int(playerPos.Z))
	fontSize := int32(11)
	cLen := rl.MeasureText(coordStr, fontSize)
	rl.DrawText(coordStr, int32(mapX+size*0.5)-cLen/2, int32(mapY+size+7), fontSize, rl.NewColor(225, 225, 235, 255))
}
