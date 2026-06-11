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

// Healer is a building whose interior restores hit points on entry.
type Healer struct {
	Name       string
	X, Y, W, H int   // building rectangle including walls
	Door       Point // doorway tile in the south wall
	Heal       int   // hit points restored per visit
}

// World is a generated overworld.
type World struct {
	W, H      int
	Tiles     []Tile
	Labels    map[int]rune // tile index -> letter inset in a wall/brick tile
	Entrances []Entrance
	Healers   []Healer
	Spawn     Point
	Seed      int64
}

// HealerAt returns the index of the healer whose interior contains (x, y),
// or -1.
func (w *World) HealerAt(x, y int) int {
	for i, h := range w.Healers {
		if x > h.X && x < h.X+h.W-1 && y > h.Y && y < h.Y+h.H-1 {
			return i
		}
	}
	return -1
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

// Generate builds a world from a seed: noise terrain, lava fields, scattered
// healer buildings, and 3-5 dungeon entrances tucked into forests or
// mountains. Every entrance and healer door is verified walkable from spawn
// without crossing lava; the rare invalid world is regenerated from a
// derived seed so the result is always playable.
func Generate(seed int64) *World {
	s := seed
	for {
		w := generate(s)
		if len(w.Entrances) >= 3 && w.entrancesReachable() {
			w.Seed = seed
			return w
		}
		s = s*31 + 7
	}
}

func generate(seed int64) *World {
	rng := rand.New(rand.NewSource(seed))
	w := &World{W: Size, H: Size, Tiles: make([]Tile, Size*Size), Labels: map[int]rune{}, Seed: seed}

	w.terrain(seed)
	w.placeBuildings(rng)
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

// placeBuildings scatters the three healer buildings across the continent
// rather than clustering them in one town: the stone DOCTOR (where the spawn
// is), then the PUB and INN well away from it, each on its own grass
// clearing and reachable on foot from the spawn.
func (w *World) placeBuildings(rng *rand.Rand) {
	specs := []struct {
		label string
		mat   Tile
		bw    int
		bh    int
		heal  int
	}{
		{"DOCTOR", TWall, 12, 7, 40},
		{"PUB", TBrick, 9, 5, 10},
		{"INN", TBrick, 8, 5, 10},
	}

	var reach []bool // filled in after the doctor fixes the spawn
	for bi, sp := range specs {
		padW, padH := sp.bw+6, sp.bh+6
		bestX, bestY, bestScore := -1, -1, -1
		// Prefer well-separated sites; relax the spacing only if the seed
		// leaves no choice.
		for _, minDist := range []int{70, 45, 25, 0} {
			for try := 0; try < 800; try++ {
				x := 2 + rng.Intn(w.W-padW-4)
				y := 2 + rng.Intn(w.H-padH-4)
				cx, cy := x+padW/2, y+padH/2
				spread := true
				for _, h := range w.Healers {
					hx, hy := h.X+h.W/2, h.Y+h.H/2
					if (cx-hx)*(cx-hx)+(cy-hy)*(cy-hy) < minDist*minDist {
						spread = false
						break
					}
				}
				if !spread {
					continue
				}
				// Later buildings must open onto ground the player can walk
				// to from the spawn.
				if bi > 0 {
					dx, dy := x+3+sp.bw/2, y+3+sp.bh-1
					if !reach[dy*w.W+dx] {
						continue
					}
				}
				score := 0
				for j := 0; j < padH; j++ {
					for i := 0; i < padW; i++ {
						if w.At(x+i, y+j) == TGrass {
							score++
						}
					}
				}
				if score > bestScore {
					bestX, bestY, bestScore = x, y, score
				}
			}
			if bestScore >= padW*padH/2 {
				break
			}
		}

		// Clear the pad to grass and raise the building inside it.
		for j := 0; j < padH; j++ {
			for i := 0; i < padW; i++ {
				w.set(bestX+i, bestY+j, TGrass)
			}
		}
		hx, hy := bestX+3, bestY+3
		w.building(hx, hy, sp.bw, sp.bh, sp.label, sp.mat)
		w.Healers = append(w.Healers, Healer{
			Name: sp.label,
			X:    hx, Y: hy, W: sp.bw, H: sp.bh,
			Door: Point{hx + sp.bw/2, hy + sp.bh - 1},
			Heal: sp.heal,
		})

		if bi == 0 {
			// Spawn on the doctor's doorstep; reachability for the rest of
			// the world is measured from here.
			w.Spawn = Point{hx + sp.bw/2, hy + sp.bh}
			reach = w.reachableFrom(w.Spawn)
		}
	}
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

// reachableFrom floods the tiles a player can sensibly walk to from p.
// Lava is excluded even though it is technically passable: anything that can
// only be reached across a lava moat counts as unreachable, so dungeon
// entrances and healer doors are never gated behind burning ground.
func (w *World) reachableFrom(p Point) []bool {
	pass := func(t Tile) bool { return t.Passable() && t != TLava }
	seen := make([]bool, w.W*w.H)
	if !pass(w.At(p.X, p.Y)) {
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
			if !seen[i] && pass(w.Tiles[i]) {
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
		exit := w.exitPad(reach, x, y)
		if exit == nil {
			continue
		}
		w.set(x, y, TEntrance)
		w.Entrances = append(w.Entrances, Entrance{Pos: Point{x, y}, Exit: *exit})
	}
	// Relaxed second pass for stingy seeds: accept any forest/mountain tile
	// with a safe approach, ignoring the spacing rules. If even this cannot
	// find three, Generate regenerates the whole world.
	for tries := 0; tries < 60000 && len(w.Entrances) < 3; tries++ {
		x, y := rng.Intn(w.W), rng.Intn(w.H)
		t := w.At(x, y)
		if t != TForest && t != TMountain {
			continue
		}
		exit := w.exitPad(reach, x, y)
		if exit == nil {
			continue
		}
		w.set(x, y, TEntrance)
		w.Entrances = append(w.Entrances, Entrance{Pos: Point{x, y}, Exit: *exit})
	}
}

// exitPad returns a tile beside (x, y) that the player can walk to from
// spawn without crossing lava, or nil if the spot is sealed off.
func (w *World) exitPad(reach []bool, x, y int) *Point {
	for _, d := range [4]Point{{0, 1}, {0, -1}, {-1, 0}, {1, 0}} {
		nx, ny := x+d.X, y+d.Y
		if nx < 0 || ny < 0 || nx >= w.W || ny >= w.H {
			continue
		}
		if reach[ny*w.W+nx] {
			return &Point{nx, ny}
		}
	}
	return nil
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

// entrancesReachable checks every critical doorway (dungeon exit pads and
// healer doors) is still walkable from spawn; signs roll back if not.
func (w *World) entrancesReachable() bool {
	reach := w.reachableFrom(w.Spawn)
	for _, e := range w.Entrances {
		if !reach[e.Exit.Y*w.W+e.Exit.X] {
			return false
		}
	}
	for _, h := range w.Healers {
		if !reach[h.Door.Y*w.W+h.Door.X] {
			return false
		}
	}
	return true
}
