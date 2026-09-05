package mcmob

import (
	"testing"

	"gocraft/pkg/voxel"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestArrowPhysicsAndPlayerHit(t *testing.T) {
	pos := rl.Vector3{X: 0, Y: 10, Z: 0}
	vel := rl.Vector3{X: 10, Y: 0, Z: 0}
	arrow := NewArrow(pos, vel, 4.0, true)

	// 1. Initial State
	if arrow.Damage != 4.0 || !arrow.IsFromMob {
		t.Fatalf("Unexpected arrow initial state: damage=%.1f, isFromMob=%v", arrow.Damage, arrow.IsFromMob)
	}

	// 2. Physics & Gravity
	dt := float32(0.1)
	arrow.Update(dt, nil, rl.Vector3{X: 100, Y: 100, Z: 100}, nil, nil)
	if arrow.Vel.Y >= 0 {
		t.Fatalf("Expected negative Y velocity due to gravity, got %.2f", arrow.Vel.Y)
	}

	// 3. Player Hit Test
	playerPos := rl.Vector3{X: 1.0, Y: 8.8, Z: 0}
	playerHealth := float32(20.0)
	hitCalled := false

	arrow.Pos = rl.Vector3{X: 1.0, Y: 9.7, Z: 0} // Close to player eye (8.8 + 0.9 = 9.7)
	arrow.Vel = rl.Vector3{X: 0, Y: 0, Z: 0}

	despawn := arrow.Update(0.01, nil, playerPos, &playerHealth, func() {
		hitCalled = true
	})

	if !despawn {
		t.Fatalf("Expected arrow to despawn upon hitting player!")
	}
	if !hitCalled {
		t.Fatalf("Expected onHit callback to be executed!")
	}
	if playerHealth != 16.0 {
		t.Fatalf("Expected player health to be 16.0 after 4 damage, got %.1f", playerHealth)
	}
}

func TestSkeletonArcheryShooting(t *testing.T) {
	world := voxel.NewVoxelWorld()
	// Floor under skeleton and player
	for x := -8; x <= 8; x++ {
		for z := -5; z <= 18; z++ {
			world.SetBlock(x, 4, z, voxel.BlockStone)
		}
	}

	skeletonPos := rl.Vector3{X: 0, Y: 5, Z: 0}
	skel := NewMob(MobSkeleton, skeletonPos)

	playerPos := rl.Vector3{X: 0, Y: 5, Z: 10} // 10 blocks away (within 16 block combat range)
	playerHealth := float32(20.0)

	// Update for 2.5 seconds to trigger 2.0s firing cooldown
	var shotArrow *Arrow
	for step := 0; step < 26; step++ {
		_, arrow := skel.Update(0.1, playerPos, &playerHealth, world, 0.0)
		if arrow != nil {
			shotArrow = arrow
			break
		}
	}

	if shotArrow == nil {
		t.Fatalf("Expected Skeleton to shoot an arrow at the player within 2.1s!")
	}

	if !shotArrow.IsFromMob {
		t.Fatalf("Expected arrow.IsFromMob to be true")
	}

	// Arrow should be flying towards player (+Z direction)
	if shotArrow.Vel.Z <= 0 {
		t.Fatalf("Expected arrow velocity Z to be positive towards player, got %.2f", shotArrow.Vel.Z)
	}
}

func TestArrowVoxelCollision(t *testing.T) {
	world := voxel.NewVoxelWorld()

	// Place solid stone block at (5, 5, 5)
	world.SetBlock(5, 5, 5, voxel.BlockStone)

	arrow := NewArrow(rl.Vector3{X: 5.2, Y: 5.2, Z: 4.2}, rl.Vector3{X: 0, Y: 0, Z: 15.0}, 3.0, true)

	// Update arrow to fly into stone block
	arrow.Update(0.1, world, rl.Vector3{X: 100, Y: 100, Z: 100}, nil, nil)

	if !arrow.IsStuck {
		t.Fatalf("Expected arrow to get stuck in the stone block!")
	}
	if arrow.Vel != (rl.Vector3{}) {
		t.Fatalf("Expected stuck arrow velocity to be zero, got %v", arrow.Vel)
	}
}
