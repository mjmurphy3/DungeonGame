package game

import (
	"fmt"
	"math"

	"dungeongame/internal/dungeon"
	"dungeongame/internal/raycast"
	"dungeongame/internal/render"
)

const (
	moveStep    = 0.22 // units per W/S keypress
	strafeStep  = 0.18 // units per Q/E keypress
	turnStep    = 0.12 // radians per A/D keypress
	playerPad   = 0.25 // collision radius against walls
	missileVel  = 8.0  // units per second
	orcChase    = 1.5  // orc chase speed, units per second
	orcWander   = 0.5
	orcSight    = 7.0
	orcReach    = 1.2 // melee range: they must close in to land a hit
	orcCooldown = 1.2 // seconds between orc strikes
	orcMeleeMax = 4   // orcs claw for 1..orcMeleeMax damage

	skelRange    = 8.0 // skeletons shoot when they can see you this close
	skelCooldown = 2.2 // seconds between arrows
	skelShotVel  = 5.0 // arrow speed, units per second
	skelShotMax  = 10  // arrows hit for 1..skelShotMax damage
	skelBackOff  = 3.0 // skeletons retreat if you get this close

	chestHeal = 5 // hit points restored by every opened chest
)

// Shot is an enemy projectile (a skeleton's bone arrow) in flight.
type Shot struct {
	X, Y, DX, DY float64
}

func (g *Game) dungeonKey(ch rune) {
	d := g.dungeons[g.curDungeon]
	sin, cos := math.Sin(g.ang), math.Cos(g.ang)
	switch ch {
	case 'w':
		g.tryMove(d, cos*moveStep, sin*moveStep)
	case 's':
		g.tryMove(d, -cos*moveStep*0.7, -sin*moveStep*0.7)
	case 'q': // strafe left
		g.tryMove(d, sin*strafeStep, -cos*strafeStep)
	case 'e': // strafe right
		g.tryMove(d, -sin*strafeStep, cos*strafeStep)
	case 'a':
		g.ang -= turnStep
	case 'd':
		g.ang += turnStep
	case ' ':
		if len(g.missiles) < 6 {
			g.missiles = append(g.missiles, Missile{
				X: g.fx + cos*0.4, Y: g.fy + sin*0.4,
				DX: cos * missileVel, DY: sin * missileVel,
			})
		}
	}
}

// tryMove slides the player axis by axis; bumping a closed door opens it.
func (g *Game) tryMove(d *dungeon.Dungeon, dx, dy float64) {
	if dx != 0 {
		nx := g.fx + dx
		lead := nx + math.Copysign(playerPad, dx)
		if d.Walkable(lead, g.fy) {
			g.fx = nx
		} else if d.At(int(lead), int(g.fy)) == dungeon.CDoor {
			d.OpenDoor(int(lead), int(g.fy))
			g.say("The door creaks open.")
		}
	}
	if dy != 0 {
		ny := g.fy + dy
		lead := ny + math.Copysign(playerPad, dy)
		if d.Walkable(g.fx, lead) {
			g.fy = ny
		} else if d.At(int(g.fx), int(lead)) == dungeon.CDoor {
			d.OpenDoor(int(g.fx), int(lead))
			g.say("The door creaks open.")
		}
	}
	// Stepping onto either ladder climbs back to the surface.
	switch d.At(int(g.fx), int(g.fy)) {
	case dungeon.CEntry, dungeon.CExit:
		g.leaveDungeon()
	}
}

func (g *Game) updateDungeon() {
	d := g.dungeons[g.curDungeon]
	g.updateMissiles(d)
	g.updateMonsters(d)
	g.updateShots(d)
	g.updateChests(d)
}

// maybeWarn flashes the "look around!" indicator when the player takes a hit
// from an attacker outside their field of view (behind or clipped offscreen).
func (g *Game) maybeWarn(ax, ay float64) {
	rel := math.Atan2(ay-g.fy, ax-g.fx) - g.ang
	for rel > math.Pi {
		rel -= 2 * math.Pi
	}
	for rel < -math.Pi {
		rel += 2 * math.Pi
	}
	// Half-FOV is atan(0.66) ~= 0.58 rad; warn a touch inside that so
	// attackers clipped at the screen edge count too.
	if math.Abs(rel) > 0.47 {
		g.warnT = 2.5
	}
}

func (g *Game) updateMissiles(d *dungeon.Dungeon) {
	alive := g.missiles[:0]
	for _, m := range g.missiles {
		dead := false
		for step := 0; step < 3 && !dead; step++ {
			m.X += m.DX * dt / 3
			m.Y += m.DY * dt / 3
			if !d.At(int(m.X), int(m.Y)).Passable() {
				dead = true // splashes on a wall or closed door
				break
			}
			for i := range d.Monsters {
				o := &d.Monsters[i]
				if o.HP <= 0 {
					continue
				}
				dx, dy := m.X-o.X, m.Y-o.Y
				if dx*dx+dy*dy < 0.45*0.45 {
					dmg := 1 + g.rng.Intn(10)
					o.HP -= dmg
					if o.HP <= 0 {
						g.foesKilled++
						g.say(fmt.Sprintf("Your missile blasts the %s for %d - it falls!", o.Name(), dmg))
						g.checkVictory()
					} else {
						g.say(fmt.Sprintf("Your missile strikes the %s for %d!", o.Name(), dmg))
					}
					dead = true
					break
				}
			}
		}
		if !dead {
			alive = append(alive, m)
		}
	}
	g.missiles = alive
}

func (g *Game) updateMonsters(d *dungeon.Dungeon) {
	for i := range d.Monsters {
		o := &d.Monsters[i]
		if o.HP <= 0 {
			o.DeadFor += dt
			continue
		}
		if o.AttackCD > 0 {
			o.AttackCD -= dt
		}
		if o.Kind == dungeon.MSkeleton {
			g.updateSkeleton(d, o)
			continue
		}
		dx, dy := g.fx-o.X, g.fy-o.Y
		dist := math.Hypot(dx, dy)
		switch {
		case dist < orcReach:
			if o.AttackCD <= 0 {
				o.AttackCD = orcCooldown
				dmg := 1 + g.rng.Intn(orcMeleeMax)
				g.damage(dmg, "An orc strikes you down.")
				g.maybeWarn(o.X, o.Y)
				if g.mode != ModeDead {
					g.say(fmt.Sprintf("The orc claws you for %d!", dmg))
				}
			}
		case dist < orcSight && d.LineOfSight(o.X, o.Y, g.fx, g.fy):
			g.moveMonster(d, o, dx/dist*orcChase*dt, dy/dist*orcChase*dt)
		default:
			g.wanderMonster(d, o, orcWander)
		}
	}
}

// updateSkeleton runs the archer brain: keep range, loose arrows on sight.
func (g *Game) updateSkeleton(d *dungeon.Dungeon, o *dungeon.Monster) {
	dx, dy := g.fx-o.X, g.fy-o.Y
	dist := math.Hypot(dx, dy)
	sees := dist < skelRange && d.LineOfSight(o.X, o.Y, g.fx, g.fy)

	if sees && o.AttackCD <= 0 {
		o.AttackCD = skelCooldown
		g.shots = append(g.shots, Shot{
			X: o.X + dx/dist*0.4, Y: o.Y + dy/dist*0.4,
			DX: dx / dist * skelShotVel, DY: dy / dist * skelShotVel,
		})
	}
	switch {
	case sees && dist < skelBackOff: // too close: back away while shooting
		g.moveMonster(d, o, -dx/dist*orcChase*dt, -dy/dist*orcChase*dt)
	case sees:
		// Hold position at range; sidle a little so it isn't a statue.
		g.wanderMonster(d, o, 0.25)
	default:
		g.wanderMonster(d, o, orcWander)
	}
}

// updateShots advances enemy arrows; each hit costs the player a little.
func (g *Game) updateShots(d *dungeon.Dungeon) {
	alive := g.shots[:0]
	for _, s := range g.shots {
		dead := false
		for step := 0; step < 2 && !dead; step++ {
			ox, oy := s.X, s.Y
			s.X += s.DX * dt / 2
			s.Y += s.DY * dt / 2
			if !d.At(int(s.X), int(s.Y)).Passable() {
				dead = true
				break
			}
			dx, dy := s.X-g.fx, s.Y-g.fy
			if dx*dx+dy*dy < 0.5*0.5 {
				dmg := 1 + g.rng.Intn(skelShotMax)
				g.damage(dmg, "A bone arrow finds your heart.")
				g.maybeWarn(ox, oy)
				if g.mode != ModeDead {
					g.say(fmt.Sprintf("A bone arrow pierces you for %d!", dmg))
				}
				dead = true
			}
		}
		if !dead {
			alive = append(alive, s)
		}
	}
	g.shots = alive
}

// moveMonster slides a monster, refusing to enter walls or crowd the player.
func (g *Game) moveMonster(d *dungeon.Dungeon, o *dungeon.Monster, dx, dy float64) {
	nx, ny := o.X+dx, o.Y+dy
	pdx, pdy := nx-g.fx, ny-g.fy
	if pdx*pdx+pdy*pdy < 0.6*0.6 {
		return // close enough; melee handles the rest
	}
	if d.Walkable(nx, o.Y) {
		o.X = nx
	}
	if d.Walkable(o.X, ny) {
		o.Y = ny
	}
}

// wanderMonster drifts a monster in its current idle direction, re-rolling
// the heading every couple of seconds.
func (g *Game) wanderMonster(d *dungeon.Dungeon, o *dungeon.Monster, speed float64) {
	o.WanderT -= dt
	if o.WanderT <= 0 {
		o.WanderT = 1 + g.rng.Float64()*2
		a := g.rng.Float64() * 2 * math.Pi
		o.DX, o.DY = math.Cos(a), math.Sin(a)
	}
	g.moveMonster(d, o, o.DX*speed*dt, o.DY*speed*dt)
}

func (g *Game) updateChests(d *dungeon.Dungeon) {
	for i := range d.Chests {
		c := &d.Chests[i]
		if c.Opened {
			continue
		}
		dx, dy := g.fx-c.X, g.fy-c.Y
		if dx*dx+dy*dy < 0.9*0.9 {
			c.Opened = true
			g.gold += c.Gold
			heal := min(chestHeal, maxHP-g.hp)
			g.hp += heal
			if heal > 0 {
				g.say(fmt.Sprintf("The chest holds %d gold and a restorative draught (+%d HP)!", c.Gold, heal))
			} else {
				g.say(fmt.Sprintf("The chest holds %d gold!", c.Gold))
			}
			g.checkVictory()
		}
	}
}

// drawDungeon renders the raycast view into the pixel buffer and blits it.
func (g *Game) drawDungeon(w, h int) {
	d := g.dungeons[g.curDungeon]
	if g.pb == nil {
		g.pb = render.NewPixelBuf(w, h)
	} else {
		g.pb.Resize(w, h)
	}

	grid := func(x, y int) raycast.WallKind {
		switch d.At(x, y) {
		case dungeon.CWall:
			return raycast.WallStone
		case dungeon.CDoor:
			return raycast.WallDoor
		}
		return raycast.WallNone
	}

	sprites := make([]raycast.Sprite, 0,
		len(d.Monsters)+len(d.Chests)+len(d.Columns)+len(d.Torches)+len(g.missiles)+len(g.shots)+2)
	sprites = append(sprites,
		raycast.Sprite{X: float64(d.Entry[0]) + 0.5, Y: float64(d.Entry[1]) + 0.5, Kind: raycast.KLadder, Scale: 0.95},
		raycast.Sprite{X: float64(d.Exit[0]) + 0.5, Y: float64(d.Exit[1]) + 0.5, Kind: raycast.KLadder, Scale: 0.95},
	)
	for _, c := range d.Columns {
		sprites = append(sprites, raycast.Sprite{X: c.X, Y: c.Y, Kind: raycast.KColumn, Scale: 1.0})
	}
	for i, t := range d.Torches {
		sprites = append(sprites, raycast.Sprite{
			X: t.X, Y: t.Y, Kind: raycast.KTorch, Frame: g.tick/5 + i, Scale: 0.38, Lift: 0.40,
		})
	}
	for _, c := range d.Chests {
		k := raycast.KChest
		if c.Opened {
			k = raycast.KChestOpen
		}
		sprites = append(sprites, raycast.Sprite{X: c.X, Y: c.Y, Kind: k, Scale: 0.5})
	}
	for i, o := range d.Monsters {
		var k raycast.SpriteKind
		scale := 0.62
		switch {
		case o.Kind == dungeon.MSkeleton && o.HP <= 0:
			k, scale = raycast.KSkeletonDead, 0.5
		case o.Kind == dungeon.MSkeleton:
			k, scale = raycast.KSkeleton, 0.64
		case o.HP <= 0:
			k, scale = raycast.KOrcDead, 0.55
		default:
			k = raycast.KOrc
		}
		sprites = append(sprites, raycast.Sprite{X: o.X, Y: o.Y, Kind: k, Frame: g.tick/7 + i, Scale: scale})
	}
	for _, m := range g.missiles {
		sprites = append(sprites, raycast.Sprite{X: m.X, Y: m.Y, Kind: raycast.KMissile, Scale: 0.14, Lift: 0.42})
	}
	for _, s := range g.shots {
		sprites = append(sprites, raycast.Sprite{X: s.X, Y: s.Y, Kind: raycast.KArrow, Scale: 0.12, Lift: 0.44})
	}

	g.rc.Render(g.pb, grid, g.fx, g.fy, g.ang, sprites, g.tick)
	g.pb.Blit(g.scr, 0, 0)
}
