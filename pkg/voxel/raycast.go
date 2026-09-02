package voxel

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// RaycastResult holds the outcome of a voxel raycast
type RaycastResult struct {
	Hit         bool
	BlockPos    rl.Vector3 // Integer coordinates of targeted block
	PlacePos    rl.Vector3 // Integer coordinates where a new block should be placed
	HitNormal   rl.Vector3 // Face normal (+Y, -Y, etc.)
	HitDistance float32
	BlockType   BlockType
}

// RaycastVoxel casts a ray through the 3D voxel grid (DDA Algorithm)
func RaycastVoxel(origin rl.Vector3, direction rl.Vector3, maxDist float32, world *VoxelWorld) RaycastResult {
	// Normalize direction
	dirLen := float32(math.Sqrt(float64(direction.X*direction.X + direction.Y*direction.Y + direction.Z*direction.Z)))
	if dirLen < 0.0001 {
		return RaycastResult{}
	}
	dir := rl.Vector3{X: direction.X / dirLen, Y: direction.Y / dirLen, Z: direction.Z / dirLen}

	stepSize := float32(0.04)
	dist := float32(0.0)

	curPos := origin
	var lastAirPos rl.Vector3 = rl.Vector3{
		X: float32(math.Floor(float64(origin.X))),
		Y: float32(math.Floor(float64(origin.Y))),
		Z: float32(math.Floor(float64(origin.Z))),
	}

	for dist < maxDist {
		curPos.X += dir.X * stepSize
		curPos.Y += dir.Y * stepSize
		curPos.Z += dir.Z * stepSize
		dist += stepSize

		bx := int(math.Floor(float64(curPos.X)))
		by := int(math.Floor(float64(curPos.Y)))
		bz := int(math.Floor(float64(curPos.Z)))

		bType := world.GetBlock(bx, by, bz)
		if bType != BlockAir && bType != BlockWater {
			// Found targeted block!
			hitPos := rl.Vector3{X: float32(bx), Y: float32(by), Z: float32(bz)}

			// Calculate face normal
			norm := rl.Vector3{
				X: lastAirPos.X - hitPos.X,
				Y: lastAirPos.Y - hitPos.Y,
				Z: lastAirPos.Z - hitPos.Z,
			}

			return RaycastResult{
				Hit:         true,
				BlockPos:    hitPos,
				PlacePos:    lastAirPos,
				HitNormal:   norm,
				HitDistance: dist,
				BlockType:   bType,
			}
		}

		lastAirPos = rl.Vector3{X: float32(bx), Y: float32(by), Z: float32(bz)}
	}

	return RaycastResult{}
}

// DrawTargetBlockOutline draws the iconic Minecraft black wireframe outline around the targeted block
func DrawTargetBlockOutline(pos rl.Vector3) {
	center := rl.Vector3{X: pos.X + 0.5, Y: pos.Y + 0.5, Z: pos.Z + 0.5}
	rl.DrawCubeWires(center, 1.004, 1.004, 1.004, rl.NewColor(30, 30, 30, 220))
}

// DrawMiningCracks draws authentic Minecraft 10-stage destroy cracking overlay on the block
func DrawMiningCracks(pos rl.Vector3, progress float32, atlas *TextureAtlas) {
	if progress <= 0.01 {
		return
	}
	stage := int(progress * 10.0)
	if stage < 0 {
		stage = 0
	}
	if stage > 9 {
		stage = 9
	}

	if atlas != nil && atlas.Texture.ID > 0 {
		u0 := float32(stage) / 16.0
		v0 := float32(5) / 16.0
		u1 := float32(stage+1) / 16.0
		v1 := float32(6) / 16.0

		x0 := pos.X - 0.003
		x1 := pos.X + 1.003
		y0 := pos.Y - 0.003
		y1 := pos.Y + 1.003
		z0 := pos.Z - 0.003
		z1 := pos.Z + 1.003

		rl.Begin(rl.Quads)
		rl.SetTexture(atlas.Texture.ID)
		rl.Color4ub(255, 255, 255, 255)

		// Top Face (+Y)
		rl.TexCoord2f(u0, v1); rl.Vertex3f(x0, y1, z1)
		rl.TexCoord2f(u1, v1); rl.Vertex3f(x1, y1, z1)
		rl.TexCoord2f(u1, v0); rl.Vertex3f(x1, y1, z0)
		rl.TexCoord2f(u0, v0); rl.Vertex3f(x0, y1, z0)

		// Bottom Face (-Y)
		rl.TexCoord2f(u0, v1); rl.Vertex3f(x0, y0, z0)
		rl.TexCoord2f(u1, v1); rl.Vertex3f(x1, y0, z0)
		rl.TexCoord2f(u1, v0); rl.Vertex3f(x1, y0, z1)
		rl.TexCoord2f(u0, v0); rl.Vertex3f(x0, y0, z1)

		// South Face (+Z)
		rl.TexCoord2f(u0, v1); rl.Vertex3f(x0, y0, z1)
		rl.TexCoord2f(u1, v1); rl.Vertex3f(x1, y0, z1)
		rl.TexCoord2f(u1, v0); rl.Vertex3f(x1, y1, z1)
		rl.TexCoord2f(u0, v0); rl.Vertex3f(x0, y1, z1)

		// North Face (-Z)
		rl.TexCoord2f(u1, v1); rl.Vertex3f(x1, y0, z0)
		rl.TexCoord2f(u0, v1); rl.Vertex3f(x0, y0, z0)
		rl.TexCoord2f(u0, v0); rl.Vertex3f(x0, y1, z0)
		rl.TexCoord2f(u1, v0); rl.Vertex3f(x1, y1, z0)

		// East Face (+X)
		rl.TexCoord2f(u0, v1); rl.Vertex3f(x1, y0, z1)
		rl.TexCoord2f(u1, v1); rl.Vertex3f(x1, y0, z0)
		rl.TexCoord2f(u1, v0); rl.Vertex3f(x1, y1, z0)
		rl.TexCoord2f(u0, v0); rl.Vertex3f(x1, y1, z1)

		// West Face (-X)
		rl.TexCoord2f(u0, v1); rl.Vertex3f(x0, y0, z0)
		rl.TexCoord2f(u1, v1); rl.Vertex3f(x0, y0, z1)
		rl.TexCoord2f(u1, v0); rl.Vertex3f(x0, y1, z1)
		rl.TexCoord2f(u0, v0); rl.Vertex3f(x0, y1, z0)

		rl.End()
	}
}


