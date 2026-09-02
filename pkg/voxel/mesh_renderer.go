package voxel

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// FaceType defines which face of a cube is being drawn
type FaceType int

const (
	FaceTop FaceType = iota
	FaceBottom
	FaceNorth // -Z
	FaceSouth // +Z
	FaceWest  // -X
	FaceEast  // +X
)

// FaceLightingMultipliers provides directional sunlight shading for each face
var FaceLightingMultipliers = map[FaceType]float32{
	FaceTop:    1.00, // Direct Sunlight
	FaceNorth:  0.82, // Angled Light
	FaceSouth:  0.82, // Angled Light
	FaceEast:   0.65, // Side Light
	FaceWest:   0.65, // Side Light
	FaceBottom: 0.48, // Underside Shadow
}

// GetBlockTextureAtlasPos returns the (col, row) tile coordinates (0..15, 0..15) in the Atlas
func GetBlockTextureAtlasPos(b BlockType, face FaceType) (int, int) {
	switch b {
	case BlockGrass:
		if face == FaceTop {
			return 0, 0
		} else if face == FaceBottom {
			return 2, 0 // Dirt
		}
		return 1, 0 // Grass side
	case BlockDirt:
		return 2, 0
	case BlockStone:
		return 3, 0
	case BlockCobblestone:
		return 4, 0
	case BlockMossyCobblestone:
		return 5, 0
	case BlockBedrock:
		return 6, 0
	case BlockSand:
		return 7, 0
	case BlockSandstone:
		if face == FaceTop {
			return 8, 0
		} else if face == FaceBottom {
			return 10, 0
		}
		return 9, 0
	case BlockOakLog:
		if face == FaceTop || face == FaceBottom {
			return 12, 0
		}
		return 11, 0
	case BlockOakPlanks:
		return 13, 0
	case BlockOakLeaves:
		return 14, 0
	case BlockGlass:
		return 15, 0
	case BlockCoalOre:
		return 0, 1
	case BlockIronOre:
		return 1, 1
	case BlockGoldOre:
		return 2, 1
	case BlockDiamondOre:
		return 3, 1
	case BlockRedstoneOre:
		return 4, 1
	case BlockEmeraldOre:
		return 5, 1
	case BlockLapisOre:
		return 6, 1
	case BlockBricks:
		return 7, 1
	case BlockTNT:
		if face == FaceTop {
			return 9, 1
		} else if face == FaceBottom {
			return 10, 1
		}
		return 8, 1
	case BlockCraftingTable:
		if face == FaceTop {
			return 11, 1
		} else if face == FaceBottom {
			return 13, 0 // Oak Planks
		}
		return 12, 1
	case BlockFurnace:
		if face == FaceNorth {
			return 13, 1 // Front
		} else if face == FaceTop || face == FaceBottom {
			return 4, 0 // Cobble
		}
		return 14, 1 // Furnace side
	case BlockBookshelf:
		if face == FaceTop || face == FaceBottom {
			return 13, 0 // Planks
		}
		return 15, 1
	case BlockTorch:
		return 0, 2
	case BlockWater:
		return 1, 2
	case ItemStick:
		return 0, 3
	case ItemDiamond:
		return 1, 3
	case ItemIronIngot:
		return 2, 3
	case ItemGoldIngot:
		return 3, 3
	case ItemCoal:
		return 4, 3
	case ItemWoodPickaxe:
		return 5, 3
	case ItemWoodAxe:
		return 6, 3
	case ItemWoodShovel:
		return 7, 3
	case ItemWoodSword:
		return 8, 3
	case ItemStonePickaxe:
		return 9, 3
	case ItemStoneAxe:
		return 10, 3
	case ItemStoneShovel:
		return 11, 3
	case ItemStoneSword:
		return 12, 3
	case ItemIronPickaxe:
		return 13, 3
	case ItemIronAxe:
		return 14, 3
	case ItemDiamondPickaxe:
		return 15, 3
	case ItemIronShovel:
		return 0, 4
	case ItemIronSword:
		return 1, 4
	case ItemDiamondAxe:
		return 2, 4
	case ItemDiamondShovel:
		return 3, 4
	case ItemDiamondSword:
		return 4, 4
	case BlockWool:
		return 5, 4
	case BlockObsidian:
		return 6, 4
	case ItemRawBeef:
		return 7, 4
	case ItemCookedBeef:
		return 8, 4
	case ItemRawPorkchop:
		return 9, 4
	case ItemCookedPorkchop:
		return 10, 4
	case ItemApple:
		return 11, 4
	case ItemBread:
		return 12, 4
	case ItemRottenFlesh:
		return 13, 4
	case ItemGunpowder:
		return 14, 4
	case ItemBone:
		return 15, 4
	case ItemArrow:
		return 10, 5
	default:
		return 0, 0
	}
}

// AtlasPixelWidth is set during atlas generation so UV inset can scale with resolution
var AtlasPixelWidth float32 = 256.0 // default for 16x16 tiles × 16 cols

// GetBlockTextureUVs returns UV (uMin, vMin, uMax, vMax) in texture atlas for block face
func GetBlockTextureUVs(b BlockType, face FaceType) (float32, float32, float32, float32) {
	col, row := GetBlockTextureAtlasPos(b, face)

	cellW := 1.0 / float32(AtlasCols)
	cellH := 1.0 / float32(AtlasRows)

	// Dynamic half-texel inset: scales correctly with any atlas resolution (16px, 64px, 128px HD, etc.)
	// This completely eliminates texture edge bleeding / colored lines between blocks
	eps := 0.5 / AtlasPixelWidth

	uMin := float32(col)*cellW + eps
	vMin := float32(row)*cellH + eps
	uMax := float32(col+1)*cellW - eps
	vMax := float32(row+1)*cellH - eps

	return uMin, vMin, uMax, vMax
}

// CalculateVertexAO calculates smooth corner ambient occlusion (0 = darkest, 3 = brightest)
func CalculateVertexAO(side1, side2, corner bool) float32 {
	if side1 && side2 {
		return 0.68 // Soft corner shadow
	}
	count := 0
	if side1 {
		count++
	}
	if side2 {
		count++
	}
	if corner {
		count++
	}
	switch count {
	case 1:
		return 0.90
	case 2:
		return 0.78
	case 3:
		return 0.68
	default:
		return 1.00 // Full ambient light
	}
}

func shadeColor(brightness float32) rl.Color {
	val := uint8(brightness * 255.0)
	return rl.NewColor(val, val, val, 255)
}
