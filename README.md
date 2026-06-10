# DungeonGame

A terminal adventure game in Go with two modes:

- **World mode** — an Ultima-style top-down scroller over a generated 256×256
  island continent ringed by open sea: animated water and lava, forests,
  mountains, and a town whose walls carry inset labels (PUB, KEEP, INN) and
  warning signs.
- **Dungeon mode** — a first-person raycaster. 3–5 dungeons hide in the
  forests and mountains, each with 6–12 rooms joined by corridors, doors that
  creak open, columns, wall torches, treasure chests, prowling orcs, and entry
  and exit ladders back to the surface.

## Screenshots

Exploring the island continent — animated coastline, forests, and the quest
tally in the stats box:

![Overworld scroller](screen1.png)

Deep in a dungeon, an orc closes in by torchlight:

![Raycast dungeon mode](screen2.png)

It does not always end well:

![You died](screen3.png)

## Running

```
go run .
```

Optional: `go run . -seed 12345` for a reproducible world.

On startup the game asks the terminal to resize to **256×64** cells (works in
Windows Terminal and most modern emulators). For the best picture, set your
terminal profile to a small fixed-width font — e.g. Cascadia Mono at 8–10 pt —
so the 256-column window fits your screen.

## Controls

| Key   | World mode            | Dungeon mode            |
|-------|-----------------------|-------------------------|
| ENTER | start (title screen)  |                         |
| W / S | walk north / south    | walk forward / back     |
| A / D | walk west / east      | turn left / right       |
| SPACE | (fizzles)             | fire magic missile      |
| R     | restart after death/victory | restart after death/victory |
| ESC   | quit                  | quit                    |

## Rules of the realm

- You start with **100 HP** and heal **1 HP every 20 seconds**, always.
- Your magic missile deals **1–10** damage; orcs have **20 HP**.
- Orcs must get adjacent to strike, and claw you for **1–4**.
- Lava burns for **10 HP per tile** — heed the warning signs.
- Walk up to a treasure chest to open it: gold, and sometimes a healing
  draught.
- Step onto either ladder in a dungeon to climb out; you emerge just outside
  the entrance you used.
- **Winning:** claim **70% of all the gold** hidden in the world's chests, or
  **slay every orc**. The bordered stats box (top-left) tracks your HP, gold
  against the goal, and orcs slain; victory earns you a sunrise.

## Development

```
go test ./...     # generation invariants, combat bounds, regen timing
go vet ./...
```
