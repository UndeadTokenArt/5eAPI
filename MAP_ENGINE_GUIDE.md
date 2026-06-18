# Map Engine Guide

This document explains how the map generator works, how Wave Function Collapse (WFC) is implemented in this project, and how to expand the tile system with better connectivity.

## What Lives Where

- `static/js/map-engine.js`: JavaScript engine for generation, propagation, SVG loading, and rendering.
- `static/js/map-tiles.js`: Tile catalog (`TILES`) and lookups (`TILE_BY_ID`, groups). This is where tile rules are defined.
- `map.go`: HTML page, UI controls, constants (`GRID`, `TILE`), map generation entrypoint (`generate()`), themes, and initialization.
- `tiles/<biome>/<asset>.svg`: Visual assets referenced by each tile's `asset` field.

## High-Level Flow

1. UI loads and builds biome/type buttons and tile SVG `<defs>`.
2. User clicks Generate.
3. `generate()` calls `wfcGenerate(currentMapType)`.
4. WFC builds a `GRID x GRID` candidate grid and collapses it with constraints.
5. If generation succeeds, `renderGrid(grid)` paints a `<use>` for each chosen tile.
6. If a contradiction happens, WFC retries (up to 8 attempts).

## Data Model

Each tile in `TILES` has this shape:

```js
{
  id: 'd-floor',
  type: 'dungeon',
  edges: { N: 'open', E: 'open', S: 'open', W: 'open' },
  weight: 10,
  asset: 'd-floor'
}
```

Meaning of each field:

- `id`: Unique tile identifier.
- `type`: Biome group (for example `dungeon`, `outdoor`). WFC only uses tiles with matching type.
- `edges`: Connection tokens per side. Neighboring tiles must match compatible tokens across touching edges.
- `weight`: Relative probability when a cell collapses.
- `asset`: SVG file name at `/tiles/<type>/<asset>.svg`.

## WFC Implementation in This Project

`wfcGenerate(mapType)` in `static/js/map-engine.js` does the full solve.

### 1) Initialize candidate grid

`initGrid()` builds a 2D array where each cell starts as a `Set` of all tile IDs for that biome.

Conceptually:

- Before collapse, each cell = many possible tiles.
- After collapse, each cell = exactly one tile ID.

### 2) Pick next cell by entropy

`minEntropyCell(grid)` finds the not-yet-collapsed cell with the fewest options (`size > 1` and minimum size).

Why: this usually reduces contradictions and search width by resolving the most constrained location first.

### 3) Collapse with weighted randomness

`weightedPick(ids)` chooses one candidate by `weight`.

If a tile has weight 10 and another has weight 2, the first is roughly 5x as likely when both are allowed.

### 4) Propagate constraints (the core)

`propagate(grid, startX, startY)` uses a queue:

1. Pop a changed cell.
2. For each direction `N/E/S/W`, inspect neighbor.
3. Compute `allowedEdges`: all edge tokens the current cell can expose toward that direction.
4. Remove any neighbor candidate whose opposite edge token is not in `allowedEdges`.
5. If neighbor changed, push neighbor into queue.
6. If any neighbor candidate set becomes empty, contradiction -> fail attempt.

Direction helpers:

- `DIRS = ['N','E','S','W']`
- `OFFSET` gives `(dx,dy)` per direction.
- `OPP` maps each direction to its opposite (`N <-> S`, `E <-> W`).

### 5) Retry on contradiction

`attempt()` runs one full solve. If contradiction occurs, it returns `null`.
`wfcGenerate` retries up to 8 times before giving up.

## Rendering Model

- `buildTileDefs()` loads all SVG assets and stores each as `<g id="tile-...">...</g>` in `<defs>`.
- `renderGrid(grid)` places `<use href="#tile-id" transform="translate(x,y)"/>` for each cell.
- `getTileSVG()` caches SVG content in `TILE_SVG_CACHE` by `<type>/<asset>`.

This makes rendering fast after initial asset load.

## How Connections Work

Connection logic is token matching across touching edges.

If tile A is left of tile B:

- A.E must match B.W.

In this implementation, matching is equality on token strings (for example `road` matches `road`, `wall` matches `wall`).

Important implication:

- Connectivity quality is entirely determined by your `edges` vocabulary and consistency.

## Expanding the Tile Set Safely

Use this checklist whenever you add tiles.

1. Add SVG file in `tiles/<type>/`.
2. Add tile entry in `TILES` with unique `id`, valid `type`, `edges`, `weight`, and `asset`.
3. Ensure all four edge tokens are intentional and consistent with existing tokens.
4. Keep rotations explicit (separate tile records) unless you implement auto-rotation.
5. Generate repeatedly to confirm no frequent contradictions or dead maps.

## Designing Better Connections

### Use a stronger edge token vocabulary

Instead of only broad tokens like `open`/`wall`, create more specific semantics:

- `corridor`
- `room`
- `door`
- `river`
- `shore`
- `cliff`
- `forest-dense`
- `forest-light`

More specific tokens reduce invalid blends and improve map readability.

### Add transition tiles

Most connection issues come from missing bridges between terrains.

Examples:

- `water <-> grass`: shoreline edges
- `road <-> grass`: road shoulder tiles
- `room <-> corridor`: doorway tiles
- `forest <-> mountain`: foothill transition tiles

If two categories should touch, provide one or more transition tiles that explicitly allow it.

### Balance weights intentionally

- High weight: filler/base tiles.
- Medium weight: common connectors.
- Low weight: distinctive landmarks/junctions.

If connectors are too rare, paths break visually.
If connectors are too common, maps look noisy and over-junctioned.

### Keep rotational families complete

For any directional tile shape, include all needed orientations:

- Straight: horizontal + vertical
- Corner: NE, SE, SW, NW
- T-junction: N, E, S, W

Missing orientations can create directional bias and increase contradictions.

## Common Failure Modes and Fixes

- Symptom: Frequent "Generation failed after 8 attempts".
  - Cause: Overly strict or incompatible edge tokens.
  - Fix: Add transition tiles, loosen token constraints, or raise connector weights.

- Symptom: Maps generate but look disconnected.
  - Cause: Too many isolated tile classes or insufficient connector frequency.
  - Fix: Increase connector weights and add bridging variants.

- Symptom: One tile appears everywhere.
  - Cause: Weight imbalance.
  - Fix: Rebalance weights and add competing alternatives.

## Practical Example: Adding a New Outdoor Road-Bridge Tile

1. Create asset: `tiles/outdoor/o-road-bridge-h.svg`.
2. Add tile in `TILES`:

```js
{ id:'o-road-bridge-h', type:'outdoor', edges:{N:'water',E:'road',S:'water',W:'road'}, weight:1, asset:'o-road-bridge-h' }
```

3. Add complementary vertical variant:

```js
{ id:'o-road-bridge-v', type:'outdoor', edges:{N:'road',E:'water',S:'road',W:'water'}, weight:1, asset:'o-road-bridge-v' }
```

4. Ensure neighboring water/road tiles exist with matching tokens.
5. Generate repeatedly and tune weights.

## Suggested Next Improvements (Optional)

If you want richer generation later, consider:

- Boundary constraints: force edge cells to specific token sets (for example no open corridor leaving map border).
- Seeded randomness: deterministic generation from a numeric seed.
- Biome-specific rule packs: custom token vocabularies per biome.
- Multi-pass generation: WFC for layout, then decorate with detail tiles.
- Automatic rotations: define one base tile + generated rotated variants.

## Quick Rule of Thumb

If you want "more tiles finding connections," focus on three levers:

1. Better edge token design.
2. More transition tiles.
3. Better weight balancing for connectors.

Those three changes usually improve output more than algorithm changes.