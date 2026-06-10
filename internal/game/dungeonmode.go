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
	turnStep    = 0.12 // radians per A/D keypress
	playerPad   = 0.25 // collision radius against walls
	missileVel  = 8.0  // units per second
	orcChase    = 1.5  // orc chase speed, units per second
	orcWander   = 0.5
	orcSight    = 7.0
	orcReach    = 1.2 // melee range: they must close in to land a hit
	orcCooldown = 1.2 // seconds between orc strikes
	orcMeleeMax = 4   // orcs claw for 1..orcMeleeMax damage
)

func (g *Game) dungeonKey(ch rune) {
	d := g.dungeons[g.curDungeon]
	switch ch {
	case 'w':
		g.tryMove(d, math.Cos(g.ang)*moveStep, math.Sin(g.ang)*moveStep)
	case 's':
		g.tryMove(d, -math.Cos(g.ang)*moveStep*0.7, -math.Sin(g.ang)*moveStep*0.7)
	case 'a':
		g.ang -= turnStep
	case 'd':
		g.ang += turnStep
	case ' ':
		if len(g.missiles) < 6 {
			dx, dy := math.Cos(g.ang), math.Sin(g.ang)
			g.missiles = append(g.missiles, Missile{
				X: g.fx + dx*0.4, Y: g.fy + dy*0.4,
				DX: dx * missileVel, DY: dy * missileVel,
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
	g.updateOrcs(d)
	g.updateChests(d)
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
			for i := range d.Orcs {
				o := &d.Orcs[i]
				if o.HP <= 0 {
					continue
				}
				dx, dy := m.X-o.X, m.Y-o.Y
				if dx*dx+dy*dy < 0.45*0.45 {
					dmg := 1 + g.rng.Intn(10)
					o.HP -= dmg
					if o.HP <= 0 {
						g.orcsKilled++
						g.say(fmt.Sprintf("Your missile blasts the orc for %d - it falls!", dmg))
						g.checkVictory()
					} else {
						g.say(fmt.Sprintf("Your missile strikes the orc for %d!", dmg))
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

func (g *Game) updateOrcs(d *dungeon.Dungeon) {
	for i := range d.Orcs {
		o := &d.Orcs[i]
		if o.HP <= 0 {
			o.DeadFor += dt
			continue
		}
		if o.AttackCD > 0 {
			o.AttackCD -= dt
		}
		dx, dy := g.fx-o.X, g.fy-o.Y
		dist := math.Hypot(dx, dy)
		switch {
		case dist < orcReach:
			if o.AttackCD <= 0 {
				o.AttackCD = orcCooldown
				dmg := 1 + g.rng.Intn(orcMeleeMax)
				g.damage(dmg, "An orc strikes you down.")
				if g.mode != ModeDead {
					g.say(fmt.Sprintf("The orc claws you for %d!", dmg))
				}
			}
		case dist < orcSight && d.LineOfSight(o.X, o.Y, g.fx, g.fy):
			g.moveOrc(d, o, dx/dist*orcChase*dt, dy/dist*orcChase*dt)
		default:
			o.WanderT -= dt
			if o.WanderT <= 0 {
				o.WanderT = 1 + g.rng.Float64()*2
				a := g.rng.Float64() * 2 * math.Pi
				o.DX, o.DY = math.Cos(a), math.Sin(a)
			}
			g.moveOrc(d, o, o.DX*orcWander*dt, o.DY*orcWander*dt)
		}
	}
}

// moveOrc slides an orc, refusing to enter walls or crowd the player.
func (g *Game) moveOrc(d *dungeon.Dungeon, o *dungeon.Orc, dx, dy float64) {
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
			if g.rng.Intn(4) == 0 && g.hp < maxHP {
				g.hp = min(g.hp+10, maxHP)
				g.say(fmt.Sprintf("The chest holds %d gold and a healing draught (+10 HP)!", c.Gold))
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

	sprites := make([]raycast.Sprite, 0, len(d.Orcs)+len(d.Chests)+len(d.Columns)+len(d.Torches)+len(g.missiles)+2)
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
	for i, o := range d.Orcs {
		if o.HP <= 0 {
			sprites = append(sprites, raycast.Sprite{X: o.X, Y: o.Y, Kind: raycast.KOrcDead, Scale: 0.55})
		} else {
			sprites = append(sprites, raycast.Sprite{X: o.X, Y: o.Y, Kind: raycast.KOrc, Frame: g.tick/7 + i, Scale: 0.62})
		}
	}
	for _, m := range g.missiles {
		sprites = append(sprites, raycast.Sprite{X: m.X, Y: m.Y, Kind: raycast.KMissile, Scale: 0.14, Lift: 0.42})
	}

	g.rc.Render(g.pb, grid, g.fx, g.fy, g.ang, sprites, g.tick)
	g.pb.Blit(g.scr, 0, 0)
}
