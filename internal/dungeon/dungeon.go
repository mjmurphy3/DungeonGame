// Package dungeon generates raycaster dungeon levels: rooms joined by
// corridors with doors, ladders, columns, torches, orcs and treasure chests.
package dungeon

import (
	"math"
	"math/rand"
)

// Cell is one dungeon grid square.
type Cell uint8

const (
	CWall Cell = iota
	CFloor
	CDoor     // closed; opens when the player walks into it
	CDoorOpen // passable doorway
	CEntry    // ladder back up to the overworld (where you climbed down)
	CExit     // ladder up at the far end; also returns to the overworld
)

// Passable reports whether the player or an orc can occupy the cell.
func (c Cell) Passable() bool {
	switch c {
	case CFloor, CDoorOpen, CEntry, CExit:
		return true
	}
	return false
}

// Room is the interior floor rectangle of a generated room.
type Room struct{ X, Y, W, H int }

func (r Room) centerX() float64 { return float64(r.X) + float64(r.W)/2 }
func (r Room) centerY() float64 { return float64(r.Y) + float64(r.H)/2 }

// MonsterKind selects a dungeon monster's behavior and looks.
type MonsterKind uint8

const (
	MOrc      MonsterKind = iota // melee bruiser: must close to adjacent
	MSkeleton                    // archer: keeps its distance, shoots bone arrows
)

// Monster is a dungeon inhabitant (20 HP). Damage rolls live in the game
// package.
type Monster struct {
	Kind     MonsterKind
	X, Y     float64
	HP       int
	AttackCD float64 // seconds until it may strike/shoot again
	DeadFor  float64 // seconds since death, drives the death frame
	WanderT  float64 // seconds until it picks a new idle direction
	DX, DY   float64 // current wander direction
}

// Name returns the monster's display name for combat messages.
func (m *Monster) Name() string {
	if m.Kind == MSkeleton {
		return "skeleton"
	}
	return "orc"
}

// Chest holds treasure and opens when the player walks up to it. Its gold is
// rolled at generation time so the world knows the total treasure available.
type Chest struct {
	X, Y   float64
	Gold   int
	Opened bool
}

// Column is a free-standing round pillar (blocks movement, rendered as a sprite).
type Column struct{ X, Y float64 }

// Torch is a flame mounted on a wall face, slightly proud of the wall.
type Torch struct{ X, Y float64 }

// Dungeon is one generated level plus its inhabitants.
type Dungeon struct {
	W, H     int
	Cells    []Cell
	Rooms    []Room
	Entry    [2]int // cell of the entry ladder
	Exit     [2]int // cell of the exit ladder
	StartX   float64
	StartY   float64
	StartA   float64 // initial facing angle
	Monsters []Monster
	Chests   []Chest
	Columns  []Column
	Torches  []Torch
}

// At returns the cell at (x, y); out of bounds reads as wall.
func (d *Dungeon) At(x, y int) Cell {
	if x < 0 || y < 0 || x >= d.W || y >= d.H {
		return CWall
	}
	return d.Cells[y*d.W+x]
}

func (d *Dungeon) set(x, y int, c Cell) {
	if x >= 0 && y >= 0 && x < d.W && y < d.H {
		d.Cells[y*d.W+x] = c
	}
}

// OpenDoor converts a closed door cell to an open one.
func (d *Dungeon) OpenDoor(x, y int) {
	if d.At(x, y) == CDoor {
		d.set(x, y, CDoorOpen)
	}
}

// Walkable is the movement test for entities with a collision radius: the
// cell must be passable and not occupied by a column.
func (d *Dungeon) Walkable(x, y float64) bool {
	if !d.At(int(x), int(y)).Passable() {
		return false
	}
	for _, c := range d.Columns {
		dx, dy := x-c.X, y-c.Y
		if dx*dx+dy*dy < 0.45*0.45 {
			return false
		}
	}
	return true
}

// LineOfSight reports whether an unobstructed straight line runs between the
// two points (walls and closed doors block it).
func (d *Dungeon) LineOfSight(x0, y0, x1, y1 float64) bool {
	dist := math.Hypot(x1-x0, y1-y0)
	steps := int(dist/0.2) + 1
	for i := 1; i < steps; i++ {
		t := float64(i) / float64(steps)
		x := x0 + (x1-x0)*t
		y := y0 + (y1-y0)*t
		if !d.At(int(x), int(y)).Passable() {
			return false
		}
	}
	return true
}

const gridSize = 40

// Generate builds a dungeon with 6-12 rooms connected by corridors. The rooms
// are linked in sequence (guaranteeing an entry-to-exit path that crosses
// rooms) plus a couple of extra loop corridors for exploration.
func Generate(seed int64) *Dungeon {
	rng := rand.New(rand.NewSource(seed))
	for {
		d := tryGenerate(rng)
		if d != nil {
			return d
		}
	}
}

func tryGenerate(rng *rand.Rand) *Dungeon {
	d := &Dungeon{W: gridSize, H: gridSize, Cells: make([]Cell, gridSize*gridSize)}

	// Carve non-overlapping room interiors (kept one wall apart).
	want := 6 + rng.Intn(7)
	for tries := 0; tries < 600 && len(d.Rooms) < want; tries++ {
		rw := 4 + rng.Intn(6) // 4..9
		rh := 4 + rng.Intn(4) // 4..7
		rx := 2 + rng.Intn(d.W-rw-4)
		ry := 2 + rng.Intn(d.H-rh-4)
		ok := true
		for _, r := range d.Rooms {
			if rx < r.X+r.W+2 && r.X < rx+rw+2 && ry < r.Y+r.H+2 && r.Y < ry+rh+2 {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		d.Rooms = append(d.Rooms, Room{rx, ry, rw, rh})
	}
	if len(d.Rooms) < 6 {
		return nil
	}
	for _, r := range d.Rooms {
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				d.set(x, y, CFloor)
			}
		}
	}

	// Chain the rooms with L-shaped corridors, then add loops.
	for i := 1; i < len(d.Rooms); i++ {
		d.corridor(rng, d.Rooms[i-1], d.Rooms[i])
	}
	for i := 0; i < 1+rng.Intn(3); i++ {
		a, b := rng.Intn(len(d.Rooms)), rng.Intn(len(d.Rooms))
		if a != b {
			d.corridor(rng, d.Rooms[a], d.Rooms[b])
		}
	}

	d.placeDoors()
	d.placeLadders()
	d.placeColumns()
	d.placeTorches(rng)
	d.populate(rng)
	return d
}

// corridor carves an L-shaped passage between two room centers.
func (d *Dungeon) corridor(rng *rand.Rand, a, b Room) {
	x0, y0 := int(a.centerX()), int(a.centerY())
	x1, y1 := int(b.centerX()), int(b.centerY())
	carveH := func(y, xa, xb int) {
		if xa > xb {
			xa, xb = xb, xa
		}
		for x := xa; x <= xb; x++ {
			if d.At(x, y) == CWall {
				d.set(x, y, CFloor)
			}
		}
	}
	carveV := func(x, ya, yb int) {
		if ya > yb {
			ya, yb = yb, ya
		}
		for y := ya; y <= yb; y++ {
			if d.At(x, y) == CWall {
				d.set(x, y, CFloor)
			}
		}
	}
	if rng.Intn(2) == 0 {
		carveH(y0, x0, x1)
		carveV(x1, y0, y1)
	} else {
		carveV(x0, y0, y1)
		carveH(y1, x0, x1)
	}
}

// inRoom reports whether the cell lies inside any room interior.
func (d *Dungeon) inRoom(x, y int) bool {
	for _, r := range d.Rooms {
		if x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
			return true
		}
	}
	return false
}

// placeDoors turns corridor cells that pierce a room's perimeter into doors:
// floor cells outside every room, adjacent to a room interior, framed by
// walls on the perpendicular axis.
func (d *Dungeon) placeDoors() {
	for y := 1; y < d.H-1; y++ {
		for x := 1; x < d.W-1; x++ {
			if d.At(x, y) != CFloor || d.inRoom(x, y) {
				continue
			}
			touchesRoom := d.inRoom(x-1, y) || d.inRoom(x+1, y) || d.inRoom(x, y-1) || d.inRoom(x, y+1)
			if !touchesRoom {
				continue
			}
			horizGap := d.At(x-1, y).Passable() && d.At(x+1, y).Passable() &&
				d.At(x, y-1) == CWall && d.At(x, y+1) == CWall
			vertGap := d.At(x, y-1).Passable() && d.At(x, y+1).Passable() &&
				d.At(x-1, y) == CWall && d.At(x+1, y) == CWall
			if horizGap || vertGap {
				d.set(x, y, CDoor)
			}
		}
	}
}

// placeLadders puts the entry ladder in room 0 and the exit ladder in the
// room whose center is farthest (walking distance) from the entry.
func (d *Dungeon) placeLadders() {
	entry := d.Rooms[0]
	ex, ey := int(entry.centerX()), int(entry.centerY())
	d.Entry = [2]int{ex, ey}
	d.set(ex, ey, CEntry)

	dist := d.bfs(ex, ey)
	best, bestD := 0, -1
	for i := 1; i < len(d.Rooms); i++ {
		cx, cy := int(d.Rooms[i].centerX()), int(d.Rooms[i].centerY())
		if dd := dist[cy*d.W+cx]; dd > bestD {
			best, bestD = i, dd
		}
	}
	far := d.Rooms[best]
	fx, fy := int(far.centerX()), int(far.centerY())
	d.Exit = [2]int{fx, fy}
	d.set(fx, fy, CExit)

	// Start the player one cell south of the entry ladder, facing away from it.
	d.StartX = float64(ex) + 0.5
	d.StartY = float64(ey) + 1.5
	if !d.At(ex, ey+1).Passable() {
		d.StartY = float64(ey) - 0.5
	}
	next := d.Rooms[1]
	d.StartA = math.Atan2(next.centerY()-d.StartY, next.centerX()-d.StartX)
}

// bfs returns walking distances from (sx, sy) over passable cells (doors count
// as passable: the player can open them). Unreached cells are -1.
func (d *Dungeon) bfs(sx, sy int) []int {
	dist := make([]int, d.W*d.H)
	for i := range dist {
		dist[i] = -1
	}
	pass := func(c Cell) bool { return c.Passable() || c == CDoor }
	if !pass(d.At(sx, sy)) {
		return dist
	}
	dist[sy*d.W+sx] = 0
	queue := [][2]int{{sx, sy}}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, dd := range [4][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
			nx, ny := c[0]+dd[0], c[1]+dd[1]
			if nx < 0 || ny < 0 || nx >= d.W || ny >= d.H {
				continue
			}
			i := ny*d.W + nx
			if dist[i] == -1 && pass(d.Cells[i]) {
				dist[i] = dist[c[1]*d.W+c[0]] + 1
				queue = append(queue, [2]int{nx, ny})
			}
		}
	}
	return dist
}

// PathExists reports whether the entry ladder can reach the exit ladder.
func (d *Dungeon) PathExists() bool {
	dist := d.bfs(d.Entry[0], d.Entry[1])
	return dist[d.Exit[1]*d.W+d.Exit[0]] >= 0
}

// placeColumns drops round pillars into the larger rooms, away from ladders.
func (d *Dungeon) placeColumns() {
	for _, r := range d.Rooms {
		if r.W < 6 || r.H < 5 {
			continue
		}
		// Symmetric pair a third of the way in from each side.
		for _, fx := range []float64{0.28, 0.72} {
			cx := float64(r.X) + fx*float64(r.W)
			cy := float64(r.Y) + 0.5*float64(r.H)
			ix, iy := int(cx), int(cy)
			if d.At(ix, iy) == CFloor {
				d.Columns = append(d.Columns, Column{math.Floor(cx) + 0.5, math.Floor(cy) + 0.5})
			}
		}
	}
}

// placeTorches mounts 1-3 torches per room on wall faces that look into the
// room, nudged slightly off the wall so the flame sprite reads as mounted.
func (d *Dungeon) placeTorches(rng *rand.Rand) {
	for _, r := range d.Rooms {
		n := 1 + rng.Intn(3)
		for i := 0; i < n; i++ {
			for try := 0; try < 12; try++ {
				// Pick a random interior edge cell and the wall beyond it.
				side := rng.Intn(4)
				var wx, wy int     // wall cell
				var tx, ty float64 // torch sprite position
				switch side {
				case 0: // north wall
					wx, wy = r.X+rng.Intn(r.W), r.Y-1
					tx, ty = float64(wx)+0.5, float64(wy)+1.12
				case 1: // south wall
					wx, wy = r.X+rng.Intn(r.W), r.Y+r.H
					tx, ty = float64(wx)+0.5, float64(wy)-0.12
				case 2: // west wall
					wx, wy = r.X-1, r.Y+rng.Intn(r.H)
					tx, ty = float64(wx)+1.12, float64(wy)+0.5
				default: // east wall
					wx, wy = r.X+r.W, r.Y+rng.Intn(r.H)
					tx, ty = float64(wx)-0.12, float64(wy)+0.5
				}
				if d.At(wx, wy) != CWall {
					continue
				}
				d.Torches = append(d.Torches, Torch{tx, ty})
				break
			}
		}
	}
}

// populate scatters orcs and skeleton archers (skipping the entry room) and
// roughly one chest per two rooms.
func (d *Dungeon) populate(rng *rand.Rand) {
	spawn := func(r Room, kind MonsterKind) {
		x := float64(r.X) + 0.5 + float64(rng.Intn(r.W))
		y := float64(r.Y) + 0.5 + float64(rng.Intn(r.H))
		if d.Walkable(x, y) {
			d.Monsters = append(d.Monsters, Monster{Kind: kind, X: x, Y: y, HP: 20})
		}
	}
	for i, r := range d.Rooms {
		if i == 0 {
			continue
		}
		for n := 1 + rng.Intn(3); n > 0; n-- {
			spawn(r, MOrc)
		}
		// At least one skeleton per dungeon (room 1), a coin flip elsewhere.
		if i == 1 || rng.Intn(2) == 0 {
			spawn(r, MSkeleton)
		}
	}
	for i := 1; i < len(d.Rooms); i += 2 {
		r := d.Rooms[i]
		for try := 0; try < 10; try++ {
			x := float64(r.X) + 0.5 + float64(rng.Intn(r.W))
			y := float64(r.Y) + 0.5 + float64(rng.Intn(r.H))
			ix, iy := int(x), int(y)
			if d.At(ix, iy) == CFloor && d.Walkable(x, y) {
				d.Chests = append(d.Chests, Chest{X: x, Y: y, Gold: 10 + rng.Intn(41)})
				break
			}
		}
	}
}
