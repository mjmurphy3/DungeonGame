// Package util provides small shared helpers: seeded value noise and math.
package util

import "math"

// hash2 produces a deterministic pseudo-random value in [0,1) from integer
// coordinates and a seed, using bit mixing (no allocation, no table).
func hash2(x, y int, seed int64) float64 {
	h := uint64(x)*0x9E3779B97F4A7C15 ^ uint64(y)*0xC2B2AE3D27D4EB4F ^ uint64(seed)
	h ^= h >> 33
	h *= 0xFF51AFD7ED558CCD
	h ^= h >> 33
	h *= 0xC4CEB9FE1A85EC53
	h ^= h >> 33
	return float64(h>>11) / float64(1<<53)
}

// smoothstep interpolation factor.
func fade(t float64) float64 { return t * t * (3 - 2*t) }

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// ValueNoise returns smooth noise in [0,1) at floating-point coordinates.
func ValueNoise(x, y float64, seed int64) float64 {
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	tx, ty := fade(x-float64(x0)), fade(y-float64(y0))
	a := hash2(x0, y0, seed)
	b := hash2(x0+1, y0, seed)
	c := hash2(x0, y0+1, seed)
	d := hash2(x0+1, y0+1, seed)
	return lerp(lerp(a, b, tx), lerp(c, d, tx), ty)
}

// FBM sums octaves of value noise, returning a value in [0,1).
func FBM(x, y float64, seed int64, octaves int, lacunarity, gain float64) float64 {
	sum, amp, norm, freq := 0.0, 1.0, 0.0, 1.0
	for i := range octaves {
		sum += amp * ValueNoise(x*freq, y*freq, seed+int64(i)*1013)
		norm += amp
		amp *= gain
		freq *= lacunarity
	}
	return sum / norm
}

// Clamp limits v to [lo, hi].
func Clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ClampI limits v to [lo, hi].
func ClampI(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
