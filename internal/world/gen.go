package world

import (
	"math"
	"math/rand"

	"dungeongame/internal/util"
)

// Size is the overworld edge length in tiles.
const Size = 256

// Point is a tile coordinate.
type Point struct{ X, Y int }

// Entrance is a dungeon doorway on the overworld. Exit is the walkable tile
// just outside it, where the player reappears when leaving the dungeon.
type Entrance struct {
	Pos  Point
	Exit Point
}

// World is a generated overworld.
type World struct {
	W, H      int
	Tiles     []Tile
	Labels    map[int]rune // tile index -> letter inset in a wall/brick tile
	Entrances []Entrance
	Spawn     Point
	Seed      int64
}

// At returns the tile at (x, y); out-of-bounds reads as deep water.
func (w *World) At(x, y int) Tile {
	if x < 0 || y < 0 || x >= w.W || y >= w.H {
		return TDeepWater
	}
	return w.Tiles[y*w.W+x]
}

func (w *World) set(x, y int, t Tile) {
	if x >= 0 && y >= 0 && x < w.W && y < w.H {
		w.Tiles[y*w.W+x] = t
	}
}

// EntranceAt returns the index of the entrance at (x, y), or -1.
func (w *World) EntranceAt(x, y int) int {
	for i, e := range w.Entrances {
		if e.Pos.X == x && e.Pos.Y == y {
			return i
		}
	}
	return -1
}

// Generate builds a world from a seed: noise terrain, lava fields, a labeled
// town, and 3-5 dungeon entrances tucked into forests or mountains, all
// verified reachable from the spawn point.
func Generate(seed int64) *World {
	rng := rand.New(rand.NewSource(seed))
	w := &World{W: Size, H: Size, Tiles: make([]Tile, Size*Size), Labels: map[int]rune{}, Seed: seed}

	w.terrain(seed)
	w.buildTown(rng)
	reach := w.reachableFrom(w.Spawn)
	w.placeEntrances(rng, reach)
	w.placeWarnings(rng)
	return w
}

func (w *World) terrain(seed int64) {
	for y := 0; y < w.H; y++ {
		for x := 0; x < w.W; x++ {
			fx, fy := float64(x), float64(y)
			elev := util.FBM(fx/44, fy/44, seed, 4, 2.0, 0.5)
			moist := util.FBM(fx/37, fy/37, seed+7777, 3, 2.0, 0.5)
			heat := util.FBM(fx/23, fy/23, seed+5555, 3, 2.0, 0.5)

			// Radial falloff sinks the land into ocean before the map edge,
			// so the continent has real coastline instead of clipped borders.
			// The fBm on top keeps the coast ragged rather than circular.
			nx := 2*fx/float64(w.W-1) - 1
			ny := 2*fy/float64(w.H-1) - 1
			d := math.Sqrt(nx*nx + ny*ny)
			fall := util.Clamp((d-0.50)/0.35, 0, 1)
			elev -= 0.6 * fall * fall

			var t Tile
			switch {
			case elev < 0.34:
				t = TDeepWater
			case elev < 0.42:
				t = TWater
			case elev < 0.45:
				t = TSand
			case elev > 0.70:
				t = TMountain
				if elev > 0.72 && heat > 0.62 {
					t = TLava // molten pools deep in the ranges
				}
			case moist > 0.58:
				t = TForest
			default:
				t = TGrass
			}
			w.Tiles[y*w.W+x] = t
		}
	}
}

// buildTown clears the grassiest window near the map center, raises a few
// labeled buildings, and sets the player spawn on the town green.
func (w *World) buildTown(rng *rand.Rand) {
	const tw, th = 34, 16
	bestX, bestY, bestScore := w.W/2, w.H/2, -1
	for try := 0; try < 400; try++ {
		x := w.W/4 + rng.Intn(w.W/2)
		y := w.H/4 + rng.Intn(w.H/2)
		score := 0
		for j := 0; j < th; j++ {
			for i := 0; i < tw; i++ {
				if w.At(x+i, y+j) == TGrass {
					score++
				}
			}
		}
		if score > bestScore {
			bestX, bestY, bestScore = x, y, score
		}
	}
	for j := -1; j <= th; j++ {
		for i := -1; i <= tw; i++ {
			w.set(bestX+i, bestY+j, TGrass)
		}
	}

	w.building(bestX, bestY, 9, 5, "PUB", TBrick)
	w.building(bestX+13, bestY, 10, 6, "KEEP", TWall)
	w.building(bestX+26, bestY, 8, 5, "INN", TBrick)

	w.Spawn = Point{bestX + tw/2, bestY + th - 2}
}

// building raises a hollow rectangle of wall tiles with a door gap on the
// south face and the label inset in the north (front-facing) wall.
func (w *World) building(x, y, bw, bh int, label string, mat Tile) {
	for j := 0; j < bh; j++ {
		for i := 0; i < bw; i++ {
			edge := i == 0 || j == 0 || i == bw-1 || j == bh-1
			if edge {
				w.set(x+i, y+j, mat)
			} else {
				w.set(x+i, y+j, TGrass)
			}
		}
	}
	w.set(x+bw/2, y+bh-1, TGrass) // doorway
	lx := x + (bw-len(label))/2
	for i, ch := range label {
		w.set(lx+i, y, mat)
		w.Labels[y*w.W+lx+i] = ch
	}
}

// reachableFrom floods the passable tiles from p and returns the visited set.
func (w *World) reachableFrom(p Point) []bool {
	seen := make([]bool, w.W*w.H)
	if !w.At(p.X, p.Y).Passable() {
		return seen
	}
	queue := []Point{p}
	seen[p.Y*w.W+p.X] = true
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, d := range [4]Point{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
			nx, ny := c.X+d.X, c.Y+d.Y
			if nx < 0 || ny < 0 || nx >= w.W || ny >= w.H {
				continue
			}
			i := ny*w.W + nx
			if !seen[i] && w.Tiles[i].Passable() {
				seen[i] = true
				queue = append(queue, Point{nx, ny})
			}
		}
	}
	return seen
}

// placeEntrances picks 3-5 forest/mountain tiles adjacent to terrain the
// player can reach, spaced apart, and carves dungeon doorways into them.
func (w *World) placeEntrances(rng *rand.Rand, reach []bool) {
	want := 3 + rng.Intn(3)
	const minDist = 36
	for tries := 0; tries < 20000 && len(w.Entrances) < want; tries++ {
		x, y := rng.Intn(w.W), rng.Intn(w.H)
		t := w.At(x, y)
		if t != TForest && t != TMountain {
			continue
		}
		dx, dy := x-w.Spawn.X, y-w.Spawn.Y
		if dx*dx+dy*dy < 30*30 {
			continue // not right on top of town
		}
		tooClose := false
		for _, e := range w.Entrances {
			ddx, ddy := x-e.Pos.X, y-e.Pos.Y
			if ddx*ddx+ddy*ddy < minDist*minDist {
				tooClose = true
				break
			}
		}
		if tooClose {
			continue
		}
		// Needs a reachable, non-lava tile beside it to serve as the exit pad.
		var exit *Point
		for _, d := range [4]Point{{0, 1}, {0, -1}, {-1, 0}, {1, 0}} {
			nx, ny := x+d.X, y+d.Y
			if nx < 0 || ny < 0 || nx >= w.W || ny >= w.H {
				continue
			}
			if reach[ny*w.W+nx] && w.At(nx, ny) != TLava {
				exit = &Point{nx, ny}
				break
			}
		}
		if exit == nil {
			continue
		}
		w.set(x, y, TEntrance)
		w.Entrances = append(w.Entrances, Entrance{Pos: Point{x, y}, Exit: *exit})
	}
	// Extremely defensive fallback: carve entrances near spawn if the noise
	// produced too little forest/mountain (not expected at these thresholds).
	for len(w.Entrances) < 3 {
		x := w.Spawn.X + len(w.Entrances)*8 - 8
		y := w.Spawn.Y + 6
		w.set(x, y, TEntrance)
		w.set(x, y-1, TMountain)
		w.Entrances = append(w.Entrances, Entrance{Pos: Point{x, y}, Exit: Point{x, y + 1}})
	}
}

// placeWarnings drops small wall-text signs near dungeon entrances and lava,
// reverting any sign that would wall off an entrance.
func (w *World) placeWarnings(rng *rand.Rand) {
	for _, e := range w.Entrances {
		w.placeSign(rng, "DANGER", e.Exit.X, e.Exit.Y, 5)
	}
	// A few lava warnings: scan for lava bordering passable ground.
	placed := 0
	for y := 0; y < w.H && placed < 3; y += 2 {
		for x := 0; x < w.W && placed < 3; x += 2 {
			if w.At(x, y) == TLava && w.At(x, y+1) == TGrass {
				if w.placeSign(rng, "BEWARE", x, y+2, 4) {
					placed++
					x += 60 // spread the signs out
				}
			}
		}
	}
}

// placeSign tries to convert a horizontal grass run near (cx, cy) into wall
// tiles carrying the text. It is rolled back if it disconnects any entrance.
func (w *World) placeSign(rng *rand.Rand, text string, cx, cy, radius int) bool {
	n := len(text)
	for try := 0; try < 30; try++ {
		x := cx - radius + rng.Intn(radius*2+1)
		y := cy - radius + rng.Intn(radius*2+1)
		ok := true
		for i := 0; i < n; i++ {
			if w.At(x+i, y) != TGrass {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		for i := 0; i < n; i++ {
			w.set(x+i, y, TWall)
			w.Labels[y*w.W+x+i] = rune(text[i])
		}
		if w.entrancesReachable() {
			return true
		}
		for i := 0; i < n; i++ { // roll back
			w.set(x+i, y, TGrass)
			delete(w.Labels, y*w.W+x+i)
		}
	}
	return false
}

func (w *World) entrancesReachable() bool {
	reach := w.reachableFrom(w.Spawn)
	for _, e := range w.Entrances {
		if !reach[e.Exit.Y*w.W+e.Exit.X] {
			return false
		}
	}
	return true
}
