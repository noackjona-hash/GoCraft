package voxel

import (
	"math"
)

// Standard 256 permutation table doubled to 512 to avoid index wrapping
var perm = [512]int{
	151, 160, 137, 91, 90, 15, 131, 13, 201, 95, 96, 53, 194, 233, 7, 225, 140, 36, 103, 30, 69, 142,
	8, 99, 37, 240, 21, 10, 23, 190, 6, 148, 247, 120, 234, 75, 0, 26, 197, 62, 94, 252, 219, 203, 117,
	35, 11, 32, 57, 177, 33, 88, 237, 149, 56, 87, 174, 20, 125, 136, 171, 168, 68, 175, 74, 165, 71,
	134, 139, 48, 27, 166, 77, 146, 158, 231, 83, 111, 229, 122, 60, 211, 133, 230, 220, 105, 92, 41,
	55, 46, 245, 40, 244, 102, 143, 54, 65, 25, 63, 161, 1, 216, 80, 73, 209, 76, 132, 187, 208, 89,
	18, 169, 200, 196, 135, 130, 116, 188, 159, 86, 164, 100, 109, 198, 173, 186, 3, 64, 52, 217, 226,
	250, 124, 123, 5, 202, 38, 147, 118, 126, 255, 82, 85, 212, 207, 206, 59, 227, 47, 16, 58, 17, 182,
	189, 28, 42, 223, 183, 170, 213, 119, 248, 152, 2, 44, 154, 163, 70, 221, 153, 101, 155, 167, 43,
	172, 9, 129, 22, 39, 253, 19, 98, 108, 110, 79, 113, 224, 232, 178, 185, 112, 104, 218, 246, 97,
	228, 251, 34, 242, 193, 238, 210, 144, 12, 191, 179, 162, 241, 81, 51, 145, 235, 249, 14, 239,
	107, 49, 192, 214, 31, 181, 199, 106, 157, 184, 84, 204, 176, 115, 121, 50, 45, 127, 4, 150, 254,
	138, 236, 205, 93, 222, 114, 67, 29, 24, 72, 243, 141, 128, 195, 78, 66, 215, 61, 156, 180,
	// Mirror copy
	151, 160, 137, 91, 90, 15, 131, 13, 201, 95, 96, 53, 194, 233, 7, 225, 140, 36, 103, 30, 69, 142,
	8, 99, 37, 240, 21, 10, 23, 190, 6, 148, 247, 120, 234, 75, 0, 26, 197, 62, 94, 252, 219, 203, 117,
	35, 11, 32, 57, 177, 33, 88, 237, 149, 56, 87, 174, 20, 125, 136, 171, 168, 68, 175, 74, 165, 71,
	134, 139, 48, 27, 166, 77, 146, 158, 231, 83, 111, 229, 122, 60, 211, 133, 230, 220, 105, 92, 41,
	55, 46, 245, 40, 244, 102, 143, 54, 65, 25, 63, 161, 1, 216, 80, 73, 209, 76, 132, 187, 208, 89,
	18, 169, 200, 196, 135, 130, 116, 188, 159, 86, 164, 100, 109, 198, 173, 186, 3, 64, 52, 217, 226,
	250, 124, 123, 5, 202, 38, 147, 118, 126, 255, 82, 85, 212, 207, 206, 59, 227, 47, 16, 58, 17, 182,
	189, 28, 42, 223, 183, 170, 213, 119, 248, 152, 2, 44, 154, 163, 70, 221, 153, 101, 155, 167, 43,
	172, 9, 129, 22, 39, 253, 19, 98, 108, 110, 79, 113, 224, 232, 178, 185, 112, 104, 218, 246, 97,
	228, 251, 34, 242, 193, 238, 210, 144, 12, 191, 179, 162, 241, 81, 51, 145, 235, 249, 14, 239,
	107, 49, 192, 214, 31, 181, 199, 106, 157, 184, 84, 204, 176, 115, 121, 50, 45, 127, 4, 150, 254,
	138, 236, 205, 93, 222, 114, 67, 29, 24, 72, 243, 141, 128, 195, 78, 66, 215, 61, 156, 180,
}

func fade(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

func lerp(t, a, b float64) float64 {
	return a + t*(b-a)
}

func grad2D(hash int, x, y float64) float64 {
	h := hash & 7
	u := x
	v := y
	if h >= 4 {
		u = y
		v = x
	}
	if (h & 1) != 0 {
		u = -u
	}
	if (h & 2) != 0 {
		v = -v
	}
	return u + v
}

func grad3D(hash int, x, y, z float64) float64 {
	h := hash & 15
	u := x
	if h < 8 {
		u = x
	} else {
		u = y
	}
	v := y
	if h < 4 {
		v = y
	} else if h == 12 || h == 14 {
		v = x
	} else {
		v = z
	}
	res := float64(0)
	if (h & 1) == 0 {
		res += u
	} else {
		res -= u
	}
	if (h & 2) == 0 {
		res += v
	} else {
		res -= v
	}
	return res
}

// Perlin2D generates continuous 2D Perlin noise in range [-1.0, 1.0]
func Perlin2D(x, y float64) float64 {
	X := int(math.Floor(x)) & 255
	Y := int(math.Floor(y)) & 255

	xf := x - math.Floor(x)
	yf := y - math.Floor(y)

	u := fade(xf)
	v := fade(yf)

	A := perm[X] + Y
	B := perm[X+1] + Y

	g00 := grad2D(perm[A], xf, yf)
	g10 := grad2D(perm[B], xf-1, yf)
	g01 := grad2D(perm[A+1], xf, yf-1)
	g11 := grad2D(perm[B+1], xf-1, yf-1)

	x1 := lerp(u, g00, g10)
	x2 := lerp(u, g01, g11)

	return lerp(v, x1, x2)
}

// Perlin3D generates continuous 3D Perlin noise in range [-1.0, 1.0] for subterranean caves
func Perlin3D(x, y, z float64) float64 {
	X := int(math.Floor(x)) & 255
	Y := int(math.Floor(y)) & 255
	Z := int(math.Floor(z)) & 255

	xf := x - math.Floor(x)
	yf := y - math.Floor(y)
	zf := z - math.Floor(z)

	u := fade(xf)
	v := fade(yf)
	w := fade(zf)

	A := perm[X] + Y
	AA := perm[A] + Z
	AB := perm[A+1] + Z
	B := perm[X+1] + Y
	BA := perm[B] + Z
	BB := perm[B+1] + Z

	g000 := grad3D(perm[AA], xf, yf, zf)
	g100 := grad3D(perm[BA], xf-1, yf, zf)
	g010 := grad3D(perm[AB], xf, yf-1, zf)
	g110 := grad3D(perm[BB], xf-1, yf-1, zf)
	g001 := grad3D(perm[AA+1], xf, yf, zf-1)
	g101 := grad3D(perm[BA+1], xf-1, yf, zf-1)
	g011 := grad3D(perm[AB+1], xf, yf-1, zf-1)
	g111 := grad3D(perm[BB+1], xf-1, yf-1, zf-1)

	x1 := lerp(u, g000, g100)
	x2 := lerp(u, g010, g110)
	y1 := lerp(v, x1, x2)

	x3 := lerp(u, g001, g101)
	x4 := lerp(u, g011, g111)
	y2 := lerp(v, x3, x4)

	return lerp(w, y1, y2)
}

// FractalNoise2D combines multiple octaves for realistic terrain geography
func FractalNoise2D(x, y float64, octaves int, persistence, lacunarity float64) float64 {
	total := float64(0)
	frequency := float64(1)
	amplitude := float64(1)
	maxValue := float64(0)

	for i := 0; i < octaves; i++ {
		total += Perlin2D(x*frequency, y*frequency) * amplitude
		maxValue += amplitude
		amplitude *= persistence
		frequency *= lacunarity
	}

	return total / maxValue
}
