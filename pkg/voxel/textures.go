package voxel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	AtlasCols = 16
	AtlasRows = 16
)

// TextureAtlas holds the generated pixel art / HD texture atlas and UV coordinates
type TextureAtlas struct {
	Texture   rl.Texture2D
	Image     *rl.Image
	BlockSize int32 // 16, 32, 64, 128, 256, 512
	Width     int32 // BlockSize * 16
	Height    int32 // BlockSize * 16
}

// GenerateTextureAtlas dynamically detects texture pack resolution (128x128 HD, 64x64, etc.) and builds HD atlas with mipmaps
func GenerateTextureAtlas() *TextureAtlas {
	blockDirs := []string{
		"assets/textures/block",
		"assets/textures/blocks",
		"assets/assets/minecraft/textures/block",
		"assets/assets/minecraft/textures/blocks",
		"assets/minecraft/textures/block",
		"assets/minecraft/textures/blocks",
		"textures/block",
		"textures/blocks",
		"assets/blocks",
	}

	itemDirs := []string{
		"assets/textures/item",
		"assets/textures/items",
		"assets/assets/minecraft/textures/item",
		"assets/assets/minecraft/textures/items",
		"assets/minecraft/textures/item",
		"assets/minecraft/textures/items",
		"textures/item",
		"textures/items",
		"assets/items",
	}

	findDir := func(dirs []string) string {
		for _, d := range dirs {
			if fi, err := os.Stat(d); err == nil && fi.IsDir() {
				return d
			}
		}
		return ""
	}

	blockDir := findDir(blockDirs)
	itemDir := findDir(itemDirs)

	// 1. Detect native texture pack resolution (e.g. 128x128 HD, 64x64, 32x32, 16x16)
	detectedTileSize := int32(16)
	if blockDir != "" {
		testFiles := []string{"dirt.png", "stone.png", "grass_block_top.png", "oak_log.png", "sand.png", "cobblestone.png"}
		for _, tf := range testFiles {
			p := filepath.Join(blockDir, tf)
			if _, err := os.Stat(p); err == nil {
				img := rl.LoadImage(p)
				if img.Width > 0 && img.Width == img.Height {
					detectedTileSize = img.Width
					rl.UnloadImage(img)
					break
				}
				if img.Width > 0 {
					rl.UnloadImage(img)
				}
			}
		}
	}

	if detectedTileSize < 16 {
		detectedTileSize = 16
	} else if detectedTileSize > 512 {
		detectedTileSize = 512
	}

	atlasW := detectedTileSize * AtlasCols
	atlasH := detectedTileSize * AtlasRows

	fmt.Printf("Detected Texture Resolution: %dx%d (HD Atlas Size: %dx%d)\n", detectedTileSize, detectedTileSize, atlasW, atlasH)

	// 2. Generate Base Procedural Atlas (16x16 per block)
	baseImg := rl.GenImageColor(16*AtlasCols, 16*AtlasRows, rl.Blank)
	generateProceduralBase(baseImg)

	// 3. Upscale base procedural canvas to match HD atlas size
	if detectedTileSize != 16 {
		rl.ImageResizeNN(baseImg, atlasW, atlasH)
	}

	// 4. Stitch custom HD texture pack PNGs at full native resolution!
	stitchedCount := stitchResourcePack(baseImg, blockDir, itemDir, detectedTileSize)
	if stitchedCount > 0 {
		fmt.Printf("Successfully loaded and stitched %d HD textures from custom Texture Pack!\n", stitchedCount)
	}

	tex := rl.LoadTextureFromImage(baseImg)
	// Generate Mipmaps for crisp HD rendering without distant noise/aliasing
	rl.GenTextureMipmaps(&tex)
	rl.SetTextureFilter(tex, rl.FilterAnisotropic16x) // High quality 16x Anisotropic Minecraft HD texture filtering

	// Set global atlas pixel width for dynamic UV half-texel inset calculation
	AtlasPixelWidth = float32(atlasW)

	return &TextureAtlas{
		Texture:   tex,
		Image:     baseImg,
		BlockSize: detectedTileSize,
		Width:     atlasW,
		Height:    atlasH,
	}
}

// findFileRecursive searches for a list of candidate filenames inside a root directory and subdirectories
func findFileRecursive(rootDir string, candidateNames ...string) string {
	for _, name := range candidateNames {
		direct := filepath.Join(rootDir, name)
		if _, err := os.Stat(direct); err == nil {
			return direct
		}
	}

	var foundPath string
	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		base := strings.ToLower(info.Name())
		if strings.HasSuffix(base, "_n.png") || strings.HasSuffix(base, "_s.png") {
			return nil
		}
		for _, name := range candidateNames {
			if strings.EqualFold(base, name) {
				foundPath = path
				return filepath.SkipDir
			}
		}
		return nil
	})
	return foundPath
}

// stitchResourcePack scans standard Minecraft resource pack folders and composites individual PNGs into the atlas
func stitchResourcePack(img *rl.Image, blockDir, itemDir string, tileSize int32) int {
	if blockDir == "" && itemDir == "" {
		return 0
	}

	blockMap := map[string][][2]int{
		"grass_block_top.png":         {{0, 0}},
		"grass_top.png":               {{0, 0}},
		"grass_block_side.png":        {{1, 0}},
		"grass_side.png":              {{1, 0}},
		"dirt.png":                    {{2, 0}},
		"stone.png":                   {{3, 0}},
		"cobblestone.png":             {{4, 0}},
		"mossy_cobblestone.png":       {{5, 0}},
		"cobblestone_mossy.png":       {{5, 0}},
		"bedrock.png":                 {{6, 0}},
		"sand.png":                    {{7, 0}},
		"sandstone_top.png":           {{8, 0}},
		"sandstone.png":               {{9, 0}},
		"sandstone_normal.png":        {{9, 0}},
		"sandstone_bottom.png":        {{10, 0}},
		"oak_log.png":                 {{11, 0}},
		"log_oak.png":                 {{11, 0}},
		"oak_log_top.png":             {{12, 0}},
		"log_oak_top.png":             {{12, 0}},
		"oak_planks.png":              {{13, 0}},
		"planks_oak.png":              {{13, 0}},
		"oak_leaves.png":              {{14, 0}},
		"oak_leaves1.png":             {{14, 0}},
		"leaves_oak.png":              {{14, 0}},
		"glass.png":                   {{15, 0}},
		"coal_ore.png":                {{0, 1}},
		"iron_ore.png":                {{1, 1}},
		"gold_ore.png":                {{2, 1}},
		"diamond_ore.png":             {{3, 1}},
		"redstone_ore.png":            {{4, 1}},
		"emerald_ore.png":             {{5, 1}},
		"lapis_ore.png":               {{6, 1}},
		"bricks.png":                  {{7, 1}},
		"brick.png":                   {{7, 1}},
		"tnt_side.png":                {{8, 1}},
		"tnt_top.png":                 {{9, 1}},
		"tnt_bottom.png":              {{10, 1}},
		"crafting_table_top.png":      {{11, 1}},
		"crafting_table_side.png":     {{12, 1}},
		"crafting_table_front.png":    {{12, 1}},
		"furnace_front.png":           {{13, 1}},
		"furnace_front_off.png":       {{13, 1}},
		"furnace_side.png":            {{14, 1}},
		"furnace_top.png":             {{14, 1}},
		"bookshelf.png":               {{15, 1}},
		"chiseled_bookshelf_side.png": {{15, 1}},
		"torch.png":                   {{0, 2}},
		"torch_on.png":                {{0, 2}},
		"water_still.png":             {{1, 2}},
		"water_flow.png":              {{1, 2}},
		"white_wool.png":              {{5, 4}},
		"wool.png":                    {{5, 4}},
		"obsidian.png":                {{6, 4}},
	}

	itemMap := map[string][][2]int{
		"stick.png":            {{0, 3}},
		"diamond.png":          {{1, 3}},
		"iron_ingot.png":       {{2, 3}},
		"gold_ingot.png":       {{3, 3}},
		"coal.png":             {{4, 3}},
		"wooden_pickaxe.png":   {{5, 3}},
		"wood_pickaxe.png":     {{5, 3}},
		"wooden_axe.png":       {{6, 3}},
		"wood_axe.png":         {{6, 3}},
		"wooden_shovel.png":    {{7, 3}},
		"wood_shovel.png":      {{7, 3}},
		"wooden_sword.png":     {{8, 3}},
		"wood_sword.png":       {{8, 3}},
		"stone_pickaxe.png":    {{9, 3}},
		"stone_axe.png":        {{10, 3}},
		"stone_shovel.png":     {{11, 3}},
		"stone_sword.png":      {{12, 3}},
		"iron_pickaxe.png":     {{13, 3}},
		"iron_axe.png":         {{14, 3}},
		"diamond_pickaxe.png":  {{15, 3}},
		"iron_shovel.png":      {{0, 4}},
		"iron_sword.png":       {{1, 4}},
		"diamond_axe.png":      {{2, 4}},
		"diamond_shovel.png":   {{3, 4}},
		"diamond_sword.png":    {{4, 4}},
		"beef.png":             {{7, 4}},
		"raw_beef.png":         {{7, 4}},
		"cooked_beef.png":      {{8, 4}},
		"steak.png":            {{8, 4}},
		"porkchop.png":         {{9, 4}},
		"raw_porkchop.png":     {{9, 4}},
		"cooked_porkchop.png":  {{10, 4}},
		"apple.png":            {{11, 4}},
		"bread.png":            {{12, 4}},
		"rotten_flesh.png":     {{13, 4}},
		"gunpowder.png":        {{14, 4}},
		"bone.png":             {{15, 4}},
		"arrow.png":            {{10, 5}},
	}

	destroyStages := [10]string{
		"destroy_stage_0.png", "destroy_stage_1.png", "destroy_stage_2.png", "destroy_stage_3.png", "destroy_stage_4.png",
		"destroy_stage_5.png", "destroy_stage_6.png", "destroy_stage_7.png", "destroy_stage_8.png", "destroy_stage_9.png",
	}

	count := 0

	loadTile := func(filePath string) *rl.Image {
		if filePath == "" {
			return nil
		}
		if strings.HasSuffix(filePath, "_n.png") || strings.HasSuffix(filePath, "_s.png") ||
			strings.Contains(filePath, "cmodels") {
			return nil
		}
		if _, err := os.Stat(filePath); err != nil {
			return nil
		}
		tile := rl.LoadImage(filePath)
		if tile.Width <= 0 || tile.Height <= 0 {
			fmt.Printf("WARNING: Failed to load texture (invalid dimensions): %s\n", filePath)
			return nil
		}
		rl.ImageFormat(tile, rl.UncompressedR8g8b8a8)
		if tile.Height > tile.Width {
			rl.ImageCrop(tile, rl.NewRectangle(0, 0, float32(tile.Width), float32(tile.Width)))
		}
		if tile.Width != tileSize || tile.Height != tileSize {
			rl.ImageResizeNN(tile, tileSize, tileSize)
		}
		return tile
	}

	// 1. Stitch block textures
	if blockDir != "" {
		// Special handling: Grass Side + Overlay composite
		dirtPath := findFileRecursive(blockDir, "dirt.png")
		sideOverlayPath := findFileRecursive(blockDir, "grass_block_side_overlay.png", "grass_side_overlay.png")
		sidePath := findFileRecursive(blockDir, "grass_block_side.png", "grass_side.png")

		dirtImg := loadTile(dirtPath)
		if dirtImg != nil {
			dstX := int32(1 * tileSize)
			dstY := int32(0 * tileSize)
			rl.ImageDraw(img, dirtImg, rl.NewRectangle(0, 0, float32(tileSize), float32(tileSize)), rl.NewRectangle(float32(dstX), float32(dstY), float32(tileSize), float32(tileSize)), rl.White)

			overlayImg := loadTile(sideOverlayPath)
			if overlayImg != nil {
				rl.ImageColorTint(overlayImg, rl.NewColor(92, 168, 56, 255))
				rl.ImageDraw(img, overlayImg, rl.NewRectangle(0, 0, float32(tileSize), float32(tileSize)), rl.NewRectangle(float32(dstX), float32(dstY), float32(tileSize), float32(tileSize)), rl.White)
				rl.UnloadImage(overlayImg)
			} else {
				sideImg := loadTile(sidePath)
				if sideImg != nil {
					rl.ImageDraw(img, sideImg, rl.NewRectangle(0, 0, float32(tileSize), float32(tileSize)), rl.NewRectangle(float32(dstX), float32(dstY), float32(tileSize), float32(tileSize)), rl.White)
					rl.UnloadImage(sideImg)
				}
			}
			rl.UnloadImage(dirtImg)
			count++
		}

		for fileName, gridPositions := range blockMap {
			path := findFileRecursive(blockDir, fileName)
			if path == "" {
				continue
			}
			tile := loadTile(path)
			if tile != nil {
				// Biome Tinting for Grass Top and Leaves
				if strings.Contains(fileName, "grass") && strings.Contains(fileName, "top") {
					rl.ImageColorTint(tile, rl.NewColor(92, 168, 56, 255)) // Lush Grass Green
				} else if strings.Contains(fileName, "leaves") {
					rl.ImageColorTint(tile, rl.NewColor(58, 140, 32, 255)) // Forest Leaf Green
				}

				for _, gridPos := range gridPositions {
					dstX := int32(gridPos[0] * int(tileSize))
					dstY := int32(gridPos[1] * int(tileSize))
					rl.ImageDrawRectangle(img, dstX, dstY, tileSize, tileSize, rl.Blank)
					rl.ImageDraw(img, tile, rl.NewRectangle(0, 0, float32(tileSize), float32(tileSize)), rl.NewRectangle(float32(dstX), float32(dstY), float32(tileSize), float32(tileSize)), rl.White)
				}
				rl.UnloadImage(tile)
				count++
			}
		}

		// Stitch 10 destroy stage crack overlay textures into Row 5 (0..9)
		for sIdx, sFile := range destroyStages {
			path := findFileRecursive(blockDir, sFile)
			tile := loadTile(path)
			if tile != nil {
				dstX := int32(sIdx * int(tileSize))
				dstY := int32(5 * int(tileSize))
				rl.ImageDrawRectangle(img, dstX, dstY, tileSize, tileSize, rl.Blank)
				rl.ImageDraw(img, tile, rl.NewRectangle(0, 0, float32(tileSize), float32(tileSize)), rl.NewRectangle(float32(dstX), float32(dstY), float32(tileSize), float32(tileSize)), rl.White)
				rl.UnloadImage(tile)
				count++
			}
		}
	}

	// 2. Stitch item textures
	if itemDir != "" {
		for fileName, gridPositions := range itemMap {
			path := findFileRecursive(itemDir, fileName)
			if path == "" {
				continue
			}
			tile := loadTile(path)
			if tile != nil {
				for _, gridPos := range gridPositions {
					dstX := int32(gridPos[0] * int(tileSize))
					dstY := int32(gridPos[1] * int(tileSize))
					rl.ImageDrawRectangle(img, dstX, dstY, tileSize, tileSize, rl.Blank)
					rl.ImageDraw(img, tile, rl.NewRectangle(0, 0, float32(tileSize), float32(tileSize)), rl.NewRectangle(float32(dstX), float32(dstY), float32(tileSize), float32(tileSize)), rl.White)
				}
				rl.UnloadImage(tile)
				count++
			}
		}
	}

	return count
}

func generateProceduralBase(img *rl.Image) {
	// Row 0: Basic Terrain & Wood
	drawGrassTop(img, 0, 0)
	drawGrassSide(img, 1, 0)
	drawDirt(img, 2, 0)
	drawStone(img, 3, 0)
	drawCobblestone(img, 4, 0)
	drawMossyCobblestone(img, 5, 0)
	drawBedrock(img, 6, 0)
	drawSand(img, 7, 0)
	drawSandstoneTop(img, 8, 0)
	drawSandstoneSide(img, 9, 0)
	drawSandstoneBottom(img, 10, 0)
	drawOakLogSide(img, 11, 0)
	drawOakLogTop(img, 12, 0)
	drawOakPlanks(img, 13, 0)
	drawOakLeaves(img, 14, 0)
	drawGlass(img, 15, 0)

	// Row 1: Ores, Bricks & Crafting
	drawCoalOre(img, 0, 1)
	drawIronOre(img, 1, 1)
	drawGoldOre(img, 2, 1)
	drawDiamondOre(img, 3, 1)
	drawRedstoneOre(img, 4, 1)
	drawEmeraldOre(img, 5, 1)
	drawLapisOre(img, 6, 1)
	drawBricks(img, 7, 1)
	drawTNTSide(img, 8, 1)
	drawTNTTop(img, 9, 1)
	drawTNTBottom(img, 10, 1)
	drawCraftingTableTop(img, 11, 1)
	drawCraftingTableSide(img, 12, 1)
	drawFurnaceFront(img, 13, 1)
	drawFurnaceSide(img, 14, 1)
	drawBookshelf(img, 15, 1)

	// Row 2: Lighting, Water & Steve Model
	drawTorch(img, 0, 2)
	drawWater(img, 1, 2)
	drawWater(img, 2, 2)
	drawSteveHeadFront(img, 3, 2)
	drawSteveHeadSide(img, 4, 2)
	drawSteveHeadTop(img, 5, 2)
	drawSteveShirt(img, 6, 2)
	drawSteveSkin(img, 7, 2)
	drawStevePants(img, 8, 2)

	// Row 3 & 4: Tools & Items
	drawStick(img, 0, 3)
	drawDiamondItem(img, 1, 3)
	drawIngot(img, 2, 3, rl.NewColor(220, 220, 220, 255)) // Iron
	drawIngot(img, 3, 3, rl.NewColor(255, 215, 40, 255))  // Gold
	drawCoalItem(img, 4, 3)
	drawTool(img, 5, 3, rl.NewColor(150, 115, 65, 255), "pickaxe")
	drawTool(img, 6, 3, rl.NewColor(150, 115, 65, 255), "axe")
	drawTool(img, 7, 3, rl.NewColor(150, 115, 65, 255), "shovel")
	drawTool(img, 8, 3, rl.NewColor(150, 115, 65, 255), "sword")
	drawTool(img, 9, 3, rl.NewColor(130, 130, 130, 255), "pickaxe")
	drawTool(img, 10, 3, rl.NewColor(130, 130, 130, 255), "axe")
	drawTool(img, 11, 3, rl.NewColor(130, 130, 130, 255), "shovel")
	drawTool(img, 12, 3, rl.NewColor(130, 130, 130, 255), "sword")
	drawTool(img, 13, 3, rl.NewColor(230, 230, 230, 255), "pickaxe")
	drawTool(img, 14, 3, rl.NewColor(230, 230, 230, 255), "axe")
	drawTool(img, 15, 3, rl.NewColor(95, 240, 250, 255), "pickaxe")
}

// Unload frees atlas resources
func (ta *TextureAtlas) Unload() {
	rl.UnloadTexture(ta.Texture)
	rl.UnloadImage(ta.Image)
}

func setPixel(img *rl.Image, col, row, px, py int, c rl.Color) {
	if px < 0 || px >= 16 || py < 0 || py >= 16 {
		return
	}
	x := int32(col*16 + px)
	y := int32(row*16 + py)
	rl.ImageDrawPixel(img, x, y, c)
}

// --- PROCEDURAL DRAWING UTILITIES ---

func drawGrassTop(img *rl.Image, col, row int) {
	c1 := rl.NewColor(89, 166, 44, 255)
	c2 := rl.NewColor(76, 147, 36, 255)
	c3 := rl.NewColor(103, 184, 53, 255)
	c4 := rl.NewColor(67, 133, 30, 255)

	palette := []rl.Color{c1, c2, c3, c4}
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			pIdx := (x*7 + y*13 + (x*y)%3) % 4
			setPixel(img, col, row, x, y, palette[pIdx])
		}
	}
}

func drawGrassSide(img *rl.Image, col, row int) {
	drawDirt(img, col, row)

	gTop := rl.NewColor(89, 166, 44, 255)
	gMid := rl.NewColor(76, 147, 36, 255)
	gHi := rl.NewColor(103, 184, 53, 255)
	gDark := rl.NewColor(55, 115, 25, 255)

	fringes := [16]int{3, 4, 3, 2, 4, 5, 3, 2, 3, 4, 5, 3, 2, 4, 3, 2}

	for x := 0; x < 16; x++ {
		fLen := fringes[x]
		for y := 0; y < fLen; y++ {
			if y == 0 {
				if x%2 == 0 {
					setPixel(img, col, row, x, y, gHi)
				} else {
					setPixel(img, col, row, x, y, gTop)
				}
			} else if y == fLen-1 {
				setPixel(img, col, row, x, y, gDark)
			} else {
				setPixel(img, col, row, x, y, gMid)
			}
		}
	}
}

func drawDirt(img *rl.Image, col, row int) {
	d1 := rl.NewColor(134, 96, 67, 255)
	d2 := rl.NewColor(121, 85, 58, 255)
	d3 := rl.NewColor(146, 107, 75, 255)
	d4 := rl.NewColor(109, 76, 52, 255)
	palette := []rl.Color{d1, d2, d3, d4}

	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			idx := (x*11 + y*17 + (x^y)) % 4
			setPixel(img, col, row, x, y, palette[idx])
		}
	}
}

func drawStone(img *rl.Image, col, row int) {
	s1 := rl.NewColor(125, 125, 125, 255)
	s2 := rl.NewColor(115, 115, 115, 255)
	s3 := rl.NewColor(138, 138, 138, 255)
	s4 := rl.NewColor(102, 102, 102, 255)
	palette := []rl.Color{s1, s2, s3, s4}

	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			idx := (x*13 + y*19 + (x*y*7)) % 4
			setPixel(img, col, row, x, y, palette[idx])
		}
	}
}

func drawCobblestone(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			shade := uint8(90 + (x*7+y*13)%45)
			if (x+y)%4 == 0 {
				shade = 70
			}
			setPixel(img, col, row, x, y, rl.NewColor(shade, shade, shade, 255))
		}
	}
}

func drawMossyCobblestone(img *rl.Image, col, row int) {
	drawCobblestone(img, col, row)
	m := rl.NewColor(60, 130, 45, 255)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if (x*3+y*7)%5 == 0 {
				setPixel(img, col, row, x, y, m)
			}
		}
	}
}

func drawBedrock(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			val := uint8(30 + (x*17+y*31)%40)
			setPixel(img, col, row, x, y, rl.NewColor(val, val, val, 255))
		}
	}
}

func drawSand(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			r := uint8(215 + (x*3+y*5)%25)
			g := uint8(200 + (x*3+y*5)%25)
			b := uint8(145 + (x*3+y*5)%25)
			setPixel(img, col, row, x, y, rl.NewColor(r, g, b, 255))
		}
	}
}

func drawSandstoneTop(img *rl.Image, col, row int) {
	drawSand(img, col, row)
}

func drawSandstoneSide(img *rl.Image, col, row int) {
	drawSand(img, col, row)
	for x := 0; x < 16; x++ {
		setPixel(img, col, row, x, 8, rl.NewColor(180, 165, 115, 255))
		setPixel(img, col, row, x, 9, rl.NewColor(170, 155, 105, 255))
	}
}

func drawSandstoneBottom(img *rl.Image, col, row int) {
	drawSand(img, col, row)
}

func drawOakLogSide(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			bark := uint8(80 + (x*11+y*3)%30)
			setPixel(img, col, row, x, y, rl.NewColor(bark+25, bark+10, bark-10, 255))
		}
	}
}

func drawOakLogTop(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			dx := float32(x - 8)
			dy := float32(y - 8)
			dist := int(dx*dx + dy*dy)
			wood := uint8(150 + (dist%20)*3)
			setPixel(img, col, row, x, y, rl.NewColor(wood, wood-30, wood-60, 255))
		}
	}
}

func drawOakPlanks(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := uint8(155 + (x*5+y*9)%25)
			if y%4 == 0 {
				c = 110
			}
			setPixel(img, col, row, x, y, rl.NewColor(c, c-30, c-70, 255))
		}
	}
}

func drawOakLeaves(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if (x+y*3)%4 == 0 {
				setPixel(img, col, row, x, y, rl.Blank)
			} else {
				g := uint8(90 + (x*7+y*11)%40)
				setPixel(img, col, row, x, y, rl.NewColor(35, g, 20, 255))
			}
		}
	}
}

func drawGlass(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if x == 0 || x == 15 || y == 0 || y == 15 || (x == y && x > 3 && x < 8) {
				setPixel(img, col, row, x, y, rl.NewColor(220, 240, 255, 200))
			} else {
				setPixel(img, col, row, x, y, rl.Blank)
			}
		}
	}
}

func drawCoalOre(img *rl.Image, col, row int) {
	drawStone(img, col, row)
	for y := 3; y < 13; y++ {
		for x := 3; x < 13; x++ {
			if (x*7+y*11)%3 == 0 {
				setPixel(img, col, row, x, y, rl.NewColor(20, 20, 20, 255))
			}
		}
	}
}

func drawIronOre(img *rl.Image, col, row int) {
	drawStone(img, col, row)
	for y := 3; y < 13; y++ {
		for x := 3; x < 13; x++ {
			if (x*7+y*11)%3 == 0 {
				setPixel(img, col, row, x, y, rl.NewColor(215, 175, 140, 255))
			}
		}
	}
}

func drawGoldOre(img *rl.Image, col, row int) {
	drawStone(img, col, row)
	for y := 3; y < 13; y++ {
		for x := 3; x < 13; x++ {
			if (x*7+y*11)%3 == 0 {
				setPixel(img, col, row, x, y, rl.NewColor(255, 220, 40, 255))
			}
		}
	}
}

func drawDiamondOre(img *rl.Image, col, row int) {
	drawStone(img, col, row)
	for y := 3; y < 13; y++ {
		for x := 3; x < 13; x++ {
			if (x*7+y*11)%3 == 0 {
				setPixel(img, col, row, x, y, rl.NewColor(90, 240, 250, 255))
			}
		}
	}
}

func drawRedstoneOre(img *rl.Image, col, row int) {
	drawStone(img, col, row)
	for y := 3; y < 13; y++ {
		for x := 3; x < 13; x++ {
			if (x*7+y*11)%3 == 0 {
				setPixel(img, col, row, x, y, rl.NewColor(230, 30, 20, 255))
			}
		}
	}
}

func drawEmeraldOre(img *rl.Image, col, row int) {
	drawStone(img, col, row)
	for y := 3; y < 13; y++ {
		for x := 3; x < 13; x++ {
			if (x*7+y*11)%3 == 0 {
				setPixel(img, col, row, x, y, rl.NewColor(30, 230, 70, 255))
			}
		}
	}
}

func drawLapisOre(img *rl.Image, col, row int) {
	drawStone(img, col, row)
	for y := 3; y < 13; y++ {
		for x := 3; x < 13; x++ {
			if (x*7+y*11)%3 == 0 {
				setPixel(img, col, row, x, y, rl.NewColor(30, 70, 220, 255))
			}
		}
	}
}

func drawBricks(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if y%4 == 0 || (y < 4 && x == 8) || (y >= 4 && y < 8 && (x == 0 || x == 15)) || (y >= 8 && y < 12 && x == 8) {
				setPixel(img, col, row, x, y, rl.NewColor(190, 180, 175, 255))
			} else {
				setPixel(img, col, row, x, y, rl.NewColor(155, 75, 60, 255))
			}
		}
	}
}

func drawTNTSide(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if y >= 6 && y <= 9 {
				setPixel(img, col, row, x, y, rl.RayWhite)
			} else {
				setPixel(img, col, row, x, y, rl.NewColor(210, 45, 30, 255))
			}
		}
	}
}

func drawTNTTop(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			setPixel(img, col, row, x, y, rl.NewColor(180, 40, 30, 255))
		}
	}
}

func drawTNTBottom(img *rl.Image, col, row int) {
	drawTNTTop(img, col, row)
}

func drawCraftingTableTop(img *rl.Image, col, row int) {
	drawOakPlanks(img, col, row)
	for y := 2; y <= 13; y++ {
		for x := 2; x <= 13; x++ {
			if x == 2 || x == 13 || y == 2 || y == 13 || x == 7 || x == 8 || y == 7 || y == 8 {
				setPixel(img, col, row, x, y, rl.NewColor(105, 75, 45, 255))
			}
		}
	}
}

func drawCraftingTableSide(img *rl.Image, col, row int) {
	drawOakPlanks(img, col, row)
	for y := 4; y <= 11; y++ {
		for x := 4; x <= 11; x++ {
			if x == 4 || x == 11 || y == 4 || y == 11 {
				setPixel(img, col, row, x, y, rl.NewColor(80, 55, 30, 255))
			}
		}
	}
}

func drawFurnaceFront(img *rl.Image, col, row int) {
	drawCobblestone(img, col, row)
	for y := 4; y <= 11; y++ {
		for x := 4; x <= 11; x++ {
			setPixel(img, col, row, x, y, rl.NewColor(30, 30, 30, 255))
		}
	}
}

func drawFurnaceSide(img *rl.Image, col, row int) {
	drawCobblestone(img, col, row)
}

func drawBookshelf(img *rl.Image, col, row int) {
	drawOakPlanks(img, col, row)
	colors := []rl.Color{rl.Red, rl.Blue, rl.Green, rl.Gold, rl.Purple}
	for y := 2; y <= 13; y++ {
		for x := 2; x <= 13; x++ {
			if y != 7 && y != 8 {
				c := colors[(x+y)%len(colors)]
				setPixel(img, col, row, x, y, c)
			}
		}
	}
}

func drawTorch(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			// Transparent background
			setPixel(img, col, row, x, y, rl.Blank)

			// Wooden Shaft (Y: 5 to 15, X: 7 to 8)
			if x >= 7 && x <= 8 && y >= 5 {
				if (x+y)%2 == 0 {
					setPixel(img, col, row, x, y, rl.NewColor(140, 100, 55, 255))
				} else {
					setPixel(img, col, row, x, y, rl.NewColor(115, 80, 40, 255))
				}
			}

			// Charcoal Top Tip (Y: 4, X: 7 to 8)
			if x >= 7 && x <= 8 && y == 4 {
				setPixel(img, col, row, x, y, rl.NewColor(50, 45, 40, 255))
			}

			// Outer Fire Flame (Y: 1 to 4, X: 6 to 9)
			if x >= 6 && x <= 9 && y >= 1 && y <= 4 {
				setPixel(img, col, row, x, y, rl.NewColor(245, 120, 20, 255))
			}

			// Inner Warm Orange Flame (Y: 1 to 3, X: 7 to 8)
			if x >= 7 && x <= 8 && y >= 1 && y <= 3 {
				setPixel(img, col, row, x, y, rl.NewColor(255, 200, 40, 255))
			}

			// Bright Core Flame (Y: 2, X: 7)
			if x == 7 && y == 2 {
				setPixel(img, col, row, x, y, rl.NewColor(255, 255, 180, 255))
			}

			// Sparks / Embers
			if (x == 6 && y == 0) || (x == 9 && y == 1) {
				setPixel(img, col, row, x, y, rl.NewColor(255, 180, 30, 220))
			}
		}
	}
}

func drawWater(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			val := uint8(190 + (x*5+y*11)%45)
			setPixel(img, col, row, x, y, rl.NewColor(35, 110, val, 180))
		}
	}
}

func drawSteveHeadFront(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if y < 5 {
				setPixel(img, col, row, x, y, rl.NewColor(65, 40, 20, 255)) // Hair
			} else if y >= 7 && y <= 8 && (x == 4 || x == 11) {
				setPixel(img, col, row, x, y, rl.NewColor(40, 60, 180, 255)) // Eyes
			} else if y >= 11 && y <= 12 && x >= 6 && x <= 9 {
				setPixel(img, col, row, x, y, rl.NewColor(95, 55, 30, 255)) // Beard/Mouth
			} else {
				setPixel(img, col, row, x, y, rl.NewColor(215, 160, 120, 255)) // Skin
			}
		}
	}
}

func drawSteveHeadSide(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if y < 7 || x < 7 {
				setPixel(img, col, row, x, y, rl.NewColor(65, 40, 20, 255))
			} else {
				setPixel(img, col, row, x, y, rl.NewColor(215, 160, 120, 255))
			}
		}
	}
}

func drawSteveHeadTop(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			setPixel(img, col, row, x, y, rl.NewColor(65, 40, 20, 255))
		}
	}
}

func drawSteveShirt(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			setPixel(img, col, row, x, y, rl.NewColor(0, 168, 178, 255))
		}
	}
}

func drawSteveSkin(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			setPixel(img, col, row, x, y, rl.NewColor(215, 160, 120, 255))
		}
	}
}

func drawStevePants(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			setPixel(img, col, row, x, y, rl.NewColor(45, 55, 130, 255))
		}
	}
}

// --- PROCEDURAL ITEMS & TOOLS ---

func drawStick(img *rl.Image, col, row int) {
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if x == y {
				setPixel(img, col, row, x, y, rl.NewColor(135, 95, 50, 255))
			} else {
				setPixel(img, col, row, x, y, rl.Blank)
			}
		}
	}
}

func drawDiamondItem(img *rl.Image, col, row int) {
	dCol := rl.NewColor(90, 235, 245, 255)
	for y := 4; y <= 11; y++ {
		for x := 4; x <= 11; x++ {
			if (x+y >= 10 && x+y <= 20) && (x-y >= -5 && x-y <= 5) {
				setPixel(img, col, row, x, y, dCol)
			} else {
				setPixel(img, col, row, x, y, rl.Blank)
			}
		}
	}
}

func drawIngot(img *rl.Image, col, row int, c rl.Color) {
	for y := 5; y <= 10; y++ {
		for x := 3; x <= 12; x++ {
			setPixel(img, col, row, x, y, c)
		}
	}
}

func drawCoalItem(img *rl.Image, col, row int) {
	cCol := rl.NewColor(30, 30, 30, 255)
	for y := 4; y <= 11; y++ {
		for x := 4; x <= 11; x++ {
			if (x*7+y*13)%2 == 0 {
				setPixel(img, col, row, x, y, cCol)
			}
		}
	}
}

func drawTool(img *rl.Image, col, row int, headCol rl.Color, tType string) {
	stickCol := rl.NewColor(135, 95, 50, 255)
	stickDark := rl.NewColor(85, 60, 30, 255)
	headDark := rl.NewColor(
		uint8(float32(headCol.R)*0.65),
		uint8(float32(headCol.G)*0.65),
		uint8(float32(headCol.B)*0.65),
		255,
	)
	headLight := rl.NewColor(
		uint8(min(255, int(float32(headCol.R)*1.25))),
		uint8(min(255, int(float32(headCol.G)*1.25))),
		uint8(min(255, int(float32(headCol.B)*1.25))),
		255,
	)

	// Clear tile to transparent first
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			setPixel(img, col, row, x, y, rl.Blank)
		}
	}

	// 1. Diagonal handle from bottom-left (2, 13) to top-right (10, 5)
	stickCoords := [][2]int{
		{2, 14}, {3, 14},
		{2, 13}, {3, 13}, {4, 13},
		{3, 12}, {4, 12}, {5, 12},
		{4, 11}, {5, 11}, {6, 11},
		{5, 10}, {6, 10}, {7, 10},
		{6, 9}, {7, 9}, {8, 9},
		{7, 8}, {8, 8}, {9, 8},
		{8, 7}, {9, 7}, {10, 7},
		{9, 6}, {10, 6},
	}
	for _, sc := range stickCoords {
		if sc[0] > sc[1] {
			setPixel(img, col, row, sc[0], sc[1], stickDark)
		} else {
			setPixel(img, col, row, sc[0], sc[1], stickCol)
		}
	}

	// 2. Draw specific tool head (Authentic Minecraft Diagonal 45-degree layouts)
	if tType == "pickaxe" {
		// Top curve
		topPixels := [][2]int{
			{6, 2}, {7, 2}, {8, 2}, {9, 2}, {10, 2},
			{5, 3}, {6, 3}, {7, 3}, {8, 3}, {9, 3}, {10, 3}, {11, 3}, {12, 3},
			{6, 4}, {10, 4}, {11, 4}, {12, 4}, {13, 4},
			{10, 5}, {11, 5}, {12, 5}, {13, 5},
			{9, 6}, {12, 6}, {13, 6},
			{12, 7}, {13, 7},
			{12, 8}, {13, 8},
			{12, 9}, {13, 9},
			{12, 10}, {13, 10},
		}
		for _, p := range topPixels {
			if p[1] <= 3 || p[0] <= 7 {
				setPixel(img, col, row, p[0], p[1], headLight)
			} else if p[0] == 13 || p[1] >= 9 {
				setPixel(img, col, row, p[0], p[1], headDark)
			} else {
				setPixel(img, col, row, p[0], p[1], headCol)
			}
		}
	} else if tType == "axe" {
		// Axe blade wedge
		axePixels := [][2]int{
			{7, 2}, {8, 2}, {9, 2},
			{6, 3}, {7, 3}, {8, 3}, {9, 3}, {10, 3},
			{6, 4}, {7, 4}, {8, 4}, {9, 4}, {10, 4}, {11, 4},
			{7, 5}, {8, 5}, {9, 5}, {10, 5}, {11, 5},
			{8, 6}, {9, 6}, {10, 6}, {11, 6},
			{9, 7}, {10, 7}, {11, 7},
			{10, 8}, {11, 8},
		}
		for _, p := range axePixels {
			if p[1] <= 3 || p[0] <= 7 {
				setPixel(img, col, row, p[0], p[1], headLight)
			} else if p[0] >= 11 || p[1] >= 7 {
				setPixel(img, col, row, p[0], p[1], headDark)
			} else {
				setPixel(img, col, row, p[0], p[1], headCol)
			}
		}
	} else if tType == "shovel" {
		// Shovel spade blade
		shovelPixels := [][2]int{
			{11, 2}, {12, 2},
			{10, 3}, {11, 3}, {12, 3}, {13, 3},
			{9, 4}, {10, 4}, {11, 4}, {12, 4}, {13, 4},
			{9, 5}, {10, 5}, {11, 5}, {12, 5},
			{8, 6}, {9, 6}, {10, 6},
		}
		for _, p := range shovelPixels {
			if p[1] <= 3 {
				setPixel(img, col, row, p[0], p[1], headLight)
			} else if p[0] >= 12 || p[1] >= 5 {
				setPixel(img, col, row, p[0], p[1], headDark)
			} else {
				setPixel(img, col, row, p[0], p[1], headCol)
			}
		}
	} else if tType == "sword" {
		// Clear handle for sword and draw authentic sword
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				setPixel(img, col, row, x, y, rl.Blank)
			}
		}
		// Pommel & Grip
		setPixel(img, col, row, 2, 14, stickDark)
		setPixel(img, col, row, 2, 13, stickDark)
		setPixel(img, col, row, 3, 14, stickDark)
		setPixel(img, col, row, 3, 13, stickCol)
		setPixel(img, col, row, 4, 12, stickCol)
		setPixel(img, col, row, 5, 11, stickCol)

		// Crossguard
		guardPixels := [][2]int{{3, 11}, {4, 10}, {5, 12}, {6, 11}, {4, 11}, {5, 10}}
		for _, gp := range guardPixels {
			setPixel(img, col, row, gp[0], gp[1], headDark)
		}

		// Blade (Diagonal)
		bladePixels := [][2]int{
			{6, 10}, {7, 9}, {8, 8}, {9, 7}, {10, 6}, {11, 5}, {12, 4}, {13, 3},
			{5, 9}, {6, 8}, {7, 7}, {8, 6}, {9, 5}, {10, 4}, {11, 3}, {12, 2},
			{7, 10}, {8, 9}, {9, 8}, {10, 7}, {11, 6}, {12, 5}, {13, 4},
		}
		for _, bp := range bladePixels {
			if bp[0] > bp[1] {
				setPixel(img, col, row, bp[0], bp[1], headLight)
			} else if bp[0] < bp[1] {
				setPixel(img, col, row, bp[0], bp[1], headDark)
			} else {
				setPixel(img, col, row, bp[0], bp[1], headCol)
			}
		}
	}
}

func drawDestroyStage(img *rl.Image, stage, row int) {
	density := (stage + 1) * 7
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			hash := (x*17 + y*31 + (x*y*7)) % 100
			if hash < density {
				setPixel(img, stage, row, x, y, rl.NewColor(0, 0, 0, 200))
			} else {
				setPixel(img, stage, row, x, y, rl.Blank)
			}
		}
	}
}
